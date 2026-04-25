package dto

type UploadDocumentResponse struct {
	DocumentID       string `json:"document_id"`
	OwnerEmail       string `json:"owner_email"`
	RecipientEmail   string `json:"recipient_email"`
	OriginalFileName string `json:"original_file_name"`
	StoredPath       string `json:"stored_path"`
	MimeType         string `json:"mime_type"`
	CreatedAt        string `json:"created_at"`
}

type SendDocumentRequest struct {
	Email string `json:"email"`
}

type SendDocumentResponse struct {
	DocumentID     string `json:"document_id"`
	RecipientEmail string `json:"recipient_email"`
	SendStatus     string `json:"send_status"`
	SentAt         string `json:"sent_at,omitempty"`
}
