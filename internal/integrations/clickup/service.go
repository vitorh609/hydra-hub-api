package clickup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	repo   Repository
	client APIClient
	cipher CredentialCipher
	logger *log.Logger
}

func NewService(repo Repository, client APIClient, cipher CredentialCipher, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}

	return &Service{
		repo:   repo,
		client: client,
		cipher: cipher,
		logger: logger,
	}
}

func (s *Service) Connect(ctx context.Context, userID string, in ConnectClickupDto) (ClickupConnectionStatusDto, error) {
	if !s.cipher.Enabled() {
		return ClickupConnectionStatusDto{}, ErrIntegrationDisabled
	}

	token := strings.TrimSpace(in.Token)
	if token == "" {
		return ClickupConnectionStatusDto{}, fmt.Errorf("token is required")
	}

	workspaces, err := s.client.GetWorkspaces(ctx, token)
	if err != nil {
		return ClickupConnectionStatusDto{}, err
	}

	if in.DefaultWorkspaceID != nil && strings.TrimSpace(*in.DefaultWorkspaceID) != "" {
		if !containsWorkspace(workspaces, *in.DefaultWorkspaceID) {
			return ClickupConnectionStatusDto{}, fmt.Errorf("defaultWorkspaceId not accessible with provided token")
		}
	}

	encrypted, err := s.cipher.Encrypt(token)
	if err != nil {
		return ClickupConnectionStatusDto{}, err
	}

	now := time.Now().UTC()
	connection, err := s.repo.UpsertConnection(ctx, Connection{
		UserID:               userID,
		TokenCiphertext:      encrypted,
		TokenKeyVersion:      s.cipher.KeyVersion(),
		DefaultWorkspaceID:   trimOptional(in.DefaultWorkspaceID),
		DefaultWorkspaceName: trimOptional(in.DefaultWorkspaceName),
		Status:               "connected",
		LastCheckedAt:        &now,
		LastError:            nil,
	})
	if err != nil {
		return ClickupConnectionStatusDto{}, err
	}

	s.logger.Printf("level=INFO component=clickup_service msg=%q user_id=%s", "clickup connection updated", userID)
	return connectionToStatus(connection, true), nil
}

func (s *Service) Status(ctx context.Context, userID string) (ClickupConnectionStatusDto, error) {
	connection, err := s.repo.GetConnectionByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			return ClickupConnectionStatusDto{Connected: false, Status: "disconnected"}, nil
		}
		return ClickupConnectionStatusDto{}, err
	}

	token, err := s.cipher.Decrypt(connection.TokenCiphertext)
	if err != nil {
		return ClickupConnectionStatusDto{}, err
	}

	now := time.Now().UTC()
	_, upstreamErr := s.client.GetWorkspaces(ctx, token)
	if upstreamErr != nil {
		lastError := internalErrorMessage(upstreamErr)
		if markErr := s.repo.MarkConnectionHealth(ctx, userID, "error", &lastError, now); markErr != nil {
			s.logger.Printf("level=WARN component=clickup_service msg=%q user_id=%s error=%q", "failed to persist clickup health status", userID, markErr.Error())
		}
		connection.Status = "error"
		connection.LastCheckedAt = &now
		connection.LastError = &lastError
		return connectionToStatus(connection, false), nil
	}

	connection.Status = "connected"
	connection.LastCheckedAt = &now
	connection.LastError = nil
	if markErr := s.repo.MarkConnectionHealth(ctx, userID, "connected", nil, now); markErr != nil {
		s.logger.Printf("level=WARN component=clickup_service msg=%q user_id=%s error=%q", "failed to persist clickup health status", userID, markErr.Error())
	}
	return connectionToStatus(connection, true), nil
}

func (s *Service) ListSpaces(ctx context.Context, userID string) ([]ClickupSpaceDto, error) {
	token, _, err := s.tokenForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	workspaces, err := s.client.GetWorkspaces(ctx, token)
	if err != nil {
		return nil, err
	}

	out := make([]ClickupSpaceDto, 0)
	for _, workspace := range workspaces {
		spaces, err := s.client.GetSpaces(ctx, token, workspace.ID)
		if err != nil {
			return nil, err
		}
		for _, space := range spaces {
			out = append(out, ClickupSpaceDto{
				ID:      space.ID,
				Name:    space.Name,
				Private: space.Private,
				Workspace: ClickupWorkspaceDto{
					ID:   workspace.ID,
					Name: workspace.Name,
				},
			})
		}
	}

	return out, nil
}

func (s *Service) ListFolders(ctx context.Context, userID string, spaceID string) (ClickupFoldersResponseDto, error) {
	token, _, err := s.tokenForUser(ctx, userID)
	if err != nil {
		return ClickupFoldersResponseDto{}, err
	}

	folders, err := s.client.GetFolders(ctx, token, spaceID)
	if err != nil {
		return ClickupFoldersResponseDto{}, err
	}

	lists, err := s.client.GetSpaceLists(ctx, token, spaceID)
	if err != nil {
		return ClickupFoldersResponseDto{}, err
	}

	out := ClickupFoldersResponseDto{
		SpaceID: spaceID,
		Folders: make([]ClickupFolderDto, 0, len(folders)),
		Lists:   make([]ClickupListDto, 0, len(lists)),
	}

	for _, folder := range folders {
		out.Folders = append(out.Folders, ClickupFolderDto{
			ID:      folder.ID,
			Name:    folder.Name,
			Hidden:  folder.Hidden,
			SpaceID: folder.SpaceID,
		})

		folderLists, err := s.client.GetFolderLists(ctx, token, folder.ID)
		if err != nil {
			return ClickupFoldersResponseDto{}, err
		}
		for _, list := range folderLists {
			out.Lists = append(out.Lists, mapList(list, spaceID))
		}
	}

	for _, list := range lists {
		out.Lists = append(out.Lists, mapList(list, spaceID))
	}

	return out, nil
}

func (s *Service) ListTasks(ctx context.Context, userID string, listID string, page int) (ClickupTasksResponseDto, error) {
	token, _, err := s.tokenForUser(ctx, userID)
	if err != nil {
		return ClickupTasksResponseDto{}, err
	}

	tasks, hasMore, err := s.client.GetTasks(ctx, token, listID, page)
	if err != nil {
		return ClickupTasksResponseDto{}, err
	}

	out := ClickupTasksResponseDto{
		ListID: listID,
		Pagination: PaginationDto{
			Page:    page,
			HasMore: hasMore,
		},
		Tasks: make([]ClickupTaskDto, 0, len(tasks)),
	}

	for _, task := range tasks {
		out.Tasks = append(out.Tasks, mapTask(task))
	}

	return out, nil
}

func (s *Service) CreateTask(ctx context.Context, userID string, listID string, in CreateClickupTaskDto) (ClickupTaskDto, error) {
	token, _, err := s.tokenForUser(ctx, userID)
	if err != nil {
		return ClickupTaskDto{}, err
	}

	task, err := s.client.CreateTask(ctx, token, listID, in)
	if err != nil {
		return ClickupTaskDto{}, err
	}

	return mapTask(task), nil
}

func (s *Service) tokenForUser(ctx context.Context, userID string) (string, Connection, error) {
	if !s.cipher.Enabled() {
		return "", Connection{}, ErrIntegrationDisabled
	}

	connection, err := s.repo.GetConnectionByUserID(ctx, userID)
	if err != nil {
		return "", Connection{}, err
	}

	token, err := s.cipher.Decrypt(connection.TokenCiphertext)
	if err != nil {
		return "", Connection{}, err
	}

	return token, connection, nil
}

func connectionToStatus(connection Connection, connected bool) ClickupConnectionStatusDto {
	return ClickupConnectionStatusDto{
		Connected:            connected,
		Status:               connection.Status,
		DefaultWorkspaceID:   connection.DefaultWorkspaceID,
		DefaultWorkspaceName: connection.DefaultWorkspaceName,
		LastCheckedAt:        connection.LastCheckedAt,
		LastError:            connection.LastError,
	}
}

func containsWorkspace(workspaces []clickupWorkspace, workspaceID string) bool {
	for _, workspace := range workspaces {
		if workspace.ID == workspaceID {
			return true
		}
	}
	return false
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func mapList(list clickupList, fallbackSpaceID string) ClickupListDto {
	var folderID *string
	if list.Folder != nil && list.Folder.ID != "" {
		id := list.Folder.ID
		folderID = &id
	}

	spaceID := fallbackSpaceID
	if list.Space != nil && list.Space.ID != "" {
		spaceID = list.Space.ID
	}

	return ClickupListDto{
		ID:       list.ID,
		Name:     list.Name,
		Content:  list.Content,
		SpaceID:  spaceID,
		FolderID: folderID,
	}
}

func mapTask(task clickupTask) ClickupTaskDto {
	var priority *string
	if task.Priority != nil && task.Priority.Priority != "" {
		priority = &task.Priority.Priority
	}

	var dueDate *time.Time
	if task.DueDate != nil && *task.DueDate != "" {
		if millis, err := strconv.ParseInt(*task.DueDate, 10, 64); err == nil {
			parsed := time.UnixMilli(millis).UTC()
			dueDate = &parsed
		}
	}

	description := trimOptional(&task.Description)
	createdAt := parseClickupTime(task.DateCreated)
	updatedAt := parseClickupTime(task.DateUpdated)

	assigneeIDs := make([]string, 0, len(task.Assignees))
	for _, assignee := range task.Assignees {
		assigneeIDs = append(assigneeIDs, strconv.FormatInt(assignee.ID, 10))
	}

	return ClickupTaskDto{
		ID:          task.ID,
		Name:        task.Name,
		Description: description,
		Status:      task.Status.Status,
		Priority:    priority,
		DueDate:     dueDate,
		URL:         task.URL,
		ListID:      task.List.ID,
		DateCreated: createdAt,
		DateUpdated: updatedAt,
		AssigneeIDs: assigneeIDs,
	}
}

func parseClickupTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(millis).UTC()
}

func internalErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return "clickup credentials are invalid"
	case errors.Is(err, ErrUpstreamUnavailable):
		return "clickup temporarily unavailable"
	case errors.Is(err, ErrBadGateway):
		return "clickup request failed"
	default:
		return "clickup request failed"
	}
}
