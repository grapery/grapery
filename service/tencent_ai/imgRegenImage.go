package tencentai

import (
	"context"
	"encoding/base64"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/utils/aiart"
)

func (t *TencentAI) ImgToImage(ctx context.Context,
	prompt *PromptParams,
	outParams *OutPutParams,
	markParams *WaterMarkParams) ([]byte, error) {
	req := aiart.NewImageToImageRequest()
	if prompt.InputImage != "" && prompt.InputImageUrl == "" {
		req.InputImage = &prompt.InputImage
	} else if prompt.InputImage == "" && prompt.InputImageUrl != "" {
		req.InputUrl = &prompt.InputImageUrl
	} else if prompt.InputImage == "" && prompt.InputImageUrl == "" {
		return nil, errors.New("input image is empty")
	} else if prompt.InputImage != "" && prompt.InputImageUrl != "" {
		req.InputImage = &prompt.InputImage
	}
	req.Prompt = &prompt.PosttivePrompt
	req.NegativePrompt = &prompt.PosttivePrompt
	req.Styles = prompt.Styles

	req.ResultConfig = &aiart.ResultConfig{
		Resolution: &outParams.Resolution,
	}
	req.Strength = &outParams.Similarity

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

	resp, err := t.AIArtClient.ImageToImageWithContext(ctx, req)
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
	return imageData, nil
}
