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

func scrapeMockServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"text":     "Example Domain content",
				"markdown": "# Example Domain\n\nContent here",
				"metadata": map[string]string{"title": "Example Domain"},
				"credits":  1,
			},
		})
	}))
}

func setupScrapeTest(t *testing.T, server *httptest.Server) {
	t.Helper()
	viper.Set("api_server_url", server.URL)
	viper.Set("access_token", "test-token")
	t.Cleanup(func() {
		viper.Set("api_server_url", "")
		viper.Set("access_token", "")
	})
}

func TestScrape(t *testing.T) {
	server := scrapeMockServer(t, func(r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/tools/web-scrape" {
			t.Errorf("path = %s, want /api/v1/tools/web-scrape", r.URL.Path)
		}
		var req api.ScrapeRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.URL != "https://example.com" {
			t.Errorf("url = %q, want https://example.com", req.URL)
		}
		if !req.IncludeMarkdown {
			t.Error("includeMarkdown = false, want true")
		}
	})
	defer server.Close()
	setupScrapeTest(t, server)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"scrape", "https://example.com"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scrape failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# Example Domain") {
		t.Errorf("output missing markdown content, got:\n%s", output)
	}
}


func TestScrapeJSON(t *testing.T) {
	server := scrapeMockServer(t, nil)
	defer server.Close()
	setupScrapeTest(t, server)
	viper.Set("output", "json")
	t.Cleanup(func() { viper.Set("output", "") })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"scrape", "https://example.com"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("scrape -o json failed: %v", err)
	}

	var result api.ScrapeResponse
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON output not parseable: %v\nOutput:\n%s", err, buf.String())
	}
	if result.Text != "Example Domain content" {
		t.Errorf("Text = %q, want Example Domain content", result.Text)
	}
}

func TestScrapeNoArgs(t *testing.T) {
	server := scrapeMockServer(t, nil)
	defer server.Close()
	setupScrapeTest(t, server)

	rootCmd.SetArgs([]string{"scrape"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing URL argument")
	}
}

