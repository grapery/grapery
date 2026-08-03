package service

// structuredStoryPanelGuidance combines narrative expansion with manga visual language for fragment/storyboard prompts.
func structuredStoryPanelGuidance() string {
	return structuredNarrativeExpansionGuidance() + "\n\n" + structuredMangaLanguageGuidance()
}

// fullBleedCanvasDirective 是「成图必须满幅出血」的唯一出处，拼进发给图片模型的英文最终提示。
// App 里每张生成图都贴屏展示，模型自绘的外框/页边距会被读成 UI 缺陷而不是画风。
func fullBleedCanvasDirective() string {
	return "Canvas directive (hard requirement): the artwork must bleed off all four edges and fill the requested output aspect ratio completely. " +
		"Do NOT draw an outer panel frame, border, keyline, rounded corners, page margin, mat or passe-partout, drop shadow around the artwork, letterbox or pillarbox bars, or any solid color band along an edge. " +
		"No paper / photo / polaroid / sticker frame, no torn-paper edge, no vignette ring, no watermark, signature, logo, caption strip, or UI chrome. " +
		"If the composition uses several internal zones, keep every separation strictly inside the canvas and still let the outermost zones run off all four edges."
}

// fragmentImageNegativePrompt 是碎片配图的反向约束，只有支持 negative prompt 的 provider 会用到；
// 与 fullBleedCanvasDirective 表达同一条规则，一正一反双重兜底。
func fragmentImageNegativePrompt() string {
	return "outer frame, picture frame, border, keyline, stroke outline around the image, page margin, white margin, black margin, matting, passe-partout, letterbox bars, pillarbox bars, rounded corners, drop shadow around artwork, vignette ring, polaroid frame, photo frame, sticker border, torn paper edge, scan edge, collage background, watermark, signature, logo, UI overlay, cropped-in artwork with empty edges"
}

// fullBleedPlanningRule 是同一条约束的中文规划版，拼进中文 LLM 规划模板，
// 让模型产出的 image_prompt / composition_plan 从源头就不描述外框。
func fullBleedPlanningRule() string {
	return `- 满幅出血（硬性）：每张图都会在 App 里贴边全屏展示，因此 image_prompt / composition_plan 禁止描述外框、边框线、描边、页边距、白边黑边、上下黑条、圆角、投影、相纸/拍立得/贴纸边框；画面内容必须延伸出画布四边。
- 若这一格用多区域/子格表达，分隔只能发生在画布内部（靠留白、色块、构图线、明暗或景深切换），最外圈区域仍要出血到画布边缘。`
}

// structuredNarrativeExpansionGuidance is shared by fragment and storyboard text/planning prompts.
// Complements structuredMangaLanguageGuidance: plot causality and subject imagination, not panel ink/SFX.
func structuredNarrativeExpansionGuidance() string {
	return `【剧情拓展与叙事主体（与漫画视觉层同等重要）】
分镜不是「同一画面的多角度翻拍」，而是「用格与格推进一段说得通的小故事」。先保证每一格承担叙事功能，再落漫画镜头与网点。

0. 先在心里建立「故事脊柱」（不要单独输出）
- 主体是谁/是什么：人、动物、器物、植物、建筑、抽象概念都可以，但必须可被画出来。
- 它想要什么：回家、被看见、保护某物、逃离、证明自己、完成一次小小复仇等。
- 阻碍是什么：空间阻隔、误解、规则、竞争者、时间流逝、主人的遗忘、物理限制。
- 它做了什么：一个具体动作或策略，能在画面里看到。
- 代价/后果是什么：失去、暴露、变形、关系改变、世界规则被揭开。
- 最后一格留下什么：余韵、反讽、继续追问、情绪释放，而不是平铺直叙地解释完。

1. 叙事增量（每格必须回答）
- 读完上一格，读者应能问出一个具体问题；本格给出部分答案，同时抛出新问题或改写前提。
- 禁止连续多格只做「同一场景换机位」而无因果；若纯氛围格，须标明节奏呼吸作用，且与前后格存在情绪/信息落差。
- 格间至少贯穿一种因果链：触发→反应→后果 / 误解→揭穿 / 蓄力→爆发 / 日常→裂隙 / 追逐→落空。
- 每格至少承担一个 beat role：setup（建立局面）、inciting（触发）、attempt（尝试）、reversal（认知翻转）、cost（代价）、payoff（呼应/余韵）。不要所有格都是 setup。

2. 叙事主体多样性（不必强行出现人类）
- 主角可以是：人物、动物、拟人化器物、植物、建筑人格、抽象概念（如「遗忘」「截止日期」）。
- 无人类时：在 visualBible.characters 登记动物或拟人主体；静态物体拟人须给出可画的表演暗示（弯折的伞骨像眉、杯沿像嘴唇、贴纸眼睛、倾斜的椅背像在回头），禁止只写「一把伞很有感情」而无可见动作与目标。
- 动物漫画：保留物种可读性，用姿态、眼神、与环境关系推进剧情；拟声/对白宜短，符合物种气质。
- 静物/物怪拟人：赋予可观察的目标、恐惧或执念；用位移、争抢、追逐光斑、倾倒、开裂等动作链表达，少用大段心理说明。
- 抽象概念拟人：必须落到具体载体（影子、雾、日历页、裂开的钟面、不断复制的便利贴），并通过载体变化讲故事。

3. 剧情拓展技法（短格数也要成立）
- 2～3 格：第一格建立「表面正常+隐患」，第二格揭示规则或代价，末格反转或余韵；不要三格同一情绪强度。
- 4 格以上：允许副线、平行因果、道具主导剧情（道具比角色更会「说了算」）。
- 从用户碎片/参考图合理外推：新增元素须与已有元素产生可解释关系（同地点、同执念、同隐喻），禁止无中生有的宏大设定。
- 无对白格：用 caption、道具位移、环境变化承担剧情说明，避免格与格语义断裂。
- 优先使用小而有力的扩展：一张车票的去向、一盏灯为什么不肯熄、一只猫为什么守着空碗。不要为了“宏大”突然加入王国战争、世界毁灭、平行宇宙，除非用户已有暗示。

4. sceneDesc / caption / content 的叙事职责
- sceneDesc 与 caption 要写「这一格在剧情链上的位置」（局势、变化、悬念），不是静态陈列清单。
- 扩写正文时：在画面感之上补足动机、冲突、后果的暗示；动物/器物主角同样需要有「想要什么、失去了什么」。
- image_prompt 的 subject/action 也要承载剧情：主体的姿态、方向、距离、手/爪/边缘/裂纹变化，应能让人看出“刚发生了什么或马上要发生什么”。

5. 与漫画视觉层的分工
- 剧情层：因果、主体、目标、转折、留白。
- 漫画视觉层：paneling、气泡、SFX、网点等把上述节拍画出来。
- 二者不可偏废：禁止「剧情薄弱但特效很满」或「剧情跳跃但画面精美」。

【剧情质量自检（输出前在心里检查）】
- 如果删掉漫画效果线/网点，这组分镜仍应有一个能讲通的故事。
- 如果没有人类角色，读者仍应知道主体是谁、想要什么、遇到什么阻碍。
- 如果只看 captions / sceneDesc，格与格仍应有“因为/但是/所以”的连接。
- 结尾避免廉价万能解法：不要轻易用“原来是一场梦”“突然出现旁白解释一切”“主角直接成功”收束。`
}

// structuredMangaLanguageGuidance is shared by fragment and storyboard prompts.
// Keep it text-only: callers embed it inside larger prompt templates.
func structuredMangaLanguageGuidance() string {
	return `【漫画视觉叙事语言（必须结构化落到 prompt 字段里）】
漫画不是“插画配文字”，而是“空间化时间”的视觉叙事。生成时必须把下面六个元素转译为可执行字段，而不是只写一句 manga style：

0. 画布与出血（先于一切版式规则）
- 一次生成得到的是「一张贴边全屏展示的图」，不是「一页带页边距的漫画书内页」。
- 画面必须出血到画布四边：不描述也不绘制外框、边框线、页边距、白边黑边、圆角、投影、上下黑条。
- 分格线、gutter、子格边界只允许存在于画布内部；最外圈的格必须被画布边缘裁掉，而不是被一圈边框包住。

1. Paneling / Koma（分镜语法）
- 分镜大小、形状、排列控制读者脑内时间：大框=慢镜头/关键时刻，小碎框=快节奏动作。
- composition / composition_plan 必须说明内部分区、跨格、阅读顺序、视线移动速度；若是单图多区域，写清上下/左右/网格与内部 gutter（仍不含外框）。

2. Speech bubbles（声音视觉化）
- dialogue=椭圆对白气泡，tail 指向 speaker；thought=云朵/串泡；低语可用虚线，喊叫可用锯齿气泡。
- comicTexts 中只放短句，不把 caption 整段塞进气泡；imagePrompt 必须要求 render exact Chinese text inside the image。

3. SFX & effect lines（拟声词与效果线）
- sfx 是画面元素，不是注释；用 bold blocky lettering、brushy lettering、jagged lettering 等说明字形。
- 动作/爆发/大喊/击碎/坠落/冲撞等语义触发 impact package：action lines、radial speed lines、motion streaking、impact burst、particles/debris/sparks。
- 语气词/反应词也是漫画节奏：疑惑/震惊用「啊？」「诶？」；沉默/压迫用「……」；期待用「要来了」「终于」；庆祝用「太好了！」；必须短、准、可读，不随机造无关文字。

3.5. Emotional beat staging（重点情绪场）
- turning_point（主要转折）：用大格/强剪影/突变光色/停顿旁白，把“局面变了”的瞬间视觉化。
- shock（震惊）：用极近景、瞳孔高光、汗滴、速度线/放射线、背景抽离、粗黑阴影、短促 sfx/interjection。
- anticipation（期待）：用留白、视线朝向画外、门缝/包裹/倒计时/手指停顿、低饱和静默、旁白框制造悬念。
- celebration（庆祝）：用上扬构图、暖色光、碎纸/星形高光/群像反应、开阔空间、短对白或欢呼 SFX。
- inner_monologue（心理描写）：用云朵思想泡、低饱和背景、脸部近景/手部细节/眼神方向，不把长段心理活动塞进气泡。

4. Character design & iconography（角色符号化）
- 角色外形、服饰、发型、标志物必须稳定且可被读者瞬间识别。
- 需要时加入漫画表情符号：sweat drop、anger mark、shock lines、tiny reaction marks，但不能遮挡脸和关键道具。

5. Tones & shading（网点、黑块、阴影）
- 黑白/漫画倾向时，优先写 dynamic screentones、heavy black ink masses、cross-hatching、dramatic etching、halftone texture。
- 冲击感模式使用 high contrast、chiaroscuro shading、deep shadows、noir lighting、gritty texture，避免 soft / bland / gentle-only 描述。

6. Gutter / closure（沟壑与闭合）
- gutter 不是空白装饰，而是让读者补全动作的时间缝隙；它只出现在格与格之间，不出现在画布与画面之间。
- 相邻格之间要设计“前一瞬/后一瞬”的闭合关系：挥刀→倒下、开门前→门后异常、拳头蓄力→碎片飞散。不要把所有动作解释完。

【结构化字段要求】
- 全局层：artStyle / styleBible.artStyle 负责媒介、线稿、网点、阴影、质感与调色。
- 镜头层：shot_type 或 composition 必须显式包含 shot scale + camera angle；相邻格不得重复同一组合。
- 版式层：layout_intent / composition_plan / composition 负责 panel grid、内部分隔、gutter、reading order；一律满幅出血，不写外框。
- 动作层：subject/action/entityBindings 负责角色位置、动作爆发瞬间、道具归属。
- 漫画元素层：comicTexts、additionalNotes 或 imagePrompt 必须落入 bubbles、SFX、effect lines、negative space for lettering；border breaking 仅当画布内部确实存在子格时才可用，不得为此凭空加一圈外框。
- 冲击感触发：检测到战斗、爆发、大喊、击碎、坠落、追逐、撞击、恐惧、高潮等语义时，必须自动加入 extreme low-angle / dramatic high-angle / Dutch angle / wide-angle distortion / radial action lines / debris / sparks / heavy ink contrast 中的合适组合。`
}
