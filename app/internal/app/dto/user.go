package dto

type RegisterUserRequest struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	PublicKeyPEM string `json:"public_key_pem,omitempty"`
}

type UpdateMyPublicKeyRequest struct {
	PublicKeyPEM string `json:"public_key_pem"`
}

type UserResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PublicKeyPEM string `json:"public_key_pem,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}
