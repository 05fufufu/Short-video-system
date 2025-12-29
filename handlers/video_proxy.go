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

	// 获取文件信息以设置正确的 Content-Type
	stat, err := object.Stat()
	if err == nil {
		c.Header("Content-Type", stat.ContentType)
	} else {
		// 兜底策略
		if strings.HasSuffix(objectPath, ".m3u8") {
			c.Header("Content-Type", "application/x-mpegURL")
		} else if strings.HasSuffix(objectPath, ".ts") {
			c.Header("Content-Type", "video/MP2T")
		} else if strings.HasSuffix(objectPath, ".jpg") || strings.HasSuffix(objectPath, ".jpeg") {
			c.Header("Content-Type", "image/jpeg")
		} else {
			c.Header("Content-Type", "video/mp4")
		}
	}

	// 将 MinIO 的流直接拷贝给外网用户
	_, _ = io.Copy(c.Writer, object)
}
