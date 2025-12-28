package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"tiktok-server/config"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// ProxyVideo 中转 MinIO 视频流
func ProxyVideo(c *gin.Context) {
	// 获取 URL 中的文件路径 (例如 /processed/123.mp4)
	objectPath := c.Param("filepath")
	objectPath = strings.TrimPrefix(objectPath, "/")

	// 从 Linux MinIO 获取文件流
	object, err := config.MinioClient.GetObject(context.Background(), config.MinioBucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到记忆影像"})
		return
	}
	defer object.Close()

	// 设置响应头，告诉浏览器这是一个 MP4 视频
	c.Header("Content-Type", "video/mp4")

	// 将 MinIO 的流直接拷贝给外网用户
	_, _ = io.Copy(c.Writer, object)
}
