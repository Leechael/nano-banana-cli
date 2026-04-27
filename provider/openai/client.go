package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/leechael/imagen/provider"
)

const defaultBaseURL = "https://api.openai.com/v1"

var aliases = map[string]string{
	"gpt-image-2":          "gpt-image-2-2026-04-21",
	"gpt-image-1.5":        "gpt-image-1.5",
	"gpt-image-1":          "gpt-image-1",
	"gpt-image-1-mini":     "gpt-image-1-mini",
	"chatgpt-image-latest": "chatgpt-image-latest",
	"dall-e-3":             "dall-e-3",
	"dall-e-2":             "dall-e-2",
	"oai-2":                "gpt-image-2-2026-04-21",
	"oai-15":               "gpt-image-1.5",
}

type tokenRate struct {
	ImageInput  float64
	TextInput   float64
	ImageOutput float64
	TextOutput  float64
}

var tokenRates = map[string]tokenRate{
	"gpt-image-2-2026-04-21": {ImageInput: 8.0, TextInput: 5.0, ImageOutput: 30.0, TextOutput: 0},
	"gpt-image-1.5":          {ImageInput: 8.0, TextInput: 5.0, ImageOutput: 32.0, TextOutput: 10.0},
	"gpt-image-1-mini":       {ImageInput: 2.5, TextInput: 2.0, ImageOutput: 8.0, TextOutput: 0},
	"gpt-image-1":            {ImageInput: 5.0, TextInput: 5.0, ImageOutput: 40.0, TextOutput: 0},
}

var imageRates = map[string]float64{
	"dall-e-2": 0.016,
	"dall-e-3": 0.040,
}

// ResolveAlias maps user-facing model names to canonical API IDs.
func ResolveAlias(input string) string {
	if v, ok := aliases[input]; ok {
		return v
	}
	return input
}

// APIError represents an error returned by the OpenAI API.
type APIError struct {
	Code    int
	Message string
	Status  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("openai api error (status %d): %s", e.Code, e.Message)
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func parseAPIErrorMessage(body []byte) string {
	var resp apiErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return string(body)
	}
	if resp.Error.Message != "" {
		return resp.Error.Message
	}
	if resp.Error.Code != "" {
		return resp.Error.Code
	}
	return string(body)
}

// Client implements provider.Provider for OpenAI image generation.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New returns a new OpenAI provider.
func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) Name() string { return "openai" }

func isGPTImageModel(m string) bool {
	switch m {
	case "gpt-image-2-2026-04-21", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini", "chatgpt-image-latest":
		return true
	}
	return strings.HasPrefix(m, "gpt-image-")
}

func maxReferences(m string) int {
	if isGPTImageModel(m) {
		return 16
	}
	if m == "dall-e-2" {
		return 1
	}
	return 0
}

func (c *Client) Capabilities(model string) provider.Capability {
	return provider.Capability{
		SupportsAspectRatio: false,
		SupportsSeed:        false,
		SupportsPerson:      false,
		SupportsThinking:    false,
		SupportsReferences:  isGPTImageModel(model) || model == "dall-e-2",
		SupportsBatch:       model != "dall-e-3",
		Sizes:               []string{"auto", "1024x1024", "1536x1024", "1024x1536"},
		MaxReferences:       maxReferences(model),
	}
}

// Generate creates or edits images via OpenAI.
func (c *Client) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.GenerateResult, error) {
	model := ResolveAlias(req.Model)
	if model == "" {
		model = "gpt-image-2-2026-04-21"
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	size, warnings := mapSize(req.Size, req.AspectRatio)

	if len(req.References) > 0 {
		return c.edit(ctx, model, req.Prompt, req.References, n, size, req.Background, warnings)
	}
	return c.generateWithBackground(ctx, model, req.Prompt, n, size, req.Background, warnings)
}

// mapSize maps CLI abstract size + aspect to OpenAI concrete size.
func mapSize(size, aspect string) (string, []string) {
	var warnings []string
	isLandscape := false
	isPortrait := false
	switch aspect {
	case "16:9", "4:3", "3:2", "21:9", "2:1", "20:9", "19.5:9":
		isLandscape = true
	case "9:16", "3:4", "2:3", "1:4", "1:8", "1:2":
		isPortrait = true
	}

	switch size {
	case "512":
		warnings = append(warnings, "size 512 not supported, using 1024x1024")
		return "1024x1024", warnings
	case "1K":
		if isLandscape {
			return "1536x1024", warnings
		}
		if isPortrait {
			return "1024x1536", warnings
		}
		return "1024x1024", warnings
	case "2K":
		if isLandscape {
			warnings = append(warnings, "size 2K not supported, using 1536x1024")
			return "1536x1024", warnings
		}
		if isPortrait {
			warnings = append(warnings, "size 2K not supported, using 1024x1536")
			return "1024x1536", warnings
		}
		warnings = append(warnings, "size 2K not supported, using 1024x1024")
		return "1024x1024", warnings
	case "4K":
		warnings = append(warnings, "size 4K not supported, using max 1536x1024")
		return "1536x1024", warnings
	case "":
		if isLandscape {
			return "1536x1024", warnings
		}
		if isPortrait {
			return "1024x1536", warnings
		}
		return "auto", warnings
	default:
		return size, warnings
	}
}

type generationRequest struct {
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	N                 int    `json:"n,omitempty"`
	Size              string `json:"size,omitempty"`
	Quality           string `json:"quality,omitempty"`
	OutputFormat      string `json:"output_format,omitempty"`
	Background        string `json:"background,omitempty"`
	Moderation        string `json:"moderation,omitempty"`
	OutputCompression int    `json:"output_compression,omitempty"`
	ResponseFormat    string `json:"response_format,omitempty"`
}

type generationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
		URL           string `json:"url,omitempty"`
	} `json:"data"`
	Usage *usageData `json:"usage,omitempty"`
}

type usageData struct {
	InputTokens         int32         `json:"input_tokens"`
	OutputTokens        int32         `json:"output_tokens"`
	TotalTokens         int32         `json:"total_tokens"`
	InputTokensDetails  *tokenDetails `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *tokenDetails `json:"output_tokens_details,omitempty"`
}

type tokenDetails struct {
	ImageTokens int32 `json:"image_tokens"`
	TextTokens  int32 `json:"text_tokens"`
}

func (c *Client) generate(ctx context.Context, model, prompt string, n int, size string, warnings []string) (*provider.GenerateResult, error) {
	return c.generateWithBackground(ctx, model, prompt, n, size, "", warnings)
}

func (c *Client) generateWithBackground(ctx context.Context, model, prompt string, n int, size, background string, warnings []string) (*provider.GenerateResult, error) {
	apiReq := generationRequest{
		Model:  model,
		Prompt: prompt,
		N:      n,
		Size:   size,
	}

	if background != "" {
		apiReq.Background = background
		if background == "transparent" {
			apiReq.OutputFormat = "png"
		}
	}

	if !isGPTImageModel(model) {
		apiReq.ResponseFormat = "b64_json"
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Code: resp.StatusCode, Message: parseAPIErrorMessage(respBody), Status: http.StatusText(resp.StatusCode)}
	}

	var apiResp generationResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return c.buildResult(model, apiResp, warnings)
}

type editImageRef struct {
	ImageURL string `json:"image_url"`
}

type editRequest struct {
	Model         string         `json:"model"`
	Prompt        string         `json:"prompt"`
	Images        []editImageRef `json:"images"`
	N             int            `json:"n,omitempty"`
	Size          string         `json:"size,omitempty"`
	Quality       string         `json:"quality,omitempty"`
	OutputFormat  string         `json:"output_format,omitempty"`
	Background    string         `json:"background,omitempty"`
	InputFidelity string         `json:"input_fidelity,omitempty"`
}

func (c *Client) edit(ctx context.Context, model, prompt string, refs []provider.Reference, n int, size, background string, warnings []string) (*provider.GenerateResult, error) {
	maxRefs := maxReferences(model)
	if maxRefs > 0 && len(refs) > maxRefs {
		return nil, fmt.Errorf("at most %d reference images are allowed for %s, got %d", maxRefs, model, len(refs))
	}

	images := make([]editImageRef, 0, len(refs))
	for _, ref := range refs {
		encoded := base64.StdEncoding.EncodeToString(ref.Data)
		images = append(images, editImageRef{ImageURL: fmt.Sprintf("data:%s;base64,%s", ref.MIMEType, encoded)})
	}

	apiReq := editRequest{
		Model:  model,
		Prompt: prompt,
		Images: images,
		N:      n,
		Size:   size,
	}

	if background != "" {
		apiReq.Background = background
		if background == "transparent" {
			apiReq.OutputFormat = "png"
		}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/images/edits", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Code: resp.StatusCode, Message: parseAPIErrorMessage(respBody), Status: http.StatusText(resp.StatusCode)}
	}

	var apiResp generationResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return c.buildResult(model, apiResp, warnings)
}

func (c *Client) buildResult(model string, apiResp generationResponse, warnings []string) (*provider.GenerateResult, error) {
	var images []provider.Image
	for _, d := range apiResp.Data {
		var data []byte
		var mimeType string
		if d.B64JSON != "" {
			var err error
			data, err = base64.StdEncoding.DecodeString(d.B64JSON)
			if err != nil {
				return nil, fmt.Errorf("decoding base64 image: %w", err)
			}
			mimeType = detectMIME(data)
		} else if d.URL != "" {
			// fallback for dall-e url mode — fetch the image
			imgData, mt, err := c.fetchImage(context.Background(), d.URL)
			if err != nil {
				return nil, fmt.Errorf("fetching image url: %w", err)
			}
			data = imgData
			mimeType = mt
		}
		if len(data) > 0 {
			images = append(images, provider.Image{Data: data, MIMEType: mimeType})
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images generated")
	}

	cost := calculateCost(model, apiResp.Usage, len(images))
	var promptTokens, outputTokens int32
	if apiResp.Usage != nil {
		promptTokens = apiResp.Usage.InputTokens
		outputTokens = apiResp.Usage.OutputTokens
	}

	return &provider.GenerateResult{
		Images:       images,
		Model:        model,
		Cost:         cost,
		PromptTokens: promptTokens,
		OutputTokens: outputTokens,
		Warnings:     warnings,
	}, nil
}

func (c *Client) fetchImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching image: status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mt := resp.Header.Get("Content-Type")
	if mt == "" {
		mt = detectMIME(data)
	}
	return data, mt, nil
}

func detectMIME(data []byte) string {
	if len(data) > 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	if len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return "image/png"
}

func calculateCost(model string, usage *usageData, imageCount int) float64 {
	if isGPTImageModel(model) {
		if usage == nil {
			return 0
		}
		rate, ok := tokenRates[model]
		if !ok {
			return 0
		}
		var cost float64
		if usage.InputTokensDetails != nil {
			cost += float64(usage.InputTokensDetails.ImageTokens) / 1e6 * rate.ImageInput
			cost += float64(usage.InputTokensDetails.TextTokens) / 1e6 * rate.TextInput
		} else {
			cost += float64(usage.InputTokens) / 1e6 * rate.ImageInput
		}
		if usage.OutputTokensDetails != nil {
			cost += float64(usage.OutputTokensDetails.ImageTokens) / 1e6 * rate.ImageOutput
			cost += float64(usage.OutputTokensDetails.TextTokens) / 1e6 * rate.TextOutput
		} else {
			cost += float64(usage.OutputTokens) / 1e6 * rate.ImageOutput
		}
		return cost
	}

	if rate, ok := imageRates[model]; ok {
		return rate * float64(imageCount)
	}
	return 0
}

// init registers the openai provider.
func init() {
	provider.Register("openai", func(apiKey string) provider.Provider {
		return New(apiKey)
	})
}

var _ provider.Provider = (*Client)(nil)
