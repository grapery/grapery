package tencentai

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	testBaseUrl = "aiart.tencentcloudapi.com"
	testRegion  = "ap-shanghai"
	testPrompt  = "木屋,别墅,小溪,秋千,桃园"
	testStyle   = "101"
	testSize    = ResolutionLevel_1024_1024
	testAppId   = ""
	testAppKey  = ""
)

func TestTencentAI_TextToImage(t *testing.T) {
	testClient, err := NewTencentAI(testAppId, testAppKey)
	if err != nil {
		t.Error("new tencent cloud ai failed: ", err.Error())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	imageData, err := testClient.TextToImage(ctx,
		&PromptParams{
			PosttivePrompt: testPrompt,
			Styles:         []*string{&testStyle},
		},
		&OutPutParams{
			Resolution: testSize,
		},
		&WaterMarkParams{
			IsMarkAI: false,
		},
	)
	if err != nil {
		t.Error("create ai image failed: ", err.Error())
		return
	}
	imgUuid, _ := uuid.NewUUID()
	curDir, _ := os.Getwd()
	imageFile, err := os.OpenFile(fmt.Sprintf(curDir+"/%s.jpg", imgUuid.String()), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		t.Error("create image data failed: ", err.Error())
		return
	}
	defer imageFile.Close()
	total, err := imageFile.Write(imageData)
	if err != nil {
		t.Error("write image data failed: ", err.Error())
		return
	}
	log.Info("write image data to file success,data length: ", total)
}
