package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL     string
	AccessToken string
	HTTPClient  *http.Client
}

type Response struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     *ErrorBody      `json:"error"`
	RequestID string          `json:"request_id"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoteListResponse struct {
	Items    []Note `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var envelope Response
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		snippet := strings.TrimSpace(string(bodyBytes))
		if snippet == "" {
			snippet = http.StatusText(resp.StatusCode)
		}
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("unexpected response (status %d): %s", resp.StatusCode, snippet)
	}

	if !envelope.Success {
		msg := "unknown error"
		if envelope.Error != nil && envelope.Error.Message != "" {
			msg = envelope.Error.Message
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("api error: %s", msg)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for successful response", resp.StatusCode)
	}

	return &envelope, nil
}

func (c *Client) GetMe(ctx context.Context) (*UserProfile, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(resp.Data, &profile); err != nil {
		return nil, fmt.Errorf("decoding user profile: %w", err)
	}
	return &profile, nil
}

func (c *Client) CreateNote(ctx context.Context, subject, content string) (*Note, error) {
	body, err := json.Marshal(map[string]string{"subject": subject, "content": content})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tools/notes", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) ListNotes(ctx context.Context, page, pageSize int) (*NoteListResponse, error) {
	path := fmt.Sprintf("/api/v1/tools/notes?page=%d&page_size=%d", page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list NoteListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding note list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tools/notes/"+id, nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) UpdateNote(ctx context.Context, id, subject, content string) (*Note, error) {
	body, err := json.Marshal(map[string]string{"subject": subject, "content": content})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPut, "/api/v1/tools/notes/"+id, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) PatchNote(ctx context.Context, id string, fields map[string]string) (*Note, error) {
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/tools/notes/"+id, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) DeleteNote(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tools/notes/"+id, nil)
	return err
}
