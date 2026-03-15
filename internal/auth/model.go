package auth

import "time"

type LoginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string      `json:"token"`
	Type      string      `json:"type"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      SessionUser `json:"user"`
}

type SessionUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthUser struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}
