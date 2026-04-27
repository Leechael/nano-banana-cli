package xai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leechael/imagen/provider"
)

func TestNewClient(t *testing.T) {
	c := New("test-key")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.http == nil {
		t.Error("http client should not be nil")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{Code: 400, Message: "bad request", Status: "Bad Request"}
	want := "xai api error (status 400): bad request"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestName(t *testing.T) {
	c := New("key")
	if c.Name() != "xai" {
		t.Errorf("Name() = %q, want xai", c.Name())
	}
}

func TestCapabilities(t *testing.T) {
	c := New("key")
	cap := c.Capabilities("grok-imagine-image")
	if !cap.SupportsAspectRatio {
		t.Error("should support aspect ratio")
	}
	if cap.SupportsSeed {
		t.Error("should not support seed")
	}
	if cap.SupportsPerson {
		t.Error("should not support person")
	}
	if cap.SupportsThinking {
		t.Error("should not support thinking")
	}
	if !cap.SupportsReferences {
		t.Error("should support references")
	}
	if !cap.SupportsBatch {
		t.Error("should support batch")
	}
	if cap.MaxReferences != 5 {
		t.Errorf("MaxReferences = %d, want 5", cap.MaxReferences)
	}
}

func TestMapSize(t *testing.T) {
	tests := []struct {
		input    string
		wantRes  string
		wantWarn bool
	}{
		{"", "", false},
		{"512", "1k", true},
		{"1K", "1k", false},
		{"2K", "2k", false},
		{"4K", "2k", true},
	}
	for _, tc := range tests {
		res, warns := mapSize(tc.input)
		if res != tc.wantRes {
			t.Errorf("mapSize(%q) res = %q, want %q", tc.input, res, tc.wantRes)
		}
		if (len(warns) > 0) != tc.wantWarn {
			t.Errorf("mapSize(%q) warnings = %v, wantWarn = %v", tc.input, warns, tc.wantWarn)
		}
	}
}

func TestGenerate_Success(t *testing.T) {
	imgData := base64.StdEncoding.EncodeToString([]byte("fake-image"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Errorf("path = %q, want /images/generations", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if !containsBearer(r.Header.Get("Authorization")) {
			t.Errorf("missing Bearer in Authorization: %q", r.Header.Get("Authorization"))
		}

		var req generateAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "grok-imagine-image" {
			t.Errorf("model = %q", req.Model)
		}
		if req.Prompt != "a cat" {
			t.Errorf("prompt = %q", req.Prompt)
		}
		if req.N != 2 {
			t.Errorf("N = %d, want 2", req.N)
		}
		if req.ResponseFormat != "b64_json" {
			t.Errorf("response_format = %q", req.ResponseFormat)
		}
		if req.AspectRatio != "1:1" {
			t.Errorf("aspect_ratio = %q", req.AspectRatio)
		}
		if req.Resolution != "2k" {
			t.Errorf("resolution = %q", req.Resolution)
		}

		resp := generateAPIResponse{
			Data:  []generateAPIImageData{{B64JSON: imgData}, {B64JSON: imgData}},
			Model: "grok-imagine-image",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	result, err := c.Generate(context.Background(), provider.GenerateRequest{
		Prompt:      "a cat",
		Model:       "grok-imagine-image",
		N:           2,
		AspectRatio: "1:1",
		Size:        "2K",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Model != "grok-imagine-image" {
		t.Errorf("model = %q", result.Model)
	}
	if len(result.Images) != 2 {
		t.Errorf("images count = %d, want 2", len(result.Images))
	}
	if string(result.Images[0].Data) != "fake-image" {
		t.Errorf("image data = %q", result.Images[0].Data)
	}
	if result.Images[0].MIMEType != "image/jpeg" {
		t.Errorf("mime = %q", result.Images[0].MIMEType)
	}
	if result.Cost < 0.13 || result.Cost > 0.15 {
		t.Errorf("cost = %f, want ~0.14", result.Cost)
	}
}

func TestGenerate_WithReferences_CallsEdits(t *testing.T) {
	imgData := base64.StdEncoding.EncodeToString([]byte("edited"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/edits" {
			t.Errorf("path = %q, want /images/edits", r.URL.Path)
		}
		var req editAPIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if len(req.Image) != 1 {
			t.Errorf("image count = %d, want 1", len(req.Image))
		}
		resp := generateAPIResponse{
			Data: []generateAPIImageData{{B64JSON: imgData}},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	result, err := c.Generate(context.Background(), provider.GenerateRequest{
		Prompt: "edit this",
		Model:  "grok-imagine-image",
		References: []provider.Reference{
			{Data: []byte("imgdata"), MIMEType: "image/png"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 {
		t.Errorf("images count = %d, want 1", len(result.Images))
	}
}

func TestGenerate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid prompt"}`))
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	_, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != 400 {
		t.Errorf("code = %d, want 400", apiErr.Code)
	}
}

func TestGenerate_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	_, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "parsing response") {
		t.Errorf("error = %q, want 'parsing response'", err.Error())
	}
}

func TestGenerate_InvalidBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateAPIResponse{
			Data: []generateAPIImageData{{B64JSON: "!!!invalid!!!"}},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	_, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "decoding base64") {
		t.Errorf("error = %q, want 'decoding base64'", err.Error())
	}
}

func TestGenerate_EmptyResponseData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateAPIResponse{Data: []generateAPIImageData{}}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	result, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 0 {
		t.Errorf("images count = %d, want 0", len(result.Images))
	}
	if result.Cost != 0.0 {
		t.Errorf("cost = %f, want 0", result.Cost)
	}
}

func TestGenerate_TooManyReferences(t *testing.T) {
	c := New("test-key")
	refs := make([]provider.Reference, 6)
	for i := range refs {
		refs[i] = provider.Reference{Data: []byte("img"), MIMEType: "image/png"}
	}
	_, err := c.Generate(context.Background(), provider.GenerateRequest{
		Prompt:     "edit",
		References: refs,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "at most 5 reference images") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestGenerate_SizeWarnings(t *testing.T) {
	imgData := base64.StdEncoding.EncodeToString([]byte("img"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateAPIRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := generateAPIResponse{Data: []generateAPIImageData{{B64JSON: imgData}}}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	result, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello", Size: "512"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for size 512")
	}

	result2, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello", Size: "4K"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result2.Warnings) == 0 {
		t.Error("expected warning for size 4K")
	}
}

func TestGenerate_ContextCancelled(t *testing.T) {
	c := New("test-key")
	c.baseURL = "http://[::1]:0"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Generate(ctx, provider.GenerateRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerate_HTTPClientDoError(t *testing.T) {
	c := New("test-key")
	c.baseURL = "http://localhost:1"

	_, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "doing request") {
		t.Errorf("error = %q, want 'doing request'", err.Error())
	}
}

func TestAPIError_Variations(t *testing.T) {
	cases := []struct {
		code   int
		msg    string
		status string
		want   string
	}{
		{400, "bad", "Bad Request", "xai api error (status 400): bad"},
		{401, "unauthorized", "Unauthorized", "xai api error (status 401): unauthorized"},
		{500, "server error", "Internal Server Error", "xai api error (status 500): server error"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.code), func(t *testing.T) {
			err := &APIError{Code: tc.code, Message: tc.msg, Status: tc.status}
			if err.Error() != tc.want {
				t.Errorf("Error() = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCostRates(t *testing.T) {
	rate, ok := costRates["grok-imagine-image"]
	if !ok {
		t.Fatal("expected grok-imagine-image in costRates")
	}
	if rate < 0.069 || rate > 0.071 {
		t.Errorf("rate = %f, want ~0.07", rate)
	}
}

func TestGenerate_ResponseModelFallback(t *testing.T) {
	imgData := base64.StdEncoding.EncodeToString([]byte("img"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateAPIResponse{
			Data:  []generateAPIImageData{{B64JSON: imgData}},
			Model: "",
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New("test-key")
	c.baseURL = server.URL

	result, err := c.Generate(context.Background(), provider.GenerateRequest{Prompt: "hello", Model: "custom-model"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "custom-model" {
		t.Errorf("model = %q, want custom-model", result.Model)
	}
}

func containsBearer(s string) bool {
	return len(s) > 7 && s[:7] == "Bearer "
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > 0 && containsSubstr(s, sub)))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
