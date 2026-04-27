package provider

import "context"

type Capability struct {
	SupportsAspectRatio bool
	SupportsSeed        bool
	SupportsPerson      bool
	SupportsThinking    bool
	SupportsReferences  bool
	SupportsBatch       bool
	Sizes               []string
	AspectRatios        []string
	MaxReferences       int
}

type Image struct {
	Data     []byte
	MIMEType string
}

type Reference struct {
	Data     []byte
	MIMEType string
}

type GenerateRequest struct {
	Model       string
	Prompt      string
	Size        string
	AspectRatio string
	References  []Reference
	Seed        *int32
	Person      string
	Thinking    string
	N           int
}

type GenerateResult struct {
	Images       []Image
	Model        string
	Cost         float64
	PromptTokens int32
	OutputTokens int32
	Warnings     []string
}

type Provider interface {
	Name() string
	Capabilities(model string) Capability
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}
