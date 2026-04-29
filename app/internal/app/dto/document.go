package dto

type UploadDocumentResponse struct {
	DocumentID       string `json:"document_id"`
	OwnerUserID      string `json:"owner_user_id"`
	SignedByUserID   string `json:"signed_by_user_id"`
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
	OwnerUserID    string `json:"owner_user_id"`
	SignedByUserID string `json:"signed_by_user_id"`
	PackageID      string `json:"package_id,omitempty"`
	RecipientEmail string `json:"recipient_email"`
	SendStatus     string `json:"send_status"`
	SentByUserID   string `json:"sent_by_user_id,omitempty"`
	SentAt         string `json:"sent_at,omitempty"`
}

type DocumentAuditResponse struct {
	DocumentID       string `json:"document_id"`
	OwnerUserID      string `json:"owner_user_id"`
	SignedByUserID   string `json:"signed_by_user_id"`
	SentByUserID     string `json:"sent_by_user_id,omitempty"`
	OwnerEmail       string `json:"owner_email"`
	RecipientEmail   string `json:"recipient_email"`
	OriginalFileName string `json:"original_file_name"`
	MimeType         string `json:"mime_type"`
	SendStatus       string `json:"send_status,omitempty"`
	CreatedAt        string `json:"created_at"`
	SignedAt         string `json:"signed_at"`
	SentAt           string `json:"sent_at,omitempty"`
}

type UserDocumentListItemResponse struct {
	DocumentID       string `json:"document_id"`
	OriginalFileName string `json:"original_file_name"`
	RecipientEmail   string `json:"recipient_email"`
	SendStatus       string `json:"send_status,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type DocumentDetailsResponse struct {
	DocumentID       string `json:"document_id"`
	OwnerUserID      string `json:"owner_user_id"`
	SignedByUserID   string `json:"signed_by_user_id"`
	SentByUserID     string `json:"sent_by_user_id,omitempty"`
	OwnerEmail       string `json:"owner_email"`
	RecipientEmail   string `json:"recipient_email"`
	OriginalFileName string `json:"original_file_name"`
	MimeType         string `json:"mime_type"`
	SendStatus       string `json:"send_status,omitempty"`
	CreatedAt        string `json:"created_at"`
	SignedAt         string `json:"signed_at"`
	SentAt           string `json:"sent_at,omitempty"`
}

type VerifyDecryptPackageMetadata struct {
	DocumentID          string `json:"document_id"`
	Version             string `json:"version"`
	EncryptionAlgorithm string `json:"encryption_algorithm"`
	KeyTransport        string `json:"key_transport"`
	SignatureAlgorithm  string `json:"signature_algorithm"`
	OriginalFileName    string `json:"original_file_name"`
	MimeType            string `json:"mime_type"`
	HashBase64          string `json:"hash_base64"`
}

type VerifyDecryptPackageResponse struct {
	Valid                   bool                         `json:"valid"`
	Error                   string                       `json:"error,omitempty"`
	Metadata                VerifyDecryptPackageMetadata `json:"metadata"`
	DecryptedDocumentBase64 string                       `json:"decrypted_document_base64,omitempty"`
}
