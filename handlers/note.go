package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"tiktok-server/config"
	"tiktok-server/models"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

func PublishNote(c *gin.Context) {
	userIDStr := c.PostForm("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	title := c.PostForm("title")
	content := c.PostForm("content")

	// Handle multiple images
	form, err := c.MultipartForm()
	var imageUrls []string

	if err == nil {
		files := form.File["images"]
		for _, file := range files {
			ext := filepath.Ext(file.Filename)
			objectName := fmt.Sprintf("notes/%d_%d_%s", userID, time.Now().UnixNano(), ext)
			
			// Open the file
			src, err := file.Open()
			if err != nil {
				continue
			}
			defer src.Close()

			// Upload to MinIO
			_, err = config.MinioClient.PutObject(context.Background(), config.MinioBucket, objectName, src, file.Size, minio.PutObjectOptions{
				ContentType: "image/jpeg", // You might want to detect this dynamically
			})
			if err != nil {
				continue
			}

			// Generate Proxy URL
			url := fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, objectName)
			imageUrls = append(imageUrls, url)
		}
	}

	imagesJson, _ := json.Marshal(imageUrls)

	note := models.Note{
		UserID:    userID,
		Title:     title,
		Content:   content,
		Images:    string(imagesJson),
		CreatedAt: time.Now(),
	}

	if err := config.DB.Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status_code": 1, "status_msg": "发布失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status_code": 0, 
		"status_msg": "发布成功",
		"note": note,
	})
}
