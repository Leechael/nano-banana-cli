package codex

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leechael/imagen/provider"
)

func TestMapSize(t *testing.T) {
	tests := []struct {
		size, aspect string
		want         string
		wantWarn     bool
	}{
		{"512", "", "1024x1024", true},
		{"1K", "", "1024x1024", false},
		{"1K", "16:9", "1536x1024", false},
		{"1K", "9:16", "1024x1536", false},
		{"2K", "", "1024x1024", true},
		{"2K", "16:9", "1536x1024", true},
		{"2K", "9:16", "1024x1536", true},
		{"4K", "", "1536x1024", true},
		{"", "", "1024x1024", false},
		{"", "16:9", "1536x1024", false},
		{"1024x1024", "", "1024x1024", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_%s", tt.size, tt.aspect)
		t.Run(name, func(t *testing.T) {
			got, warns := mapSize(tt.size, tt.aspect)
			if got != tt.want {
				t.Errorf("mapSize(%q, %q) = %q, want %q", tt.size, tt.aspect, got, tt.want)
			}
			if tt.wantWarn && len(warns) == 0 {
				t.Errorf("expected warnings, got none")
			}
			if !tt.wantWarn && len(warns) > 0 {
				t.Errorf("unexpected warnings: %v", warns)
			}
		})
	}
}

func TestResolveTier(t *testing.T) {
	m, ok := resolveTier("gpt-image-2-low")
	if !ok || m.Quality != "low" {
		t.Errorf("resolveTier(low) = %+v, %v", m, ok)
	}
	m, ok = resolveTier("gpt-image-2-high")
	if !ok || m.Quality != "high" {
		t.Errorf("resolveTier(high) = %+v, %v", m, ok)
	}
	m, ok = resolveTier("unknown")
	if ok || m.Quality != "medium" {
		t.Errorf("resolveTier(unknown) = %+v, %v", m, ok)
	}
}

func TestGenerateSuccess(t *testing.T) {
	imgData := []byte("fake-png-data")
	b64 := base64.StdEncoding.EncodeToString(imgData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("unexpected auth: %s", auth)
		}
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "text/event-stream") {
			t.Errorf("expected Accept to contain text/event-stream, got: %s", accept)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		// Write SSE events
		fmt.Fprintf(w, "event: response.created\n")
		fmt.Fprintf(w, "data: {\"type\":\"response.created\"}\n\n")
		fmt.Fprintf(w, "event: response.output_item.done\n")
		fmt.Fprintf(w, "data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"image_generation_call\",\"result\":\"%s\"}}\n\n", b64)
		fmt.Fprintf(w, "event: response.completed\n")
		fmt.Fprintf(w, "data: {\"type\":\"response.completed\"}\n\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := New("test-token")
	client.baseURL = server.URL

	res, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "codex-2",
		Prompt: "a cat",
		Size:   "1K",
		N:      1,
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if len(res.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(res.Images))
	}
	if string(res.Images[0].Data) != string(imgData) {
		t.Error("image data mismatch")
	}
	if res.Images[0].MIMEType != "image/png" {
		t.Errorf("mime = %q, want image/png", res.Images[0].MIMEType)
	}
	if res.Model != "codex-2" {
		t.Errorf("model = %q, want codex-2", res.Model)
	}
}

func TestGeneratePartialImage(t *testing.T) {
	imgData := []byte("fake-png-data")
	b64 := base64.StdEncoding.EncodeToString(imgData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"type\":\"response.image_generation_call.partial_image\",\"partial_image_b64\":\"%s\"}\n\n", b64)
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	client := New("test-token")
	client.baseURL = server.URL

	res, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "codex-2-medium",
		Prompt: "a cat",
		Size:   "1K",
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(res.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(res.Images))
	}
}

func TestGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"invalid token","type":"auth_error"}}`)
	}))
	defer server.Close()

	client := New("bad-token")
	client.baseURL = server.URL

	_, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "codex-2",
		Prompt: "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	ok := false
	if ae, isAE := err.(*APIError); isAE {
		apiErr = ae
		ok = true
	}
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != 401 {
		t.Errorf("code = %d, want 401", apiErr.Code)
	}
	if apiErr.Message != "invalid token" {
		t.Errorf("message = %q, want 'invalid token'", apiErr.Message)
	}
}

func TestExtractImageB64(t *testing.T) {
	c := New("test")

	// Test output_item.done event
	input := `event: response.output_item.done
data: {"type":"response.output_item.done","item":{"type":"image_generation_call","result":"b64data1"}}

`
	got, err := c.extractImageB64(strings.NewReader(input))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if got != "b64data1" {
		t.Errorf("got %q, want b64data1", got)
	}

	// Test partial_image event
	input2 := `data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"b64data2"}

`
	got2, err := c.extractImageB64(strings.NewReader(input2))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if got2 != "b64data2" {
		t.Errorf("got %q, want b64data2", got2)
	}

	// Test empty stream
	got3, err := c.extractImageB64(strings.NewReader(""))
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if got3 != "" {
		t.Errorf("got %q, want empty", got3)
	}
}

func TestResolveAlias(t *testing.T) {
	if got := ResolveAlias("codex-2"); got != "gpt-image-2-medium" {
		t.Errorf("ResolveAlias(codex-2) = %q, want gpt-image-2-medium", got)
	}
	if got := ResolveAlias("codex-2-low"); got != "gpt-image-2-low" {
		t.Errorf("ResolveAlias(codex-2-low) = %q, want gpt-image-2-low", got)
	}
	if got := ResolveAlias("unknown"); got != "unknown" {
		t.Errorf("ResolveAlias(unknown) = %q, want unknown", got)
	}
}

func TestDigString(t *testing.T) {
	m := map[string]any{
		"item": map[string]any{
			"result": "found",
		},
	}
	if got := digString(m, "item", "result"); got != "found" {
		t.Errorf("digString = %q, want found", got)
	}
	if got := digString(m, "missing"); got != "" {
		t.Errorf("digString missing = %q, want empty", got)
	}
}
