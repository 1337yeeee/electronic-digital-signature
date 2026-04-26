package model

import (
	"time"
)

const (
	VerificationStatusPending = "pending"
	VerificationStatusValid   = "valid"
	VerificationStatusInvalid = "invalid"
)

type Message struct {
	ID                 string    `gorm:"primaryKey;size:40" json:"id"`
	CreatedByUserID    string    `gorm:"size:40;not null" json:"created_by_user_id"`
	SignerID           string    `gorm:"size:40" json:"signer_id"`
	Message            string    `json:"message"`
	Hash               []byte    `gorm:"type:bytea" json:"-"`
	Signature          []byte    `gorm:"type:bytea" json:"-"`
	VerificationStatus string    `gorm:"size:20" json:"verification_status"`
	SignedAt           time.Time `json:"signed_at"`
	CreatedAt          time.Time `json:"created_at"`
}
