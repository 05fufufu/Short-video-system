package handlers

import (
	"encoding/json"
	"tiktok-server/config"
	"tiktok-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

func FeedAction(c *gin.Context) {
	var videos []models.Video
	cacheKey := "feed:latest"

	// 1. 先查 Redis
	val, err := config.RDB.Get(config.Ctx, cacheKey).Result()
	if err == nil {
		json.Unmarshal([]byte(val), &videos)
		c.JSON(200, gin.H{"status_code": 0, "video_list": videos, "source": "cache"})
		return
	}

	// 2. 查 MySQL (只查已发布的 status=1)
	config.DB.Where("status = ?", 1).Order("created_at desc").Limit(30).Find(&videos)

	// 3. 回写 Redis
	if len(videos) > 0 {
		jsonBytes, _ := json.Marshal(videos)
		config.RDB.Set(config.Ctx, cacheKey, jsonBytes, 10*time.Second)
	}

	c.JSON(200, gin.H{"status_code": 0, "video_list": videos, "source": "db"})
}
