package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/viper"
)

func setupDriveTest(t *testing.T, server *httptest.Server) {
	t.Helper()
	viper.Set("api_server_url", server.URL)
	viper.Set("access_token", "test-token")
	t.Cleanup(func() {
		viper.Set("api_server_url", "")
		viper.Set("access_token", "")
	})
}

func driveListMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"items": []map[string]any{
					{
						"id":         "obj-1",
						"path":       "/docs/readme.txt",
						"sizeBytes":  1024,
						"mimeType":   "text/plain",
						"visibility": "private",
						"properties": map[string]any{},
						"createdAt":  "2025-06-01T00:00:00Z",
						"updatedAt":  "2025-06-01T00:00:00Z",
					},
					{
						"id":         "obj-2",
						"path":       "/images/photo.png",
						"sizeBytes":  2048000,
						"mimeType":   "image/png",
						"visibility": "private",
						"properties": map[string]any{},
						"createdAt":  "2025-06-02T00:00:00Z",
						"updatedAt":  "2025-06-02T00:00:00Z",
					},
				},
				"total":    2,
				"page":     1,
				"pageSize": 30,
			},
		})
	}))
}

func TestDriveList(t *testing.T) {
	server := driveListMockServer(t, nil)
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive list failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/docs/readme.txt") {
		t.Errorf("output missing '/docs/readme.txt', got:\n%s", output)
	}
	if !strings.Contains(output, "/images/photo.png") {
		t.Errorf("output missing '/images/photo.png', got:\n%s", output)
	}
	// Summary line now goes to stderr, not stdout
	if strings.Contains(output, "Showing") {
		t.Errorf("stdout should not contain summary line, got:\n%s", output)
	}
}

func TestDriveListWithPrefix(t *testing.T) {
	server := driveListMockServer(t, func(r *http.Request) {
		if got := r.URL.Query().Get("prefix"); got != "/docs" {
			t.Errorf("prefix = %q, want /docs", got)
		}
	})
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "list", "--prefix", "/docs"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive list --prefix failed: %v", err)
	}
}

func TestDriveListJSON(t *testing.T) {
	server := driveListMockServer(t, nil)
	defer server.Close()
	setupDriveTest(t, server)
	viper.Set("output", "json")
	t.Cleanup(func() { viper.Set("output", "") })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive list -o json failed: %v", err)
	}

	var list api.DriveListResponse
	if err := json.Unmarshal(buf.Bytes(), &list); err != nil {
		t.Fatalf("JSON output not parseable as DriveListResponse: %v\nOutput:\n%s", err, buf.String())
	}
	if len(list.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(list.Items))
	}
	if list.Items[0].Path != "/docs/readme.txt" {
		t.Errorf("Items[0].Path = %q, want /docs/readme.txt", list.Items[0].Path)
	}
}

func TestDriveView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         "obj-1",
				"path":       "/docs/readme.txt",
				"sizeBytes":  1024,
				"mimeType":   "text/plain",
				"etag":       "abc123",
				"visibility": "private",
				"properties": map[string]any{},
				"createdAt":  "2025-06-01T00:00:00Z",
				"updatedAt":  "2025-06-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "view", "/docs/readme.txt"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive view failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "obj-1") {
		t.Errorf("output missing ID, got:\n%s", output)
	}
	if !strings.Contains(output, "/docs/readme.txt") {
		t.Errorf("output missing path, got:\n%s", output)
	}
}

func TestDriveViewByUUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/drive/objects/550e8400-e29b-41d4-a716-446655440000" {
			t.Errorf("path = %s, want /api/v1/drive/objects/550e8400-e29b-41d4-a716-446655440000", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         "550e8400-e29b-41d4-a716-446655440000",
				"path":       "/docs/readme.txt",
				"sizeBytes":  1024,
				"mimeType":   "text/plain",
				"visibility": "private",
				"properties": map[string]any{},
				"createdAt":  "2025-06-01T00:00:00Z",
				"updatedAt":  "2025-06-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "view", "550e8400-e29b-41d4-a716-446655440000"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive view by UUID failed: %v", err)
	}
}

func TestDriveDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         "obj-1",
				"path":       "/docs/readme.txt",
				"sizeBytes":  1024,
				"mimeType":   "text/plain",
				"visibility": "private",
				"properties": map[string]any{},
				"createdAt":  "2025-06-01T00:00:00Z",
				"updatedAt":  "2025-06-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "delete", "/docs/readme.txt"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive delete failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Deleted: /docs/readme.txt (id=obj-1)") {
		t.Errorf("output missing delete confirmation with ID, got:\n%s", output)
	}
}

func TestDriveUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/drive/upload":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":        "upload-1",
					"uploadUrl": "http://" + r.Host + "/presigned-put",
					"expiresIn": 3600,
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/presigned-put":
			w.Header().Set("ETag", `"etag-abc"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/complete"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "obj-new",
					"path":       "/test.txt",
					"sizeBytes":  11,
					"mimeType":   "text/plain",
					"visibility": "private",
					"properties": map[string]any{},
					"createdAt":  "2025-06-01T00:00:00Z",
					"updatedAt":  "2025-06-01T00:00:00Z",
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setupDriveTest(t, server)

	// Create a temp file
	tmpFile := t.TempDir() + "/test.txt"
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "upload", tmpFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive upload failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Uploaded: /test.txt") || !strings.Contains(output, "id=obj-new") {
		t.Errorf("output missing upload confirmation with ID, got:\n%s", output)
	}
	if !strings.Contains(output, "size=") {
		t.Errorf("output missing size= field, got:\n%s", output)
	}
}

func TestDriveDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/download"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"downloadUrl": "http://" + r.Host + "/presigned-get",
					"expiresIn":   3600,
				},
			})
		case r.URL.Path == "/presigned-get":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("file contents here"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"drive", "download", "/docs/readme.txt"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive download failed: %v", err)
	}

	output := buf.String()
	if output != "file contents here" {
		t.Errorf("download output = %q, want 'file contents here'", output)
	}
}

func TestDriveDownloadToDest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/drive/object" && r.URL.Query().Get("path") != "":
			// Metadata request for MIME type
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "obj-1",
					"path":       "/docs/readme.txt",
					"sizeBytes":  13,
					"mimeType":   "text/plain",
					"visibility": "private",
					"properties": map[string]any{},
					"createdAt":  "2025-06-01T00:00:00Z",
					"updatedAt":  "2025-06-01T00:00:00Z",
				},
			})
		case strings.HasSuffix(r.URL.Path, "/download"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"downloadUrl": "http://" + r.Host + "/presigned-get",
					"expiresIn":   3600,
				},
			})
		case r.URL.Path == "/presigned-get":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("saved to file"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setupDriveTest(t, server)

	dest := t.TempDir() + "/downloaded.txt"

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"drive", "download", "/docs/readme.txt", "--dest", dest})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive download --dest failed: %v", err)
	}

	// stdout should be empty (file goes to dest)
	if buf.Len() != 0 {
		t.Errorf("stdout should be empty, got: %q", buf.String())
	}

	// stderr should have status message with MIME type
	stderrOutput := errBuf.String()
	if !strings.Contains(stderrOutput, "Downloaded") {
		t.Errorf("stderr missing download status, got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "type=text/plain") {
		t.Errorf("stderr missing MIME type, got: %q", stderrOutput)
	}

	// Verify file contents
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(data) != "saved to file" {
		t.Errorf("file contents = %q, want 'saved to file'", string(data))
	}
}

func TestDriveListCmdHasFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"limit", "L"},
		{"page", "p"},
		{"prefix", ""},
	}
	for _, tt := range flags {
		f := driveListCmd.Flags().Lookup(tt.name)
		if f == nil {
			t.Errorf("--%s flag not found", tt.name)
			continue
		}
		if tt.shorthand != "" && f.Shorthand != tt.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
		}
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"not-a-uuid", false},
		{"/some/path", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isUUID(tt.input); got != tt.want {
			t.Errorf("isUUID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/docs/readme.txt", "/docs/readme.txt"},
		{"docs/readme.txt", "/docs/readme.txt"},
		{"readme.txt", "/readme.txt"},
	}
	for _, tt := range tests {
		if got := normalizePath(tt.input); got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDriveListSummaryOnStderr(t *testing.T) {
	server := driveListMockServer(t, nil)
	defer server.Close()
	setupDriveTest(t, server)

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"drive", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("drive list failed: %v", err)
	}

	if !strings.Contains(errBuf.String(), "Showing 2 of 2 files") {
		t.Errorf("stderr missing summary line, got: %q", errBuf.String())
	}
}

func TestDriveUploadUnsupportedFormat(t *testing.T) {
	// upload should reject -o table
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/drive/upload":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":        "upload-1",
					"uploadUrl": "http://" + r.Host + "/presigned-put",
					"expiresIn": 3600,
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/presigned-put":
			w.Header().Set("ETag", `"etag-abc"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/complete"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "obj-new",
					"path":       "/test.txt",
					"sizeBytes":  11,
					"mimeType":   "text/plain",
					"visibility": "private",
					"properties": map[string]any{},
					"createdAt":  "2025-06-01T00:00:00Z",
					"updatedAt":  "2025-06-01T00:00:00Z",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	setupDriveTest(t, server)
	viper.Set("output", "table")
	t.Cleanup(func() { viper.Set("output", "") })

	tmpFile := t.TempDir() + "/test.txt"
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	rootCmd.SetArgs([]string{"drive", "upload", tmpFile})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error = %q, want 'unsupported output format'", err.Error())
	}
}

func TestDriveDeleteUnsupportedFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         "obj-1",
				"path":       "/docs/readme.txt",
				"sizeBytes":  1024,
				"mimeType":   "text/plain",
				"visibility": "private",
				"properties": map[string]any{},
				"createdAt":  "2025-06-01T00:00:00Z",
				"updatedAt":  "2025-06-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()
	setupDriveTest(t, server)
	viper.Set("output", "table")
	t.Cleanup(func() { viper.Set("output", "") })

	rootCmd.SetArgs([]string{"drive", "delete", "/docs/readme.txt"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error = %q, want 'unsupported output format'", err.Error())
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1024 B (1.0 KB)"},
		{1048576, "1048576 B (1.0 MB)"},
		{1073741824, "1073741824 B (1.0 GB)"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.input); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
