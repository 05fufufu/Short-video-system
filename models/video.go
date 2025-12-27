package models

import "time"

type Video struct {
	ID            int64     `json:"id"`
	AuthorID      int64     `json:"author_id"`
	PlayURL       string    `json:"play_url"`
	CoverURL      string    `json:"cover_url"`
	Title         string    `json:"title"`
	Status        int       `json:"status"` // 0:处理中 1:已发布
	FavoriteCount int       `json:"favorite_count"`
	CreatedAt     time.Time `json:"created_at"`
}

func (Video) TableName() string {
	return "videos"
}
