package service

// structuredStoryPanelGuidance is intentionally short. It is consumed by planning
// models, not by image models; provider-specific rendering constraints are added
// only when the final image prompt is assembled.
func structuredStoryPanelGuidance() string {
	return structuredNarrativeExpansionGuidance() + "\n\n" + structuredMangaLanguageGuidance()
}

// fullBleedCanvasDirective is the single source of truth for the final image canvas.
func fullBleedCanvasDirective() string {
	return "Canvas directive (hard requirement): the artwork must bleed off all four edges and fill the requested output aspect ratio completely. " +
		"Do NOT draw an outer panel frame, border, keyline, rounded corners, page margin, mat or passe-partout, drop shadow around the artwork, letterbox or pillarbox bars, or any solid color band along an edge. " +
		"No paper / photo / polaroid / sticker frame, no torn-paper edge, no vignette ring, no watermark, signature, logo, caption strip, or UI chrome. " +
		"If the composition uses several internal zones, keep every separation strictly inside the canvas and still let the outermost zones run off all four edges."
}

func fragmentImageNegativePrompt() string {
	return "outer frame, picture frame, border, keyline, stroke outline around the image, page margin, white margin, black margin, matting, passe-partout, letterbox bars, pillarbox bars, rounded corners, drop shadow around artwork, vignette ring, polaroid frame, photo frame, sticker border, torn paper edge, scan edge, collage background, watermark, signature, logo, UI overlay, cropped-in artwork with empty edges"
}

func fullBleedPlanningRule() string {
	return `- 满幅出血：画面内容延伸出画布四边；禁止外框、页边距、白边黑边、圆角、投影、相纸或贴纸边框。
- 多区域构图的分隔只发生在画布内部，最外圈区域仍须出血到边缘。`
}

func structuredNarrativeExpansionGuidance() string {
	return `【叙事规划契约】
- 忠实于用户输入与已提取的视觉事实；补充内容必须能由现有角色、地点、道具或主题合理推出。
- 先确定主体、目标、阻碍、可见动作和后果，再分配到各格；不要输出这段内部推理。
- 每格只承担一个主要节拍：建立、触发、尝试、转折、代价或回响；相邻格的局势、认知或情绪至少变化一项。
- 氛围格可以不推进事件，但必须改变节奏或信息，并且不能连续出现。
- 动机与情绪必须落到可画的姿态、距离、方向、表情、道具或环境变化，禁止用抽象形容词代替画面。
- 结尾回应前文并留下余韵；除非用户已有暗示，不使用梦境揭晓、旁白解释一切或突然成功等万能收束。`
}

func structuredMangaLanguageGuidance() string {
	return `【分镜规划契约】
- 同一角色、地点和关键道具的不可变特征保持一致；剧情造成的变化必须在当前格说明。
- sceneDesc/caption 写本格发生的变化；imagePrompt 写可执行视觉事实，不复述整段剧情。
- composition/shot_type 明确景别、机位、主体位置和阅读顺序；相邻格不要重复同一景别与角度组合。
- layout 只在确有两个以上同时发生的节拍时使用内部子格，否则优先单一连续场景；不得为了“像漫画”机械分格。
- comicTexts 是唯一允许出现的图中文字来源：短、少、与剧情有关，不得生成额外文字。dialogue/thought 的 speaker 必须绑定本格角色。
- 图片模型无法可靠绘制指定字形时，省略字形和空气泡/空旁白框，仅保留自然留白；不得用乱码或近似字替代。
- 动作高潮使用与语义匹配的低角度、倾斜机位、速度线、碎屑或强对比；安静场景不要堆叠冲击特效。
- 输出前检查：只看各格 caption 仍能读出因果，只看 imagePrompt 仍能画出不同且连续的画面。`
}
