package model

import "time"

type Document struct {
	ID               string     `gorm:"primaryKey;size:40" json:"id"`
	OwnerUserID      string     `gorm:"size:40;not null" json:"owner_user_id"`
	OwnerEmail       string     `gorm:"size:255;not null" json:"owner_email"`
	RecipientEmail   string     `gorm:"size:255;not null" json:"recipient_email"`
	OriginalFileName string     `gorm:"size:255;not null" json:"original_file_name"`
	StoredPath       string     `gorm:"size:500;not null" json:"stored_path"`
	MimeType         string     `gorm:"size:120;not null" json:"mime_type"`
	Hash             []byte     `gorm:"type:bytea" json:"-"`
	Signature        []byte     `gorm:"type:bytea" json:"-"`
	EncryptedPath    string     `gorm:"size:500" json:"encrypted_path"`
	SendStatus       string     `gorm:"size:20" json:"send_status"`
	LastSentByUserID string     `gorm:"size:40" json:"last_sent_by_user_id"`
	LastSentToEmail  string     `gorm:"size:255" json:"last_sent_to_email"`
	SendError        string     `gorm:"size:1000" json:"send_error"`
	CreatedAt        time.Time  `json:"created_at"`
	SignedAt         time.Time  `json:"signed_at"`
	SentAt           *time.Time `json:"sent_at"`
}
