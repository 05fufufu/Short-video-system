package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"tiktok-server/config" // 👈 引用刚才写的 config 包
	"tiktok-server/models" // 👈 引用刚才写的 models 包

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// PublishAction 处理视频上传
func PublishAction(c *gin.Context) {
	// 1. 获取文件
	file, header, err := c.Request.FormFile("data")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 2. 生成文件名
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), "video", ext)

	// 3. 上传 MinIO (使用 config.MinioClient)
	ctx := context.Background()
	info, err := config.MinioClient.PutObject(ctx, config.MinioBucket, filename, file, header.Size, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MinIO 上传失败: " + err.Error()})
		return
	}

	// 4. 拼接 URL
	playURL := fmt.Sprintf("http://%s/%s/%s", config.MinioEndpoint, config.MinioBucket, filename)
	coverURL := "http://localhost:9000/images/default.jpg"

	// 5. 存入数据库 (使用 config.DB)
	newVideo := models.Video{
		AuthorID: 1,
		PlayURL:  playURL,
		CoverURL: coverURL,
		Title:    c.PostForm("title"),
		Status:   0,
	}

	if err := config.DB.Create(&newVideo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库保存失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "上传成功！",
		"video_url": playURL,
		"size":      info.Size,
	})
}
