package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Attachment struct {
	ID                 string `json:"id"`
	Filename           string `json:"filename"`
	ContentType        string `json:"contentType"`
	ContentDisposition string `json:"contentDisposition"`
	ContentID          string `json:"contentId"`
	Size               int64  `json:"size"`
}

type Email struct {
	ID          string        `json:"id"`
	MailboxID   string        `json:"mailboxId"`
	Direction   string        `json:"direction"`
	From        string        `json:"from"`
	To          []string      `json:"to"`
	CC          []string      `json:"cc"`
	BCC         []string      `json:"bcc"`
	ReplyTo     []string      `json:"replyTo"`
	Subject     string        `json:"subject"`
	Snippet     string        `json:"snippet,omitempty"`
	Status      string        `json:"status"`
	Attachments []*Attachment `json:"attachments"`
	ReadAt      *time.Time    `json:"readAt,omitempty"`
	ArchivedAt  *time.Time    `json:"archivedAt,omitempty"`
	SentAt      *time.Time    `json:"sentAt,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	DeletedAt   *time.Time    `json:"deletedAt,omitempty"`
}

type EmailDetail struct {
	Email
	TextBody string `json:"textBody"`
	HTMLBody string `json:"htmlBody"`
}

type EmailListResponse struct {
	Items    []*Email `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type SendEmailRequest struct {
	MailboxID string   `json:"mailboxId"`
	To        []string `json:"to"`
	CC        []string `json:"cc,omitempty"`
	BCC       []string `json:"bcc,omitempty"`
	ReplyTo   string   `json:"replyTo,omitempty"`
	Subject   string   `json:"subject"`
	HTML      string   `json:"html,omitempty"`
	Text      string   `json:"text,omitempty"`
}

type Mailbox struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	DisplayName string    `json:"displayName"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateMailboxRequest struct {
	Address     string `json:"address"`
	DisplayName string `json:"displayName,omitempty"`
}

type MailboxPatch struct {
	DisplayName *string `json:"displayName,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

func (c *Client) SendEmail(ctx context.Context, req *SendEmailRequest) (*EmailDetail, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/send", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) ListEmails(ctx context.Context, page, pageSize int, direction string, isRead *bool, archived *bool) (*EmailListResponse, error) {
	path := fmt.Sprintf("/api/v1/email?page=%d&pageSize=%d", page, pageSize)
	if direction != "" {
		path += "&direction=" + url.QueryEscape(direction)
	}
	if isRead != nil {
		path += fmt.Sprintf("&isRead=%t", *isRead)
	}
	if archived != nil {
		path += fmt.Sprintf("&archived=%t", *archived)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list EmailListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding email list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetEmail(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/email/"+id, nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) MarkEmailAsRead(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/"+id+"/mark-as-read", nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) MarkEmailAsUnread(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/"+id+"/mark-as-unread", nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) ArchiveEmail(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/"+id+"/archive", nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) UnarchiveEmail(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/"+id+"/unarchive", nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) DeleteEmail(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/email/"+id, nil)
	return err
}

func (c *Client) ListTrashedEmails(ctx context.Context, page, pageSize int, direction string) (*EmailListResponse, error) {
	path := fmt.Sprintf("/api/v1/email/trash?page=%d&pageSize=%d", page, pageSize)
	if direction != "" {
		path += "&direction=" + url.QueryEscape(direction)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list EmailListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding trashed email list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetTrashedEmail(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/email/trash/"+id, nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) RestoreEmail(ctx context.Context, id string) (*EmailDetail, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/trash/"+id+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var email EmailDetail
	if err := json.Unmarshal(resp.Data, &email); err != nil {
		return nil, fmt.Errorf("decoding email: %w", err)
	}
	return &email, nil
}

func (c *Client) PermanentlyDeleteEmail(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/email/trash/"+id, nil)
	return err
}

func (c *Client) CreateMailbox(ctx context.Context, req *CreateMailboxRequest) (*Mailbox, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/email/mailboxes", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var mailbox Mailbox
	if err := json.Unmarshal(resp.Data, &mailbox); err != nil {
		return nil, fmt.Errorf("decoding mailbox: %w", err)
	}
	return &mailbox, nil
}

func (c *Client) ListMailboxes(ctx context.Context) ([]*Mailbox, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/email/mailboxes", nil)
	if err != nil {
		return nil, err
	}

	var mailboxes []*Mailbox
	if err := json.Unmarshal(resp.Data, &mailboxes); err != nil {
		return nil, fmt.Errorf("decoding mailboxes: %w", err)
	}
	return mailboxes, nil
}

func (c *Client) PatchMailbox(ctx context.Context, id string, patch *MailboxPatch) (*Mailbox, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/email/mailboxes/"+id, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var mailbox Mailbox
	if err := json.Unmarshal(resp.Data, &mailbox); err != nil {
		return nil, fmt.Errorf("decoding mailbox: %w", err)
	}
	return &mailbox, nil
}

func (c *Client) DeleteMailbox(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/email/mailboxes/"+id, nil)
	return err
}
