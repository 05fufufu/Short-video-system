package models

import "time"

type Notification struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"` // Receiver
	SenderID   int64     `json:"sender_id"`
	ActionType int       `json:"action_type"` // 1: like, 2: comment
	VideoID    int64     `json:"video_id"`
	NoteID     int64     `json:"note_id"`
	Content    string    `json:"content"`
	IsRead     int       `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }
