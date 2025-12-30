package models

import "time"

type Like struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	VideoID   int64     `json:"video_id"`
	NoteID    int64     `json:"note_id"`
	CreatedAt time.Time `json:"created_at"`
	IsDeleted int       `json:"is_deleted"` // 0: valid, 1: deleted
}

func (Like) TableName() string { return "likes" }
