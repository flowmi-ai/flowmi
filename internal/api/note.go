package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Note struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Subject   string     `json:"subject"`
	Content   string     `json:"content"`
	Labels    []string   `json:"labels"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type NoteListResponse struct {
	Items    []Note `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type CreateNoteRequest struct {
	Subject string   `json:"subject"`
	Content string   `json:"content"`
	Labels  []string `json:"labels,omitempty"`
}

type NotePatch struct {
	Subject *string   `json:"subject,omitempty"`
	Content *string   `json:"content,omitempty"`
	Labels  *[]string `json:"labels,omitempty"`
}

func (c *Client) CreateNote(ctx context.Context, subject, content string, labels []string) (*Note, error) {
	body, err := json.Marshal(CreateNoteRequest{
		Subject: subject,
		Content: content,
		Labels:  labels,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/notes", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) ListNotes(ctx context.Context, page, pageSize int, labels []string, status, query string) (*NoteListResponse, error) {
	path := fmt.Sprintf("/api/v1/notes?page=%d&page_size=%d", page, pageSize)
	if len(labels) > 0 {
		path += "&labels=" + url.QueryEscape(strings.Join(labels, ","))
	}
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}
	if query != "" {
		path += "&q=" + url.QueryEscape(query)
	}
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
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/notes/"+id, nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) PatchNote(ctx context.Context, id string, patch *NotePatch) (*Note, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/notes/"+id, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) DeleteNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/notes/"+id, nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}

func (c *Client) RestoreNote(ctx context.Context, id string) (*Note, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/notes/"+id+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var note Note
	if err := json.Unmarshal(resp.Data, &note); err != nil {
		return nil, fmt.Errorf("decoding note: %w", err)
	}
	return &note, nil
}
