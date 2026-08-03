package service

import (
	"encoding/json"
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

// fragmentPanelPlanLayoutAddon 将客户端指定的对白选项并入规划提示（仅多格流水线）。
func fragmentPanelPlanLayoutAddon(req domain.FragmentPanelGenerationRequest) string {
	var parts []string
	switch strings.TrimSpace(req.DialogueMode) {
	case "none":
		parts = append(parts, "本任务：各格画面不要出现对白气泡或旁白框。")
	case "auto":
		parts = append(parts, "在叙事需要时使用椭圆形对白气泡或旁白框；caption 可与气泡文案呼应。")
	case "from_user_input":
		parts = append(parts, "对白尽量来自用户文字；caption 使用自然中文并与气泡一致。")
	}
	return strings.Join(parts, "\n")
}

func panelPlanLayoutWithVisualEvidence(layoutAddon string, evidence []domain.FragmentVisualEvidence) string {
	if len(evidence) == 0 {
		return layoutAddon
	}
	b, err := json.Marshal(evidence)
	if err != nil {
		return layoutAddon
	}
	parts := []string{strings.TrimSpace(layoutAddon)}
	parts = append(parts, "参考图多模态视觉事实 JSON：\n"+string(b)+"\n分镜规划必须优先沿用这些可见事实；若创作补充未在图片中出现，不得写入 visualBible 的 immutableTraits。")
	return strings.TrimSpace(strings.Join(parts, "\n"))
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
		// 多格输出 JSON 体量更大，压低单格下限以减少输出被截断、panels 数量不足。
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

%s

【一、世界观一致性】
- 若多格出现同一角色，外貌（发型、服装颜色款式、体型、标志特征）应一致，除非剧情明确改变（换装、受伤等）。
- 空间与地貌保持可辨识连续性，避免无因果的跳变。
- 各格 image_prompt 的 artStyle 开头应统一，形成同一套视觉签名。

【二、剧情拓展（硬性，与漫画视觉层并重）】
- 每一格须携带叙事增量：相对上一格，局势、认知或情绪至少有一项可感知变化；禁止连续多格仅换机位、剧情静止。
- 整组 panels 须串成微型因果链；主角可以是人物、动物、拟人器物或静物，须有可观察的目标/恐惧/执念。
- 静物/物怪拟人：用可画的姿态与动作链表达（倾斜、滚落、开裂、争抢光斑），visualBible 登记拟人主体 name。
- 从用户文字与参考图合理外推，禁止与锚点无关的设定硬塞。
- 输出前在心里给每格分配一个 beat role（无需新增 JSON 字段）：setup / inciting / attempt / reversal / cost / payoff。caption、composition_plan、image_prompt 的 subject/action 必须体现该格 role。
- %d 格中至少要出现一个“局势改变”的格（inciting/reversal/cost 之一）；不能全部是 setup 或纯氛围。

【三、叙事与镜头自由度】
- 鼓励跳切、闪回、视角换位、静帧氛围格、象征画面；氛围格须承担节奏呼吸，与前后格存在信息或情绪落差。

【四、构图多样性（硬性）】
- 相邻两格不得使用相同「景别+角度」组合；理想情况是连续三格内不重复。
- 景别与角度工具：特写/近景/中景/全景/远景、平视/俯拍/仰拍/Dutch angle/鸟瞰/虫视角等；可从非人类视角制造惊喜。

【五、自动布局决策（每格必须独立判断）】
- 每一格仍是「一次文生图得到的一张贴边全屏图」，但图片内部可以是：(A) 单一连续场景；或 (B) 多个区域/子格（条漫式分区、上下分镜、左右对照、2×2 等），用留白、粗线或清晰边界分隔，并交待阅读顺序。按剧情选 A 或 B，不要为每格机械重复同一种版式。
- 每格必须输出 layout_intent、composition_plan、shot_type、visual_hierarchy。
- layout_intent 使用简短英文 snake_case，例如：single_subject_focus、split_foreground_background、wide_establishing、diagonal_motion、symmetrical_faceoff、detail_insert、layered_depth、negative_space_tension、comic_single_panel、comic_two_panel_grid、comic_strip、intra_image_multi_panel、stacked_vertical_zones、split_screen_two_beat、grid_four_beat。
- composition_plan 用中文或英文自然语言写清「区域怎么分、每块放什么」：若多区域，说明上下/左右/网格位置、每区主体与动作、gutter/间距、阅读顺序；若单场景，说明主体位置、前中后景、留白、引导线、视觉重心。
- visual_hierarchy 说明主视觉、次视觉、背景信息的优先级，避免所有元素平均铺开。
- shot_type 使用英文短语，例如 close_up、medium_shot、wide_shot、overhead、low_angle、dutch_angle、detail_insert；多区域时可用 wide_shot 概括整图或注明 per-zone。
- 布局必须服务该格剧情功能（例如铺垫+反转可在一张图内用上下两区完成）。
- 这四个字段是“文本阶段的前置漫画规划”，后续图片阶段会直接消费：不得留空、不得用模板占位、不得所有格重复同一值。
- 若故事含「冲击/对抗/追逐/坠落/爆发」语义，至少两格必须在 layout_intent 或 composition_plan 中显式体现冲击镜头语法（如 diagonal_motion、extreme_angle、impact_burst、radial_lines、subject_overflowing_frame_edge）。
- 如该格适合漫画表达，必须在 composition_plan / image_prompt 中写清内部分区、gutter、气泡预留位置，并输出 comic_texts：narration=旁白框、dialogue=角色对白气泡、thought=内心气泡、sfx=拟声/语气音效字。
- comic_texts 中的中文文字是最终图片中要直接画出来的文字，不是给 App 叠加的占位数据；image_prompt 必须明确要求图片模型 render the exact Chinese text inside the image。
- 数量上限：每格最多 1 narration、1-2 dialogue、最多 1 sfx、最多 1 thought；每条中文建议不超过 12 个汉字；禁止额外随机文字。

【六、光影与色彩】
- 格间可改变光型以配合情绪，但要可解释；色温与饱和度变化应服务于叙事走向。

【七、caption 写法】
- 每格 caption 为一句简洁中文：叙事主体、在做什么、相对上一格局势如何变化、何种氛围；不要写成章回标题或纯静态陈列。

【八、image_prompt 写法（英文，给文生图/参考生图模型）】
- 必须按以下 8 层依次写成一个连贯英文段落，层与层之间用句号分隔；至少 %d 个英文单词，覆盖全部 8 层，禁止空泛词。
  (1) artStyle — 具体技法混合，勿只写 "anime" / "illustration"。
  (2) subject — 谁/什么在画中，外貌、姿态、表情、手持物。
  (3) environment — 完整空间与层次。
  (4) composition — 景别+角度+重心+引导线；若为单图多区域，则说明分区方式、各区内容与阅读顺序。
  (5) lighting — 光源方向、类型、色温、阴影与高光。
  (6) colorPalette — 主色、点缀、对比、分布。
  (7) mood — 复合情绪，勿单一形容词。
  (8) extra details — 微粒、反光、景深、材质、天气等提升质感的细节。

【九、视觉圣经 visualBible（与普通故事碎片 Step1 JSON 字段名一致，必须输出）】
- visualBible 与 panels、参考图、用户文字必须自洽；immutableTraits 使用与用户文字相同的自然语言（中文或英文，与用户输入一致）。
- characters 最多 3 项，props 最多 5 项，locations 1–2 项；每项必须有全局唯一 key（小写英文+下划线，如 char_main、prop_bag、loc_cafe）。
- **每个 characters[] 条目必须包含非空 name：**与 captions、用户文字中出现的称呼保持一致优先；若无姓名则用与用户语言一致的简短识别名（如中文 2～8 字），供下游「故事角色」展示。**禁止不写 name，或仅占位符号。** locations / props 若包含 name 字段也需可称呼的简称。
- immutableTraits 为字符串数组：每条描述一个不可随意更改的视觉事实。
- characters 可以是人、动物、拟人器物、植物或抽象概念的可视化载体；若主体不是人，immutableTraits 必须写清物种/材质/轮廓/拟人表演特征，避免后续被画成人类。
- styleBible.artStyle 必须用英文写出可执行的总体画法（媒介、线稿/渲染、时代感），供各格 image_prompt 的 artStyle 层对齐；其他 styleBible 字段可选。

【十、reference_keys（每格）】
- 每一格必须包含 reference_keys：1–5 个字符串，且必须来自 visualBible 中已声明的 key；若无任何资产 key 可引用（极少见）则该格 reference_keys 为 []。
- 禁止自造不在 visualBible 中出现的 key。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输出格式（仅此 JSON，不要 markdown 围栏、不要前后解释）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{"visualBible":{"styleBible":{"artStyle":"English overall art direction"},"characters":[{"key":"char_main","name":"与 captions/用户一致的称呼","immutableTraits":["..."]}],"props":[],"locations":[]},"panels":[{"index":0,"image_prompt":"English eight-layer description as one paragraph, min %d words","caption":"一句中文","reference_keys":["char_main"],"layout_intent":"wide_establishing","composition_plan":"主体在左下三分之一，远处环境占据右上区域，前景物形成遮挡和纵深。","shot_type":"wide_shot","visual_hierarchy":"主视觉：角色轮廓；次视觉：关键道具；背景：地点氛围","comic_texts":[{"type":"narration","text":"旁白短句","position":"top-left"},{"type":"dialogue","text":"角色台词","speaker":"char_main","position":"speech-bubble"},{"type":"sfx","text":"砰！","position":"mid-frame"}]},{"index":1,"image_prompt":"...","caption":"一句中文","reference_keys":["char_main"],"layout_intent":"comic_two_panel_grid","composition_plan":"单图垂直分为上下两区：上区特写手部与钥匙；下区中景同一角色推门，中区 gutter 隔开，先读上再读下，并在下区右上角预留对白气泡。","shot_type":"wide_shot","visual_hierarchy":"上区手部第一；下区全身动作第二","comic_texts":[]}, ...]}

硬性规则：
- visualBible 必须存在且包含 styleBible.artStyle；characters、props、locations 可为空数组但键必须存在；若有任何 character 条目，该项 **必须包含非空的 name 字段**。
- "panels" 数组恰好 %d 项，index 依次为 0 到 %d。
- image_prompt：仅英文；每格至少 %d 词；八层齐全；各格 artStyle 描述应一致；禁止在每格都要求「像素级复制参考图」——锚定身份与氛围，鼓励每格有独立构图与叙事增量。
- layout_intent、composition_plan、shot_type、visual_hierarchy 必须存在且服务当前格剧情，不得所有格重复。
- captions 连起来必须像一个可读的小故事；至少三格时，不能每句都是“某主体在某地看着某物”的静态句式。
- composition_plan 必须说明这一格为何采用该布局来服务剧情功能（例如制造误会、揭示代价、放大失败、保留悬念），不是只描述画面摆放。
- 不允许“先不规划，后续生图再决定漫画元素”的写法；漫画相关结构必须在本 JSON 一次性给全。
- 若某一格采用单图内多区域/子格，composition_plan 与 image_prompt 的 composition 层须一致写出分区、gutter、阅读顺序。
- comic_texts 可为空数组；若 dialogue/thought 存在，speaker 必须是该格 reference_keys 中的角色 key；文字必须短（建议 <=12 汉字），不要把整段 caption 放入气泡；每格最多 1 narration、1-2 dialogue、最多 1 sfx、最多 1 thought；所有 comic_texts 都必须作为图中文字直接绘制在最终图片内，且不允许额外随机文字。
- caption：仅中文，每格一行，无 # 号、无 markdown。
- 相邻两格 image_prompt 中的 composition（景别+角度）必须明显不同。
- 不要输出 JSON 之外的任何字符。`,
		panelCount,
		narr,
		panelCount-1,
		st,
		styleDesc,
		ui,
		structuredStoryPanelGuidance(),
		panelCount,
		minWords,
		minWords,
		panelCount,
		panelCount-1,
		minWords,
	)
	body += "\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n画布与出血（硬性，优先级高于任何版式偏好）\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" + fullBleedPlanningRule() + "\n"
	if a := strings.TrimSpace(layoutAddon); a != "" {
		body += "\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n版式与对白（用户指定；须融入分镜规划与 caption）\n" + a + "\n"
	}

	return renderPromptDSL(PromptDSL{
		Role:         "你是一位漫画分镜导演与结构化提示词工程师。",
		Task:         "根据参考图锚点与用户文字，输出 panels[] 与 visualBible 的结构化 JSON。",
		Inputs:       map[string]any{"userInput": ui, "styleSlug": st, "styleDesc": styleDesc, "panelCount": panelCount, "minWordsEach": minWords, "layoutAddon": strings.TrimSpace(layoutAddon), "narrativeHint": narr},
		GlobalConfig: structuredStoryPanelGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Paneling / Camera / Action / Comic Elements Rules", Kind: "text", Body: body},
		},
	})
}
