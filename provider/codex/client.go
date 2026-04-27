package codex

import (
	"bufio"
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

const (
	defaultBaseURL    = "https://chatgpt.com/backend-api/codex"
	defaultChatModel  = "gpt-5.4"
	defaultAPIModel   = "gpt-image-2"
	defaultQuality    = "medium"
	defaultBackground = "opaque"
)

var aliases = map[string]string{
	"codex-2":        "gpt-image-2-medium",
	"codex-2-low":    "gpt-image-2-low",
	"codex-2-medium": "gpt-image-2-medium",
	"codex-2-high":   "gpt-image-2-high",
}

type tierMeta struct {
	Quality string
	Display string
}

var tierMap = map[string]tierMeta{
	"gpt-image-2-low":    {Quality: "low", Display: "GPT Image 2 (Low)"},
	"gpt-image-2-medium": {Quality: "medium", Display: "GPT Image 2 (Medium)"},
	"gpt-image-2-high":   {Quality: "high", Display: "GPT Image 2 (High)"},
}

// ResolveAlias maps user-facing names to canonical tier IDs.
func ResolveAlias(input string) string {
	if v, ok := aliases[input]; ok {
		return v
	}
	return input
}

func resolveTier(modelID string) (tierMeta, bool) {
	if m, ok := tierMap[modelID]; ok {
		return m, true
	}
	// fallback: if suffix matches a known quality
	for _, m := range tierMap {
		if strings.HasSuffix(modelID, "-"+m.Quality) {
			return m, true
		}
	}
	return tierMap["gpt-image-2-medium"], false
}

// APIError represents an error returned by the Codex API.
type APIError struct {
	Code    int
	Message string
	Status  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("codex api error (status %d): %s", e.Code, e.Message)
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

// Client implements provider.Provider for OpenAI Codex (ChatGPT OAuth).
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New returns a new Codex provider.
func New(apiKey string) *Client {
	baseURL := defaultBaseURL
	if v := os.Getenv("CODEX_BASE_URL"); v != "" {
		baseURL = strings.TrimSuffix(v, "/")
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *Client) Name() string { return "codex" }

func (c *Client) Capabilities(_ string) provider.Capability {
	return provider.Capability{
		SupportsAspectRatio: false,
		SupportsSeed:        false,
		SupportsPerson:      false,
		SupportsThinking:    false,
		SupportsReferences:  false,
		SupportsBatch:       false,
		SupportsQuality:     true,
		Sizes:               []string{"1024x1024", "1536x1024", "1024x1536"},
		MaxReferences:       0,
	}
}

// Generate creates an image via Codex Responses API image_generation tool.
func (c *Client) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.GenerateResult, error) {
	rawModel := req.Model
	if rawModel == "" {
		rawModel = "codex-2"
	}
	modelID := ResolveAlias(rawModel)
	if modelID == "" {
		modelID = "gpt-image-2-medium"
	}
	meta, _ := resolveTier(modelID)

	// Allow --quality flag to override the tier default.
	quality := meta.Quality
	if req.Quality != "" {
		quality = req.Quality
	}

	size, warnings := mapSize(req.Size, req.AspectRatio)
	background := defaultBackground
	if req.Background != "" {
		background = req.Background
	}

	apiReq := responsesRequest{
		Model:        defaultChatModel,
		Store:        false,
		Stream:       true,
		Instructions: "You are an assistant that must fulfill image generation requests by using the image_generation tool when provided.",
		Input: []inputItem{
			{
				Type: "message",
				Role: "user",
				Content: []contentItem{
					{Type: "input_text", Text: req.Prompt},
				},
			},
		},
		Tools: []toolItem{
			{
				Type:          "image_generation",
				Model:         defaultAPIModel,
				Size:          size,
				Quality:       quality,
				OutputFormat:  "png",
				Background:    background,
				PartialImages: 1,
			},
		},
		ToolChoice: toolChoice{
			Type: "allowed_tools",
			Mode: "required",
			Tools: []toolRef{
				{Type: "image_generation"},
			},
		},
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
	httpReq.Header.Set("Origin", "https://chatgpt.com")
	httpReq.Header.Set("Referer", "https://chatgpt.com/")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{Code: resp.StatusCode, Message: parseAPIErrorMessage(bodyBytes), Status: http.StatusText(resp.StatusCode)}
	}

	b64, err := c.extractImageB64(resp.Body)
	if err != nil {
		return nil, err
	}

	if b64 == "" {
		return nil, fmt.Errorf("codex response contained no image_generation_call result")
	}

	imgData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 image: %w", err)
	}

	return &provider.GenerateResult{
		Images: []provider.Image{
			{Data: imgData, MIMEType: "image/png"},
		},
		Model:    rawModel,
		Warnings: warnings,
	}, nil
}

// mapSize maps CLI abstract size + aspect to Codex size.
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
		return "1024x1024", warnings
	default:
		return size, warnings
	}
}

// extractImageB64 reads an SSE stream and extracts the base64 image.
func (c *Client) extractImageB64(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	// Increase buffer beyond the default 64 KiB — SSE data lines can carry
	// large base64-encoded image payloads.
	const maxCapacity = 8 * 1024 * 1024 // 8 MiB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	var currentEvent string
	var currentData strings.Builder

	var b64 string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if currentData.Len() > 0 {
				b64 = c.tryExtractB64([]byte(currentData.String()), b64)
				currentEvent = ""
				currentData.Reset()
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			if currentData.Len() > 0 {
				currentData.WriteString("\n")
			}
			currentData.WriteString(strings.TrimPrefix(line, "data: "))
		}
		_ = currentEvent // event name can also be inside JSON; we check both
	}
	if currentData.Len() > 0 {
		b64 = c.tryExtractB64([]byte(currentData.String()), b64)
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading sse stream: %w", err)
	}
	return b64, nil
}

func (c *Client) tryExtractB64(data []byte, currentBest string) string {
	// Try event-type extraction first.
	var evt struct {
		Type string `json:"type"`
		Item *struct {
			Type   string `json:"type"`
			Result string `json:"result"`
		} `json:"item"`
		PartialImageB64 string `json:"partial_image_b64"`
	}
	if err := json.Unmarshal(data, &evt); err != nil {
		return currentBest
	}

	// Event names from Python SDK (event.type or data.type).
	switch evt.Type {
	case "response.output_item.done":
		if evt.Item != nil && evt.Item.Type == "image_generation_call" && evt.Item.Result != "" {
			return evt.Item.Result
		}
	case "response.image_generation_call.partial_image":
		if evt.PartialImageB64 != "" {
			return evt.PartialImageB64
		}
	}

	// Fallback: scan for any image_generation_call result in the data.
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return currentBest
	}
	if result := digString(generic, "item", "result"); result != "" {
		return result
	}
	if result := digString(generic, "result"); result != "" {
		return result
	}
	if partial := digString(generic, "partial_image_b64"); partial != "" {
		return partial
	}

	return currentBest
}

func digString(m map[string]any, keys ...string) string {
	current := m
	for i, k := range keys {
		v, ok := current[k]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		}
		next, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

// --- Request / response types ---

type responsesRequest struct {
	Model        string      `json:"model"`
	Store        bool        `json:"store"`
	Stream       bool        `json:"stream"`
	Instructions string      `json:"instructions"`
	Input        []inputItem `json:"input"`
	Tools        []toolItem  `json:"tools"`
	ToolChoice   toolChoice  `json:"tool_choice"`
}

type inputItem struct {
	Type    string        `json:"type"`
	Role    string        `json:"role,omitempty"`
	Content []contentItem `json:"content,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type toolItem struct {
	Type          string `json:"type"`
	Model         string `json:"model"`
	Size          string `json:"size"`
	Quality       string `json:"quality"`
	OutputFormat  string `json:"output_format"`
	Background    string `json:"background"`
	PartialImages int    `json:"partial_images"`
}

type toolChoice struct {
	Type  string    `json:"type"`
	Mode  string    `json:"mode"`
	Tools []toolRef `json:"tools"`
}

type toolRef struct {
	Type string `json:"type"`
}

// init registers the codex provider.
func init() {
	provider.Register("codex", func(apiKey string) provider.Provider {
		return New(apiKey)
	})
}

var _ provider.Provider = (*Client)(nil)
