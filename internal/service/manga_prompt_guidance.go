package service

// structuredMangaLanguageGuidance is shared by fragment and storyboard prompts.
// Keep it text-only: callers embed it inside larger prompt templates.
func structuredMangaLanguageGuidance() string {
	return `【漫画视觉叙事语言（必须结构化落到 prompt 字段里）】
漫画不是“插画配文字”，而是“空间化时间”的视觉叙事。生成时必须把下面六个元素转译为可执行字段，而不是只写一句 manga style：

1. Paneling / Koma（分镜语法）
- 分镜大小、形状、排列控制读者脑内时间：大框=慢镜头/关键时刻，小碎框=快节奏动作。
- composition / composition_plan 必须说明格框、跨格、阅读顺序、视线移动速度；若是单图多区域，写清上下/左右/网格与 gutter。

2. Speech bubbles（声音视觉化）
- dialogue=椭圆对白气泡，tail 指向 speaker；thought=云朵/串泡；低语可用虚线，喊叫可用锯齿气泡。
- comicTexts 中只放短句，不把 caption 整段塞进气泡；imagePrompt 必须要求 render exact Chinese text inside the image。

3. SFX & effect lines（拟声词与效果线）
- sfx 是画面元素，不是注释；用 bold blocky lettering、brushy lettering、jagged lettering 等说明字形。
- 动作/爆发/大喊/击碎/坠落/冲撞等语义触发 impact package：action lines、radial speed lines、motion streaking、impact burst、particles/debris/sparks。
- 语气词/反应词也是漫画节奏：疑惑/震惊用「啊？」「诶？」；沉默/压迫用「……」；期待用「要来了」「终于」；庆祝用「太好了！」；必须短、准、可读，不随机造无关文字。

3.5. Emotional beat staging（重点情绪场）
- turning_point（主要转折）：用大格/断裂边框/强剪影/突变光色/停顿旁白，把“局面变了”的瞬间视觉化。
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
- gutter 不是空白装饰，而是让读者补全动作的时间缝隙。
- 相邻格之间要设计“前一瞬/后一瞬”的闭合关系：挥刀→倒下、开门前→门后异常、拳头蓄力→碎片飞散。不要把所有动作解释完。

【结构化字段要求】
- 全局层：artStyle / styleBible.artStyle 负责媒介、线稿、网点、阴影、质感与调色。
- 镜头层：shot_type 或 composition 必须显式包含 shot scale + camera angle；相邻格不得重复同一组合。
- 版式层：layout_intent / composition_plan / composition 负责 panel grid、border、gutter、reading order。
- 动作层：subject/action/entityBindings 负责角色位置、动作爆发瞬间、道具归属。
- 漫画元素层：comicTexts、additionalNotes 或 imagePrompt 必须落入 bubbles、SFX、effect lines、border breaking、negative space for lettering。
- 冲击感触发：检测到战斗、爆发、大喊、击碎、坠落、追逐、撞击、恐惧、高潮等语义时，必须自动加入 extreme low-angle / dramatic high-angle / Dutch angle / wide-angle distortion / radial action lines / debris / sparks / heavy ink contrast 中的合适组合。`
}
