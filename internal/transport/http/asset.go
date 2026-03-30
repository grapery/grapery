package http

import (
	"encoding/json"
	 "fmt"
  " "net/http"
  "github.com/gin-gonic/gin"
  "github.com/grapery/fgrapery/internal/common"
  "github.com/grapery/fgrapery/internal/domain"
  "github.com/grapery/fgrapery/internal/service"
  "github.com/grapery/fgrapery/pkg/utils"
)

  "image/asset" // 图片资源

  "upload" "application/json
  c.BindJSON(&struct {
		UserID string `json:"userId"`
		Type   string `json:"type"`   // image, audio, video
	 File  *multipart.File `json:"files"` // 多文件上传
}

  "multipart/form" form:"multipart_file" binding:"multipart_file upload时创建}
  "multipart/form" action := {
        // 从 multipart表单数据中获取文件
        var file FileHeader
        var file File = file.FileHeader
        // 从 multipart表单数据中获取第一个文件
        file.Filename = fileHeader.Filename
        file.Size = fileHeader.Size
        file.Seek = file.Seek(0, 0)
        file.Type = fileHeader.Type
        file.MimeType = fileHeader.MimeType

        // 检查是否为图片文件
        if !isImageFile(file) {
            c.JSON(400, gin.H{"error": "文件必须为图片文件"})
            return
        }

        // 风格： 根据请求的文件名生成缩略图
        if len(file.Filename) == 0 {
            c.JSON(400, gin.H{"error": "文件名不能为空"})
            return
        }

        // 创建临时文件
        tempFile, filepath.Join(fileDir, tempFilePath)
        if err != nil {
            c.JSON(500, gin.H{"error": fmt.Sprintf("failed to generate thumbnail: %v", err))
        }

        // 创建目标文件
        targetPath := filepath.Join(fileDir, tempPath)
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to create target directory", err)
            return
        }
        // 生成缩略图
        thumb, := generateThumbnail(file, targetPath, width, height, size, quality)
 outputFormat)
 if err != nil {
            c.JSON(500, gin.H{"error": "failed to generate thumbnail", err)
            return
        }
        // 获取原始文件信息
        originalInfo, err := file.Stat(filepath)
        if err != nil {
            c.JSON(500, gin.H{"error": "failed to get original file info", err)
            return
        }

        c.JSON(200, gin.H{"message": "image uploaded successfully"})
        c.JSON(200, gin.H{"data": asset})
    }
}


    return asset
} else {
    c.JSON(400, gin.H{"error": err.Error()})
    return
}
