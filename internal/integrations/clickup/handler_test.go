package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-hydra-hub/internal/auth"
)

type serviceStub struct {
	connectStatus ClickupConnectionStatusDto
	connectErr    error
}

func (s *serviceStub) Connect(_ context.Context, _ string, _ ConnectClickupDto) (ClickupConnectionStatusDto, error) {
	return s.connectStatus, s.connectErr
}
func (s *serviceStub) Status(_ context.Context, _ string) (ClickupConnectionStatusDto, error) {
	return ClickupConnectionStatusDto{}, nil
}
func (s *serviceStub) ListSpaces(_ context.Context, _ string) ([]ClickupSpaceDto, error) {
	return nil, nil
}
func (s *serviceStub) ListFolders(_ context.Context, _ string, _ string) (ClickupFoldersResponseDto, error) {
	return ClickupFoldersResponseDto{}, nil
}
func (s *serviceStub) ListTasks(_ context.Context, _ string, _ string, _ int) (ClickupTasksResponseDto, error) {
	return ClickupTasksResponseDto{}, nil
}
func (s *serviceStub) CreateTask(_ context.Context, _ string, _ string, _ CreateClickupTaskDto) (ClickupTaskDto, error) {
	return ClickupTaskDto{}, nil
}

func TestHandlerConnectReturnsUnauthorizedWithoutUser(t *testing.T) {
	handler := NewHandler(&serviceStub{})
	req := httptest.NewRequest(http.MethodPost, "/integrations/clickup/connect", bytes.NewBufferString(`{"token":"x"}`))
	rec := httptest.NewRecorder()

	handler.Connect(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerConnectReturnsStatusPayload(t *testing.T) {
	handler := NewHandler(&serviceStub{
		connectStatus: ClickupConnectionStatusDto{Connected: true, Status: "connected"},
	})

	body, _ := json.Marshal(ConnectClickupDto{Token: "secret"})
	req := httptest.NewRequest(http.MethodPost, "/integrations/clickup/connect", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithUser(req.Context(), auth.SessionUser{ID: "user-1"}))
	rec := httptest.NewRecorder()

	handler.Connect(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
