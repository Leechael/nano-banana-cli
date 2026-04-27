package google

import (
	"context"
	"errors"
	"mime"
	"strings"

	"github.com/leechael/imagen/provider"

	"google.golang.org/genai"
)

var costRates = map[string]costRate{
	"gemini-3.1-flash-image-preview": {Input: 0.25, ImageOutput: 60},
	"gemini-3-pro-image-preview":     {Input: 2.0, ImageOutput: 120},
}

const defaultModel = "gemini-3.1-flash-image-preview"

type costRate struct {
	Input       float64
	ImageOutput float64
}

var validAspects = map[string]bool{
	"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true,
	"3:2": true, "2:3": true, "4:5": true, "5:4": true, "21:9": true,
	"1:4": true, "1:8": true, "4:1": true, "8:1": true,
}

// Client implements provider.Provider for Google Gemini.
type Client struct {
	apiKey string
}

// New returns a new Google Gemini provider.
func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) Name() string { return "google" }

func (c *Client) Capabilities(model string) provider.Capability {
	aspectRatios := make([]string, 0, len(validAspects))
	for k := range validAspects {
		aspectRatios = append(aspectRatios, k)
	}
	return provider.Capability{
		SupportsAspectRatio: true,
		SupportsSeed:        true,
		SupportsPerson:      true,
		SupportsThinking:    model == "gemini-3.1-flash-image-preview",
		SupportsReferences:  true,
		SupportsBatch:       true,
		Sizes:               []string{"512", "1K", "2K", "4K"},
		AspectRatios:        aspectRatios,
		MaxReferences:       10,
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

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: c.apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, err
	}

	var result provider.GenerateResult
	result.Model = model

	for i := 0; i < n; i++ {
		res, err := c.callOnce(ctx, client, model, req)
		if err != nil {
			return nil, err
		}
		result.Images = append(result.Images, res.Images...)
		result.PromptTokens += res.PromptTokens
		result.OutputTokens += res.OutputTokens
		result.Cost += res.Cost
		result.Warnings = append(result.Warnings, res.Warnings...)
	}

	return &result, nil
}

func (c *Client) callOnce(ctx context.Context, client *genai.Client, model string, req provider.GenerateRequest) (*provider.GenerateResult, error) {
	parts := []*genai.Part{}
	for _, ref := range req.References {
		parts = append(parts, genai.NewPartFromBytes(ref.Data, ref.MIMEType))
	}
	parts = append(parts, genai.NewPartFromText(req.Prompt))
	contents := []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}

	imgCfg := &genai.ImageConfig{ImageSize: imageSize(req.Size)}
	if req.AspectRatio != "" {
		imgCfg.AspectRatio = req.AspectRatio
	}
	if req.Person != "" {
		imgCfg.PersonGeneration = req.Person
	}

	cfg := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE", "TEXT"},
		ImageConfig:        imgCfg,
		Tools:              []*genai.Tool{{GoogleSearch: &genai.GoogleSearch{}}},
	}
	if req.Seed != nil {
		cfg.Seed = req.Seed
	}
	if req.Thinking != "" {
		cfg.ThinkingConfig = &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevel(strings.ToUpper(req.Thinking))}
	}

	resp, err := client.Models.GenerateContent(ctx, model, contents, cfg)
	if err != nil {
		return nil, err
	}

	var images []provider.Image
	for _, cand := range resp.Candidates {
		if cand == nil || cand.Content == nil {
			continue
		}
		for _, p := range cand.Content.Parts {
			if p.InlineData != nil && len(p.InlineData.Data) > 0 {
				images = append(images, provider.Image{
					Data:     p.InlineData.Data,
					MIMEType: p.InlineData.MIMEType,
				})
			}
		}
	}

	if len(images) == 0 {
		return nil, errors.New("no images generated")
	}

	var cost float64
	var promptTokens, outputTokens int32
	if resp.UsageMetadata != nil {
		promptTokens = resp.UsageMetadata.PromptTokenCount
		outputTokens = resp.UsageMetadata.CandidatesTokenCount
		cost = calculateCost(model, promptTokens, outputTokens)
	}

	return &provider.GenerateResult{
		Images:       images,
		Model:        model,
		Cost:         cost,
		PromptTokens: promptTokens,
		OutputTokens: outputTokens,
	}, nil
}

func imageSize(size string) string {
	if size == "512" {
		return "1K"
	}
	return size
}

func calculateCost(model string, promptTokens, outputTokens int32) float64 {
	rate, ok := costRates[model]
	if !ok {
		rate = costRates[defaultModel]
	}
	return (float64(promptTokens)/1_000_000)*rate.Input + (float64(outputTokens)/1_000_000)*rate.ImageOutput
}

// ExtFromMime returns a file extension for a MIME type.
func ExtFromMime(mt string) string {
	preferred := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
	if ext, ok := preferred[mt]; ok {
		return ext
	}
	if exts, err := mime.ExtensionsByType(mt); err == nil && len(exts) > 0 {
		return exts[0]
	}
	if strings.Contains(mt, "/") {
		return "." + strings.SplitN(mt, "/", 2)[1]
	}
	return ".png"
}

// CalculateCost is exported for testing.
func CalculateCost(model string, promptTokens, outputTokens int32) float64 {
	return calculateCost(model, promptTokens, outputTokens)
}

// init registers the google provider.
func init() {
	provider.Register("google", func(apiKey string) provider.Provider {
		return New(apiKey)
	})
}

// AsProvider wraps Client as a provider.Provider (already satisfies the interface).
var _ provider.Provider = (*Client)(nil)
