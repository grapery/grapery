package service

import (
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// fragmentPanelGeminiReferenceImagePreamble Gemini 多模态：附图前的中文锚点说明（与碎片「参考图理解」语义对齐）。
const fragmentPanelGeminiReferenceImagePreamble = "上方为用户参考图。请先在心里完成锚点判断（不要单独输出成段说明）：" +
	"识别主体（人物/物体/建筑/自然物）、室内外空间、时间感与光线、主色与情绪、构图重心；" +
	"判断用户可能希望保留的身份特征与世界氛围。" +
	"将该图视为故事世界与视觉风格的锚点，而不是要求每一格都像素级复刻同一 photograph。"

// panelPlanNarrativeRhythm 与 fragment_generation expandScenes 同源的分格叙事节奏指引。
func panelPlanNarrativeRhythm(panelCount int) string {
	switch {
	case panelCount == 1:
		return "1 格：这一帧必须是整条故事中最有视觉冲击力的瞬间。不要选平淡的叙述时刻——选悬念最浓的那一秒、反转刚发生的那一帧、或者一个让人立刻想问\"之前到底发生了什么\"的定格。想象电影海报：一个画面就让观众脑补出一整部电影。让这一帧的构图、光影、角色表情本身就在讲故事。若故事有高潮，这就是高潮被凝固的那一毫秒；若没有高潮，就选最让人不安的那一秒——一切看似正常但有什么不太对。"
	case panelCount == 2:
		return "2 格：核心技法是\"认知落差\"——第一格建立预期，第二格打破它。可用：视角落差、情绪落差、尺度落差、时间落差。观众看完应产生\"等等，怎么会这样？\"第二格的第一反应是意外，第二反应才是理解。"
	case panelCount == 3:
		return "3 格：不要呆板三幕式。第一格抛引子，第二格可突然转向（换视角/时空/闯入元素），第三格收束但可留白。可用假结局、环形呼应、打破第四面墙、时间嵌套。惊喜感比工整重要。"
	default:
		return fmt.Sprintf("%d 格：一条故事线但不要线性平铺。格间可穿插视角突变、时空闪回、超现实片段、画外元素闯入；可穿插纯氛围格（停顿即节奏）。建立世界与角色后，中间制造\"没想到\"，结尾可呼应、留悬念或推翻设定——让读者觉得这趟视觉旅程值得。", panelCount)
	}
}

// fragmentPanelPlanLayoutAddon 将客户端指定的版式/对白选项并入规划提示（仅多格流水线）。
func fragmentPanelPlanLayoutAddon(req domain.FragmentPanelGenerationRequest) string {
	var parts []string
	switch strings.TrimSpace(req.LayoutPreset) {
	case "strip5_top2_middle_wide_bottom2":
		parts = append(parts, "版式目标：竖版 5 格条漫节奏——第 1 行两格并排、第 2 行一条全宽横条大格、第 3 行两格并排；分镜顺序与信息递进应符合该阅读流（各格仍是独立插图，构图留出条漫呼吸感）。")
	}
	switch strings.TrimSpace(req.GutterStyle) {
	case "white_thin":
		parts = append(parts, "格间留白：想象细白 gutter 的现代条漫分隔。")
	case "black_thin":
		parts = append(parts, "格间描边：想象细黑线分隔的经典漫画 gutter。")
	}
	switch strings.TrimSpace(req.DialogueMode) {
	case "none":
		parts = append(parts, "本任务：各格画面不要出现对白气泡或旁白框。")
	case "auto":
		parts = append(parts, "在叙事需要时使用椭圆形对白气泡或旁白框；caption 可与气泡文案呼应。")
	case "from_user_input":
		parts = append(parts, "对白尽量来自用户文字；caption 使用自然中文并与气泡一致。")
	}
	if o := strings.TrimSpace(req.OutputMode); o != "" {
		parts = append(parts, fmt.Sprintf("输出策略（供规划理解）：%s。", o))
	}
	return strings.Join(parts, "\n")
}

// buildFragmentPanelPlanUserPrompt 分镜规划主提示词：与 fragment_generation_service.expandScenes 方法论同构，输出仍为 panels[].image_prompt + caption JSON。
func buildFragmentPanelPlanUserPrompt(userInput, style string, panelCount int, layoutAddon string) string {
	ui := strings.TrimSpace(userInput)
	st := strings.TrimSpace(style)
	if st == "" {
		st = "fantasy"
	}
	styleDesc := fragmentStyleDesc(st)
	narr := panelPlanNarrativeRhythm(panelCount)
	minWords := 70
	if panelCount >= 5 {
		// 5 格（含竖版条漫版式）单行 image_prompt 更长，压低下限以减少输出被截断、panels 数量不足。
		minWords = 52
	}

	body := fmt.Sprintf(`你是一位脑洞大开、同时精通电影摄影和漫画分镜的视觉故事导演，兼具叙事把控力、摄影师的画面敏感度和概念艺术家的想象力。用户提供了「一张参考图」+「文字意向」。你的任务是规划 %d 个分镜 panel（连环画式故事碎片）：每一格都是一幅能独立抓眼、按顺序读又能连成全片的视觉作品。你不是在「把参考图描 N 遍」，而是在「以参考图为世界锚点，用画面推进故事」。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
叙事节奏指引
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
阶段一 · 参考图理解（内心完成，不要写成单独长篇散文输出）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

按下列清单在脑中完成判断（可直接用于后续分镜，无需复述）：
- 主体层：人物/生物/物体的外观、姿态、服装与标志性特征；是否有遮挡、背影、局部特写。
- 环境层：室内/室外、空间层次（前中后景）、建筑或自然地标、季节与温度暗示。
- 光影层：主光源方向与类型、色温、阴影软硬、高光与轮廓。
- 色彩层：主色、点缀色、饱和度倾向。
- 构图层：景别（特写/中景/全景等）、视角（平拍/俯仰等）、画面重心。
- 用户意图猜测：用户想保留什么（角色身份、旅行/场景氛围、情绪基调），哪些可以自由延展。

阶段二 · 分镜与用户文字融合
- 将用户文字（中文或英文）与你的画面理解合并：故事应从参考图延展、多样化——新时刻、新动作、新景别、新时间感；保持世界观与可识主体的连续性。
- 禁止每一格都只是「微调同一 pose 的参考照片」。第 0 格可较贴近锚图建立局面，但仍需有导演级构图选择；第 1～%d 格必须明显改变景别、机位、动作或叙事节拍，除非用户明确要求重复。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输入
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

风格倾向（slug，下游配图会沿用）：%s
风格说明：%s

用户文字：
%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
创作方法论（与「碎片多场景扩写」对齐的精简版）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

【一、世界观一致性】
- 若多格出现同一角色，外貌（发型、服装颜色款式、体型、标志特征）应一致，除非剧情明确改变（换装、受伤等）。
- 空间与地貌保持可辨识连续性，避免无因果的跳变。
- 各格 image_prompt 的 artStyle 开头应统一，形成同一套视觉签名。

【二、叙事与镜头自由度】
- 鼓励跳切、闪回、视角换位、静帧氛围格、象征画面；不必格格推进剧情——氛围格可以是节奏的一部分。

【三、构图多样性（硬性）】
- 相邻两格不得使用相同「景别+角度」组合；理想情况是连续三格内不重复。
- 景别与角度工具：特写/近景/中景/全景/远景、平视/俯拍/仰拍/Dutch angle/鸟瞰/虫视角等；可从非人类视角制造惊喜。

【四、光影与色彩】
- 格间可改变光型以配合情绪，但要可解释；色温与饱和度变化应服务于叙事走向。

【五、caption 写法】
- 每格 caption 为一句简洁中文，让读者一眼明白这一格在故事中的画面感（谁在做什么、何种氛围），不要写成剧情提纲或章回标题。

【六、image_prompt 写法（英文，给文生图/参考生图模型）】
- 必须按以下 8 层依次写成一个连贯英文段落，层与层之间用句号分隔；至少 %d 个英文单词，覆盖全部 8 层，禁止空泛词。
  (1) artStyle — 具体技法混合，勿只写 "anime" / "illustration"。
  (2) subject — 谁/什么在画中，外貌、姿态、表情、手持物。
  (3) environment — 完整空间与层次。
  (4) composition — 景别+角度+重心+引导线。
  (5) lighting — 光源方向、类型、色温、阴影与高光。
  (6) colorPalette — 主色、点缀、对比、分布。
  (7) mood — 复合情绪，勿单一形容词。
  (8) extra details — 微粒、反光、景深、材质、天气等提升质感的细节。

【七、视觉圣经 visualBible（与普通故事碎片 Step1 JSON 字段名一致，必须输出）】
- visualBible 与 panels、参考图、用户文字必须自洽；immutableTraits 使用与用户文字相同的自然语言（中文或英文，与用户输入一致）。
- characters 最多 3 项，props 最多 5 项，locations 1–2 项；每项必须有全局唯一 key（小写英文+下划线，如 char_main、prop_bag、loc_cafe）。
- immutableTraits 为字符串数组：每条描述一个不可随意更改的视觉事实。
- styleBible.artStyle 必须用英文写出可执行的总体画法（媒介、线稿/渲染、时代感），供各格 image_prompt 的 artStyle 层对齐；其他 styleBible 字段可选。

【八、reference_keys（每格）】
- 每一格必须包含 reference_keys：1–5 个字符串，且必须来自 visualBible 中已声明的 key；若无任何资产 key 可引用（极少见）则该格 reference_keys 为 []。
- 禁止自造不在 visualBible 中出现的 key。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输出格式（仅此 JSON，不要 markdown 围栏、不要前后解释）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{"visualBible":{"styleBible":{"artStyle":"English overall art direction"},"characters":[{"key":"char_main","immutableTraits":["..."]}],"props":[],"locations":[]},"panels":[{"index":0,"image_prompt":"English eight-layer description as one paragraph, min %d words","caption":"一句中文","reference_keys":["char_main"]}, ...]}

硬性规则：
- visualBible 必须存在且包含 styleBible.artStyle；characters、props、locations 可为空数组但键必须存在。
- "panels" 数组恰好 %d 项，index 依次为 0 到 %d。
- image_prompt：仅英文；每格至少 %d 词；八层齐全；各格 artStyle 描述应一致；禁止在每格都要求「像素级复制参考图」——锚定身份与氛围，鼓励每格有独立构图与叙事增量。
- caption：仅中文，每格一行，无 # 号、无 markdown。
- 相邻两格 image_prompt 中的 composition（景别+角度）必须明显不同。
- 不要输出 JSON 之外的任何字符。`,
		panelCount,
		narr,
		panelCount-1,
		st,
		styleDesc,
		ui,
		minWords,
		minWords,
		panelCount,
		panelCount-1,
		minWords,
	)
	if a := strings.TrimSpace(layoutAddon); a != "" {
		body += "\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n版式与对白（用户指定；须融入分镜规划与 caption）\n" + a + "\n"
	}
	return body
}
