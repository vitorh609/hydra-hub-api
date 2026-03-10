package account_settings

import (
	"time"
)

type AccountSettings struct {
	UserId       string    `json:"user_id"`
	AvatarBase64 string    `json:"avatar_base64"`
	Name         string    `json:"name"`
	Location     string    `json:"location"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateAccountSettingsInput struct {
	AvatarBase64 string `json:"avatar_base64"`
	Name         string `json:"name"`
	Location     string `json:"location"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
}

type UpdateAccountSettingsInput struct {
	AvatarBase64 *string `json:"avatarBase64"`
	Name         *string `json:"name"`
	Location     *string `json:"location"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Address      *string `json:"address"`
}
