package dto

type VerifyClientSignatureRequest struct {
	Message         string `json:"message" binding:"required"`
	SignatureBase64 string `json:"signature_base64" binding:"required"`
	PublicKeyBase64 string `json:"public_key_base64" binding:"required"`
}

type VerifyClientSignatureResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type ServerPublicKeyResponse struct {
	Algorithm       string `json:"algorithm"`
	PublicKeyBase64 string `json:"public_key_base64"`
}

type IssueServerMessageRequest struct {
	Message string `json:"message,omitempty"`
}

type IssueServerMessageResponse struct {
	Message         string `json:"message"`
	Algorithm       string `json:"algorithm"`
	HashBase64      string `json:"hash_base64"`
	SignatureBase64 string `json:"signature_base64"`
}
