package model

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:40" json:"id"`
	Email        string    `gorm:"size:255;not null;uniqueIndex" json:"email"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	PublicKeyPEM string    `gorm:"type:text" json:"public_key_pem"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
