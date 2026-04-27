package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		{"", "", "auto", false},
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

func TestCapabilities(t *testing.T) {
	c := New("test-key")

	gpt2 := c.Capabilities("gpt-image-2-2026-04-21")
	if !gpt2.SupportsReferences {
		t.Error("gpt-image-2 should support references")
	}
	if gpt2.MaxReferences != 16 {
		t.Errorf("gpt-image-2 max refs = %d, want 16", gpt2.MaxReferences)
	}
	if gpt2.SupportsBatch != true {
		t.Error("gpt-image-2 should support batch")
	}

	d3 := c.Capabilities("dall-e-3")
	if d3.SupportsBatch {
		t.Error("dall-e-3 should not support batch")
	}
	if d3.SupportsReferences {
		t.Error("dall-e-3 should not support references")
	}

	d2 := c.Capabilities("dall-e-2")
	if !d2.SupportsReferences {
		t.Error("dall-e-2 should support references")
	}
	if d2.MaxReferences != 1 {
		t.Errorf("dall-e-2 max refs = %d, want 1", d2.MaxReferences)
	}
}

func TestGenerateSuccess(t *testing.T) {
	imgData := []byte("fake-png-data")
	b64 := base64.StdEncoding.EncodeToString(imgData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", auth)
		}

		var req generationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-image-2-2026-04-21" {
			t.Errorf("model = %s, want gpt-image-2-2026-04-21", req.Model)
		}

		resp := generationResponse{
			Created: 1234567890,
			Data: []struct {
				B64JSON       string `json:"b64_json"`
				RevisedPrompt string `json:"revised_prompt,omitempty"`
				URL           string `json:"url,omitempty"`
			}{{
				B64JSON: b64,
			}},
			Usage: &usageData{
				InputTokens:  50,
				OutputTokens: 100,
				TotalTokens:  150,
				InputTokensDetails: &tokenDetails{
					ImageTokens: 40,
					TextTokens:  10,
				},
				OutputTokensDetails: &tokenDetails{
					ImageTokens: 100,
					TextTokens:  0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New("test-key")
	client.baseURL = server.URL

	res, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "gpt-image-2",
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
	if res.PromptTokens != 50 {
		t.Errorf("promptTokens = %d, want 50", res.PromptTokens)
	}
	if res.OutputTokens != 100 {
		t.Errorf("outputTokens = %d, want 100", res.OutputTokens)
	}
	// cost = (40/1e6)*8 + (10/1e6)*5 + (100/1e6)*30 = 0.00032 + 0.00005 + 0.003 = 0.00337
	expectedCost := 0.00337
	if res.Cost != expectedCost {
		t.Errorf("cost = %.5f, want %.5f", res.Cost, expectedCost)
	}
}

func TestEditSuccess(t *testing.T) {
	imgData := []byte("fake-png-data")
	b64 := base64.StdEncoding.EncodeToString(imgData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/edits" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req editRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Images) != 2 {
			t.Errorf("images = %d, want 2", len(req.Images))
		}
		if req.Images[0].ImageURL == "" {
			t.Error("expected image_url in first image ref")
		}

		resp := generationResponse{
			Data: []struct {
				B64JSON       string `json:"b64_json"`
				RevisedPrompt string `json:"revised_prompt,omitempty"`
				URL           string `json:"url,omitempty"`
			}{{
				B64JSON: b64,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New("test-key")
	client.baseURL = server.URL

	res, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "gpt-image-2",
		Prompt: "edit this",
		Size:   "1K",
		N:      1,
		References: []provider.Reference{
			{Data: []byte("ref1"), MIMEType: "image/png"},
			{Data: []byte("ref2"), MIMEType: "image/png"},
		},
	})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if len(res.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(res.Images))
	}
}

func TestGenerateAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "invalid model",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	client := New("test-key")
	client.baseURL = server.URL

	_, err := client.Generate(context.Background(), provider.GenerateRequest{
		Model:  "bad-model",
		Prompt: "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != 400 {
		t.Errorf("code = %d, want 400", apiErr.Code)
	}
	if apiErr.Message != "invalid model" {
		t.Errorf("message = %q, want 'invalid model'", apiErr.Message)
	}
}

func TestDetectMIME(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{[]byte("\x89PNG\r\n\x1a\n"), "image/png"},
		{[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{[]byte("RIFFxxxxWEBPxxxx"), "image/webp"},
		{[]byte("unknown"), "image/png"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := detectMIME(tt.data)
			if got != tt.want {
				t.Errorf("detectMIME = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveAlias(t *testing.T) {
	if got := ResolveAlias("oai-2"); got != "gpt-image-2-2026-04-21" {
		t.Errorf("ResolveAlias(oai-2) = %q, want gpt-image-2-2026-04-21", got)
	}
	if got := ResolveAlias("gpt-image-2"); got != "gpt-image-2-2026-04-21" {
		t.Errorf("ResolveAlias(gpt-image-2) = %q, want gpt-image-2-2026-04-21", got)
	}
	if got := ResolveAlias("unknown"); got != "unknown" {
		t.Errorf("ResolveAlias(unknown) = %q, want unknown", got)
	}
}
