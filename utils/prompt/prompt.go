package prompt

const (
	ZhipuStoryGeneratePrompt = `Please enter the story name: {{name}}, 
	and the author name: {{author}}`
	ZhipuStoryBoardPrompt    = `Please enter the story board name: {{name}}`
	ZhipuStoryBoardBackgroud = `Please enter the story board background: {{background}}`

	ZhipuStoryScenePrompt      = `Please enter the story scene name: {{name}}`
	ZhipuStorySceneImagePrompt = `Please enter the story scene image: {{image}}`

	ZhipuStoryRolePrompt      = `Please enter the story role name: {{name}}`
	ZhipuStoryRoleImagePrompt = `Please enter the story role image: {{image}}`

	ZhipuNegativePrompt = `Please enter the negative sentence: {{sentence}}`
	ZhipuPositivePrompt = `Please enter the positive sentence: {{sentence}}`
)
