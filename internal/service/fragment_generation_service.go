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

// processFragmentGeneration 处理碎片生成流程：
// Step 1: 元素提取 + 文案生成（一次 LLM 调用）
// Step 2: 场景扩展（将元素+文案扩展为 N 个场景，每个场景含独立的图片提示词）
// Step 3: 逐场景生成图片
func (s *FragmentGenerationService) processFragmentGeneration(ctx context.Context, taskID string) {
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 5, "starting")

	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		return
	}

	// ── Step 1: 元素提取 + 文案生成 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 10, "extracting_elements")

	elemResult, err := s.extractElementsAndGenerateContent(ctx, task.UserID, task.Request)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Element extraction failed: %v", err))
		return
	}

	resolvedAR := domain.NormalizeFragmentAspectRatio(task.Request.AspectRatio)
	if resolvedAR == "" {
		resolvedAR = domain.NormalizeFragmentAspectRatio(elemResult.AspectRatio)
	}
	if resolvedAR == "" {
		resolvedAR = domain.FragmentAspectDefault
	}

	// ── Step 2: 场景扩展 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 35, "expanding_scenes")

	sceneCount := task.Request.ImageCount
	if sceneCount <= 0 {
		sceneCount = 1
	}
	scenesResult, err := s.expandScenes(ctx, task.UserID, task.Request, elemResult, sceneCount, resolvedAR)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Scene expansion failed: %v", err))
		return
	}

	// ── Step 3: 逐场景生成图片 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 55, "generating_images")

	imageResult, err := s.generateImagesFromScenes(ctx, task.UserID, taskID, resolvedAR, scenesResult.Scenes)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Image generation failed: %v", err))
		return
	}

	// ── 汇总结果 ──
	result := &domain.FragmentGenerationResult{
		Content:     elemResult.Content,
		ImageUrls:   imageResult.ImageUrls,
		AspectRatio: resolvedAR,
		TokensUsed:  elemResult.TokensUsed + scenesResult.TokensUsed + imageResult.TokensUsed,
	}

	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
		s.logger.Error("Failed to update result", zap.Error(err))
	}

	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "completed", 100, "completed")

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
			if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
				s.logger.Warn("Failed to persist draftFragmentId on generation task", zap.Error(err))
			}
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
			if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
				s.logger.Warn("Failed to persist draftFragmentId on generation task", zap.Error(err))
			}
		}
	}

	s.logger.Info("Fragment generation completed",
		zap.String("task_id", taskID),
		zap.Int("tokens_used", result.TokensUsed),
		zap.Int("image_count", len(result.ImageUrls)),
		zap.Int("scenes", len(scenesResult.Scenes)))
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
	Elements    fragmentStoryElements `json:"elements"`
	Content     string                `json:"content"`
	AspectRatio string                `json:"aspectRatio"`
	TokensUsed  int
}

// fragmentStoryElements 从用户输入和参考图中提取的结构化元素。
type fragmentStoryElements struct {
	Weather     string   `json:"weather"`     // 天气：晴朗、雨天、暴风雪等
	Objects     []string `json:"objects"`     // 关键物品
	Scenes      []string `json:"scenes"`      // 场景类型：室内/室外/森林/城市等
	TimeOfDay   string   `json:"timeOfDay"`   // 时间：清晨/正午/黄昏/深夜等
	Location    string   `json:"location"`    // 地点描述
	Characters  []string `json:"characters"`  // 人物/角色描述
	Tendency    string   `json:"tendency"`    // 情感倾向/叙事方向
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

	raw, tokensUsed, inferredAR, err := s.aiService.GenerateTextForFragment(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI extraction+story generation failed: %w", err)
	}

	result, perr := parseExtractionResult(raw)
	if perr != nil {
		s.logger.Warn("extraction JSON parse failed, using raw text as content",
			zap.Error(perr),
			zap.String("snippet", truncateRunes(raw, 120)))
		// 回退：JSON 解析失败时把原文作为 content
		result = &fragmentElementExtractionResult{
			Content:     truncateRunes(raw, 500),
			AspectRatio: inferredAR,
		}
	}
	if result.AspectRatio == "" {
		result.AspectRatio = inferredAR
	}
	result.TokensUsed = tokensUsed
	return result, nil
}

// buildExtractionAndStoryPrompt 构建元素提取 + 文案生成的组合提示词。
func (s *FragmentGenerationService) buildExtractionAndStoryPrompt(req domain.FragmentGenerationRequest, hasImages bool) string {
	styleDesc := fragmentStyleDesc(req.Style)
	moodDesc := fragmentMoodDesc(req.Mood)
	lengthDesc := fragmentLengthDesc(req.Length)

	imgNote := ""
	if hasImages {
		imgNote = `

【参考图分析】用户提供了参考图，请像导演分析 storyboard 一样拆解画面：

第一步——整体印象（2 秒内你看到了什么）：
- 画面的情绪基调是什么？（温暖/压抑/神秘/欢快/孤独……）
- 视线最先被什么吸引？为什么？（色彩对比？位置？大小？）

第二步——拆解视觉元素：
- 人物/主体：外观细节（发型、服装款式与材质、体型、年龄感）、姿态（站/坐/跑/蜷缩）、表情或情绪暗示、与镜头的关系（正面/侧脸/背影/剪影）
- 环境：室内还是室外、建筑风格或自然地貌、季节和温度感、前景/中景/背景的层次
- 光影：光源位置和方向（侧光/逆光/顶光/散射光）、色温（暖黄/冷蓝/中性白）、阴影的形状和浓淡
- 色彩：主色调和点缀色、饱和度高低、是否有明显的色彩对比或渐变
- 构图：景别（特写/近景/中景/远景）、角度（平视/俯拍/仰拍）、画面重心位置
- 细节彩蛋：画面中容易被忽略但有趣的细节（墙上的贴纸、口袋里露出的纸条、远处的剪影）

第三步——将视觉信息翻译为故事元素：
- 把画面中的人物外貌提取为 characters 的视觉特征
- 把环境和光影转化为 scenes 和 weather 的氛围描述
- 把色彩和构图转化为 tendency 的情绪方向
- 确保生成的故事与参考图有画面上的呼应，不是只基于文字做文字推理`
	}

	return fmt.Sprintf(`你是一位擅长在短篇幅中制造惊喜的故事碎片创作助手，同时具备电影美术指导的画面感知力。你需要完成两件事：

━━━━━━━━━━━━━━━━━━ 第一件事：元素提取 ━━━━━━━━━━━━━━━━━━

从用户的文字描述%s中提炼出以下故事元素。注意：元素不是关键词罗列，而是带有画面感和叙事暗示的具体描述——想象你要把这些元素交给一位从未读过原文的插画师，让他凭这些描述就能画出对的画面。

- weather（天气）：不只是"晴/雨"，要写出天气如何影响画面和情绪。好的示例："暴雨将至，天色发黄，空气黏稠得像裹了一层保鲜膜"；差的示例："阴天"。如果天气不重要就写"无特别天气"，但优先把天气当作情绪的放大器。
- objects（物品）：画面中会出现的关键物品（最多 5 个）。每个物品写清：外观+材质+状态+与角色的空间关系。好的示例："一把半透明的红伞，伞面有几道裂痕，被随手靠在咖啡店门边"；差的示例："伞"。
- scenes（场景）：故事发生的空间（最多 3 个）。写出空间的感官细节——不只是视觉，还有声音、气味、温度、触感。好的示例："老式电车内部，木质座椅磨得发亮，窗外是模糊的樱花隧道，车厢里有铁锈和甜食混合的味道"；差的示例："电车上"。
- timeOfDay（时间）：时段 + 该时段特有的光线和氛围。好的示例："黄昏，太阳刚好卡在两栋楼之间，整条街被染成橘红色，路人的影子拖得很长"；差的示例："傍晚"。
- location（地点）：具体到有辨识度的地点。包含时代感（现代/复古/未来）、空间尺度（开阔/逼仄/纵深）、空间特征（材质、色彩、标志性元素）。好的示例："一个被废弃的室内游乐场，旋转木马还在缓慢转，彩灯只剩红和蓝还亮着"。
- characters（人物）：角色的视觉身份卡（最多 3 个）。包含：体型轮廓、穿着（款式+颜色+材质）、标志性特征（疤/纹身/配饰/走路姿势）、当前状态（在做什么/表情/情绪）。好的示例："瘦高的女生，穿oversized黑卫衣帽子压得很低，手里攥着一杯已经凉透的拿铁，站在斑马线中间像在犹豫要不要过去"。
- tendency（倾向）：用一句话概括这段故事想传达的核心感受。不是分类标签，而是"如果这是一张专辑封面，封面上只有这一句话"。好的示例："所有出口都标着入口的方向"；差的示例："悬疑风格"。

━━━━━━━━━━━━━━━━━━ 第二件事：碎片故事 ━━━━━━━━━━━━━━━━━━

基于提取的元素，写一个让人看完想转发的故事碎片。

用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s

故事心法（按重要性排序）：

【第一句就是钩子】
- 不要用"有一天""在很久以前""那是一个"开头——这些是读者滑走的信号
- 好的开头示例："钥匙插进锁孔的时候，她发现门是从里面锁着的。"
- 用动作、对话、或者一个反常的细节开局

【用画面讲故事，不用形容词堆砌】
- ❌ "她感到无比悲伤和孤独"  →  ✅ "她把第二碗面推到对面空着的座位前"
- ❌ "天空非常美丽壮观"    →  ✅ "云层裂开一道缝，光柱打在她脚边半米处"
- 动词 > 形容词，具体名词 > 抽象概念，可观察的行为 > 内心独白

【留白是最强的笔法】
- 最好的结尾不是句号，而是让读者自己脑补下一秒发生了什么
- 不要解释你的隐喻——如果读者需要你解释，说明隐喻不够好
- 可以在最高潮处戛然而止，比写完整个结局更有余韵

【一个意外的转折就够了】
- 不需要反转再反转，一个"等等，什么？"的时刻比三个小反转有效
- 转折最好来自前面埋下的细节，而不是凭空出现
- 可以是视角的转折（"我"其实不是人类）、时间的转折（这已经是回忆）、身份的转折

【世界观用细节建立，不要用旁白解释】
- ❌ "这是一个魔法与科技并存的世界"  →  ✅ "咖啡杯飘在半空，她懒得去接"
- 一个反常的日常细节比三句世界观说明更有说服力
- 让读者自己拼凑规则，不要喂给他们

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
{
  "elements": {
    "weather": "天气氛围描述（中文）",
    "objects": ["物品1：具体描述（外观+材质+状态）", "物品2"],
    "scenes": ["场景1：感官细节（视觉+听觉+嗅觉+触感）", "场景2"],
    "timeOfDay": "时段+光线氛围（中文）",
    "location": "地点+时代感+空间特征（中文）",
    "characters": ["角色1：外观+穿着+标志性特征+当前状态", "角色2"],
    "tendency": "核心感受一句话——像专辑封面上的一行字（中文）"
  },
  "content": "碎片故事正文（中文），直接可读，不要标题",
  "aspectRatio": "推荐长宽比：1:1、16:9、9:16、3:4、4:3，结合画面构图选择；不确定用 16:9"
}`,
		imgNote,
		req.UserInput, styleDesc, moodDesc, lengthDesc, req.Language)
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
	return &out, nil
}

// ────────────────────── Step 2: 场景扩展 ──────────────────────

// fragmentSceneExpansionResult 场景扩展输出。
type fragmentSceneExpansionResult struct {
	Scenes     []fragmentExpandedScene `json:"scenes"`
	TokensUsed int
}

// fragmentExpandedScene 扩展出的单个场景，含中文描述和英文图片提示词。
type fragmentExpandedScene struct {
	Index         int    `json:"index"`
	SceneDesc     string `json:"sceneDesc"`     // 中文场景描述（面向读者）
	ImagePrompt   string `json:"imagePrompt"`   // 英文图片生成提示词（面向图片模型）
}

func (s *FragmentGenerationService) expandScenes(ctx context.Context, userID string, req domain.FragmentGenerationRequest, elemResult *fragmentElementExtractionResult, sceneCount int, aspectRatio string) (*fragmentSceneExpansionResult, error) {
	elementsJSON, _ := json.Marshal(elemResult.Elements)

	var narrativeHint string
	switch {
	case sceneCount == 1:
		narrativeHint = "1 格：这一帧必须是整条故事中最有视觉冲击力的瞬间。不要选平淡的叙述时刻——选悬念最浓的那一秒、反转刚发生的那一帧、或者一个让人立刻想问\"之前到底发生了什么\"的定格。想象电影海报：一个画面就让观众脑补出一整部电影。让这一帧的构图、光影、角色表情本身就在讲故事。"
	case sceneCount == 2:
		narrativeHint = "2 格：核心技法是\"认知落差\"——第一格建立预期，第二格打破它。可以用的落差类型：视角落差（第一格是人眼中的世界，第二格是虫子眼中的同一场景）、情绪落差（温馨 → 惊悚，平静 → 狂喜）、尺度落差（微观 → 宏观，局部 → 全景）、时间落差（现在 → 十年前，白天 → 深夜）。观众看完两格后的反应应该是\"等等，怎么会这样？\"或\"所以第一格里那个其实是……\"。"
	case sceneCount == 3:
		narrativeHint = "3 格：不要三幕式！三幕式会产出四平八稳但无趣的内容。自由节奏才是王道——第一格抛出引子（一个画面就让人好奇），第二格可以突然转向（换视角、换时空、换叙事者、或者一个完全出乎意料的元素闯入），第三格收束但不给答案——留一个让人回味的尾巴。可以尝试的结构：假结局（第三格看似结束但其实暗示了更大的故事）、环形结构（第三格呼应第一格但信息量不同）、打破第四面墙（角色意识到了读者的存在）。惊喜感比工整重要十倍。"
	default:
		narrativeHint = fmt.Sprintf("%d 格：一条故事线但绝不要线性平铺直叙。在格与格之间可以大胆穿插——视角突变（从人类视角突然切到猫的视角、从地面仰视突然变卫星俯瞰）、时空跳跃或闪回（这一格还是现在，下一格突然是十年前，再下一格又回到现在但时间已过）、打破第四面墙（角色看向\"镜头\"外、文字溢出画框）、超现实梦境（前一秒还在教室，下一秒教室漂浮在星空中）、意料之外的新元素闯入（一个路过的蝴蝶改变了整条故事线）。开头建立世界观和角色，中间尽情放飞制造\"没想到\"的惊喜，结尾可以呼应开头、可以留悬念、可以推翻之前所有设定——只要让观众看完觉得\"这趟旅程值了\"。", sceneCount)
	}

	prompt := fmt.Sprintf(`你是一位脑洞大开、同时精通电影摄影和漫画分镜的视觉故事创作者。你的任务是把一段故事文字拆解为 %d 个画面格——每一格都是一幅能独立吸引目光、按顺序浏览又能串联成完整故事的视觉作品。

━━━━━━━━━━━━ 叙事节奏指引 ━━━━━━━━━━━━

%s

━━━━━━━━━━━━ 输入素材 ━━━━━━━━━━━━

故事元素（从用户输入中提取）：
%s

故事正文：
%s

风格：%s
情绪：%s
画面长宽比：%s

━━━━━━━━━━━━ 创作方法论 ━━━━━━━━━━━━

【世界观一致性】
- 角色的外貌（发型、服装、体型、标志性特征）必须在每一格中完全一致——除非剧情需要角色发生变化
- 空间设定（室内布局、城市天际线、自然地貌）保持可辨识的连续性
- artStyle（艺术风格）全程统一——如果第一格是\"水彩插画风\"，最后一格也必须是水彩插画风

【叙事自由度】
- 允许并且鼓励：跳切（时间突然推进）、闪回（突然回到过去）、视角反转（突然换成配角的视角）、超现实片段（梦境/幻觉/想象）、意外闯入的新元素（一个路过的角色改变了走向）
- 每一格都可以是不同类型的画面：写实场景、意象画面、特写细节、全景鸟瞰、角色内心世界的可视化
- 不必每一格都在推进剧情——有时一格氛围画面（雨中的空椅子、黄昏的空荡街道）比剧情画面更有力量

【构图多样性——拒绝重复】
每格必须使用不同的镜头语言，以下是可用的构图类型库（混搭使用）：
- 景别：特写（眼睛/手/物品细节）、近景（胸部以上）、中景（半身）、全景（全身+环境）、远景（人物渺小）、极远景（地标级画面）
- 角度：平视、俯拍（上帝视角）、仰拍（角色显得高大/压迫）、Dutch angle（倾斜，制造不安感）、鸟瞰（正上方往下看）、虫眼（贴地往上拍）
- 视角：人类正常视角、猫/狗视角（低角度）、鸟的视角（高空俯瞰）、鱼的视角（水下往上看）、物品视角（钥匙孔视角、镜子中的倒影、手机屏幕视角）
- 硬性规则：相邻两格不能使用相同的景别+角度组合

【光影与色彩——情绪的调色板】
- 光影是情绪最强大的工具：逆光=神秘/英雄感、侧光=戏剧冲突/立体感、顶光=压抑/审判感、散射光=日常/平静、剪影=未知/威胁
- 色温配合情绪弧线：暖色调（安全/怀旧/温馨）→ 冷色调（疏离/科技/忧郁）、高饱和（活力/梦幻）→ 低饱和（压抑/回忆/末日）
- 光源方向和强度在格间可以变化，但要服务于情绪走向——比如故事走向紧张，光线可以从明亮逐步变暗

【sceneDesc 写法——面向读者的画面描述】
- 一到两句话，让读者在脑中\"看到\"这一格的画面
- 要传达：谁在画面里、在做什么、什么氛围、在故事的哪个节点
- 不要写成剧情概要，要写成\"如果你闭上眼睛想象这一格，你会看到什么\"
- 好的示例：\"她站在空荡荡的站台，身后是已经驶远的列车尾灯，手里还攥着没来得及递出去的信\"
- 差的示例：\"第二幕，她错过了火车\"

【imagePrompt 写法——面向图片模型的视觉指令】
- 这是在给一位顶级插画师下 brief，必须具体到每个视觉决策
- 必须按以下 8 层结构依次描述，用句号分隔：

  (1) artStyle：整体艺术风格——不是笼统的\"anime\"，而是具体的混合风格
      示例：\"cinematic watercolor with digital color grading\"
      示例：\"charcoal sketch style with selective watercolor highlights\"
      示例：\"hyperrealistic 3D render with soft bloom lighting\"

  (2) subject：画面中的核心主体——谁/什么在画面中，具体到外观和姿态
      要包含：角色外貌（发型发色、服装款式和颜色、体型）、姿态（站/坐/跑/蹲）、表情或情绪暗示
      示例：\"a tall woman in an oversized black hoodie with the hood pulled low, standing still at the center of a crosswalk, holding a cold takeaway coffee cup with both hands, her expression unreadable\"

  (3) environment：背景和场景——角色所处的空间
      要包含：空间类型、建筑或自然特征、深度暗示（前景/中景/背景各有什么）
      示例：\"an abandoned indoor amusement park, faded carousel slowly rotating in the background, only red and blue neon lights still working, dusty floor with scattered tickets\"

  (4) composition：构图——镜头如何框取画面
      要包含：景别+角度+画面重心位置
      示例：\"medium shot from a low angle, subject positioned at right third, carousel filling the left background\"

  (5) lighting：光照——光源如何塑造画面
      要包含：光源方向、色温、阴影特征、高光位置
      示例：\"red and blue neon lights casting colored shadows from the left, face partially in shadow with blue highlight on cheekbone, no natural light\"

  (6) colorPalette：色彩方案——画面的色彩构成
      要包含：主色调、点缀色、对比度
      示例：\"dominant dark teal and navy blue, accent warm red from neon sign, high contrast between light sources and deep shadows\"

  (7) mood：情绪氛围——这一格应该让观众感受到什么
      示例：\"melancholic solitude with a hint of nostalgia, quiet and still\"

  (8) extra details：额外细节——让画面更丰富的元素
      示例：\"dust particles floating in the neon light, faint reflection on the glossy floor, shallow depth of field with bokeh on background lights\"

━━━━━━━━━━━━ 输出格式 ━━━━━━━━━━━━

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明文字、不要注释）：
{
  "scenes": [
    {
      "index": 0,
      "sceneDesc": "中文：这格画面的内容描述，1-2句，让读者脑中浮现画面",
      "imagePrompt": "English: structured visual description following the 8-layer format above (artStyle. subject. environment. composition. lighting. colorPalette. mood. extra details). At least 80 words. Be specific and concrete — every word should help the image model make a visual decision."
    }
  ]
}

━━━━━━━━━━━━ 硬性规则 ━━━━━━━━━━━━

- scenes 数组恰好 %d 项，index 从 0 到 %d
- imagePrompt 必须是英文，sceneDesc 必须是中文
- 每个 imagePrompt 至少 80 个英文单词，必须覆盖上述 8 个视觉层
- 所有格的 artStyle 描述必须以相同的风格前缀开头，确保视觉风格统一
- 角色外貌（发型、服装颜色款式、体型）在各格之间完全一致
- 相邻两格的 composition 必须有明显差异（不同景别或不同角度，不能连续两格正面中景）
- imagePrompt 中不要出现\"copy the reference image\"或\"exactly like the reference\"之类的指令
- sceneDesc 中不要出现格式化标记（不要 #、不要 **、不要列表）`,
		sceneCount,
		narrativeHint,
		string(elementsJSON),
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

// ────────────────────── Step 3: 逐场景生成图片 ──────────────────────

func (s *FragmentGenerationService) generateImagesFromScenes(ctx context.Context, userID, genTaskID, aspectRatio string, scenes []fragmentExpandedScene) (*domain.FragmentImageGenerationResult, error) {
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
		imgInput, _ := json.Marshal(map[string]string{
			"prompt":      scene.ImagePrompt,
			"aspectRatio": ar,
		})

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
