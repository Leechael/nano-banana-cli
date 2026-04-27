package provider

import (
	"fmt"
	"strings"
)

var aliases = map[string]map[string]string{
	"google": {
		"nb":     "gemini-2.5-flash-image",
		"flash":  "gemini-3.1-flash-image-preview",
		"nb2":    "gemini-3.1-flash-image-preview",
		"pro":    "gemini-3-pro-image-preview",
		"nb-pro": "gemini-3-pro-image-preview",
	},
	"xai": {
		"grok":         "grok-imagine-image",
		"grok-imagine": "grok-imagine-image",
	},
	"openai": {
		"gpt-image-2":          "gpt-image-2-2026-04-21",
		"gpt-image-1.5":        "gpt-image-1.5",
		"gpt-image-1":          "gpt-image-1",
		"gpt-image-1-mini":     "gpt-image-1-mini",
		"chatgpt-image-latest": "chatgpt-image-latest",
		"dall-e-3":             "dall-e-3",
		"dall-e-2":             "dall-e-2",
		"oai-2":                "gpt-image-2-2026-04-21",
		"oai-15":               "gpt-image-1.5",
	},
}

// providerOrder defines deterministic resolution order for bare aliases.
var providerOrder = []string{"google", "xai", "openai"}

var registry = map[string]func(apiKey string) Provider{}

// Register adds a provider factory to the registry.
func Register(name string, factory func(apiKey string) Provider) {
	registry[name] = factory
}

// Get returns a constructed provider by name using the given API key.
func Get(providerName, apiKey string) (Provider, error) {
	factory, ok := registry[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}
	return factory(apiKey), nil
}

// ParseModel resolves a user-facing model spec to (providerName, canonicalModelID).
// Accepted forms:
//
//	"provider/model"           explicit
//	"provider/alias"           explicit, alias resolved within provider
//	"alias"                    bare alias resolved across all providers
//	"provider/unknown-model"   passthrough — providerName known, modelID as-is
//
// Returns error if input doesn't contain "/" and no alias matches.
func ParseModel(input string) (providerName, modelID string, err error) {
	if strings.Contains(input, "/") {
		parts := strings.SplitN(input, "/", 2)
		providerName = parts[0]
		raw := parts[1]
		if provAliases, ok := aliases[providerName]; ok {
			if resolved, ok := provAliases[strings.ToLower(raw)]; ok {
				return providerName, resolved, nil
			}
		}
		return providerName, raw, nil
	}

	lower := strings.ToLower(input)
	for _, prov := range providerOrder {
		if provAliases, ok := aliases[prov]; ok {
			if resolved, ok := provAliases[lower]; ok {
				return prov, resolved, nil
			}
		}
	}
	return "", "", fmt.Errorf("unknown model or alias %q — use \"provider/model\" form or a known alias", input)
}
