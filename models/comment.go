package models

import "time"

type Comment struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	VideoID   int64     `json:"video_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"create_date"`
}

func (Comment) TableName() string { return "comments" }
