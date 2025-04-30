package models

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
}

type LogoutRequest struct {
	UserID string `json:"id"`
}
