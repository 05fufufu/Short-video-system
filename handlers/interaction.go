package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"tiktok-server/config"
	"tiktok-server/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

type LikeMessage struct {
	UserID  int64 `json:"user_id"`
	VideoID int64 `json:"video_id"`
	NoteID  int64 `json:"note_id"`
	Action  int   `json:"action"`
}

func FavoriteAction(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	noteID, _ := strconv.ParseInt(c.Query("note_id"), 10, 64)
	action, _ := strconv.Atoi(c.Query("action_type"))

	// 1. 操作 Redis
	var redisKey string
	if noteID > 0 {
		redisKey = fmt.Sprintf("note_likes:%d", noteID)
	} else {
		redisKey = fmt.Sprintf("video_likes:%d", videoID)
	}

	if action == 1 {
		config.RDB.SAdd(config.Ctx, redisKey, userID)
	} else {
		config.RDB.SRem(config.Ctx, redisKey, userID)
	}

	// 2. 发 MQ
	msg := LikeMessage{UserID: userID, VideoID: videoID, NoteID: noteID, Action: action}
	body, _ := json.Marshal(msg)
	config.MQChannel.Publish("", "like_queue", false, false, amqp.Publishing{
		ContentType: "application/json", Body: body,
	})

	c.JSON(200, gin.H{"status_code": 0, "status_msg": "操作成功"})
}

func FavoriteStatus(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	noteID, _ := strconv.ParseInt(c.Query("note_id"), 10, 64)

	var redisKey string
	if noteID > 0 {
		redisKey = fmt.Sprintf("note_likes:%d", noteID)
	} else {
		redisKey = fmt.Sprintf("video_likes:%d", videoID)
	}

	isFav := config.RDB.SIsMember(config.Ctx, redisKey, userID).Val()
	c.JSON(200, gin.H{"status_code": 0, "is_favorite": isFav})
}

// ReceivedLikesList 获取收到的点赞
func ReceivedLikesList(c *gin.Context) {
	userID := c.Query("user_id")

	// 1. 找到该用户发布的所有视频 ID 和笔记 ID
	var videos []models.Video
	config.DB.Where("author_id = ?", userID).Find(&videos)

	var notes []models.Note
	config.DB.Where("user_id = ?", userID).Find(&notes)

	videoMap := make(map[int64]models.Video)
	noteMap := make(map[int64]models.Note)
	var videoIDs []int64
	var noteIDs []int64

	for _, v := range videos {
		videoIDs = append(videoIDs, v.ID)
		videoMap[v.ID] = v
	}
	for _, n := range notes {
		noteIDs = append(noteIDs, n.ID)
		noteMap[n.ID] = n
	}

	if len(videoIDs) == 0 && len(noteIDs) == 0 {
		c.JSON(200, gin.H{"status_code": 0, "list": []interface{}{}})
		return
	}

	// 2. 找到这些视频/笔记的点赞记录
	var likes []models.Like
	query := config.DB.Where("is_deleted = 0")
	
	if len(videoIDs) > 0 && len(noteIDs) > 0 {
		query = query.Where("video_id IN ? OR note_id IN ?", videoIDs, noteIDs)
	} else if len(videoIDs) > 0 {
		query = query.Where("video_id IN ?", videoIDs)
	} else if len(noteIDs) > 0 {
		query = query.Where("note_id IN ?", noteIDs)
	}
	
	query.Find(&likes)

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

		var title, cover string
		if like.VideoID > 0 {
			if v, ok := videoMap[like.VideoID]; ok {
				title = v.Title
				cover = fixURL(v.CoverURL)
			}
		} else if like.NoteID > 0 {
			if n, ok := noteMap[like.NoteID]; ok {
				title = n.Title
				// 解析笔记封面
				var imgs []string
				json.Unmarshal([]byte(n.Images), &imgs)
				if len(imgs) > 0 {
					cover = imgs[0]
				} else {
					cover = "https://via.placeholder.com/320x180/eef2ff/8aa9ff?text=Note"
				}
			}
		}

		result = append(result, gin.H{
			"video_title":  title,
			"video_cover":  cover,
			"liker_id":     like.UserID,
			"liker_name":   u.Nickname,
			"liker_avatar": u.Avatar,
			"type":         func() string { if like.NoteID > 0 { return "note" } else { return "video" } }(),
		})
	}

	c.JSON(200, gin.H{"status_code": 0, "list": result})
}

func CommentAction(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	noteID, _ := strconv.ParseInt(c.Query("note_id"), 10, 64)
	content := c.Query("comment_text")

	comment := models.Comment{
		UserID:    userID,
		VideoID:   videoID,
		NoteID:    noteID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := config.DB.Create(&comment).Error; err != nil {
		c.JSON(500, gin.H{"status_code": 1, "status_msg": "评论失败"})
		return
	}

	// 发送通知
	go func() {
		var authorID int64
		if noteID > 0 {
			var note models.Note
			config.DB.Select("user_id").First(&note, noteID)
			authorID = note.UserID
		} else {
			var video models.Video
			config.DB.Select("author_id").First(&video, videoID)
			authorID = video.AuthorID
		}

		if authorID != 0 && authorID != userID {
			notif := models.Notification{
				UserID:     authorID,
				SenderID:   userID,
				ActionType: 2, // comment
				VideoID:    videoID,
				NoteID:     noteID,
				Content:    content,
				CreatedAt:  time.Now(),
				IsRead:     0,
			}
			config.DB.Create(&notif)
		}
	}()

	c.JSON(200, gin.H{"status_code": 0, "status_msg": "评论成功", "comment": comment})
}

func NotificationList(c *gin.Context) {
	userID := c.Query("user_id")
	var notifs []models.Notification
	config.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&notifs)

	var result []gin.H
	for _, n := range notifs {
		var u struct {
			Nickname string
			Avatar   string
		}
		config.GetUserDB(n.SenderID).Table("users").Select("nickname, avatar").Where("id = ?", n.SenderID).First(&u)

		// 修复头像 URL
		if strings.Contains(u.Avatar, "/video_file/") && !strings.Contains(u.Avatar, config.MinioPublicServer) {
			parts := strings.Split(u.Avatar, "/video_file/")
			if len(parts) >= 2 {
				u.Avatar = fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, parts[1])
			}
		}

		var title, cover string
		if n.VideoID > 0 {
			var v models.Video
			if err := config.DB.Select("title, cover_url").First(&v, n.VideoID).Error; err == nil {
				title = v.Title
				cover = fixURL(v.CoverURL)
			}
		} else if n.NoteID > 0 {
			var note models.Note
			if err := config.DB.Select("title, images").First(&note, n.NoteID).Error; err == nil {
				title = note.Title
				var imgs []string
				json.Unmarshal([]byte(note.Images), &imgs)
				if len(imgs) > 0 {
					cover = imgs[0]
				} else {
					cover = "https://via.placeholder.com/320x180/eef2ff/8aa9ff?text=Note"
				}
			}
		}

		result = append(result, gin.H{
			"id":            n.ID,
			"sender_name":   u.Nickname,
			"sender_avatar": u.Avatar,
			"action_type":   n.ActionType,
			"content":       n.Content,
			"created_at":    n.CreatedAt.Format("01-02 15:04"),
			"video_id":      n.VideoID,
			"note_id":       n.NoteID,
			"target_title":  title,
			"target_cover":  cover,
		})
	}
	c.JSON(200, gin.H{"status_code": 0, "list": result})
}

func CommentList(c *gin.Context) {
	videoID, _ := strconv.ParseInt(c.Query("video_id"), 10, 64)
	noteID, _ := strconv.ParseInt(c.Query("note_id"), 10, 64)

	var comments []models.Comment
	if noteID > 0 {
		config.DB.Where("note_id = ?", noteID).Order("created_at desc").Find(&comments)
	} else {
		config.DB.Where("video_id = ?", videoID).Order("created_at desc").Find(&comments)
	}

	var result []gin.H
	for _, cmt := range comments {
		var u struct {
			Nickname string
			Avatar   string
		}
		config.GetUserDB(cmt.UserID).Table("users").Select("nickname, avatar").Where("id = ?", cmt.UserID).First(&u)

		// 修复头像 URL
		if strings.Contains(u.Avatar, "/video_file/") && !strings.Contains(u.Avatar, config.MinioPublicServer) {
			parts := strings.Split(u.Avatar, "/video_file/")
			if len(parts) >= 2 {
				u.Avatar = fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, parts[1])
			}
		}

		result = append(result, gin.H{
			"id":            cmt.ID,
			"user_id":       cmt.UserID,
			"content":       cmt.Content,
			"create_date":   cmt.CreatedAt.Format("01-02 15:04"),
			"user_nickname": u.Nickname,
			"user_avatar":   u.Avatar,
		})
	}

	c.JSON(200, gin.H{"status_code": 0, "comment_list": result})
}
