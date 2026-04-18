package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
)

type FragmentGenerationService struct {
	fragmentGenRepo *repository.FragmentGenerationRepository
	fragmentRepo    *repository.FragmentRepository
	aiService       *AIService
	logger          *zap.Logger
}

func NewFragmentGenerationService(
	fragmentGenRepo *repository.FragmentGenerationRepository,
	fragmentRepo *repository.FragmentRepository,
	aiService *AIService,
	logger *zap.Logger,
) *FragmentGenerationService {
	return &FragmentGenerationService{
		fragmentGenRepo: fragmentGenRepo,
		fragmentRepo:    fragmentRepo,
		aiService:       aiService,
		logger:          logger,
	}
}

// GenerateFragment 创建生成任务并立即落库占位草稿（与多格参考图流程对齐），返回 task 与 draftFragmentId。
func (s *FragmentGenerationService) GenerateFragment(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*domain.FragmentGenerationTask, string, error) {
	taskID := uuid.New().String()
	task := &domain.FragmentGenerationTask{
		ID:        taskID,
		UserID:    userID,
		Status:    "pending",
		Request:   req,
		Progress:  0,
		CreatedAt: currentTime(),
	}

	if err := s.fragmentGenRepo.Create(ctx, task); err != nil {
		return nil, "", fmt.Errorf("failed to create generation task: %w", err)
	}

	nowMs := time.Now().UnixMilli()
	draft := &domain.Fragment{
		BaseModel: common.BaseModel{
			ID:        uuid.New().String(),
			CreatedAt: nowMs,
			UpdatedAt: nowMs,
		},
		UserID:          userID,
		CreatorID:       userID,
		Content:         "生成中…",
		MediaURLs:       []string{},
		ImageUrls:       "[]",
		Visibility:      domain.FragmentVisibilityPrivate,
		IsDraft:         true,
		SourceType:      string(domain.FragmentSourceAIGeneration),
		SourceID:        taskID,
		EngagementStats: common.EngagementStats{},
	}
	if err := s.fragmentRepo.Create(ctx, draft); err != nil {
		if delErr := s.fragmentGenRepo.Delete(ctx, taskID); delErr != nil {
			s.logger.Warn("rollback: failed to delete generation task after draft create error",
				zap.String("task_id", taskID), zap.Error(delErr))
		}
		return nil, "", fmt.Errorf("failed to create placeholder draft: %w", err)
	}

	s.logger.Info("Fragment generation task created",
		zap.String("task_id", taskID),
		zap.String("draft_id", draft.ID),
		zap.String("user_id", userID),
		zap.Int("image_count", req.ImageCount))

	go s.processFragmentGeneration(context.Background(), taskID)

	return task, draft.ID, nil
}

const (
	fragmentMaxAnchorCharacters     = 3
	fragmentMaxAnchorProps          = 3
	fragmentMaxAnchorLocations      = 2
	fragmentMaxSceneReferenceImages = 6
)

// processFragmentGeneration 处理碎片生成流程（方案 B）：
// Step 1: 元素提取 + 文案 + VisualBible
// Step 2: 场景扩展（含 referenceKeys）
// Step 3: 锚点图
// Step 4: 逐场景出图（多参考图）
// Step 5: 一致性检查（仅记录）
func (s *FragmentGenerationService) processFragmentGeneration(ctx context.Context, taskID string) {
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 5, "starting")

	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		return
	}

	// ── Step 1: 元素提取 + 文案 + VisualBible ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 10, "extracting_elements")

	elemResult, err := s.extractElementsAndGenerateContent(ctx, task.UserID, task.Request)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Element extraction failed: %v", err))
		return
	}

	totalTokens := elemResult.TokensUsed

	resolvedAR := domain.NormalizeFragmentAspectRatio(task.Request.AspectRatio)
	if resolvedAR == "" {
		resolvedAR = domain.NormalizeFragmentAspectRatio(elemResult.AspectRatio)
	}
	if resolvedAR == "" {
		resolvedAR = domain.FragmentAspectDefault
	}

	// ── Step 2: 场景扩展 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 28, "expanding_scenes")

	sceneCount := task.Request.ImageCount
	if sceneCount <= 0 {
		sceneCount = 1
	}
	scenesResult, err := s.expandScenes(ctx, task.UserID, task.Request, elemResult, sceneCount, resolvedAR)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Scene expansion failed: %v", err))
		return
	}
	totalTokens += scenesResult.TokensUsed

	// ── Step 3: 锚点图 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 45, "generating_anchor_images")
	anchorMap, anchorRecords, anchorTok := s.generateAnchorImages(ctx, task.UserID, taskID, task.Request, elemResult.VisualBible, resolvedAR)
	totalTokens += anchorTok

	// ── Step 4: 逐场景生成图片 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 62, "generating_images")
	userRefs := fragmentPrefillHTTPImageURLs(task.Request.ImageUrls, 2)
	imageResult, err := s.generateImagesFromScenes(ctx, task.UserID, taskID, resolvedAR, scenesResult.Scenes, anchorMap, userRefs)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Image generation failed: %v", err))
		return
	}
	totalTokens += imageResult.TokensUsed

	// ── Step 5: 一致性检查（不阻断） ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 82, "checking_consistency")
	issues, checkTok := s.checkFragmentConsistency(ctx, elemResult.VisualBible, scenesResult.Scenes, imageResult.ImageUrls)
	totalTokens += checkTok
	if len(issues) > 0 {
		for _, iss := range issues {
			s.logger.Info("fragment consistency issue",
				zap.String("task_id", taskID),
				zap.Int("scene_index", iss.SceneIndex),
				zap.String("severity", iss.Severity),
				zap.String("detail", iss.Detail))
		}
	}

	result := &domain.FragmentGenerationResult{
		Content:           elemResult.Content,
		ImageUrls:         imageResult.ImageUrls,
		AspectRatio:       resolvedAR,
		TokensUsed:        totalTokens,
		VisualBible:       elemResult.VisualBible,
		AnchorImages:      anchorRecords,
		ConsistencyIssues: issues,
	}

	// 更新占位草稿
	now := time.Now().UnixMilli()
	imgCount := len(result.ImageUrls)
	if imgCount < 1 {
		imgCount = 1
	}
	style := task.Request.Style
	caption := captionFromGenerationContent(result.Content)

	existing, gerr := s.fragmentRepo.GetBySource(ctx, string(domain.FragmentSourceAIGeneration), taskID)
	if gerr == nil && existing != nil {
		existing.Content = result.Content
		existing.MediaURLs = append([]string(nil), result.ImageUrls...)
		existing.ImageUrls = stringifyGenerationImageURLs(result.ImageUrls)
		existing.Style = &style
		existing.FragmentCount = &imgCount
		existing.UpdatedAt = now
		if caption != "" {
			existing.Caption = caption
		}
		if err := s.fragmentRepo.Update(ctx, existing); err != nil {
			s.logger.Error("Failed to update draft fragment from generation",
				zap.String("task_id", taskID),
				zap.String("draft_id", existing.ID),
				zap.Error(err))
		} else {
			result.DraftFragmentID = existing.ID
		}
	} else {
		fragment := &domain.Fragment{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:          task.UserID,
			CreatorID:       task.UserID,
			Content:         result.Content,
			MediaURLs:       append([]string(nil), result.ImageUrls...),
			ImageUrls:       stringifyGenerationImageURLs(result.ImageUrls),
			Style:           &style,
			FragmentCount:   &imgCount,
			Visibility:      domain.FragmentVisibilityPrivate,
			IsDraft:         true,
			SourceType:      string(domain.FragmentSourceOriginal),
			SourceID:        taskID,
			EngagementStats: common.EngagementStats{},
		}
		if caption != "" {
			fragment.Caption = caption
		}
		if err := s.fragmentRepo.Create(ctx, fragment); err != nil {
			s.logger.Error("Failed to create draft fragment from generation",
				zap.String("task_id", taskID),
				zap.Error(err))
		} else {
			result.DraftFragmentID = fragment.ID
		}
	}

	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
		s.logger.Error("Failed to update result", zap.Error(err))
	}

	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "completed", 100, "completed")

	s.logger.Info("Fragment generation completed",
		zap.String("task_id", taskID),
		zap.Int("tokens_used", result.TokensUsed),
		zap.Int("image_count", len(result.ImageUrls)),
		zap.Int("scenes", len(scenesResult.Scenes)),
		zap.Int("anchor_images", len(anchorRecords)))
}

func stringifyGenerationImageURLs(urls []string) string {
	if len(urls) == 0 {
		return "[]"
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func captionFromGenerationContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > 32 {
		return string(runes[:32])
	}
	return s
}

// ────────────────────── Step 1: 元素提取 + 文案生成 ──────────────────────

// fragmentElementExtractionResult 元素提取 + 文案生成的输出。
type fragmentElementExtractionResult struct {
	Elements    fragmentStoryElements       `json:"elements"`
	VisualBible *domain.FragmentVisualBible `json:"visualBible,omitempty"`
	Content     string                      `json:"content"`
	AspectRatio string                      `json:"aspectRatio"`
	TokensUsed  int
}

// fragmentStoryElements 从用户输入和参考图中提取的结构化元素。
type fragmentStoryElements struct {
	Weather    string   `json:"weather"`    // 天气：晴朗、雨天、暴风雪等
	Objects    []string `json:"objects"`    // 关键物品
	Scenes     []string `json:"scenes"`     // 场景类型：室内/室外/森林/城市等
	TimeOfDay  string   `json:"timeOfDay"`  // 时间：清晨/正午/黄昏/深夜等
	Location   string   `json:"location"`   // 地点描述
	Characters []string `json:"characters"` // 人物/角色描述
	Tendency   string   `json:"tendency"`   // 情感倾向/叙事方向
}

func (s *FragmentGenerationService) extractElementsAndGenerateContent(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*fragmentElementExtractionResult, error) {
	hasImages := len(fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)) > 0
	prompt := s.buildExtractionAndStoryPrompt(req, hasImages)

	payload := map[string]interface{}{"prompt": prompt}
	if imgs := fragmentPrefillHTTPImageURLs(req.ImageUrls, 10); len(imgs) > 0 {
		payload["imageUrls"] = imgs
	}
	inputBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal extraction input: %w", err)
	}

	aiReq := domain.AITask{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     domain.AITaskGenerateFragmentContent,
		Status:   domain.AITaskStatusProcessing,
		Provider: "",
		Input:    string(inputBytes),
	}

	raw, tokensUsed, err := s.aiService.GenerateFragmentExtractionJSON(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI extraction+story generation failed: %w", err)
	}

	result, perr := parseExtractionResult(raw)
	if perr != nil {
		s.logger.Warn("extraction JSON parse failed, using raw text as content",
			zap.Error(perr),
			zap.String("snippet", truncateRunes(raw, 120)))
		if c, ar, e2 := parseFragmentMultimodalStoryJSON(raw); e2 == nil {
			result = &fragmentElementExtractionResult{
				Content:     c,
				AspectRatio: ar,
			}
		} else {
			result = &fragmentElementExtractionResult{
				Content: truncateRunes(raw, 500),
			}
		}
	}
	if result.AspectRatio == "" {
		if _, ar, e2 := parseFragmentMultimodalStoryJSON(raw); e2 == nil {
			result.AspectRatio = ar
		}
	}
	result.TokensUsed = tokensUsed
	return result, nil
}

// buildExtractionAndStoryPrompt 构建元素提取 + 文案生成的组合提示词。
func (s *FragmentGenerationService) buildExtractionAndStoryPrompt(req domain.FragmentGenerationRequest, hasImages bool) string {
	styleDesc := fragmentStyleDesc(req.Style)
	moodDesc := fragmentMoodDesc(req.Mood)
	lengthDesc := fragmentLengthDesc(req.Length)
	language := strings.TrimSpace(req.Language)
	if language == "" {
		language = "中文"
	}

	imgNote := ""
	if hasImages {
		imgNote = `

【参考图深度分析】用户提供了参考图，你不仅要"看懂"这张图，还要像一位电影美术指导一样把它拆解为可复用的视觉资产库。

第一阶段——直觉反应（先不要分析，记录你的第一感觉）：
- 这张图给你的第一情绪是什么？用一个形容词回答（如：孤独的、温暖的、诡异的、怀旧的……）
- 你的视线最先落在画面的哪个位置？是被什么吸引过去的？（色彩对比？人物表情？光线的方向？某个反常的细节？）
- 如果这是一部电影的一帧截图，你会猜测这部电影是什么类型？什么年代？讲什么故事？

第二阶段——系统性拆解（逐层分析画面的视觉构成）：

  A. 人物/主体层：
  - 外观精确描述：年龄感（少年/青年/中年/老年）、体型（高矮胖瘦）、发型发色、肤色
  - 穿着细节：服装款式（T恤/衬衫/裙子/外套）、颜色、材质（棉/丝/皮/毛）、是否有图案或文字
  - 姿态与动作：站/坐/跑/蹲/躺/跳、手在做什么、身体朝向（正面/45度侧/全侧/背面）、重心偏移
  - 面部信息：表情（或模糊/遮挡/背影）、视线方向（看镜头/看画外/低头/仰望）
  - 标志性视觉特征：纹身、疤痕、饰品（项链/戒指/耳环）、帽子/围巾/背包等配饰

  B. 环境空间层：
  - 空间类型：室内/室外/半室外（走廊/阳台/天台）、人造环境/自然环境/混合
  - 建筑或地貌：建筑风格（现代/古典/赛博朋克/废墟）、年代痕迹（新旧程度、破损、苔藓、灰尘）
  - 季节与温度：季节暗示（落叶/新绿/积雪/蝉鸣）、温度感（炎热蒸腾/寒冷刺骨/凉爽舒适）
  - 空间层次：前景有什么（遮挡物/框架元素）、中景主体、背景延伸（天际线/远山/墙壁截止）
  - 纵深感：是否有明确的透视线（道路/走廊/河流引导视线）、空间是开阔还是封闭

  C. 光影层：
  - 光源类型与方向：自然光（太阳从哪个方向）还是人造光（灯光/火光/屏幕光）、主光源位置
  - 色温：暖黄光（日落/烛火/白炽灯）、冷蓝光（月光/荧光灯/电子屏幕）、中性白（正午日光）
  - 阴影：阴影的浓淡（硬阴影/柔阴影）、阴影的形状和方向、是否有趣味性阴影图案
  - 高光：画面中最亮的部分在哪里、高光是集中还是散射、是否有镜头光晕或光斑

  D. 色彩层：
  - 主色调：画面面积最大的颜色是什么、这个颜色的情绪暗示
  - 点缀色：与主色调形成对比或互补的颜色、这些颜色出现在画面中的哪些位置
  - 饱和度：整体是高饱和（鲜艳）还是低饱和（灰调/褪色）、饱和度在画面中是否均匀
  - 色彩渐变：画面中是否有色彩渐变（天空从蓝到橙、光线从暖到冷的过渡）

  E. 构图层：
  - 景别：特写（面部或细节）/ 近景（上半身）/ 中景（全身）/ 远景（人物渺小）
  - 角度：平视/俯拍/仰拍/Dutch angle（倾斜）
  - 画面重心：主体在画面的什么位置（居中/三分法/边缘）
  - 引导线：画面中是否有线条（道路/建筑边缘/光影分界）引导视线

  F. 细节彩蛋层：
  - 画面中容易被忽略但有趣的微观细节（墙角的涂鸦、书桌上的便利贴、远处的动物剪影、口袋里露出的信封一角、地上的水洼倒影）
  - 这些细节可能暗示了什么故事信息

第三阶段——视觉信息的故事化转译：
- 人物外观 → characters 的视觉身份卡（发型/服装/标志性特征）
- 环境与季节 → scenes 的空间氛围描述
- 光影与色彩 → weather 的情绪化天气描述
- 构图与细节 → tendency 的核心感受方向
- 确保生成的故事与参考图在画面上有真实呼应，而不是只基于文字做文字推理
- 从参考图中提取的元素应该让读者感觉"这个故事确实可能发生在那张图片里"

第四阶段——冲突裁决与创意边界：
- 若参考图与用户文字存在轻微冲突：以用户文字为剧情主线，以参考图为视觉锚点
- 允许创造惊喜与反转，但不能引入与图文都无关的核心设定
- 每个关键转折都要能在 elements 或原始输入中找到伏笔或视觉依据`
	}

	return fmt.Sprintf(`你是一位兼具文学才华和电影美术指导素养的故事碎片创作大师。你同时拥有小说家的叙事直觉、摄影师的画面敏感度、和漫画编辑的商业嗅觉——你知道什么样的内容能让人停下手指、点开、读完、然后截图转发。你需要完成两件事：

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第一件事：元素提取（把模糊的感觉变成精确的视觉指令）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

从用户的文字描述%s中提炼出以下故事元素。

核心原则：元素不是关键词罗列，而是带有画面感和叙事暗示的具体描述。检验标准是——把这些元素交给一位从未读过原文的插画师，他仅凭这些描述就能画出"对的画面"。

逐项说明（每项都附有好的和差的示例，请向好的看齐）：

1. weather（天气）—— 情绪的物理化
   天气不是背景板，它是情绪的放大器和隐喻。不要只写"晴天/雨天"，要写出天气如何渗透进画面和角色的感受中。
   好的示例："暴雨将至，天色发黄，空气黏稠得像裹了一层保鲜膜，远处传来闷雷但还没下雨"
   好的示例："深秋的晴夜，冷得能看见自己呼出的白气，月光把地上的落叶冻成了银色"
   差的示例："阴天" / "下雨"
   如果天气确实不重要，写"无特别天气"——但优先考虑天气能如何强化你想讲的故事的情绪。

2. objects（物品）—— 故事的道具箱（最多 5 个）
   每个物品都是一个潜在的叙事线索。写出：外观 + 材质 + 当前状态 + 与角色的空间关系。
   好的示例："一把半透明的红伞，伞面有几道裂痕，被随手靠在咖啡店门边，伞尖的水渍在地砖上洇开一小片"
   好的示例："一只翻盖手机，外壳贴满了已经褪色的贴纸，屏幕上的时间停在 3:17"
   差的示例："伞" / "手机"

3. scenes（场景）—— 五感的空间（最多 3 个）
   场景不只是一个地点名称，它是一个可以用五感去体验的空间。写出视觉之外的感官细节。
   好的示例："老式电车内部，木质座椅磨得发亮，窗外是模糊的樱花隧道，车厢里有铁锈和甜食混合的味道，地板随着转弯发出低沉的吱嘎声"
   好的示例："凌晨三点的便利店，冷柜嗡嗡作响，日光灯管其中一根在闪，空气里有加热便当和消毒水的混合味道"
   差的示例："电车上" / "便利店"

4. timeOfDay（时间）—— 光线即情绪
   不要只写时段，要写出那个时段特有的光线、温度、以及它暗示的情绪氛围。
   好的示例："黄昏，太阳刚好卡在两栋楼之间，整条街被染成橘红色，路人的影子拖得像另一个人"
   好的示例："凌晨四点半，天还没亮但已经不是完全的黑，东边的天际线有一道极细的灰蓝色"
   差的示例："傍晚" / "早上"

5. location（地点）—— 有故事感的空间
   写出地点的辨识度、时代感、和空间性格（开阔/逼仄/纵深/封闭）。
   好的示例："一个被废弃的室内游乐场，旋转木马还在缓慢转，彩灯只剩红和蓝还亮着，地上散着褪色的入场券和爆米花桶"
   好的示例："高铁站的 7 号站台，电子屏显示着已经发车的班次，不锈钢长椅上只剩一个没喝完的纸杯"
   差的示例："游乐场" / "火车站"

6. characters（人物）—— 视觉身份卡（最多 3 个）
   想象你在给一位从未见过这个角色的画师做口头描述。包含：体型轮廓、穿着（款式+颜色+材质）、标志性视觉特征、当前在做的事情和表情。
   好的示例："瘦高的女生，穿oversized黑卫衣帽子压得很低，露出一截染了蓝色的发尾，手里攥着一杯已经凉透的拿铁，站在斑马线中间像在犹豫要不要过去"
   好的示例："中年男人，头发灰白但梳得整齐，穿着一件洗到发白的蓝色工装外套，右手插在口袋里，左手拎着一个系着红绳的旧皮箱"
   差的示例："一个女生" / "一个男人"

7. tendency（倾向）—— 这段故事的一行宣发语
   不是分类标签，而是"如果这是一张专辑封面，封面上只印这一句话"。要让读者看到这句话就想点进去看故事。
   好的示例："所有出口都标着入口的方向"
   好的示例："她等的那班列车三年前就停运了"
   好的示例："世界上最远的距离不是生与死，是你坐在对面但信号只有一格"
   差的示例："悬疑风格" / "温馨治愈"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第二件事：碎片故事（用文字创造让人截图转发的瞬间）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

基于提取的元素，写一个让人看完想转发的故事碎片。

用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s（elements 与 content 的自然语言都必须与该语言一致）

创作优先级（从高到低，不能颠倒）：
- 第一优先级：贴合用户输入与参考图锚点
- 第二优先级：画面感与可传播性
- 第三优先级：反转、奇观和风格实验

故事创作心法（这是你的创作工具箱，按主题分类）：

【钩子：第一句就锁住读者】
- 绝对不要用的开头："有一天""在很久以前""那是一个""故事开始于"——这些是读者滑走的信号
- 用以下方式之一开局：
  · 反常细节："钥匙插进锁孔的时候，她发现门是从里面锁着的。"
  · 动作进行时："他跑到便利店的时候，雨已经停了。但他的伞还是撑着的。"
  · 对话切入："你刚才说你有妹妹？" "我说过吗？"
  · 悬念前置："那张照片里多出来的人，不是后期P的。"
  · 感官冲击："空气里有烧焦的味道，但不是食物。"

【画面感：用文字画画】
核心原则：动词 > 形容词，具体名词 > 抽象概念，可观察的行为 > 内心独白。
- 不要写"她感到无比悲伤和孤独"，要写"她把第二碗面推到对面空着的座位前"
- 不要写"天空非常美丽壮观"，要写"云层裂开一道缝，光柱打在她脚边半米处"
- 不要写"他很紧张"，要写"他把同一页书翻了三次"
- 不要写"气氛很尴尬"，要写"两个人同时伸手去拿桌上的遥控器，又同时缩回去"
- 角色的情绪要"演"出来，不要"说"出来：不要写"她很害怕"，写"她的手在口袋里攥成了拳头，指甲掐进掌心"

【留白：不说完比说完更高级】
- 最好的结尾不是句号，而是让读者自己脑补下一秒发生了什么
- 不要解释你的隐喻——如果读者需要你解释，说明隐喻不够好
- 可以在最高潮处戛然而止，比写完整个结局更有余韵
- 留白的位置通常在这些地方：做出选择的那一刻、真相揭晓的前一秒、转身之后
- 技巧：最后一句可以是完全平静的日常动作——越是平静，越衬托前面的波澜

【转折：一个就够】
- 不需要反转再反转，一个"等等，什么？"的时刻比三个小反转有效十倍
- 好的转折来自前面埋下的细节——读者回头看时会发现"原来早就有暗示了"
- 转折类型参考：
  · 视角的转折："我"其实不是人类（是一只猫、一个AI、一颗树）
  · 时间的转折：这已经是十年前的回忆 / 这段话是遗书 / 这是未来的人在看旧照片
  · 身份的转折：对面的人其实不认识她 / "妹妹"其实是想象中的
  · 空间的转折：门外面不是走廊，是天空 / 一直以为是室内的场景其实在外面

【世界观：用细节播种，让读者自己收获】
- 不要写"这是一个魔法与科技并存的世界"，要写"咖啡杯飘在半空，她懒得去接"
- 不要写"未来社会严禁自由恋爱"，要写"结婚申请表上有一栏：请填写恋爱许可证编号"
- 一个反常的日常细节比三句世界观说明更有说服力
- 让读者自己拼凑规则，不要喂给他们
- 世界观的细节应该"长"在叙事里，而不是"贴"在上面——通过角色的行为和环境的描写自然呈现

【节奏控制：快慢交替】
- 短句加速（制造紧张感、转折冲击）："她回头。没人。"
- 长句减速（铺陈氛围、内心沉淀）："她站在站台上看着列车尾灯在隧道口慢慢缩成一个红点，手里的信封被风吹得哗哗响，但她没有追过去"
- 一段安静的描写之后突然一个短句转折，效果拔群
- 句子长度本身就是情绪的呼吸——全篇匀速的文字是没有生命力的

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第三件事：视觉圣经 visualBible（多图一致性锚点，必须输出）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
- visualBible 必须与 elements、content 自洽；immutableTraits 使用与 elements/content 相同的自然语言（由上方 language 决定）。
- characters最多 3 项，props 最多 5 项，locations 1–2 项；每项必须有全局唯一 key（小写英文+下划线，如 char_main、prop_umbrella、loc_station）。
- immutableTraits 为字符串数组：每条描述一个不可随意更改的视觉事实（发色、服装剪裁与主色、道具外形、建筑特征等）。
- styleBible.artStyle 必须用英文写出可执行的总体画法（媒介、线稿/渲染、时代感、参考流派），供后续所有配图 prompt 对齐；其他 styleBible 字段可选。

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
{
  "elements": {
    "weather": "天气氛围描述——写出天气如何渗透进画面和情绪（与上方语言要求一致）",
    "objects": ["物品1：外观+材质+状态+与角色空间关系", "物品2"],
    "scenes": ["场景1：五感空间描述（视觉+听觉+嗅觉+触感+温度）", "场景2"],
    "timeOfDay": "时段+该时段特有的光线氛围（与上方语言要求一致）",
    "location": "具体地点+时代感+空间性格（开阔/封闭/纵深）+标志性细节（与上方语言要求一致）",
    "characters": ["角色1：体型+穿着（款式颜色材质）+标志性特征+当前动作和表情", "角色2"],
    "tendency": "核心感受一句话——像专辑封面或电影海报上印的那一行字（与上方语言要求一致）"
  },
  "visualBible": {
    "styleBible": {
      "artStyle": "English: overall rendering and line quality for all images",
      "lineQuality": "optional English",
      "palette": "optional English",
      "lightingMood": "optional English"
    },
    "characters": [
      { "key": "char_main", "name": "optional", "immutableTraits": ["trait1", "trait2"] }
    ],
    "props": [
      { "key": "prop_key", "immutableTraits": ["..."] }
    ],
    "locations": [
      { "key": "loc_key", "immutableTraits": ["..."] }
    ]
  },
  "content": "碎片故事正文（与上方语言要求一致），直接可读，不要标题，不要前缀说明",
  "aspectRatio": "推荐长宽比：1:1、16:9、9:16、3:4、4:3，结合画面构图选择；不确定用 16:9"
}`,
		imgNote,
		req.UserInput, styleDesc, moodDesc, lengthDesc, language)
}

// parseExtractionResult 解析元素提取 + 文案的 JSON 输出。
func parseExtractionResult(raw string) (*fragmentElementExtractionResult, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var out fragmentElementExtractionResult
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse extraction JSON: %w", err)
	}
	if strings.TrimSpace(out.Content) == "" {
		return nil, fmt.Errorf("extraction result missing content")
	}
	normalizeFragmentVisualBible(&out.VisualBible)
	return &out, nil
}

func normalizeFragmentVisualBible(b **domain.FragmentVisualBible) {
	if b == nil || *b == nil {
		return
	}
	bb := *b
	for i := range bb.Characters {
		if strings.TrimSpace(bb.Characters[i].Key) == "" {
			bb.Characters[i].Key = fmt.Sprintf("char_%d", i)
		}
	}
	for i := range bb.Props {
		if strings.TrimSpace(bb.Props[i].Key) == "" {
			bb.Props[i].Key = fmt.Sprintf("prop_%d", i)
		}
	}
	for i := range bb.Locations {
		if strings.TrimSpace(bb.Locations[i].Key) == "" {
			bb.Locations[i].Key = fmt.Sprintf("loc_%d", i)
		}
	}
	if bb.StyleBible != nil {
		empty := strings.TrimSpace(bb.StyleBible.ArtStyle) == "" &&
			strings.TrimSpace(bb.StyleBible.LineQuality) == "" &&
			strings.TrimSpace(bb.StyleBible.Palette) == "" &&
			strings.TrimSpace(bb.StyleBible.LightingMood) == ""
		if empty {
			bb.StyleBible = nil
		}
	}
	hasAssets := len(bb.Characters) > 0 || len(bb.Props) > 0 || len(bb.Locations) > 0 || bb.StyleBible != nil
	if !hasAssets {
		*b = nil
	}
}

func collectVisualBibleKeys(b *domain.FragmentVisualBible) []string {
	if b == nil {
		return nil
	}
	var keys []string
	for _, c := range b.Characters {
		if k := strings.TrimSpace(c.Key); k != "" {
			keys = append(keys, k)
		}
	}
	for _, p := range b.Props {
		if k := strings.TrimSpace(p.Key); k != "" {
			keys = append(keys, k)
		}
	}
	for _, loc := range b.Locations {
		if k := strings.TrimSpace(loc.Key); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func formatVisualBibleKeyListForPrompt(b *domain.FragmentVisualBible) string {
	if len(collectVisualBibleKeys(b)) == 0 {
		return "（无可用 key；每格 referenceKeys 输出 []）"
	}
	return strings.Join(collectVisualBibleKeys(b), ", ")
}

func mergeFragmentSceneReferenceImages(userURLs []string, scene fragmentExpandedScene, anchorMap map[string]string, maxTotal int) []string {
	if maxTotal <= 0 {
		maxTotal = fragmentMaxSceneReferenceImages
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, u := range userURLs {
		add(u)
		if len(out) >= maxTotal {
			return out
		}
	}
	for _, k := range scene.ReferenceKeys {
		if u := anchorMap[strings.TrimSpace(k)]; u != "" {
			add(u)
			if len(out) >= maxTotal {
				return out
			}
		}
	}
	return out
}

func (s *FragmentGenerationService) generateAnchorImages(ctx context.Context, userID, genTaskID string, req domain.FragmentGenerationRequest, bible *domain.FragmentVisualBible, aspectRatio string) (map[string]string, []domain.FragmentAnchorImage, int) {
	outMap := make(map[string]string)
	if bible == nil {
		return outMap, nil, 0
	}
	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	styleEN := ""
	if bible.StyleBible != nil {
		styleEN = strings.TrimSpace(bible.StyleBible.ArtStyle)
	}
	styleZH := fragmentStyleDesc(req.Style)
	moodZH := fragmentMoodDesc(req.Mood)
	userRef := fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)

	totalTok := 0
	var records []domain.FragmentAnchorImage
	firstChar := true

	for i, ch := range bible.Characters {
		if i >= fragmentMaxAnchorCharacters {
			break
		}
		key := strings.TrimSpace(ch.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(ch.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Character design reference, single person, neutral standing pose facing camera, full body visible, clean simple studio background, high detail concept art. Immutable traits: %s. Overall mood (reference): %s. Style tag: %s.",
			styleEN, traits, moodZH, styleZH)
		if n := strings.TrimSpace(ch.Name); n != "" {
			prompt = n + ". " + prompt
		}
		var refs []string
		if firstChar && len(userRef) > 0 {
			refs = append(refs, userRef...)
			firstChar = false
		}
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, refs)
		if err != nil {
			s.logger.Warn("anchor character image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "character", ImageURL: url})
		totalTok += tok
	}

	for i, p := range bible.Props {
		if i >= fragmentMaxAnchorProps {
			break
		}
		key := strings.TrimSpace(p.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(p.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Single hero prop object on neutral surface, studio product shot, sharp focus, no people. Object traits: %s. Style: %s.",
			styleEN, traits, styleZH)
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, nil)
		if err != nil {
			s.logger.Warn("anchor prop image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "prop", ImageURL: url})
		totalTok += tok
	}

	for i, loc := range bible.Locations {
		if i >= fragmentMaxAnchorLocations {
			break
		}
		key := strings.TrimSpace(loc.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(loc.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Wide environmental establishing shot, empty of people, architectural or landscape concept art, clear readable space. Location: %s. Style: %s.",
			styleEN, traits, styleZH)
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, nil)
		if err != nil {
			s.logger.Warn("anchor location image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "location", ImageURL: url})
		totalTok += tok
	}

	return outMap, records, totalTok
}

func (s *FragmentGenerationService) generateOneFragmentImage(ctx context.Context, userID, genTaskID, prompt, aspectRatio string, refURLs []string) (string, int, error) {
	payload := map[string]interface{}{
		"prompt":      prompt,
		"aspectRatio": aspectRatio,
	}
	if len(refURLs) > 0 {
		payload["referenceImages"] = refURLs
	}
	imgInput, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}
	aiReq := domain.AITask{
		ID:                uuid.New().String(),
		UserID:            userID,
		Type:              domain.AITaskGenerateFragmentImages,
		Status:            domain.AITaskStatusProcessing,
		Provider:          "",
		Input:             string(imgInput),
		RelatedEntityID:   genTaskID,
		RelatedEntityType: "fragment_generation",
	}
	return s.aiService.GenerateImageForFragment(ctx, &aiReq)
}

func (s *FragmentGenerationService) checkFragmentConsistency(ctx context.Context, bible *domain.FragmentVisualBible, scenes []fragmentExpandedScene, imageURLs []string) ([]domain.FragmentConsistencyIssue, int) {
	vbBytes, _ := json.Marshal(bible)
	scenesBytes, _ := json.Marshal(scenes)
	var urlLines strings.Builder
	for i, u := range imageURLs {
		fmt.Fprintf(&urlLines, "%d: %s\n", i, u)
	}
	prompt := fmt.Sprintf(`你是视觉一致性审计员。基于「视觉圣经」、各格场景描述与最终配图 URL 列表，列出可能的不一致（角色外观漂移、关键道具缺失、环境与圣经矛盾等）。若无法访问图片，仅依据文字推断；无问题时 issues 为空数组。

视觉圣经 JSON：
%s

场景 JSON（含 sceneDesc、referenceKeys、index）：
%s

最终图片 URL（按生成顺序排列；尽量与场景 index 对应）：
%s

只输出 JSON：{"issues":[{"sceneIndex":0,"severity":"low|medium|high","detail":"中文简述"}]}`,
		string(vbBytes), string(scenesBytes), urlLines.String())

	raw, tokens, err := s.aiService.GenerateFragmentAuxJSON(ctx, prompt)
	if err != nil {
		s.logger.Warn("fragment consistency check request failed", zap.Error(err))
		return nil, 0
	}
	issues, perr := parseFragmentConsistencyIssues(raw)
	if perr != nil {
		s.logger.Warn("fragment consistency check parse failed", zap.Error(perr), zap.String("snippet", truncateRunes(raw, 200)))
		return nil, tokens
	}
	return issues, tokens
}

func parseFragmentConsistencyIssues(raw string) ([]domain.FragmentConsistencyIssue, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var env struct {
		Issues []domain.FragmentConsistencyIssue `json:"issues"`
	}
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, err
	}
	return env.Issues, nil
}

// ────────────────────── Step 2: 场景扩展 ──────────────────────

// fragmentSceneExpansionResult 场景扩展输出。
type fragmentSceneExpansionResult struct {
	Scenes     []fragmentExpandedScene `json:"scenes"`
	TokensUsed int
}

// fragmentExpandedScene 扩展出的单个场景，含中文描述和英文图片提示词。
type fragmentExpandedScene struct {
	Index         int      `json:"index"`
	SceneDesc     string   `json:"sceneDesc"`               // 中文场景描述（面向读者）
	ImagePrompt   string   `json:"imagePrompt"`             // 英文图片生成提示词（面向图片模型）
	ReferenceKeys []string `json:"referenceKeys,omitempty"` // 引用 visualBible / 锚点图的 key
}

func (s *FragmentGenerationService) expandScenes(ctx context.Context, userID string, req domain.FragmentGenerationRequest, elemResult *fragmentElementExtractionResult, sceneCount int, aspectRatio string) (*fragmentSceneExpansionResult, error) {
	elementsJSON, _ := json.Marshal(elemResult.Elements)
	vbJSON := "{}"
	if elemResult.VisualBible != nil {
		if b, err := json.Marshal(elemResult.VisualBible); err == nil {
			vbJSON = string(b)
		}
	}
	keyHint := formatVisualBibleKeyListForPrompt(elemResult.VisualBible)

	var narrativeHint string
	switch {
	case sceneCount == 1:
		narrativeHint = "1 格：这一帧必须是整条故事中最有视觉冲击力的瞬间。不要选平淡的叙述时刻——选悬念最浓的那一秒、反转刚发生的那一帧、或者一个让人立刻想问\"之前到底发生了什么\"的定格。想象电影海报：一个画面就让观众脑补出一整部电影。让这一帧的构图、光影、角色表情本身就在讲故事。如果故事有高潮，这就是高潮被凝固的那一毫秒；如果故事没有高潮，这就是最让人不安的那一秒——一切看似正常但有什么东西不太对。"
	case sceneCount == 2:
		narrativeHint = "2 格：核心技法是\"认知落差\"——第一格建立预期，第二格打破它。具体可以用的落差类型：视角落差（第一格是人眼中的世界，第二格是虫子眼中的同一场景——完全不同的尺度，完全不同的恐惧）、情绪落差（温馨 → 惊悚，平静 → 狂喜，搞笑 → 心碎）、尺度落差（微观 → 宏观，一只蚂蚁的特写 → 整个城市的俯瞰）、时间落差（现在 → 十年前，白天 → 深夜，春天 → 冬天）。观众看完两格后的反应应该是\"等等，怎么会这样？\"或\"所以第一格里那个其实是……\"。第二格的第一反应应该是意外，第二反应才是理解。"
	case sceneCount == 3:
		narrativeHint = "3 格：不要三幕式！三幕式会产出四平八稳但无趣的内容。自由节奏才是王道——第一格抛出引子（一个画面就让人好奇\"这是哪里/这个人是谁/发生了什么\"），第二格可以突然转向（换视角、换时空、换叙事者、或者一个完全出乎意料的元素闯入），第三格收束但不给答案——留一个让人回味的尾巴。可以尝试的高级结构：假结局（第三格看似结束但其实暗示了更大的故事）、环形结构（第三格呼应第一格但信息量不同，读者对比两格才发现真相）、打破第四面墙（角色意识到了读者的存在）、时间嵌套（第三格揭示第一格其实是回忆中的回忆）。惊喜感比工整重要十倍。"
	default:
		narrativeHint = fmt.Sprintf("%d 格：一条故事线但绝不要线性平铺直叙。在格与格之间可以大胆穿插——视角突变（从人类视角突然切到猫的视角、从地面仰视突然变卫星俯瞰、从一个角色的POV切到另一个角色的POV看着同一个场景）、时空跳跃或闪回（这一格还是现在，下一格突然是十年前，再下一格又回到现在但时间已过）、打破第四面墙（角色看向\"镜头\"外、文字溢出画框、角色似乎在跟读者说话）、超现实梦境（前一秒还在教室，下一秒教室漂浮在星空中、水面变成了天空、书本里的文字活了过来）、意料之外的新元素闯入（一个路过的蝴蝶改变了整条故事线、一只手从画外伸进来拿走了桌上的杯子）。开头建立世界观和角色，中间尽情放飞制造\"没想到\"的惊喜，结尾可以呼应开头、可以留悬念、可以推翻之前所有设定——只要让观众看完觉得\"这趟旅程值了\"。中间某些格可以是氛围画面——不推进剧情但传递情绪（空椅子、飘落的羽毛、雨中的路灯）——这种\"停顿\"本身是节奏的一部分。", sceneCount)
	}

	prompt := fmt.Sprintf(`你是一位脑洞大开、同时精通电影摄影和漫画分镜的视觉故事创作者，兼具导演的叙事把控力、摄影师的画面敏感度和概念艺术家的想象力。你的任务是把一段故事文字拆解为 %d 个画面格——每一格都是一幅能独立吸引目光、按顺序浏览又能串联成完整故事的视觉作品。你不是在"配图"，你是在"用画面重新讲故事"。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
叙事节奏指引
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输入素材
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

故事元素（从用户输入中提取的结构化元素）：
%s

视觉圣经 visualBible（JSON；用于每格 referenceKeys；若无则为 {}）：
%s

referenceKeys 可用的稳定 key 列表（必须从下列 key 中选择子集填入每格；禁止自造 key；若列表说明为无 key 则每格 referenceKeys=[]）：
%s

故事正文（AI 生成的碎片故事文案）：
%s

风格：%s
情绪：%s
画面长宽比：%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
创作方法论（你的导演工具箱）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

【一、世界观一致性——这是底线，不是上限】
- 角色的外貌（发型、服装、体型、标志性特征）必须在每一格中完全一致——除非剧情需要角色发生变化（被雨淋湿、受伤、换装等需要明确交代）
- 空间设定（室内布局、城市天际线、自然地貌）保持可辨识的连续性——读者应该感觉这些格发生在同一个世界里
- artStyle（艺术风格）全程统一——如果第一格是"水彩插画风"，最后一格也必须是水彩插画风。风格是故事的视觉签名
- 色彩基调可以随情绪渐变，但不要突然跳到完全不同的色彩体系

【二、叙事自由度——打破常规才是你的常规】
- 允许并且鼓励：
  · 跳切：时间突然推进，中间的过程留白让读者脑补
  · 闪回：突然回到过去，揭示之前没展示的信息
  · 视角反转：突然换成配角、动物、甚至无生命物体的视角
  · 超现实片段：梦境/幻觉/想象——可以让光影与空间反常，但仍保持同一主风格体系
  · 意外闯入：可以有新角色/物体突然出现，但必须与已有元素或故事正文存在因果关联
  · 静帧与留白：一格完全静止的画面（空房间、雨中的长椅、桌面上的物品）——不推进剧情但传递情绪，这种"停顿"本身就是节奏
- 每一格都可以是不同类型的画面：
  · 写实叙事场景（角色在行动）
  · 氛围/意境画面（雨滴落在水洼、风铃在摇晃、夕阳照进空教室）
  · 特写细节（一只手、一封信、地上的影子）
  · 全景鸟瞰（城市天际线、从太空看地球）
  · 角色内心可视化（恐惧具象化成黑色的手、回忆用虚线勾勒、想象用不同的色调区分）
  · 象征/隐喻画面（断裂的桥、倒走的钟、镜中不同的自己）
- 不必每一格都在推进剧情——有时一格氛围画面比剧情画面更有力量。读者会在安静的画面中感受到前面积累的情绪

【三、构图多样性——你的镜头语言词库】
每格必须使用不同的镜头语言，以下是你可用的完整构图类型库（混搭使用，拒绝连续两格相同组合）：

  景别工具箱：
  - 极特写：一只眼睛、嘴唇、手指尖、物品的某个局部——传递极度的亲密或紧张
  - 特写：面部表情、手部动作、物品全貌——情绪和细节的放大镜
  - 近景：胸部以上，能看到表情和上半身动作——对话和情感交流的标配
  - 中景：膝盖以上，能看到角色全身姿态和环境的一小部分——叙事的主力景别
  - 全景：全身 + 周围环境，角色与环境的关系清晰——交代场景和空间关系
  - 远景：人物在画面中很小，环境占据主导——表达孤独、渺小、天地之大
  - 极远景/大远景：地标级画面，几乎看不到人物——故事的"呼吸"，给读者喘息的空间

  角度工具箱：
  - 平视：最接近人眼的日常视角，真实感最强
  - 俯拍/高角度：上帝视角，角色显得渺小无助，或展示场景的空间布局
  - 仰拍/低角度：角色显得高大/威严/压迫，或者模拟儿童/动物视角
  - Dutch angle（倾斜角度）：画面歪斜，制造不安、失衡、精神异常的感觉
  - 鸟瞰：正上方往下拍，展示平面的图案和布局（桌面、街道、棋盘）
  - 虫眼：贴地往上拍，夸大物体的高度和压迫感

  非人类视角工具箱（这是制造惊喜的秘密武器）：
  - 猫/狗视角：低角度，人类的腿变成了柱子，世界变得巨大
  - 鸟的视角：高空俯瞰，一切变得像模型，人变成了小点
  - 鱼的视角：从水下往上看，水面是扭曲的亮面，岸上的世界是摇晃的
  - 物品视角：钥匙孔视角（窥视感）、镜中倒影（虚实对照）、手机屏幕视角（被观看的感觉）、时钟视角（往下看人）、书本视角（被翻开的感觉）
  - 完全抽象：用色块和线条表达情绪而非具象画面——适合内心独白或超现实段落

  硬性规则：相邻两格不能使用相同的景别+角度组合。这是最低要求——理想状态是每 3 格内不重复

【四、光影与色彩——情绪的精密调色板】
  光影工具（选择与场景情绪匹配的光影方案）：
  - 逆光/轮廓光：主体变成剪影或边缘发光 → 神秘、英雄感、告别、未知
  - 侧光：一半亮一半暗，强烈的明暗对比 → 戏剧冲突、内心矛盾、揭示秘密
  - 顶光：从正上方打下来，眼窝和鼻子下方投下浓重阴影 → 压抑、审判、精神压力
  - 底光：从下方照亮面部（鬼故事经典打光） → 恐怖、诡异、非自然
  - 散射光/柔光：没有明确方向，阴影柔和 → 日常、平静、回忆、安全
  - 剪影：完全背光，只剩轮廓 → 未知、威胁、悬念、分离
  - 斑驳光：透过树叶/窗户/格栅的碎光 → 怀旧、监狱/困住、梦境

  色温与情绪的映射：
  - 暖黄/橙色（日落、烛火） → 安全、怀旧、温馨、即将结束
  - 冷蓝/青色（月光、荧光灯） → 疏离、科技、忧郁、冷静
  - 中性白（正午日光） → 真实、日常、客观
  - 红色（灯光、血色、警报） → 危险、激情、愤怒、警告
  - 绿色（自然、毒气、监控） → 自然、生长、毒性、被监视

  色彩饱和度的情绪曲线：
  - 高饱和 → 活力、梦境、奇幻、童年回忆
  - 中饱和 → 现实、日常、叙事进行时
  - 低饱和/去饱和 → 压抑、回忆褪色、末日、疲惫

  格间光影变化规则：
  - 光源方向和强度在格间可以变化，但要服务于情绪走向
  - 例如：故事走向紧张 → 光线从明亮逐步变暗，阴影逐渐拉长
  - 例如：故事走向释然 → 从冷色调逐渐回暖，阴影变柔和
  - 允许一格突然跳到完全不同的光影方案（如闪回用怀旧柔光，回到现实用冷硬侧光）

【五、sceneDesc 写法——面向读者的画面脚本】
- 一到两句话，让读者在脑中"看到"这一格的画面
- 要传达四个信息：谁在画面里、在做什么、什么氛围、在故事的哪个节点
- 不要写成剧情概要，要写成"如果你闭上眼睛想象这一格，你会看到什么"
- 好的示例："她站在空荡荡的站台，身后是已经驶远的列车尾灯，手里还攥着没来得及递出去的信"
- 差的示例："第二幕，她错过了火车"（这是剧情概要，不是画面描述）
- 好的示例："特写：一只手慢慢松开，信封从指缝间滑落，背面写着'已过期'三个字"
- 差的示例："信掉了"（太简略，没有画面感）

【六、imagePrompt 写法——给图片模型下达的精确视觉指令】
- 这是在给一位顶级概念艺术家下 brief，必须具体到每个视觉决策
- 必须按以下 8 层结构依次描述，每层用句号分隔，形成一段完整的英文视觉描述：

  第 (1) 层 artStyle —— 整体艺术风格
  不要用笼统的"anime"或"illustration"。要写出具体的混合风格，让画师一眼知道用什么技法。
  好的示例："cinematic watercolor with digital color grading, muted palette with selective vivid accents"
  好的示例："charcoal sketch style with selective watercolor highlights, rough texture, visible stroke marks"
  好的示例："hyperrealistic 3D render with soft bloom lighting, slight lens aberration"
  好的示例："retro cel-shaded animation style, flat colors with bold ink outlines, 90s anime aesthetic"
  差的示例："anime style" / "illustration"

  第 (2) 层 subject —— 画面核心主体
  谁或什么在画面中，具体到可以被直接画出来。包括：角色完整外貌（发型发色、服装款式颜色材质、体型年龄）、当前姿态（站/坐/跑/蹲/转身）、表情或情绪暗示、手中持有的物品。
  好的示例："a tall slender woman in her mid-20s with shoulder-length dyed blue hair tips, wearing an oversized black hoodie with the hood down, standing still at the center of a pedestrian crossing, holding a paper coffee cup with both hands, her gaze directed slightly downward, expression neutral but tired"
  差的示例："a woman standing"（缺少所有视觉细节）

  第 (3) 层 environment —— 背景和场景空间
  角色所处的完整空间，包括空间类型、建筑/自然特征、前景中景背景的层次。
  好的示例："an abandoned indoor amusement park at night, a faded carousel with peeling paint horses slowly rotating in the midground, only red and blue neon tubes still flickering, dusty concrete floor scattered with old admission tickets and collapsed cotton candy cones, a collapsed banner reading GRAND OPENING visible in the far background"
  差的示例："an amusement park"（没有细节、没有层次）

  第 (4) 层 composition —— 镜头构图
  景别 + 角度 + 画面重心位置 + 引导线（如有）。
  好的示例："medium shot from a low angle, subject positioned at the right third of the frame, the carousel filling the left two-thirds of the background, leading lines from the floor tiles converging toward the subject"
  差的示例："medium shot"（缺少角度和构图位置）

  第 (5) 层 lighting —— 光照方案
  光源方向、类型、色温、阴影特征、高光位置。要具体到画师能据此画出光影。
  好的示例："primary light from the red neon sign on the left casting warm crimson shadows across the subject's face, secondary cool blue light from a flickering neon tube behind creating a rim light on the subject's hair, no ambient natural light, deep shadows in the background with no visible detail"
  差的示例："neon lights"（方向？色温？阴影？）

  第 (6) 层 colorPalette —— 完整色彩方案
  主色调 + 点缀色 + 对比度 + 色彩分布。
  好的示例："dominant dark teal and navy blue in the shadows, accent warm red from the neon sign creating high-contrast focal points, muted purple undertones in the midtones, skin tones slightly desaturated, overall low-key color scheme"
  差的示例："dark colors with red"（太笼统）

  第 (7) 层 mood —— 情绪氛围
  这一格应该让观众感受到什么。用复合情绪词，不要只写一个形容词。
  好的示例："melancholic solitude with a hint of nostalgia, quiet and still, the feeling of being the last person in a place that used to be full of laughter"
  差的示例："sad"（太简单）

  第 (8) 层 extra details —— 额外的画面丰富层
  让画面从"正确"升级到"有灵魂"的细节：微粒、反光、景深、材质质感、天气效果。
  好的示例："dust particles drifting through the beams of neon light, faint blurry reflection of the carousel lights on the glossy floor, shallow depth of field with bokeh balls on the background neon signs, a single old ticket caught mid-air as if just dropped"
  差的示例："some dust"（太简单）

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输出格式
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明文字、不要注释）：
{
  "scenes": [
    {
      "index": 0,
      "sceneDesc": "中文：这格画面的内容描述，1-2句，让读者脑中浮现画面",
      "imagePrompt": "English: (1) artStyle. (2) subject. (3) environment. (4) composition. (5) lighting. (6) colorPalette. (7) mood. (8) extra details. At least 70 words total. Be specific and concrete — every word should help the image model make a visual decision. Do not use vague terms.",
      "referenceKeys": ["char_main", "prop_laptop", "loc_office"]
    }
  ]
}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
硬性规则（不可违反）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

- scenes 数组恰好 %d 项，index 从 0 到 %d
- imagePrompt 必须是英文，sceneDesc 必须是中文
- 每个 imagePrompt 至少 70 个英文单词，必须覆盖上述全部 8 个视觉层
- 所有格的 artStyle 必须以相同的风格描述开头，确保视觉风格统一
- 角色外貌（发型、服装颜色款式、体型、标志性特征）在各格之间完全一致
- 相邻两格的 composition 必须有明显差异（不同景别或不同角度，不能连续两格正面中景）
- 每一格必须引用至少 2 个可追溯锚点（来自 elements 或故事正文的具体角色/物品/场景细节），禁止无依据“硬反转”
- 每一格必须包含 referenceKeys：1–5 个字符串，且必须是上方「稳定 key 列表」中的 key；若列表为空则 referenceKeys 为 []
- imagePrompt 中绝对不要出现"copy the reference image"或"exactly like the reference"——每格都应该是原创的视觉创作
- sceneDesc 中不要出现格式化标记（不要 #、不要 **、不要列表符号）
- 不要在 JSON 之外输出任何文字（包括开头和结尾的说明）`,
		sceneCount,
		narrativeHint,
		string(elementsJSON),
		vbJSON,
		keyHint,
		elemResult.Content,
		req.Style,
		req.Mood,
		aspectRatio,
		sceneCount,
		sceneCount-1)

	payloadBytes, _ := json.Marshal(map[string]interface{}{"prompt": prompt})
	aiReq := domain.AITask{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     domain.AITaskGenerateFragmentContent,
		Status:   domain.AITaskStatusProcessing,
		Provider: "",
		Input:    string(payloadBytes),
	}

	raw, tokensUsed, _, err := s.aiService.GenerateTextForFragment(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI scene expansion failed: %w", err)
	}

	scenes, perr := parseSceneExpansion(raw, sceneCount)
	if perr != nil {
		s.logger.Warn("scene expansion JSON parse failed, generating single fallback scene",
			zap.Error(perr))
		// 回退：生成一个默认场景
		fallback := fmt.Sprintf("%s, %s style, %s mood, aspect ratio %s",
			truncateRunes(elemResult.Content, 100), req.Style, req.Mood, aspectRatio)
		scenes = []fragmentExpandedScene{
			{Index: 0, SceneDesc: truncateRunes(elemResult.Content, 50), ImagePrompt: fallback},
		}
	}

	return &fragmentSceneExpansionResult{
		Scenes:     scenes,
		TokensUsed: tokensUsed,
	}, nil
}

// parseSceneExpansion 解析场景扩展 JSON 输出。
func parseSceneExpansion(raw string, want int) ([]fragmentExpandedScene, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var env struct {
		Scenes []fragmentExpandedScene `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, fmt.Errorf("parse scene expansion JSON: %w", err)
	}
	if len(env.Scenes) == 0 {
		return nil, fmt.Errorf("scene expansion returned 0 scenes")
	}
	// 校验每个场景都有 imagePrompt
	for i, sc := range env.Scenes {
		if strings.TrimSpace(sc.ImagePrompt) == "" {
			return nil, fmt.Errorf("scene %d has empty imagePrompt", i)
		}
		sc.Index = i
		env.Scenes[i] = sc
	}
	return env.Scenes, nil
}

// ────────────────────── Step 4: 逐场景生成图片（多参考图） ──────────────────────

func (s *FragmentGenerationService) generateImagesFromScenes(ctx context.Context, userID, genTaskID, aspectRatio string, scenes []fragmentExpandedScene, anchorMap map[string]string, userRefURLs []string) (*domain.FragmentImageGenerationResult, error) {
	if len(scenes) == 0 {
		return &domain.FragmentImageGenerationResult{
			ImageUrls:  []string{},
			TokensUsed: 0,
		}, nil
	}

	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}

	var allImageUrls []string
	totalTokens := 0

	for _, scene := range scenes {
		refImgs := mergeFragmentSceneReferenceImages(userRefURLs, scene, anchorMap, fragmentMaxSceneReferenceImages)
		payload := map[string]interface{}{
			"prompt":      scene.ImagePrompt,
			"aspectRatio": ar,
		}
		if len(refImgs) > 0 {
			payload["referenceImages"] = refImgs
		}
		imgInput, _ := json.Marshal(payload)

		aiReq := domain.AITask{
			ID:                uuid.New().String(),
			UserID:            userID,
			Type:              domain.AITaskGenerateFragmentImages,
			Status:            domain.AITaskStatusProcessing,
			Provider:          "",
			Input:             string(imgInput),
			RelatedEntityID:   genTaskID,
			RelatedEntityType: "fragment_generation",
		}

		imageURL, tokens, err := s.aiService.GenerateImageForFragment(ctx, &aiReq)
		if err != nil {
			s.logger.Warn("Image generation failed for scene",
				zap.Error(err),
				zap.Int("scene_index", scene.Index))
			continue
		}
		allImageUrls = append(allImageUrls, imageURL)
		totalTokens += tokens
	}

	return &domain.FragmentImageGenerationResult{
		ImageUrls:  allImageUrls,
		TokensUsed: totalTokens,
	}, nil
}

// ────────────────────── 风格/情绪/长度映射 ──────────────────────

func fragmentStyleDesc(style string) string {
	switch strings.TrimSpace(style) {
	case "fantasy":
		return "奇幻风格，充满魔法和冒险元素"
	case "realistic":
		return "现实主义风格，贴近日常生活"
	case "anime":
		return "动漫风格，角色形象鲜明"
	case "scifi":
		return "科幻风格，未来科技感"
	default:
		if style != "" {
			return fmt.Sprintf("视觉与叙事贴近「%s」风格（漫画/插画向）", style)
		}
		return "生动有趣的故事风格"
	}
}

func fragmentMoodDesc(mood string) string {
	switch strings.TrimSpace(mood) {
	case "happy":
		return "轻松愉快，积极向上"
	case "sad":
		return "感人至深，略带忧伤"
	case "mysterious":
		return "神秘莫测，引人入胜"
	case "romantic":
		return "浪漫温馨，情感细腻"
	default:
		return "情感丰富，引人共鸣"
	}
}

func fragmentLengthDesc(length string) string {
	switch strings.TrimSpace(length) {
	case "short":
		return "50-100字"
	case "medium":
		return "100-200字"
	case "long":
		return "200-500字"
	default:
		return "100-200字"
	}
}

// currentTime 获取当前时间戳
func currentTime() int64 {
	return time.Now().Unix()
}

// GetTask retrieves a generation task by ID
func (s *FragmentGenerationService) GetTask(ctx context.Context, taskID string) (*domain.FragmentGenerationTask, error) {
	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err), zap.String("task_id", taskID))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// ListTasks retrieves a list of generation tasks for a user
func (s *FragmentGenerationService) ListTasks(ctx context.Context, userID string, page, limit int) ([]*domain.FragmentGenerationTask, int64, error) {
	offset := (page - 1) * limit
	tasks, total, err := s.fragmentGenRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list tasks",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.Int("page", page),
			zap.Int("limit", limit))
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, total, nil
}

// CancelTask cancels a pending or processing generation task
func (s *FragmentGenerationService) CancelTask(ctx context.Context, taskID, userID string) error {
	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Verify ownership
	if task.UserID != userID {
		return fmt.Errorf("unauthorized: task does not belong to user")
	}

	// Only pending or processing tasks can be cancelled
	if task.Status != "pending" && task.Status != "processing" {
		return fmt.Errorf("task cannot be cancelled: current status is %s", task.Status)
	}

	// Update status to cancelled
	if err := s.fragmentGenRepo.UpdateStatus(ctx, taskID, "cancelled", task.Progress, "cancelled by user"); err != nil {
		s.logger.Error("Failed to cancel task", zap.Error(err), zap.String("task_id", taskID))
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	s.logger.Info("Task cancelled", zap.String("task_id", taskID), zap.String("user_id", userID))
	return nil
}
