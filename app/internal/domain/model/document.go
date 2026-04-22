package model

type Document struct {
	Message
	MimeType string `gorm:"size:40" json:"mime_type"`
	Path     string `gorm:"size:255" json:"path"`
	Hash     string `gorm:"size:255" json:"hash"`
}
