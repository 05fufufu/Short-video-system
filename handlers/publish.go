package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"tiktok-server/config"
	"tiktok-server/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

// 必须定义这个结构体，供两个函数使用
type TranscodeMessage struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	AuthorID int64  `json:"author_id"`
	CoverURL string `json:"cover_url"`
}

// === 1. 投稿功能 (确保这个函数还在!) ===
func PublishAction(c *gin.Context) {
	file, header, err := c.Request.FormFile("data")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件获取失败"})
		return
	}

	userIDStr := c.PostForm("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	ext := filepath.Ext(header.Filename)
	rawFilename := fmt.Sprintf("raw/%d_%s", time.Now().Unix(), "video"+ext)
	_, err = config.MinioClient.PutObject(context.Background(), config.MinioBucket, rawFilename, file, header.Size, minio.PutObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MinIO 上传失败"})
		return
	}

	var coverURL string
	coverFile, coverHeader, err := c.Request.FormFile("cover")
	if err == nil {
		coverName := fmt.Sprintf("covers/%d_%s", time.Now().Unix(), coverHeader.Filename)
		_, err = config.MinioClient.PutObject(context.Background(), config.MinioBucket, coverName, coverFile, coverHeader.Size, minio.PutObjectOptions{})
		if err == nil {
			coverURL = fmt.Sprintf("http://%s/%s/%s", config.MinioEndpoint, config.MinioBucket, coverName)
		}
	}
	if coverURL == "" {
		coverURL = "https://via.placeholder.com/320x180/ff758c/ffffff?text=Magic+Girl"
	}

	msg := TranscodeMessage{
		FileName: rawFilename,
		Title:    c.PostForm("title"),
		AuthorID: userID,
		CoverURL: coverURL,
	}
	body, _ := json.Marshal(msg)

	config.MQChannel.Publish("", "transcode_queue", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})

	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "上传成功"})
}

// === 2. 删除功能 (刚才新加的) ===
func DeleteAction(c *gin.Context) {
	videoID := c.Query("video_id")
	userID := c.Query("user_id")

	var video models.Video
	if err := config.DB.Where("id = ?", videoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	if fmt.Sprintf("%d", video.AuthorID) != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "非本人无法撤回"})
		return
	}

	ctx := context.Background()
	parts := strings.Split(video.PlayURL, config.MinioBucket+"/")
	if len(parts) > 1 {
		_ = config.MinioClient.RemoveObject(ctx, config.MinioBucket, parts[1], minio.RemoveObjectOptions{})
	}

	config.DB.Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM comments WHERE video_id = ?", videoID)
		tx.Exec("DELETE FROM likes WHERE video_id = ?", videoID)
		return tx.Delete(&video).Error
	})

	config.RDB.Del(config.Ctx, "feed:latest")
	c.JSON(http.StatusOK, gin.H{"status_code": 0, "status_msg": "已抹除"})
}
