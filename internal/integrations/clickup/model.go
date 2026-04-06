package clickup

import "time"

type Connection struct {
	UserID               string
	TokenCiphertext      string
	TokenKeyVersion      string
	DefaultWorkspaceID   *string
	DefaultWorkspaceName *string
	Status               string
	LastCheckedAt        *time.Time
	LastError            *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ConnectClickupDto struct {
	Token                string  `json:"token"`
	DefaultWorkspaceID   *string `json:"defaultWorkspaceId,omitempty"`
	DefaultWorkspaceName *string `json:"defaultWorkspaceName,omitempty"`
}

type ClickupConnectionStatusDto struct {
	Connected            bool       `json:"connected"`
	Status               string     `json:"status"`
	DefaultWorkspaceID   *string    `json:"defaultWorkspaceId,omitempty"`
	DefaultWorkspaceName *string    `json:"defaultWorkspaceName,omitempty"`
	LastCheckedAt        *time.Time `json:"lastCheckedAt,omitempty"`
	LastError            *string    `json:"lastError,omitempty"`
}

type ClickupWorkspaceDto struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ClickupSpaceDto struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Private   bool                `json:"private"`
	Workspace ClickupWorkspaceDto `json:"workspace"`
}

type ClickupFolderDto struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Hidden  bool   `json:"hidden"`
	SpaceID string `json:"spaceId"`
}

type ClickupListDto struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Content  *string `json:"content,omitempty"`
	SpaceID  string  `json:"spaceId"`
	FolderID *string `json:"folderId,omitempty"`
}

type ClickupTaskDto struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    *string    `json:"priority,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	URL         *string    `json:"url,omitempty"`
	ListID      string     `json:"listId"`
	DateCreated time.Time  `json:"dateCreated"`
	DateUpdated time.Time  `json:"dateUpdated"`
	AssigneeIDs []string   `json:"assigneeIds,omitempty"`
}

type CreateClickupTaskDto struct {
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Priority    *int       `json:"priority,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	AssigneeIDs []string   `json:"assigneeIds,omitempty"`
}

type ClickupFoldersResponseDto struct {
	SpaceID string             `json:"spaceId"`
	Folders []ClickupFolderDto `json:"folders"`
	Lists   []ClickupListDto   `json:"lists"`
}

type PaginationDto struct {
	Page    int  `json:"page"`
	HasMore bool `json:"hasMore"`
}

type ClickupTasksResponseDto struct {
	ListID     string           `json:"listId"`
	Pagination PaginationDto    `json:"pagination"`
	Tasks      []ClickupTaskDto `json:"tasks"`
}
