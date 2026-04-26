package dto

type UploadDocumentResponse struct {
	DocumentID       string `json:"document_id"`
	OwnerUserID      string `json:"owner_user_id"`
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
	DocumentID       string `json:"document_id"`
	PackageID        string `json:"package_id,omitempty"`
	RecipientEmail   string `json:"recipient_email"`
	SendStatus       string `json:"send_status"`
	LastSentByUserID string `json:"last_sent_by_user_id,omitempty"`
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
