package tencentai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/utils/aiart"
)

// 生成图支持分辨率，不传默认为"512*512"
// 1:1尺寸支持"512:512"，"1024:1024"
// 3:4尺寸支持"512:704"，"768:1024"
// 9:16尺寸支持"448:704"
// 4:3尺寸支持"704:512"，"1024:768"
// 16:9尺寸支持"704:448"
// 1:2尺寸支持”512:1024“
//
// 注意：此字段可能返回 null，表示取不到有效值。

type PromptParams struct {
	PosttivePrompt string
	NegativePrompt string
	Styles         []*string

	//
	InputImage    string
	InputImageUrl string
}
type OutPutParams struct {
	Resolution string
	Similarity float64
}

type WaterMarkParams struct {
	IsMarkAI bool
	LogoUrl  string
	LogoImg  string

	StartX int64
	StartY int64
	Width  int64
	Length int64
}

func (t *TencentAI) TextToImage(ctx context.Context,
	prompt *PromptParams,
	outParams *OutPutParams,
	markParams *WaterMarkParams,
) ([]byte, error) {
	req := aiart.NewTextToImageRequest()
	req.Prompt = &prompt.PosttivePrompt
	req.NegativePrompt = &prompt.PosttivePrompt
	req.Styles = prompt.Styles

	req.ResultConfig = &aiart.ResultConfig{
		Resolution: &outParams.Resolution,
	}
	var temp int64 = 0
	if !markParams.IsMarkAI {
		temp = 0
		req.LogoAdd = &temp
	} else {
		temp = 1
		req.LogoAdd = &temp
		req.LogoParam.LogoUrl = &markParams.LogoUrl
		req.LogoParam.LogoImage = &markParams.LogoImg
		req.LogoParam.LogoRect = &aiart.LogoRect{
			X:      &markParams.StartX,
			Y:      &markParams.StartY,
			Width:  &markParams.Width,
			Height: &markParams.Length,
		}
	}

	resp, err := t.AIArtClient.TextToImageWithContext(ctx, req)
	if err != nil {
		log.Errorf("call tencent cloud api failed: %s", err.Error())
		return nil, err
	}
	log.Printf("resp req id %d resp data length: %d",
		resp.Response.RequestId, len(*resp.Response.ResultImage))
	if len(*resp.Response.ResultImage) <= 0 {
		log.Errorf("image base64 data is empty: %d", len(*resp.Response.ResultImage))
		return nil, errors.New("image base64 data is empty")
	}
	imageData, err := base64.StdEncoding.DecodeString(*resp.Response.ResultImage)
	if err != nil {
		log.Errorf("base64 decode failed: %s", err.Error())
		return nil, err
	}
	imgUuid, _ := uuid.NewUUID()
	imageFile, err := os.OpenFile(fmt.Sprintf("%s.jpg", imgUuid.String()), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("create image data failed: %s", err.Error())
		return nil, err
	}
	defer imageFile.Close()
	total, err := imageFile.Write(imageData)
	if err != nil {
		log.Errorf("write image data failed: %s", err.Error())
		return nil, err
	}
	log.Infof("write image data to file success,data length: %d ", total)
	return imageData, nil
}
