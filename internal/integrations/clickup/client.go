package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.clickup.com/api/v2"

type APIClient interface {
	GetWorkspaces(ctx context.Context, token string) ([]clickupWorkspace, error)
	GetSpaces(ctx context.Context, token string, workspaceID string) ([]clickupSpace, error)
	GetFolders(ctx context.Context, token string, spaceID string) ([]clickupFolder, error)
	GetSpaceLists(ctx context.Context, token string, spaceID string) ([]clickupList, error)
	GetFolderLists(ctx context.Context, token string, folderID string) ([]clickupList, error)
	GetTasks(ctx context.Context, token string, listID string, page int) ([]clickupTask, bool, error)
	CreateTask(ctx context.Context, token string, listID string, in CreateClickupTaskDto) (clickupTask, error)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
	maxRetries int
}

func NewClient(logger *log.Logger) *Client {
	if logger == nil {
		logger = log.Default()
	}

	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
		logger:     logger,
		maxRetries: 2,
	}
}

func (c *Client) GetWorkspaces(ctx context.Context, token string) ([]clickupWorkspace, error) {
	var response struct {
		Teams []clickupWorkspace `json:"teams"`
	}

	if err := c.doJSON(ctx, token, http.MethodGet, "/team", nil, &response); err != nil {
		return nil, err
	}
	return response.Teams, nil
}

func (c *Client) GetSpaces(ctx context.Context, token string, workspaceID string) ([]clickupSpace, error) {
	var response struct {
		Spaces []clickupSpace `json:"spaces"`
	}

	if err := c.doJSON(ctx, token, http.MethodGet, "/team/"+url.PathEscape(workspaceID)+"/space", nil, &response); err != nil {
		return nil, err
	}
	return response.Spaces, nil
}

func (c *Client) GetFolders(ctx context.Context, token string, spaceID string) ([]clickupFolder, error) {
	params := url.Values{}
	params.Set("archived", "false")

	var response struct {
		Folders []clickupFolder `json:"folders"`
	}
	if err := c.doJSON(ctx, token, http.MethodGet, "/space/"+url.PathEscape(spaceID)+"/folder", params, &response); err != nil {
		return nil, err
	}
	return response.Folders, nil
}

func (c *Client) GetSpaceLists(ctx context.Context, token string, spaceID string) ([]clickupList, error) {
	params := url.Values{}
	params.Set("archived", "false")

	var response struct {
		Lists []clickupList `json:"lists"`
	}
	if err := c.doJSON(ctx, token, http.MethodGet, "/space/"+url.PathEscape(spaceID)+"/list", params, &response); err != nil {
		return nil, err
	}
	return response.Lists, nil
}

func (c *Client) GetFolderLists(ctx context.Context, token string, folderID string) ([]clickupList, error) {
	params := url.Values{}
	params.Set("archived", "false")

	var response struct {
		Lists []clickupList `json:"lists"`
	}
	if err := c.doJSON(ctx, token, http.MethodGet, "/folder/"+url.PathEscape(folderID)+"/list", params, &response); err != nil {
		return nil, err
	}
	return response.Lists, nil
}

func (c *Client) GetTasks(ctx context.Context, token string, listID string, page int) ([]clickupTask, bool, error) {
	params := url.Values{}
	params.Set("archived", "false")
	params.Set("page", strconv.Itoa(page))

	var response struct {
		Tasks    []clickupTask `json:"tasks"`
		LastPage bool          `json:"last_page"`
	}
	if err := c.doJSON(ctx, token, http.MethodGet, "/list/"+url.PathEscape(listID)+"/task", params, &response); err != nil {
		return nil, false, err
	}
	return response.Tasks, !response.LastPage, nil
}

func (c *Client) CreateTask(ctx context.Context, token string, listID string, in CreateClickupTaskDto) (clickupTask, error) {
	payload := map[string]any{
		"name": in.Name,
	}
	if in.Description != nil {
		payload["description"] = *in.Description
	}
	if in.Status != nil {
		payload["status"] = *in.Status
	}
	if in.Priority != nil {
		payload["priority"] = *in.Priority
	}
	if in.DueDate != nil {
		payload["due_date"] = in.DueDate.UTC().UnixMilli()
	}
	if len(in.AssigneeIDs) > 0 {
		assignees := make([]int64, 0, len(in.AssigneeIDs))
		for _, id := range in.AssigneeIDs {
			parsed, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return clickupTask{}, fmt.Errorf("invalid clickup assignee id: %w", err)
			}
			assignees = append(assignees, parsed)
		}
		payload["assignees"] = assignees
	}

	var response clickupTask
	if err := c.doJSON(ctx, token, http.MethodPost, "/list/"+url.PathEscape(listID)+"/task", nil, &response, payload); err != nil {
		return clickupTask{}, err
	}
	return response, nil
}

func (c *Client) doJSON(ctx context.Context, token string, method string, path string, params url.Values, out any, body ...any) error {
	var requestPayload []byte
	if len(body) > 0 && body[0] != nil {
		raw, err := json.Marshal(body[0])
		if err != nil {
			return fmt.Errorf("marshal clickup request: %w", err)
		}
		requestPayload = raw
	}

	endpoint := c.baseURL + path
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var requestBody io.Reader
		if requestPayload != nil {
			requestBody = bytes.NewReader(requestPayload)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
		if err != nil {
			return fmt.Errorf("build clickup request: %w", err)
		}
		req.Header.Set("Authorization", token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = classifyTransportError(err)
			if !shouldRetryTransport(err) || attempt == c.maxRetries {
				c.logger.Printf("level=WARN component=clickup_client msg=%q method=%s path=%s attempt=%d error=%q", "clickup transport failure", method, path, attempt+1, lastErr.Error())
				return lastErr
			}
			c.logger.Printf("level=WARN component=clickup_client msg=%q method=%s path=%s attempt=%d", "clickup transport retry", method, path, attempt+1)
			time.Sleep(backoff(attempt))
			continue
		}

		err = decodeResponse(resp, out)
		resp.Body.Close()
		if err == nil {
			return nil
		}

		lastErr = err
		var upstreamErr *UpstreamError
		if errors.As(err, &upstreamErr) && upstreamErr.Retryable && attempt < c.maxRetries {
			c.logger.Printf("level=WARN component=clickup_client msg=%q method=%s path=%s attempt=%d status=%d", "clickup upstream retry", method, path, attempt+1, upstreamErr.StatusCode)
			time.Sleep(backoff(attempt))
			continue
		}

		c.logger.Printf("level=WARN component=clickup_client msg=%q method=%s path=%s attempt=%d error=%q", "clickup request failed", method, path, attempt+1, err.Error())
		return err
	}

	if lastErr == nil {
		lastErr = ErrBadGateway
	}
	return lastErr
}

type UpstreamError struct {
	StatusCode int
	Message    string
	Retryable  bool
}

func (e *UpstreamError) Error() string {
	return e.Message
}

func decodeResponse(resp *http.Response, out any) error {
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read clickup response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || len(payload) == 0 {
			return nil
		}
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("decode clickup response: %w", err)
		}
		return nil
	}

	message := sanitizeUpstreamMessage(payload, resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidCredentials
	case http.StatusTooManyRequests:
		return &UpstreamError{StatusCode: resp.StatusCode, Message: ErrUpstreamUnavailable.Error(), Retryable: true}
	default:
		if resp.StatusCode >= 500 {
			return &UpstreamError{StatusCode: resp.StatusCode, Message: ErrBadGateway.Error(), Retryable: true}
		}
		return &UpstreamError{StatusCode: resp.StatusCode, Message: message, Retryable: false}
	}
}

func sanitizeUpstreamMessage(payload []byte, statusCode int) string {
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err == nil {
		if value, ok := body["err"].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
		if value, ok := body["error"].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
		if value, ok := body["ECODE"].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}

	switch {
	case statusCode >= 500:
		return ErrBadGateway.Error()
	case statusCode == http.StatusTooManyRequests:
		return ErrUpstreamUnavailable.Error()
	default:
		return "clickup request failed"
	}
}

func shouldRetryTransport(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}

func classifyTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrUpstreamUnavailable
	}
	return ErrBadGateway
}

func backoff(attempt int) time.Duration {
	return time.Duration(attempt+1) * 200 * time.Millisecond
}

type clickupWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type clickupSpace struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

type clickupFolder struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Hidden  bool   `json:"hidden"`
	SpaceID string `json:"space_id"`
}

type clickupList struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Content *string `json:"content"`
	Folder  *struct {
		ID string `json:"id"`
	} `json:"folder"`
	Space *struct {
		ID string `json:"id"`
	} `json:"space"`
}

type clickupTask struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      struct {
		Status string `json:"status"`
	} `json:"status"`
	Priority *struct {
		Priority string `json:"priority"`
	} `json:"priority"`
	DueDate     *string `json:"due_date"`
	URL         *string `json:"url"`
	DateCreated string  `json:"date_created"`
	DateUpdated string  `json:"date_updated"`
	List        struct {
		ID string `json:"id"`
	} `json:"list"`
	Assignees []struct {
		ID int64 `json:"id"`
	} `json:"assignees"`
}
