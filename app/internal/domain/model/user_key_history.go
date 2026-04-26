package model

import "time"

type UserKeyHistory struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       string    `gorm:"size:40;index;not null" json:"user_id"`
	PublicKeyPEM string    `gorm:"type:text;not null" json:"public_key_pem"`
	CreatedAt    time.Time `json:"created_at"`
}
