package tencentcloud

import (
	"io/ioutil"
	"testing"
)

func TestUploadObject(t *testing.T) {
	imageData, _ := ioutil.ReadFile("./for_teat.png")
	got, err := UploadObject(imageData, "png")
	if err != nil {
		t.Error(err)
	}
	t.Log(got)
}
