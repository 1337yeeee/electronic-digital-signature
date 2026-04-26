package dto

type VerifyClientSignatureRequest struct {
	Message         string `json:"message" binding:"required"`
	SignatureBase64 string `json:"signature_base64" binding:"required"`
	PublicKey       string `json:"public_key" binding:"required"`
}

type VerifyUserSignatureRequest struct {
	Message         string `json:"message" binding:"required"`
	SignatureBase64 string `json:"signature_base64" binding:"required"`
}

type VerifyClientSignatureResponse struct {
	Valid        bool   `json:"valid"`
	SignerType   string `json:"signer_type,omitempty"`
	SignerUserID string `json:"signer_user_id,omitempty"`
	Error        string `json:"error,omitempty"`
}

type ServerPublicKeyResponse struct {
	Algorithm    string `json:"algorithm"`
	PublicKeyPEM string `json:"public_key_pem"`
}

type IssueServerMessageRequest struct {
	Message string `json:"message,omitempty"`
}

type IssueServerMessageResponse struct {
	MessageID       string `json:"message_id"`
	SignerType      string `json:"signer_type"`
	SignerUserID    string `json:"signer_user_id,omitempty"`
	CreatedByUserID string `json:"created_by_user_id,omitempty"`
	CreatedAt       string `json:"created_at"`
	Message         string `json:"message"`
	Algorithm       string `json:"algorithm"`
	HashBase64      string `json:"hash_base64"`
	SignatureBase64 string `json:"signature_base64"`
}
