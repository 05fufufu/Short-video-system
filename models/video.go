package models

import "time"

// Video 对应数据库的 videos 表
type Video struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	PlayURL   string    `json:"play_url"`
	CoverURL  string    `json:"cover_url"`
	Title     string    `json:"title"`
	Status    int       `json:"status"` // 0:处理中 1:已发布
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名（GORM 默认是复数，明确指定更安全）
func (Video) TableName() string {
	return "videos"
}
