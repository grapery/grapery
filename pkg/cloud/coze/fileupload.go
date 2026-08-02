package coze

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/coze-dev/coze-go"
	cozego "github.com/coze-dev/coze-go"
)

func (c HuoShanCozeClient) UploadFile(ctx context.Context, src io.Reader, filename string, filePath string) (string, error) {
	authCli := cozego.NewTokenAuth(c.GetAPIKey())
	cozeCli := cozego.NewCozeAPI(authCli, cozego.WithBaseURL(os.Getenv("COZE_API_BASE")))
	// Upload file
	req := cozego.UploadFilesReq{
		File: cozego.NewUploadFile(src, filename),
	}
	uploadResp, err := cozeCli.Files.Upload(ctx, &req)
	if err != nil {
		fmt.Println("Error uploading file:", err)
		return "", nil
	}
	fileInfo := uploadResp.FileInfo

	// Wait for file processing
	time.Sleep(time.Second)

	// Retrieve file
	retrievedResp, err := cozeCli.Files.Retrieve(ctx, &coze.RetrieveFilesReq{
		FileID: fileInfo.ID,
	})
	if err != nil {
		fmt.Println("Error retrieving file:", err)
		return "", nil
	}
	fmt.Println(retrievedResp.FileInfo)
	return retrievedResp.FileInfo.ID, nil
}
