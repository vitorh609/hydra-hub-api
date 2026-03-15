package users

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        *string   `json:"phone,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateUserInput struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	PasswordHash string  `json:"password_hash"`
	Phone        *string `json:"phone"`
}

type UpdateUserInput struct {
	Name         *string `json:"name"`
	Email        *string `json:"email"`
	Password     *string `json:"password"`
	PasswordHash *string `json:"password_hash"`
	Phone        *string `json:"phone"`
}
