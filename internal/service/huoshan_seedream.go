package service

import (
	"strconv"
	"strings"

	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
	huoshanpkg "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
)

// 火山方舟 Seedream 组图官方约束摘要（doubao-seedream-5-0）。
const (
	HuoshanMaxMultiReferenceImages = 14 // 多参考：2～14 张
	huoshanRefPlusGeneratedBudget   = 15 // 参考张数 + 生成张数 ≤ 15（多参考组图）
	maxGeneratedTextOnlySet         = 15 // 文生组图上限
	maxGeneratedSingleRefSet        = 14 // 单参考组图上限
	huoshanSequentialAuto           = "auto"
	huoshanSequentialDisabled       = "disabled"
)

func mergeStringOptions(base map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(patch) == 0 {
		return nil
	}
	out := make(map[string]interface{})
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}

// ClampHuoshanMultiReferenceURLs 裁剪参考图至多 14 张（方舟多参考上限）；保持顺序与非空字符串。
func ClampHuoshanMultiReferenceURLs(urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	out := make([]string, 0, min(len(urls), HuoshanMaxMultiReferenceImages))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		out = append(out, u)
		if len(out) >= HuoshanMaxMultiReferenceImages {
			break
		}
	}
	return out
}

// OptimizeHuoshanReferencesAndOutputs 满足：多参考时可先裁参考再在仍超限时降低生成数以满足（参考数+生成数≤15）。
// wantOutputs 为目标生成张数；返回调整后的 refs 与张数。
func OptimizeHuoshanReferencesAndOutputs(urls []string, wantOutputs int) ([]string, int) {
	if wantOutputs < 1 {
		wantOutputs = 1
	}
	refs := ClampHuoshanMultiReferenceURLs(urls)
	refc := len(refs)

	// 多于 14 已由 Clamp；单张/无双参考走下面 cap。
	for refc > 2 && wantOutputs > huoshanMaxOutputsForRefCount(refc) {
		refs = refs[:refc-1]
		refc--
	}

	capOut := huoshanMaxOutputsForRefCount(refc)
	if wantOutputs > capOut {
		wantOutputs = capOut
	}
	if wantOutputs < 1 {
		wantOutputs = 1
	}
	return refs, wantOutputs
}

func huoshanMaxOutputsForRefCount(refc int) int {
	switch {
	case refc <= 0:
		return maxGeneratedTextOnlySet
	case refc == 1:
		return maxGeneratedSingleRefSet
	default:
		// refc ∈ [2,14]
		budget := huoshanRefPlusGeneratedBudget - refc
		if budget < 1 {
			budget = 1
		}
		return budget
	}
}

func computeHuoshanSeedreamPatch(refCount int, textToImage bool, batchHint int, _ map[string]interface{}, forceHuoshanBatchSemantics bool) map[string]interface{} {
	want := batchHint
	if want < 1 {
		want = 1
	}
	// 故事板 / 碎片在火山上一律走组图（sequential auto + *Set），单张亦然（max_images=1）。
	multiImageSet := forceHuoshanBatchSemantics || want > 1

	var mode string
	var seq string
	var cappedMax int

	if multiImageSet {
		seq = huoshanSequentialAuto
		cappedMax = min(want, huoshanMaxOutputsForRefCount(refCount))
		switch {
		case refCount <= 0:
			mode = huoshanpkg.Seedream50ModeTextSet
		case refCount == 1:
			mode = huoshanpkg.Seedream50ModeImageSingleSet
		default:
			mode = huoshanpkg.Seedream50ModeImageMultiSet
		}
	} else {
		seq = huoshanSequentialDisabled
		cappedMax = 1
		switch {
		case refCount <= 0 || textToImage:
			mode = huoshanpkg.Seedream50ModeTextSingle
		case refCount == 1:
			mode = huoshanpkg.Seedream50ModeImageSingleSingle
		default:
			mode = huoshanpkg.Seedream50ModeImageMultiSingle
		}
	}

	patch := map[string]interface{}{
		"mode":                        mode,
		"sequential_image_generation": seq,
	}

	if seq == huoshanSequentialAuto {
		patch["max_images"] = cappedMax
		patch["maxImages"] = cappedMax
		patch["sequential_image_generation_options"] = map[string]interface{}{
			"max_images": cappedMax,
			"maxImages":  cappedMax,
		}
	} else {
		// 清空组图 options，避免历史 options 遗留 max_images / auto 语义
		patch["sequential_image_generation_options"] = nil
		patch["max_images"] = 1
		patch["maxImages"] = 1
	}
	return patch
}

// HuoshanUseBatchSemantics 故事板场景图、碎片配图等在火山上一律按组图链路（batch）请求，与张数是否为 1 无关。
func HuoshanUseBatchSemantics(relatedEntityType string, meta map[string]interface{}) bool {
	t := strings.ToLower(strings.TrimSpace(relatedEntityType))
	switch t {
	case "fragment_generation", "fragment_panel_generation", "storyboard_scene", "storyboard":
		return true
	}
	if len(meta) == 0 {
		return false
	}
	for _, key := range []string{"related_entity_type", "relatedEntityType"} {
		raw, ok := meta[key].(string)
		if ok && HuoshanUseBatchSemantics(strings.TrimSpace(raw), nil) {
			return true
		}
	}
	return false
}

// PrepareHuoshanGenerateImageRequest 统一：裁剪参考图、在满足「参考+生成≤15」下调整期望张数、注入 Seedream mode/sequential/max_images。
func PrepareHuoshanGenerateImageRequest(req *GenerateImageRequest) {
	if req == nil || !strings.EqualFold(strings.TrimSpace(req.Provider), "huoshan") {
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = huoshanpkg.DefaultHuoshanImageModelID
	}
	hint := req.OutputCount
	if hint < 1 {
		hint = 1
	}
	refs, out := OptimizeHuoshanReferencesAndOutputs(req.ReferenceImages, hint)
	req.ReferenceImages = refs
	req.OutputCount = out

	refc := len(refs)
	textToImage := refc == 0
	force := HuoshanUseBatchSemantics(strings.TrimSpace(req.RelatedEntityType), req.Metadata)
	patch := computeHuoshanSeedreamPatch(refc, textToImage, req.OutputCount, nil, force)
	req.Options = mergeStringOptions(req.Options, patch)
}

// PrepareHuoshanGenAPIImageRequest 直连 GenAPI 场景（故事板等）：逻辑同上。
func PrepareHuoshanGenAPIImageRequest(genReq *genapi.GenerateRequest) {
	if genReq == nil {
		return
	}
	if strings.TrimSpace(genReq.Model) == "" {
		genReq.Model = huoshanpkg.DefaultHuoshanImageModelID
	}
	refURLs := genReq.ReferenceImages
	hint := genReq.OutputCount
	if hint < 1 {
		hint = 1
	}
	refs, out := OptimizeHuoshanReferencesAndOutputs(refURLs, hint)
	genReq.ReferenceImages = refs
	genReq.OutputCount = out
	if len(genReq.ReferenceImages) == 0 && genReq.Operation == genapi.OperationImageToImage {
		genReq.Operation = genapi.OperationTextToImage
		genReq.ReferenceImageURL = ""
	} else if len(genReq.ReferenceImages) > 0 && genReq.Operation == genapi.OperationImageToImage {
		genReq.ReferenceImageURL = genReq.ReferenceImages[0]
	}
	refc := len(genReq.ReferenceImages)
	textToImage := genReq.Operation == genapi.OperationTextToImage || refc == 0
	force := HuoshanUseBatchSemantics("", genReq.Metadata)
	patch := computeHuoshanSeedreamPatch(refc, textToImage, genReq.OutputCount, nil, force)
	genReq.Options = mergeStringOptions(genReq.Options, patch)
}

// ApplyHuoshanSeedreamToGenerateImageRequest（兼容）：仅写入 options patch，一般不单独使用；统一入口优先 PrepareHuoshanGenerateImageRequest。
func ApplyHuoshanSeedreamToGenerateImageRequest(req *GenerateImageRequest, maxImagesHint int) {
	if req == nil || !strings.EqualFold(strings.TrimSpace(req.Provider), "huoshan") {
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = huoshanpkg.DefaultHuoshanImageModelID
	}
	refc := len(req.ReferenceImages)
	textToImage := refc == 0
	want := req.OutputCount
	if maxImagesHint > want {
		want = maxImagesHint
	}
	if want < 1 {
		want = 1
	}
	patch := computeHuoshanSeedreamPatch(refc, textToImage, want, nil, false)
	req.Options = mergeStringOptions(req.Options, patch)
}

func ApplyHuoshanSeedreamToGenAPIRequest(genReq *genapi.GenerateRequest, maxImagesHint int) {
	if genReq == nil {
		return
	}
	if strings.TrimSpace(genReq.Model) == "" {
		genReq.Model = huoshanpkg.DefaultHuoshanImageModelID
	}
	refc := len(genReq.ReferenceImages)
	textToImage := genReq.Operation == genapi.OperationTextToImage || refc == 0
	want := genReq.OutputCount
	if maxImagesHint > want {
		want = maxImagesHint
	}
	if want < 1 {
		want = 1
	}
	patch := computeHuoshanSeedreamPatch(refc, textToImage, want, nil, false)
	genReq.Options = mergeStringOptions(genReq.Options, patch)
}

// effectiveMaxImagesFromOptions 从 Options 读取组图张数（配额估算用）。
func effectiveMaxImagesFromOptions(opts map[string]interface{}) int {
	if len(opts) == 0 {
		return 0
	}
	for _, key := range []string{"max_images", "maxImages"} {
		if v, ok := opts[key]; ok {
			switch t := v.(type) {
			case int:
				if t > 0 {
					return t
				}
			case int32:
				if int(t) > 0 {
					return int(t)
				}
			case int64:
				if int(t) > 0 {
					return int(t)
				}
			case float64:
				if t > 0 {
					return int(t)
				}
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil && n > 0 {
					return n
				}
			}
		}
	}
	// sequential_image_generation_options.max_images
	if m, ok := opts["sequential_image_generation_options"]; ok {
		if sm, ok := m.(map[string]interface{}); ok {
			for _, key := range []string{"max_images", "maxImages"} {
				if v, ok := sm[key]; ok {
					switch t := v.(type) {
					case int:
						if t > 0 {
							return t
						}
					case float64:
						if t > 0 {
							return int(t)
						}
					case string:
						if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil && n > 0 {
							return n
						}
					}
				}
			}
		}
	}
	return 0
}
