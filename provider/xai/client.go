package xai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/leechael/imagen/provider"
)

const defaultBaseURL = "https://api.x.ai/v1"
const defaultModel = "grok-imagine-image"
const maxBatchSize = 10

var costRates = map[string]float64{
	"grok-imagine-image": 0.07,
}

type APIError struct {
	Code    int
	Message string
	Status  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("xai api error (status %d): %s", e.Code, e.Message)
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func parseAPIErrorMessage(body []byte) string {
	var resp apiErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return string(body)
	}
	if resp.Error != "" {
		return resp.Error
	}
	if resp.Message != "" {
		return resp.Message
	}
	if resp.Code != "" {
		return resp.Code
	}
	return string(body)
}

type generateAPIRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	AspectRatio    string `json:"aspect_ratio,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format"`
}

type editAPIRequest struct {
	Model          string   `json:"model"`
	Prompt         string   `json:"prompt"`
	Image          []string `json:"image"`
	N              int      `json:"n,omitempty"`
	AspectRatio    string   `json:"aspect_ratio,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	ResponseFormat string   `json:"response_format"`
}

type generateAPIResponse struct {
	Data  []generateAPIImageData `json:"data"`
	Model string                 `json:"model"`
}

type generateAPIImageData struct {
	B64JSON string `json:"b64_json"`
}

// Client implements provider.Provider for xAI (Grok).
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New returns a new xAI provider.
func New(apiKey string) *Client {
	baseURL := defaultBaseURL
	if v := os.Getenv("XAI_BASE_URL"); v != "" {
		baseURL = strings.TrimSuffix(v, "/")
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) Name() string { return "xai" }

func (c *Client) Capabilities(_ string) provider.Capability {
	return provider.Capability{
		SupportsAspectRatio: true,
		SupportsSeed:        false,
		SupportsPerson:      false,
		SupportsThinking:    false,
		SupportsQuality:     true,
		SupportsReferences:  true,
		SupportsBatch:       true,
		Sizes:               []string{"1K", "2K"},
		MaxReferences:       5,
	}
}

func (c *Client) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.GenerateResult, error) {
	model := req.Model
	if model == "" {
		model = defaultModel
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	resolution, warnings := mapSize(req.Size)

	if len(req.References) == 0 {
		return c.generate(ctx, model, req.Prompt, n, req.AspectRatio, resolution, req.Quality, warnings)
	}
	return c.edit(ctx, model, req.Prompt, req.References, n, req.AspectRatio, resolution, req.Quality, warnings)
}

func mapSize(size string) (resolution string, warnings []string) {
	switch strings.ToUpper(size) {
	case "":
		return "", nil
	case "512":
		return "1k", []string{"size 512 not supported, using 1k"}
	case "1K":
		return "1k", nil
	case "2K":
		return "2k", nil
	case "4K":
		return "2k", []string{"size 4K not supported, using 2k"}
	default:
		return strings.ToLower(size), nil
	}
}

func (c *Client) generate(ctx context.Context, model, prompt string, n int, aspectRatio, resolution, quality string, warnings []string) (*provider.GenerateResult, error) {
	result := &provider.GenerateResult{Model: model, Warnings: warnings}
	for remaining := n; remaining > 0; {
		batch := min(remaining, maxBatchSize)
		res, err := c.callGenerate(ctx, model, prompt, batch, aspectRatio, resolution, quality)
		if err != nil {
			return nil, err
		}
		result.Images = append(result.Images, res.Images...)
		result.Cost += res.Cost
		remaining -= batch
	}
	return result, nil
}

func (c *Client) callGenerate(ctx context.Context, model, prompt string, n int, aspectRatio, resolution, quality string) (*provider.GenerateResult, error) {
	apiReq := generateAPIRequest{
		Model:          model,
		Prompt:         prompt,
		N:              n,
		ResponseFormat: "b64_json",
	}
	if aspectRatio != "" {
		apiReq.AspectRatio = aspectRatio
	}
	if resolution != "" {
		apiReq.Resolution = resolution
	}
	if quality != "" {
		apiReq.Quality = quality
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

	var apiResp generateAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return c.buildResult(model, apiResp, nil)
}

func (c *Client) edit(ctx context.Context, model, prompt string, refs []provider.Reference, n int, aspectRatio, resolution, quality string, warnings []string) (*provider.GenerateResult, error) {
	if len(refs) > 5 {
		return nil, fmt.Errorf("at most 5 reference images are allowed, got %d", len(refs))
	}

	images := make([]string, 0, len(refs))
	for _, ref := range refs {
		encoded := base64.StdEncoding.EncodeToString(ref.Data)
		images = append(images, fmt.Sprintf("data:%s;base64,%s", ref.MIMEType, encoded))
	}

	result := &provider.GenerateResult{Model: model, Warnings: warnings}
	for remaining := n; remaining > 0; {
		batch := min(remaining, maxBatchSize)
		res, err := c.callEdit(ctx, model, prompt, images, batch, aspectRatio, resolution, quality)
		if err != nil {
			return nil, err
		}
		result.Images = append(result.Images, res.Images...)
		result.Cost += res.Cost
		remaining -= batch
	}
	return result, nil
}

func (c *Client) callEdit(ctx context.Context, model, prompt string, images []string, n int, aspectRatio, resolution, quality string) (*provider.GenerateResult, error) {
	apiReq := editAPIRequest{
		Model:          model,
		Prompt:         prompt,
		Image:          images,
		N:              n,
		ResponseFormat: "b64_json",
	}
	if aspectRatio != "" {
		apiReq.AspectRatio = aspectRatio
	}
	if resolution != "" {
		apiReq.Resolution = resolution
	}
	if quality != "" {
		apiReq.Quality = quality
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

	var apiResp generateAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return c.buildResult(model, apiResp, nil)
}

func (c *Client) buildResult(model string, apiResp generateAPIResponse, warnings []string) (*provider.GenerateResult, error) {
	resultModel := apiResp.Model
	if resultModel == "" {
		resultModel = model
	}

	var images []provider.Image
	for _, img := range apiResp.Data {
		data, err := base64.StdEncoding.DecodeString(img.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decoding base64 image: %w", err)
		}
		images = append(images, provider.Image{
			Data:     data,
			MIMEType: "image/jpeg",
		})
	}

	var cost float64
	if rate, ok := costRates[resultModel]; ok {
		cost = float64(len(images)) * rate
	}

	return &provider.GenerateResult{
		Images:   images,
		Model:    resultModel,
		Cost:     cost,
		Warnings: warnings,
	}, nil
}

// init registers the xai provider.
func init() {
	provider.Register("xai", func(apiKey string) provider.Provider {
		return New(apiKey)
	})
}

var _ provider.Provider = (*Client)(nil)
