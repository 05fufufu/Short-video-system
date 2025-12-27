package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"tiktok-server/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
)

// 修改消息结构体，增加 CoverURL 字段
type TranscodeMessage struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	AuthorID int64  `json:"author_id"`
	CoverURL string `json:"cover_url"` // 新增：携带封面地址
}

func PublishAction(c *gin.Context) {
	// 1. 获取视频文件
	file, header, err := c.Request.FormFile("data")
	if err != nil {
		c.JSON(400, gin.H{"error": "视频文件获取失败"})
		return
	}

	// 2. 上传视频到 MinIO
	ext := filepath.Ext(header.Filename)
	rawFilename := fmt.Sprintf("raw/%d_%s", time.Now().Unix(), "video"+ext)
	_, err = config.MinioClient.PutObject(context.Background(), config.MinioBucket, rawFilename, file, header.Size, minio.PutObjectOptions{})
	if err != nil {
		c.JSON(500, gin.H{"error": "MinIO 视频上传失败"})
		return
	}

	// 3. 处理封面 (关键修改)
	coverURL := "" // 默认空
	coverFile, coverHeader, err := c.Request.FormFile("cover")

	// 如果用户上传了封面
	if err == nil {
		coverName := fmt.Sprintf("covers/%d_%s", time.Now().Unix(), coverHeader.Filename)
		_, err = config.MinioClient.PutObject(context.Background(), config.MinioBucket, coverName, coverFile, coverHeader.Size, minio.PutObjectOptions{})
		if err == nil {
			// 生成封面 URL (使用 Linux IP)
			coverURL = fmt.Sprintf("http://%s/%s/%s", config.MinioEndpoint, config.MinioBucket, coverName)
		}
	} else {
		// 如果没传封面，给一个默认的魔法少女图
		coverURL = "https://via.placeholder.com/320x180/ff9a9e/ffffff?text=Magic+Girl"
	}

	// 4. 发消息给 MQ (带上封面URL)
	msg := TranscodeMessage{
		FileName: rawFilename,
		Title:    c.PostForm("title"),
		AuthorID: 1,
		CoverURL: coverURL, // 传给 Worker
	}
	body, _ := json.Marshal(msg)

	config.MQChannel.Publish("", "transcode_queue", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})

	c.JSON(200, gin.H{
		"status_code": 0,
		"status_msg":  "上传成功",
	})
}
