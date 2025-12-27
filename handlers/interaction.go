package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"tiktok-server/config"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

type LikeMessage struct {
	UserID  int64 `json:"user_id"`
	VideoID int64 `json:"video_id"`
	Action  int   `json:"action"`
}

func FavoriteAction(c *gin.Context) {
	userID := int64(1)
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	action, _ := strconv.Atoi(c.Query("action_type"))

	// 1. 操作 Redis
	redisKey := fmt.Sprintf("video_likes:%d", videoID)
	if action == 1 {
		config.RDB.SAdd(config.Ctx, redisKey, userID)
	} else {
		config.RDB.SRem(config.Ctx, redisKey, userID)
	}

	// 2. 发 MQ
	msg := LikeMessage{UserID: userID, VideoID: videoID, Action: action}
	body, _ := json.Marshal(msg)
	config.MQChannel.Publish("", "like_queue", false, false, amqp.Publishing{
		ContentType: "application/json", Body: body,
	})

	c.JSON(200, gin.H{"status_code": 0, "status_msg": "操作成功"})
}
