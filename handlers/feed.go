package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	IsFavorite bool `json:"is_favorite"`       // 是否已点赞
}

func fixURL(u string) string {
	if u == "" {
		return ""
	}
	// 如果是相对路径或已经是公网地址，直接返回
	if strings.Contains(u, config.MinioPublicServer) {
		return u
	}
	// 替换 localhost 或内网 IP 为公网域名
	if strings.Contains(u, "/video_file/") {
		parts := strings.Split(u, "/video_file/")
		if len(parts) >= 2 {
			return fmt.Sprintf("http://%s/video_file/%s", config.MinioPublicServer, parts[1])
		}
	}
	return u
}

func FeedAction(c *gin.Context) {
	userIDStr := c.Query("user_id")
	var userID int64
	if userIDStr != "" {
		userID, _ = strconv.ParseInt(userIDStr, 10, 64)
	}

	var items []FeedItem
	cacheKey := "feed:mixed:latest"

	// 1. 尝试读缓存 (如果用户未登录，可以用缓存；登录用户需要实时查点赞状态，所以跳过缓存或缓存不含状态)
	// 为了简单，这里暂时先跳过缓存，或者后续优化为"基础数据缓存+状态实时查"
	// val, err := config.RDB.Get(config.Ctx, cacheKey).Result()
	// if err == nil && userID == 0 { ... } 

	// 2. 查视频
	var videos []models.Video
	config.DB.Where("status = ?", 1).Order("created_at desc").Limit(30).Find(&videos)

	// 3. 查笔记
	var notes []models.Note
	config.DB.Order("created_at desc").Limit(30).Find(&notes)

	// 4. 合并
	for _, v := range videos {
		isFav := false
		if userID > 0 {
			isFav = config.RDB.SIsMember(config.Ctx, fmt.Sprintf("video_likes:%d", v.ID), userID).Val()
		}

		items = append(items, FeedItem{
			ID:         v.ID,
			Type:       "video",
			Title:      v.Title,
			CoverURL:   fixURL(v.CoverURL),
			AuthorID:   v.AuthorID,
			CreatedAt:  v.CreatedAt,
			PlayURL:    fixURL(v.PlayURL),
			IsFavorite: isFav,
		})
	}

	for _, n := range notes {
		isFav := false
		if userID > 0 {
			isFav = config.RDB.SIsMember(config.Ctx, fmt.Sprintf("note_likes:%d", n.ID), userID).Val()
		}

		// 解析图片JSON取第一张作为封面
		var imgs []string
		json.Unmarshal([]byte(n.Images), &imgs)
		cover := "https://via.placeholder.com/320x180/eef2ff/8aa9ff?text=Note"
		if len(imgs) > 0 {
			cover = imgs[0]
		}

		items = append(items, FeedItem{
			ID:         n.ID,
			Type:       "note",
			Title:      n.Title,
			CoverURL:   cover,
			AuthorID:   n.UserID,
			CreatedAt:  n.CreatedAt,
			Content:    n.Content,
			Images:     n.Images,
			IsFavorite: isFav,
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

// SearchAction 搜索接口
func SearchAction(c *gin.Context) {
	keyword := c.Query("keyword")
	var items []FeedItem

	// 1. 搜视频
	var videos []models.Video
	config.DB.Where("title LIKE ? AND status = 1", "%"+keyword+"%").Order("created_at desc").Find(&videos)

	// 2. 搜笔记
	var notes []models.Note
	config.DB.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Order("created_at desc").Find(&notes)

	// 3. 合并
	for _, v := range videos {
		items = append(items, FeedItem{
			ID:        v.ID,
			Type:      "video",
			Title:     v.Title,
			CoverURL:  fixURL(v.CoverURL),
			AuthorID:  v.AuthorID,
			CreatedAt: v.CreatedAt,
			PlayURL:   fixURL(v.PlayURL),
		})
	}
	for _, n := range notes {
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

	c.JSON(200, gin.H{"status_code": 0, "video_list": items})
}
