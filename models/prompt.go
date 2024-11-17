package models

const (
	ComicStyle  = "Japanese Anime"
	CinanaStyle = "Manga"
	AnimeStyle  = "Anime"
)

const (
	LayoutClassicStyle   = "Classic Comic Style"
	LayoutFourPanelStyle = "Four Pannel"
)

const (
	NegativePrompt = `
	bad anatomy, bad hands, missing fingers, extra fingers, three hands, three legs, bad arms, missing legs, missing arms, poorly drawn face, bad face, fused face, cloned face, three crus, fused feet, fused thigh, extra crus, ugly fingers, horn, cartoon, cg, 3d, unreal, animate, amputation, disconnected limbs
	`
	PositivePrompt = ``
)

type PromptTemplate struct {
	Name           string `json:"name"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt"`
}

var PreDefineTemplate = []PromptTemplate{
	{
		Name:           "(No style)",
		Prompt:         "{prompt}",
		NegativePrompt: "",
	},
	{
		Name:           "Japanese Anime",
		Prompt:         "anime artwork illustrating {prompt}. created by japanese anime studio. highly emotional. best quality, high resolution",
		NegativePrompt: "low quality, low resolution",
	},
	{
		Name:           "Cinematic",
		Prompt:         "cinematic still {prompt} . emotional, harmonious, vignette, highly detailed, high budget, bokeh, cinemascope, moody, epic, gorgeous, film grain, grainy",
		NegativePrompt: "anime, cartoon, graphic, text, painting, crayon, graphite, abstract, glitch, deformed, mutated, ugly, disfigured",
	},
	{
		Name:           "Disney Charactor",
		Prompt:         "A Pixar animation character of {prompt} . pixar-style, studio anime, Disney, high-quality",
		NegativePrompt: "lowres, bad anatomy, bad hands, text, bad eyes, bad arms, bad legs, error, missing fingers, extra digit, fewer digits, cropped, worst quality, low quality, normal quality, jpeg artifacts, signature, watermark, blurry, grayscale, noisy, sloppy, messy, grainy, highly detailed, ultra textured, photo",
	},
	{
		Name:           "Photographic",
		Prompt:         "cinematic photo {prompt} . 35mm photograph, film, bokeh, professional, 4k, highly detailed",
		NegativePrompt: "drawing, painting, crayon, sketch, graphite, impressionist, noisy, blurry, soft, deformed, ugly",
	},
	{
		Name:           "Comic book",
		Prompt:         "comic {prompt} . graphic illustration, comic art, graphic novel art, vibrant, highly detailed",
		NegativePrompt: "photograph, deformed, glitch, noisy, realistic, stock photo",
	},
	{
		Name:           "Line art",
		Prompt:         "line art drawing {prompt} . professional, sleek, modern, minimalist, graphic, line art, vector graphics",
		NegativePrompt: "anime, photorealistic, 35mm film, deformed, glitch, blurry, noisy, off-center, deformed, cross-eyed, closed eyes, bad anatomy, ugly, disfigured, mutated, realism, realistic, impressionism, expressionism, oil, acrylic",
	},
}

type Prompt struct {
	IDBase
	Background     string
	Content        string
	NegativePrompt string
	PositivePrompt string
	TemplateId     int64
	UserID         int64
	Platform       string
	GenStatus      int
	StartTime      int64
	FinishTime     int64
	TokenInput     int64
	TokenOutput    int64
}

func NewPrompt() *Prompt {
	return &Prompt{}
}
func (p *Prompt) TableName() string {
	return "prompt"
}
