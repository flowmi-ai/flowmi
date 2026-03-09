package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Column struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Position   int             `json:"position"`
	IsRequired bool            `json:"isRequired"`
	Options    json.RawMessage `json:"options,omitempty"`
}

type Table struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Columns     []*Column  `json:"columns"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
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
	DeletedAt *time.Time     `json:"deletedAt,omitempty"`
}

type RowListResponse struct {
	Items    []*Row `json:"items"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type AggregateFunc struct {
	Fn     string `json:"fn"`
	Column string `json:"column,omitempty"`
	Alias  string `json:"alias"`
}

type AggregateResponse struct {
	Results map[string]any `json:"results"`
}

type QueryRequest struct {
	Filter    *QueryFilter     `json:"filter,omitempty"`
	Sort      []*QuerySort     `json:"sort,omitempty"`
	Aggregate []*AggregateFunc `json:"aggregate,omitempty"`
	GroupBy   []string         `json:"groupBy,omitempty"`
	Page      int              `json:"page,omitempty"`
	PageSize  int              `json:"pageSize,omitempty"`
}

type GroupByResponse struct {
	Groups   []map[string]any `json:"groups"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
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

func (c *Client) GroupByRows(ctx context.Context, tableID string, req *QueryRequest) (*GroupByResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/rows/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var grouped GroupByResponse
	if err := json.Unmarshal(resp.Data, &grouped); err != nil {
		return nil, fmt.Errorf("decoding group by response: %w", err)
	}
	return &grouped, nil
}

func (c *Client) AggregateRows(ctx context.Context, tableID string, req *QueryRequest) (*AggregateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/rows/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var agg AggregateResponse
	if err := json.Unmarshal(resp.Data, &agg); err != nil {
		return nil, fmt.Errorf("decoding aggregate response: %w", err)
	}
	return &agg, nil
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

// Table trash methods

func (c *Client) ListTrashedTables(ctx context.Context, page, pageSize int) (*TableListResponse, error) {
	path := fmt.Sprintf("/api/v1/tables/trash?page=%d&pageSize=%d", page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list TableListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding trashed table list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetTrashedTable(ctx context.Context, id string) (*Table, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tables/trash/"+id, nil)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) RestoreTable(ctx context.Context, id string) (*Table, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/trash/"+id+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var table Table
	if err := json.Unmarshal(resp.Data, &table); err != nil {
		return nil, fmt.Errorf("decoding table: %w", err)
	}
	return &table, nil
}

func (c *Client) PermanentlyDeleteTable(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tables/trash/"+id, nil)
	return err
}

// Row trash methods

func (c *Client) ListTrashedRows(ctx context.Context, tableID string, page, pageSize int) (*RowListResponse, error) {
	path := fmt.Sprintf("/api/v1/tables/%s/rows/trash?page=%d&pageSize=%d", tableID, page, pageSize)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var list RowListResponse
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		return nil, fmt.Errorf("decoding trashed row list: %w", err)
	}
	return &list, nil
}

func (c *Client) GetTrashedRow(ctx context.Context, tableID, rowID string) (*Row, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/tables/"+tableID+"/rows/trash/"+rowID, nil)
	if err != nil {
		return nil, err
	}

	var row Row
	if err := json.Unmarshal(resp.Data, &row); err != nil {
		return nil, fmt.Errorf("decoding row: %w", err)
	}
	return &row, nil
}

func (c *Client) RestoreRow(ctx context.Context, tableID, rowID string) (*Row, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/tables/"+tableID+"/rows/trash/"+rowID+"/restore", nil)
	if err != nil {
		return nil, err
	}

	var row Row
	if err := json.Unmarshal(resp.Data, &row); err != nil {
		return nil, fmt.Errorf("decoding row: %w", err)
	}
	return &row, nil
}

func (c *Client) PermanentlyDeleteRow(ctx context.Context, tableID, rowID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/tables/"+tableID+"/rows/trash/"+rowID, nil)
	return err
}
