package models

import "time"

type Note struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Images    string    `json:"images"` // JSON array of image URLs
	CreatedAt time.Time `json:"created_at"`
}

func (Note) TableName() string {
	return "notes"
}
