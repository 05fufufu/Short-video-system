package service

import (
	"encoding/json"
	"log"
	"tiktok-server/config"
	"tiktok-server/models"

	"gorm.io/gorm/clause"
)

type LikeMessage struct {
	UserID  int64 `json:"user_id"`
	VideoID int64 `json:"video_id"`
	Action  int   `json:"action"`
}

func StartLikeWorker() {
	msgs, _ := config.MQChannel.Consume("like_queue", "", true, false, false, false, nil)
	go func() {
		log.Println("❤️ 点赞 Worker 已启动...")
		for d := range msgs {
			var msg LikeMessage
			json.Unmarshal(d.Body, &msg)

			like := models.Like{UserID: msg.UserID, VideoID: msg.VideoID, IsDeleted: 0}
			if msg.Action == 2 {
				like.IsDeleted = 1
			}

			config.DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "video_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"is_deleted"}),
			}).Create(&like)

			log.Printf("✅ 异步点赞入库: User %d -> Video %d", msg.UserID, msg.VideoID)
		}
	}()
}
