package genapi

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MediaGenerationProvider 是图片与视频统一落地的 provider（Seedream 出图 / Seedance 出片）。
const MediaGenerationProvider = "huoshan"

// mediaGenerationDeniedProviders 列出不再承担出图/出片的 provider。
// Gemini（含 Imagen 与 Veo）仍用于文本与多模态理解，但媒体生成一律改判到火山。
// 策略放在分发口而不是各调用点，这样遗漏的老调用点与将来新增的调用点都无法绕过。
var mediaGenerationDeniedProviders = map[string]struct{}{
	normalizeProviderName(string(ProviderGemini)): {},
}

// MediaGenerationDenied 判断某个 provider 是否已被排除在出图/出片之外。
func MediaGenerationDenied(name string) bool {
	_, denied := mediaGenerationDeniedProviders[normalizeProviderName(name)]
	return denied
}

func mediaGenerationDenied(name string) bool {
	return MediaGenerationDenied(name)
}

// resolveImageGenerationProvider 返回真正用于出图的 provider，并说明是否发生了改判。
func (g *GenAPI) resolveImageGenerationProvider(requested string) (string, bool) {
	return g.resolveMediaProvider(requested, func(name string) bool {
		g.mu.RLock()
		defer g.mu.RUnlock()
		_, ok := g.imageProviders[name]
		return ok
	})
}

// resolveVideoGenerationProvider 返回真正用于出片的 provider，并说明是否发生了改判。
func (g *GenAPI) resolveVideoGenerationProvider(requested string) (string, bool) {
	return g.resolveMediaProvider(requested, func(name string) bool {
		g.mu.RLock()
		defer g.mu.RUnlock()
		_, ok := g.videoProviders[name]
		return ok
	})
}

func (g *GenAPI) resolveMediaProvider(requested string, isRegistered func(string) bool) (string, bool) {
	name := normalizeProviderName(requested)
	if name == "" {
		name = normalizeProviderName(MediaGenerationProvider)
	}
	if !mediaGenerationDenied(name) {
		return name, false
	}
	target := normalizeProviderName(MediaGenerationProvider)
	if g == nil || !isRegistered(target) {
		// 目标 provider 不可用时保持原样，让上层报出「未注册」，而不是静默回到被禁的 provider。
		return name, false
	}
	return target, true
}

// foreignMediaModelMarkers 标记其它厂商的模型族。provider 被改判后必须清掉这类 model id，
// 否则方舟会收到一个它不认识的模型名而直接失败。
var foreignMediaModelMarkers = []string{"gemini", "imagen", "veo", "nano-banana", "nano_banana"}

// dropForeignMediaModel 清空不属于当前 provider 的模型名，让 provider 回落到自己的默认模型。
func dropForeignMediaModel(req *GenerateRequest) {
	if req == nil {
		return
	}
	model := strings.ToLower(strings.TrimSpace(req.Model))
	if model == "" {
		return
	}
	for _, marker := range foreignMediaModelMarkers {
		if strings.Contains(model, marker) {
			req.Model = ""
			return
		}
	}
}

// huoshanImageSize 取调用方给定的像素尺寸；只给了长宽比时按比例换算，避免落到 provider 默认尺寸。
func huoshanImageSize(req *GenerateRequest) string {
	if req == nil {
		return ""
	}
	if size := strings.TrimSpace(req.Size); size != "" {
		return size
	}
	if aspect := strings.TrimSpace(req.AspectRatio); aspect != "" {
		return HuoshanPixelSizeForAspectRatio(aspect)
	}
	return ""
}

// huoshanMinImagePixels 是 Seedream 组图接口的总像素下限（方舟当前要求约 2560×1440）。
const huoshanMinImagePixels = 3686400

const (
	huoshanMinImageSide = 512
	huoshanMaxImageSide = 4096
)

// HuoshanPixelSizeForAspectRatio 把 "W:H" 换算成方舟接受的像素尺寸。
// 方舟只认像素尺寸、不认长宽比，所以只给了 AspectRatio 的调用方需要在这里补齐，
// 否则会拿到 provider 默认尺寸、比例全错。
func HuoshanPixelSizeForAspectRatio(aspectRatio string) string {
	w, h := parseAspectRatio(aspectRatio)
	scale := math.Sqrt(float64(huoshanMinImagePixels) / (w * h))
	return fmt.Sprintf("%dx%d", clampImageSide(w*scale), clampImageSide(h*scale))
}

// parseAspectRatio 解析 "16:9" 这类比例，无法解析时回退到 16:9。
func parseAspectRatio(aspectRatio string) (float64, float64) {
	parts := strings.Split(strings.TrimSpace(aspectRatio), ":")
	if len(parts) == 2 {
		w, wErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		h, hErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if wErr == nil && hErr == nil && w > 0 && h > 0 {
			return w, h
		}
	}
	return 16, 9
}

// clampImageSide 向上取到 8 的倍数以避免像素数落回下限之下，再夹到方舟允许的边长区间。
func clampImageSide(side float64) int {
	v := int(math.Ceil(side/8)) * 8
	if v < huoshanMinImageSide {
		return huoshanMinImageSide
	}
	if v > huoshanMaxImageSide {
		return huoshanMaxImageSide
	}
	return v
}
