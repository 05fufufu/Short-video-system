package models

import "time"

type User struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	Username        string    `json:"username"`
	Password        string    `json:"-"`
	Nickname        string    `json:"nickname"`
	Avatar          string    `json:"avatar"`
	BackgroundImage string    `json:"background_image"`
	CreatedAt       time.Time `json:"created_at"`
}

func (User) TableName() string {
	return "users"
}
