package model

import (
	"time"
)

type Message struct {
	ID        string    `gorm:"primaryKey;size:40" json:"id"`
	UserID    string    `gorm:"size:40" json:"user_id"`
	Message   string    `json:"message"`
	SignedAt  time.Time `json:"signed_at"`
	CreatedAt time.Time `json:"created_at"`
}
