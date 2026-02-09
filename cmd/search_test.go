package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowmi/flowmi/internal/api"
	"github.com/spf13/viper"
)

func webSearchMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"searchParameters": map[string]any{"q": "test query", "type": "search", "engine": "google"},
				"organic": []map[string]any{
					{"title": "Result One", "link": "https://example.com/1", "snippet": "First result", "position": 1},
					{"title": "Result Two", "link": "https://example.com/2", "snippet": "Second result", "position": 2},
				},
				"credits": 1,
			},
		})
	}))
}

func imageSearchMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"searchParameters": map[string]any{"q": "test query", "type": "images", "engine": "google"},
				"images": []map[string]any{
					{"title": "Image One", "imageUrl": "https://example.com/img1.png", "imageWidth": 800, "imageHeight": 600, "domain": "example.com", "position": 1},
				},
				"credits": 1,
			},
		})
	}))
}

func newsSearchMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"searchParameters": map[string]any{"q": "test query", "type": "news", "engine": "google"},
				"news": []map[string]any{
					{"title": "News One", "link": "https://example.com/news1", "snippet": "Breaking news", "date": "2 hours ago", "source": "Test News", "position": 1},
				},
				"credits": 1,
			},
		})
	}))
}

func setupSearchTest(t *testing.T, server *httptest.Server) {
	t.Helper()
	viper.Set("api_server_url", server.URL)
	viper.Set("access_token", "test-token")
	t.Cleanup(func() {
		viper.Set("api_server_url", "")
		viper.Set("access_token", "")
	})
}

func TestSearchDefault(t *testing.T) {
	server := webSearchMockServer(t, func(r *http.Request) {
		if r.URL.Path != "/api/v1/tools/web-search" {
			t.Errorf("path = %s, want /api/v1/tools/web-search", r.URL.Path)
		}
	})
	defer server.Close()
	setupSearchTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "test", "query"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Result One") {
		t.Errorf("output missing 'Result One', got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 2 results") {
		t.Errorf("output missing summary line, got:\n%s", output)
	}
}

func TestSearchWeb(t *testing.T) {
	server := webSearchMockServer(t, func(r *http.Request) {
		var req api.WebSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang concurrency" {
			t.Errorf("q = %q, want golang concurrency", req.Q)
		}
		if req.Num != 5 {
			t.Errorf("num = %d, want 5", req.Num)
		}
	})
	defer server.Close()
	setupSearchTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "web", "golang", "concurrency", "-L", "5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search web failed: %v", err)
	}
}

func TestSearchWebJSON(t *testing.T) {
	server := webSearchMockServer(t, nil)
	defer server.Close()
	setupSearchTest(t, server)
	viper.Set("output", "json")
	t.Cleanup(func() { viper.Set("output", "") })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "web", "test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search web -o json failed: %v", err)
	}

	var result api.WebSearchResponse
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON output not parseable: %v\nOutput:\n%s", err, buf.String())
	}
	if len(result.Organic) != 2 {
		t.Errorf("len(Organic) = %d, want 2", len(result.Organic))
	}
}

func TestSearchImages(t *testing.T) {
	server := imageSearchMockServer(t, func(r *http.Request) {
		var req api.ImageSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang gopher" {
			t.Errorf("q = %q, want golang gopher", req.Q)
		}
		if req.TBS != "isz:l" {
			t.Errorf("tbs = %q, want isz:l", req.TBS)
		}
	})
	defer server.Close()
	setupSearchTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "images", "golang", "gopher", "--size", "large"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search images failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Image One") {
		t.Errorf("output missing 'Image One', got:\n%s", output)
	}
	if !strings.Contains(output, "800\u00d7600") {
		t.Errorf("output missing dimensions, got:\n%s", output)
	}
}

func TestSearchNews(t *testing.T) {
	server := newsSearchMockServer(t, func(r *http.Request) {
		var req api.NewsSearchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q != "golang release" {
			t.Errorf("q = %q, want golang release", req.Q)
		}
		if req.TBS != "qdr:w" {
			t.Errorf("tbs = %q, want qdr:w", req.TBS)
		}
	})
	defer server.Close()
	setupSearchTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "news", "golang", "release", "--time", "week"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("search news failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "News One") {
		t.Errorf("output missing 'News One', got:\n%s", output)
	}
	if !strings.Contains(output, "Test News") {
		t.Errorf("output missing source, got:\n%s", output)
	}
}

func TestSearchNoQuery(t *testing.T) {
	server := webSearchMockServer(t, nil)
	defer server.Close()
	setupSearchTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestSearchCmdHasFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"limit", "L"},
		{"page", "p"},
		{"country", ""},
		{"language", ""},
	}
	for _, tt := range flags {
		f := searchCmd.PersistentFlags().Lookup(tt.name)
		if f == nil {
			t.Errorf("--%s flag not found", tt.name)
			continue
		}
		if tt.shorthand != "" && f.Shorthand != tt.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
		}
	}
}

func TestSearchNewsCmdHasTimeFlag(t *testing.T) {
	f := searchNewsCmd.Flags().Lookup("time")
	if f == nil {
		t.Fatal("--time flag not found on search news")
	}
	if f.Shorthand != "t" {
		t.Errorf("--time shorthand = %q, want t", f.Shorthand)
	}
}

func TestSearchImagesCmdHasSizeFlag(t *testing.T) {
	f := searchImagesCmd.Flags().Lookup("size")
	if f == nil {
		t.Fatal("--size flag not found on search images")
	}
	if f.Shorthand != "s" {
		t.Errorf("--size shorthand = %q, want s", f.Shorthand)
	}
}
