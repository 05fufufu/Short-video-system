package handlers

import (
	"encoding/json"
	"sort"
	"tiktok-server/config"
	"tiktok-server/models"
	"time"

	"github.com/gin-gonic/gin"
)

// FeedItem 统一的动态结构
type FeedItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"` // "video" or "note"
	Title     string    `json:"title"`
	CoverURL  string    `json:"cover_url"`
	AuthorID  int64     `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	
	// 详情页需要的字段
	PlayURL string `json:"play_url,omitempty"` // 视频专用
	Content string `json:"content,omitempty"`  // 笔记专用
	Images  string `json:"images,omitempty"`   // 笔记专用
}

func FeedAction(c *gin.Context) {
	var items []FeedItem
	cacheKey := "feed:mixed:latest"

	// 1. 尝试读缓存
	val, err := config.RDB.Get(config.Ctx, cacheKey).Result()
	if err == nil {
		json.Unmarshal([]byte(val), &items)
		c.JSON(200, gin.H{"status_code": 0, "video_list": items, "source": "cache"})
		return
	}

	// 2. 查视频
	var videos []models.Video
	config.DB.Where("status = ?", 1).Order("created_at desc").Limit(30).Find(&videos)

	// 3. 查笔记
	var notes []models.Note
	config.DB.Order("created_at desc").Limit(30).Find(&notes)

	// 4. 合并
	for _, v := range videos {
		items = append(items, FeedItem{
			ID:        v.ID,
			Type:      "video",
			Title:     v.Title,
			CoverURL:  v.CoverURL,
			AuthorID:  v.AuthorID,
			CreatedAt: v.CreatedAt,
			PlayURL:   v.PlayURL,
		})
	}

	for _, n := range notes {
		// 解析图片JSON取第一张作为封面
		var imgs []string
		json.Unmarshal([]byte(n.Images), &imgs)
		cover := "https://via.placeholder.com/320x180/eef2ff/8aa9ff?text=Note"
		if len(imgs) > 0 {
			cover = imgs[0]
		}

		items = append(items, FeedItem{
			ID:        n.ID,
			Type:      "note",
			Title:     n.Title,
			CoverURL:  cover,
			AuthorID:  n.UserID,
			CreatedAt: n.CreatedAt,
			Content:   n.Content,
			Images:    n.Images,
		})
	}

	// 5. 排序 (按时间倒序)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	// 截取前 30 条
	if len(items) > 30 {
		items = items[:30]
	}

	// 6. 回写缓存
	if len(items) > 0 {
		jsonBytes, _ := json.Marshal(items)
		config.RDB.Set(config.Ctx, cacheKey, jsonBytes, 10*time.Second)
	}

	c.JSON(200, gin.H{"status_code": 0, "video_list": items, "source": "db"})
}
