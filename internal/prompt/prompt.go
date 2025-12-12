package prompt

const (
	// 1. 生成故事文本的正面提示词
	StoryTextPositivePrompt = `Generate a creative and engaging story with the following details:
	Story Name: {{name}}
	Author: {{author}}
	Background: {{background}}
	
	Please create a compelling narrative that captures the reader's imagination and maintains a positive, uplifting tone throughout the story.`

	// 2. 生成故事文本的负面提示词
	StoryTextNegativePrompt = `Avoid the following elements in story generation:
	- Inappropriate or offensive content
	- Violence, gore, or disturbing imagery
	- Hate speech or discriminatory language
	- Copyrighted material or plagiarism
	- Overly simplistic or clichéd plotlines
	- Inconsistent character development
	- Poor grammar or unclear writing style`

	// 3. 生成故事图片的正面提示词
	StoryImagePositivePrompt = `Create a beautiful and artistic illustration for the story scene:
	Scene Name: {{name}}
	Scene Description: {{description}}
	
	Style: High-quality, detailed artwork with vibrant colors, proper lighting, and engaging composition. The image should capture the mood and atmosphere of the story scene.`

	// 4. 生成故事图片的负面提示词
	StoryImageNegativePrompt = `Avoid the following in image generation:
	- Low quality, blurry, or pixelated images
	- Inappropriate or adult content
	- Violence, gore, or disturbing imagery
	- Text or watermarks in the image
	- Poor composition or awkward poses
	- Inconsistent art style
	- Copyrighted characters or logos`

	// 5. 生成故事视频的正面提示词
	StoryVideoPositivePrompt = `Create an engaging video sequence for the story:
	Scene Name: {{name}}
	Scene Description: {{description}}
	
	Style: Smooth camera movements, high-quality visuals, appropriate pacing, and engaging storytelling through visual narrative. The video should bring the story to life with dynamic scenes and emotional impact.`

	// 6. 生成故事视频的负面提示词
	StoryVideoNegativePrompt = `Avoid the following in video generation:
	- Low resolution or poor video quality
	- Inappropriate or adult content
	- Violence, gore, or disturbing scenes
	- Copyrighted music or audio
	- Jarring camera movements or poor editing
	- Inconsistent visual style
	- Overly long or boring sequences
	- Poor audio quality or inappropriate sound effects`
)

/*

为实现细腻的创意控制，建议在提示词中包含以下维度：

主体（Subject）
明确图像中的核心对象或角色。
示例：一位眼神冷峻、拥有发光蓝色光学元件的机器人咖啡师
构图（Composition）
指定镜头视角与取景方式。
示例：低角度广角镜头、人像特写、电影级POV视角
动作（Action）
描述正在发生的动态行为。
示例：正在施展火焰魔法、在雨中奔跑、搅拌一杯豆蔻奶茶
地点（Location）
设定场景环境与空间背景。
示例：火星殖民地的露天咖啡馆、维多利亚时代图书馆、黄金时刻的麦田
风格（Style）
定义整体美学与艺术表现形式。
示例：3D Pixar动画风格、黑色电影（film noir）、水彩插画、90年代产品摄影
编辑指令（Edit Directive）
针对已有图像进行具体修改。
示例：将背景中的汽车移除、把衬衫颜色改为靛蓝色
细节控制（Advanced Details）
包含专业级摄影与设计参数：
纵横比（如 9:16、21:9）
相机设置（如 f/1.8 浅景深、50mm 镜头）
灯光与氛围（如黄金时刻逆光、赛博朋克霓虹补光）
色彩分级（如青橙电影色调、高对比度黑白）
文本集成（如“EXPLORE”以粗体无衬线白色字体置于顶部）
事实约束（如“确保19世纪服饰历史准确性”）
参考图角色说明（如“图像A作为角色姿态，图像B作为风格参考”）
*/
