package google

import (
	api "github.com/grapery/common-protoc/gen"
)

// ImageDimensions 图片尺寸
type ImageDimensions struct {
	Width  int32
	Height int32
}

// CalculateImageDimensions 根据Ratio计算图片尺寸（以1000为基数）
func CalculateImageDimensions(ratio api.ImageRatios) ImageDimensions {
	base := int32(1000)

	switch ratio {
	case api.ImageRatios_Ratio1_1:
		// 1:1 正方形
		return ImageDimensions{Width: base, Height: base}
	case api.ImageRatios_Ratio4_3:
		// 4:3 横向
		return ImageDimensions{Width: base, Height: base * 3 / 4}
	case api.ImageRatios_Ratio16_9:
		// 16:9 宽屏
		return ImageDimensions{Width: base, Height: base * 9 / 16}
	case api.ImageRatios_Ratio3_2:
		// 3:2 横向
		return ImageDimensions{Width: base, Height: base * 2 / 3}
	case api.ImageRatios_Ratio3_4:
		// 3:4 竖向
		return ImageDimensions{Width: base * 3 / 4, Height: base}
	case api.ImageRatios_Ratio2_3:
		// 2:3 竖向
		return ImageDimensions{Width: base * 2 / 3, Height: base}
	default:
		// 默认使用16:9
		return ImageDimensions{Width: base, Height: base * 9 / 16}
	}
}

// GetAspectRatioString 获取宽高比字符串
func GetAspectRatioString(ratio api.ImageRatios) string {
	switch ratio {
	case api.ImageRatios_Ratio1_1:
		return "1:1"
	case api.ImageRatios_Ratio4_3:
		return "4:3"
	case api.ImageRatios_Ratio16_9:
		return "16:9"
	case api.ImageRatios_Ratio3_2:
		return "3:2"
	case api.ImageRatios_Ratio3_4:
		return "3:4"
	case api.ImageRatios_Ratio2_3:
		return "2:3"
	default:
		return "16:9"
	}
}

// GetAspectRatioFromString 从字符串获取宽高比
func GetAspectRatioFromString(ratioStr string) api.ImageRatios {
	switch ratioStr {
	case "1:1":
		return api.ImageRatios_Ratio1_1
	case "4:3":
		return api.ImageRatios_Ratio4_3
	case "16:9":
		return api.ImageRatios_Ratio16_9
	case "3:2":
		return api.ImageRatios_Ratio3_2
	case "3:4":
		return api.ImageRatios_Ratio3_4
	case "2:3":
		return api.ImageRatios_Ratio2_3
	default:
		return api.ImageRatios_Ratio16_9
	}
}
