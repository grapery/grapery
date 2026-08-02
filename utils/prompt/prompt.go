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
