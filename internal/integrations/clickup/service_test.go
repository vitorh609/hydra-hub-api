package clickup

import (
	"context"
	"io"
	"log"
	"testing"
	"time"
)

type repoStub struct {
	connection   Connection
	getErr       error
	upserted     Connection
	healthStatus string
	healthErr    *string
}

func (r *repoStub) UpsertConnection(_ context.Context, connection Connection) (Connection, error) {
	r.upserted = connection
	connection.CreatedAt = time.Now().UTC()
	connection.UpdatedAt = connection.CreatedAt
	r.connection = connection
	return connection, nil
}

func (r *repoStub) EnsureSchema(_ context.Context) error { return nil }

func (r *repoStub) GetConnectionByUserID(_ context.Context, userID string) (Connection, error) {
	if r.getErr != nil {
		return Connection{}, r.getErr
	}
	if r.connection.UserID != userID {
		return Connection{}, ErrConnectionNotFound
	}
	return r.connection, nil
}

func (r *repoStub) MarkConnectionHealth(_ context.Context, _ string, status string, lastError *string, _ time.Time) error {
	r.healthStatus = status
	r.healthErr = lastError
	return nil
}

type clientStub struct {
	workspaces []clickupWorkspace
	tasks      []clickupTask
	hasMore    bool
	err        error
}

func (c *clientStub) GetWorkspaces(_ context.Context, _ string) ([]clickupWorkspace, error) {
	return c.workspaces, c.err
}
func (c *clientStub) GetSpaces(_ context.Context, _ string, _ string) ([]clickupSpace, error) {
	return nil, c.err
}
func (c *clientStub) GetFolders(_ context.Context, _ string, _ string) ([]clickupFolder, error) {
	return nil, c.err
}
func (c *clientStub) GetSpaceLists(_ context.Context, _ string, _ string) ([]clickupList, error) {
	return nil, c.err
}
func (c *clientStub) GetFolderLists(_ context.Context, _ string, _ string) ([]clickupList, error) {
	return nil, c.err
}
func (c *clientStub) GetTasks(_ context.Context, _ string, _ string, _ int) ([]clickupTask, bool, error) {
	return c.tasks, c.hasMore, c.err
}
func (c *clientStub) CreateTask(_ context.Context, _ string, _ string, _ CreateClickupTaskDto) (clickupTask, error) {
	return clickupTask{}, c.err
}

type cipherStub struct {
	encrypted string
	decrypted string
	enabled   bool
	err       error
}

func (c *cipherStub) Encrypt(_ string) (string, error) { return c.encrypted, c.err }
func (c *cipherStub) Decrypt(_ string) (string, error) { return c.decrypted, c.err }
func (c *cipherStub) KeyVersion() string               { return "v1" }
func (c *cipherStub) Enabled() bool                    { return c.enabled }

func TestServiceConnectEncryptsAndStoresConnection(t *testing.T) {
	repo := &repoStub{}
	client := &clientStub{
		workspaces: []clickupWorkspace{{ID: "team-1", Name: "Workspace"}},
	}
	cipher := &cipherStub{encrypted: "cipher", enabled: true}
	service := &Service{
		repo:   repo,
		client: client,
		cipher: cipher,
		logger: nilLogger(),
	}

	status, err := service.Connect(context.Background(), "user-1", ConnectClickupDto{
		Token:              "secret-token",
		DefaultWorkspaceID: stringPtr("team-1"),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !status.Connected || repo.upserted.TokenCiphertext != "cipher" {
		t.Fatalf("expected encrypted connection to be saved, got status=%+v upsert=%+v", status, repo.upserted)
	}
}

func TestServiceStatusReturnsDisconnectedWhenMissingConnection(t *testing.T) {
	repo := &repoStub{getErr: ErrConnectionNotFound}
	service := &Service{
		repo:   repo,
		client: &clientStub{},
		cipher: &cipherStub{enabled: true},
		logger: nilLogger(),
	}

	status, err := service.Status(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if status.Connected || status.Status != "disconnected" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestServiceStatusMarksInvalidCredentials(t *testing.T) {
	repo := &repoStub{
		connection: Connection{
			UserID:          "user-1",
			TokenCiphertext: "cipher",
			Status:          "connected",
		},
	}
	service := &Service{
		repo:   repo,
		client: &clientStub{err: ErrInvalidCredentials},
		cipher: &cipherStub{enabled: true, decrypted: "secret"},
		logger: nilLogger(),
	}

	status, err := service.Status(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if status.Connected {
		t.Fatalf("expected disconnected status, got %+v", status)
	}
	if repo.healthStatus != "error" || repo.healthErr == nil {
		t.Fatalf("expected health error to be persisted, got status=%q err=%v", repo.healthStatus, repo.healthErr)
	}
}

func TestMapTaskParsesNormalizedFields(t *testing.T) {
	task := clickupTask{
		ID:          "task-1",
		Name:        "Task",
		Description: "Desc",
		DueDate:     stringPtr("1712345678000"),
		DateCreated: "1712345678000",
		DateUpdated: "1712345688000",
		URL:         stringPtr("https://app.clickup.com/t/1"),
		Assignees: []struct {
			ID int64 "json:\"id\""
		}{{ID: 11}, {ID: 22}},
	}
	task.Status.Status = "open"
	task.List.ID = "list-1"
	task.Priority = &struct {
		Priority string "json:\"priority\""
	}{Priority: "high"}

	got := mapTask(task)

	if got.ID != "task-1" || got.ListID != "list-1" || got.Status != "open" {
		t.Fatalf("unexpected task mapping: %+v", got)
	}
	if got.DueDate == nil || got.Priority == nil || len(got.AssigneeIDs) != 2 {
		t.Fatalf("expected normalized optional fields, got %+v", got)
	}
}

func nilLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func stringPtr(v string) *string { return &v }
