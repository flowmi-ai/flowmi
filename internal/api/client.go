package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	RequestID string          `json:"requestId"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Hint    string         `json:"hint,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

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

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, &Error{
			Code:    CodeNetworkError,
			Message: fmt.Sprintf("creating request: %s", err),
			Cause:   err,
		}
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &Error{
			Code:    CodeNetworkError,
			Message: fmt.Sprintf("executing request: %s", err),
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &Error{
			Code:    CodeNetworkError,
			Message: fmt.Sprintf("reading response: %s", err),
			Cause:   err,
		}
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
		return nil, &Error{
			Code:       CodeUnexpectedResp,
			Message:    fmt.Sprintf("unexpected response (status %d): %s", resp.StatusCode, snippet),
			StatusCode: resp.StatusCode,
			Cause:      err,
		}
	}

	if !envelope.Success {
		code := CodeUnknownError
		msg := "unknown error"
		var hint string
		var details map[string]any
		if envelope.Error != nil {
			if envelope.Error.Code != "" {
				code = envelope.Error.Code
			}
			if envelope.Error.Message != "" {
				msg = envelope.Error.Message
			}
			hint = envelope.Error.Hint
			details = envelope.Error.Details
		}
		return nil, &Error{
			Code:       code,
			Message:    msg,
			RequestID:  envelope.RequestID,
			StatusCode: resp.StatusCode,
			Hint:       hint,
			Details:    details,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{
			Code:       CodeUnexpectedResp,
			Message:    fmt.Sprintf("unexpected status %d for successful response", resp.StatusCode),
			StatusCode: resp.StatusCode,
		}
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

// Search & Scrape types

type WebSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
}

type ImageSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
	TBS  string `json:"tbs,omitempty"`
}

type NewsSearchRequest struct {
	Q    string `json:"q"`
	GL   string `json:"gl,omitempty"`
	HL   string `json:"hl,omitempty"`
	Num  int    `json:"num,omitempty"`
	Page int    `json:"page,omitempty"`
	TBS  string `json:"tbs,omitempty"`
}

type ScrapeRequest struct {
	URL             string `json:"url"`
	IncludeMarkdown bool   `json:"includeMarkdown,omitempty"`
}

type SearchParameters struct {
	Q      string `json:"q"`
	GL     string `json:"gl,omitempty"`
	HL     string `json:"hl,omitempty"`
	Num    int    `json:"num,omitempty"`
	Type   string `json:"type"`
	Engine string `json:"engine"`
}

type OrganicResult struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Position int    `json:"position"`
}

type WebSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	Organic          []*OrganicResult  `json:"organic"`
	Credits          int               `json:"credits"`
}

type ImageResult struct {
	Title        string `json:"title"`
	ImageURL     string `json:"imageUrl"`
	ImageWidth   int    `json:"imageWidth"`
	ImageHeight  int    `json:"imageHeight"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Source       string `json:"source"`
	Domain       string `json:"domain"`
	Link         string `json:"link"`
	Position     int    `json:"position"`
}

type ImageSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	Images           []*ImageResult    `json:"images"`
	Credits          int               `json:"credits"`
}

type NewsResult struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Snippet  string `json:"snippet"`
	Date     string `json:"date"`
	Source   string `json:"source"`
	ImageURL string `json:"imageUrl,omitempty"`
	Position int    `json:"position"`
}

type NewsSearchResponse struct {
	SearchParameters *SearchParameters `json:"searchParameters"`
	News             []*NewsResult     `json:"news"`
	Credits          int               `json:"credits"`
}

type ScrapeResponse struct {
	Text     string            `json:"text"`
	Markdown string            `json:"markdown,omitempty"`
	Metadata map[string]string `json:"metadata"`
	Credits  int               `json:"credits"`
}

// Table types

type Column struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Position   int             `json:"position"`
	IsRequired bool            `json:"isRequired"`
	Options    json.RawMessage `json:"options,omitempty"`
}

type Table struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Columns     []*Column `json:"columns"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TableListResponse struct {
	Items    []*Table `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

type CreateTableRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Columns     []*CreateColumnInput `json:"columns,omitempty"`
}

type CreateColumnInput struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	IsRequired bool            `json:"isRequired,omitempty"`
	Position   *int            `json:"position,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
}

type TablePatch struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ColumnPatch struct {
	Name       *string         `json:"name,omitempty"`
	Type       *string         `json:"type,omitempty"`
	IsRequired *bool           `json:"isRequired,omitempty"`
	Position   *int            `json:"position,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
}

type Row struct {
	ID        string         `json:"id"`
	TableID   string         `json:"tableId"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type RowListResponse struct {
	Items    []*Row `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type QueryRequest struct {
	Filter   *QueryFilter `json:"filter,omitempty"`
	Sort     []*QuerySort `json:"sort,omitempty"`
	Page     int          `json:"page,omitempty"`
	PageSize int          `json:"pageSize,omitempty"`
}

type QueryFilter struct {
	And []*QueryCondition `json:"and,omitempty"`
	Or  []*QueryCondition `json:"or,omitempty"`
}

type QueryCondition struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  any    `json:"value,omitempty"`
}

type QuerySort struct {
	Column    string `json:"column"`
	Direction string `json:"direction,omitempty"`
}

// Drive types

type DriveObject struct {
	ID         string         `json:"id"`
	Path       string         `json:"path"`
	SizeBytes  int64          `json:"sizeBytes"`
	MimeType   string         `json:"mimeType,omitempty"`
	ETag       string         `json:"etag,omitempty"`
	Visibility string         `json:"visibility"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type DriveListResponse struct {
	Items    []DriveObject `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type InitUploadRequest struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	MimeType  string `json:"mimeType,omitempty"`
}

type InitUploadResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"uploadUrl"`
	ExpiresIn int    `json:"expiresIn"`
}

type CompleteUploadRequest struct {
	ETag      string `json:"etag"`
	SizeBytes int64  `json:"sizeBytes"`
}

type DownloadResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresIn   int    `json:"expiresIn"`
}

// Email types

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
	Status      string        `json:"status"`
	Attachments []*Attachment `json:"attachments"`
	SentAt      *time.Time    `json:"sentAt,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
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

// Email API methods

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

func (c *Client) ListEmails(ctx context.Context, page, pageSize int, direction string) (*EmailListResponse, error) {
	path := fmt.Sprintf("/api/v1/email?page=%d&pageSize=%d", page, pageSize)
	if direction != "" {
		path += "&direction=" + url.QueryEscape(direction)
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

func (c *Client) DeleteEmail(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/email/"+id, nil)
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

// Table API methods

func (c *Client) ListTables(ctx context.Context, page, pageSize int) (*TableListResponse, error) {
	path := fmt.Sprintf("/api/v1/tables?page=%d&pageSize=%d", page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list TableListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding table list: %w", err)
	}
	return &list, nil
}

func (c *Client) CreateTable(ctx context.Context, req *CreateTableRequest) (*Table, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) GetTable(ctx context.Context, id string) (*Table, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tables/"+id, nil)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) PatchTable(ctx context.Context, id string, patch *TablePatch) (*Table, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/tables/"+id, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) DeleteTable(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tables/"+id, nil)
	return err
}

func (c *Client) AddColumn(ctx context.Context, tableID string, input *CreateColumnInput) (*Table, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/columns", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) PatchColumn(ctx context.Context, tableID, colID string, patch *ColumnPatch) (*Table, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/tables/"+tableID+"/columns/"+colID, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) DeleteColumn(ctx context.Context, tableID, colID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tables/"+tableID+"/columns/"+colID, nil)
	return err
}

func (c *Client) ListRows(ctx context.Context, tableID string, page, pageSize int) (*RowListResponse, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/rows?page=%d&pageSize=%d", tableID, page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list RowListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding row list: %w", err)
	}
	return &list, nil
}

func (c *Client) CreateRow(ctx context.Context, tableID string, data map[string]any) (*Row, error) {
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/rows", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var row Row
	if err := json.Unmarshal(resp.Data, &row); err != nil {
		return nil, fmt.Errorf("decoding row: %w", err)
	}
	return &row, nil
}

func (c *Client) GetRow(ctx context.Context, tableID, rowID string) (*Row, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tables/"+tableID+"/rows/"+rowID, nil)
	if err != nil {
		return nil, err
	}

	var row Row
	if err := json.Unmarshal(resp.Data, &row); err != nil {
		return nil, fmt.Errorf("decoding row: %w", err)
	}
	return &row, nil
}

func (c *Client) PatchRow(ctx context.Context, tableID, rowID string, data map[string]any) (*Row, error) {
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/tables/"+tableID+"/rows/"+rowID, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var row Row
	if err := json.Unmarshal(resp.Data, &row); err != nil {
		return nil, fmt.Errorf("decoding row: %w", err)
	}
	return &row, nil
}

func (c *Client) DeleteRow(ctx context.Context, tableID, rowID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tables/"+tableID+"/rows/"+rowID, nil)
	return err
}

func (c *Client) QueryRows(ctx context.Context, tableID string, req *QueryRequest) (*RowListResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/rows/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var list RowListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding row list: %w", err)
	}
	return &list, nil
}

func (c *Client) WebSearch(ctx context.Context, req *WebSearchRequest) (*WebSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/web", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result WebSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding web search response: %w", err)
	}
	return &result, nil
}

func (c *Client) ImageSearch(ctx context.Context, req *ImageSearchRequest) (*ImageSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/images", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ImageSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding image search response: %w", err)
	}
	return &result, nil
}

func (c *Client) NewsSearch(ctx context.Context, req *NewsSearchRequest) (*NewsSearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/search/news", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result NewsSearchResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding news search response: %w", err)
	}
	return &result, nil
}

func (c *Client) Scrape(ctx context.Context, req *ScrapeRequest) (*ScrapeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/scrape", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result ScrapeResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding scrape response: %w", err)
	}
	return &result, nil
}

// Drive API methods

func (c *Client) ListDriveObjects(ctx context.Context, page, pageSize int, prefix string) (*DriveListResponse, error) {
	path := fmt.Sprintf("/api/v1/drive/objects?page=%d&pageSize=%d", page, pageSize)
	if prefix != "" {
		path += "&prefix=" + url.QueryEscape(prefix)
	}
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list DriveListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding drive object list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/objects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) GetDriveObjectByPath(ctx context.Context, path string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/object?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) DeleteDriveObject(ctx context.Context, id string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/drive/objects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) DeleteDriveObjectByPath(ctx context.Context, path string) (*DriveObject, error) {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/drive/object?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) InitUpload(ctx context.Context, req *InitUploadRequest) (*InitUploadResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/drive/upload", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result InitUploadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding init upload response: %w", err)
	}
	return &result, nil
}

func (c *Client) CompleteUpload(ctx context.Context, id, etag string, sizeBytes int64) (*DriveObject, error) {
	body, err := json.Marshal(&CompleteUploadRequest{
		ETag:      etag,
		SizeBytes: sizeBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/drive/upload/"+id+"/complete", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var obj DriveObject
	if err := json.Unmarshal(resp.Data, &obj); err != nil {
		return nil, fmt.Errorf("decoding drive object: %w", err)
	}
	return &obj, nil
}

func (c *Client) GetDownloadURL(ctx context.Context, id string) (*DownloadResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/objects/"+id+"/download", nil)
	if err != nil {
		return nil, err
	}

	var result DownloadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding download response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetDownloadURLByPath(ctx context.Context, path string) (*DownloadResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/drive/object/download?path="+url.QueryEscape(path), nil)
	if err != nil {
		return nil, err
	}

	var result DownloadResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("decoding download response: %w", err)
	}
	return &result, nil
}

// UploadToPresignedURL uploads binary data to a presigned R2 URL.
// Returns the ETag from the response header.
func (c *Client) UploadToPresignedURL(ctx context.Context, uploadURL, contentType string, body io.Reader, size int64) (string, error) {
	httpClient := &http.Client{Timeout: 10 * time.Minute}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("creating upload request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.ContentLength = size

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading to presigned URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	etag := resp.Header.Get("ETag")
	// Strip surrounding quotes from ETag if present
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		etag, _ = strconv.Unquote(etag)
	}
	return etag, nil
}

// DownloadFromURL downloads binary data from a presigned R2 URL.
// Caller must close the returned ReadCloser.
func (c *Client) DownloadFromURL(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	httpClient := &http.Client{Timeout: 10 * time.Minute}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading from presigned URL: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed (status %d)", resp.StatusCode)
	}

	return resp.Body, nil
}
