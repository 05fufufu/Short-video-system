package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"tiktok-server/config"
	"tiktok-server/models"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

type LikeMessage struct {
	UserID  int64 `json:"user_id"`
	VideoID int64 `json:"video_id"`
	Action  int   `json:"action"`
}

func FavoriteAction(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
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

// ReceivedLikesList 获取收到的点赞
func ReceivedLikesList(c *gin.Context) {
	userID := c.Query("user_id")

	// 1. 找到该用户发布的所有视频 ID
	var videos []models.Video
	if err := config.DB.Where("author_id = ?", userID).Find(&videos).Error; err != nil {
		c.JSON(200, gin.H{"status_code": 0, "list": []interface{}{}})
		return
	}

	videoMap := make(map[int64]models.Video)
	var videoIDs []int64
	for _, v := range videos {
		videoIDs = append(videoIDs, v.ID)
		videoMap[v.ID] = v
	}

	if len(videoIDs) == 0 {
		c.JSON(200, gin.H{"status_code": 0, "list": []interface{}{}})
		return
	}

	// 2. 找到这些视频的点赞记录
	var likes []models.Like
	config.DB.Where("video_id IN ? AND is_deleted = 0", videoIDs).Find(&likes)

	// 3. 组装数据
	var result []gin.H
	for _, like := range likes {
		// 查点赞人的信息
		var u struct {
			Nickname string
			Avatar   string
		}
		config.GetUserDB(like.UserID).Table("users").Select("nickname, avatar").Where("id = ?", like.UserID).First(&u)

		// 修复头像 URL
		if strings.Contains(u.Avatar, "/video_file/") && !strings.Contains(u.Avatar, config.MinioPublicServer) {
			parts := strings.Split(u.Avatar, "/video_file/")
			if len(parts) >= 2 {
				u.Avatar = fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, parts[1])
			}
		}

		video := videoMap[like.VideoID]
		result = append(result, gin.H{
			"video_title":  video.Title,
			"video_cover":  fixURL(video.CoverURL),
			"liker_id":     like.UserID,
			"liker_name":   u.Nickname,
			"liker_avatar": u.Avatar,
		})
	}

	c.JSON(200, gin.H{"status_code": 0, "list": result})
}
