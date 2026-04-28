package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	icli "github.com/leechael/imagen/internal/cli"
)

type statusProviderConfig struct {
	Name           string
	KeyEnvNames    []string
	BaseURLEnvName string
	DefaultBaseURL string
	AuthURL        func(baseURL, apiKey string) string
	AuthHeader     func(apiKey string) (string, string)
	CreditsCheck   func(ctx context.Context) (string, string)
}

type statusReport struct {
	OK        bool             `json:"ok"`
	Providers []providerStatus `json:"providers"`
	At        time.Time        `json:"at"`
}

type providerStatus struct {
	Provider       string `json:"provider"`
	APIKeySet      bool   `json:"api_key_set"`
	APIKeyEnv      string `json:"api_key_env,omitempty"`
	APIKeyMasked   string `json:"api_key_masked,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
	BaseURLEnv     string `json:"base_url_env,omitempty"`
	BaseURLChanged bool   `json:"base_url_changed,omitempty"`
	AuthStatus     string `json:"auth_status"`
	AuthMessage    string `json:"auth_message,omitempty"`
	CreditsStatus  string `json:"credits_status"`
	CreditsMessage string `json:"credits_message,omitempty"`
}

func printStatus(ctx context.Context, mode icli.OutputMode) error {
	report := statusReport{
		OK:        true,
		Providers: checkProviderStatuses(ctx),
		At:        time.Now().UTC(),
	}

	switch mode {
	case icli.ModeJSON:
		return json.NewEncoder(os.Stdout).Encode(report)
	case icli.ModePlain:
		for _, p := range report.Providers {
			key := "missing"
			if p.APIKeySet {
				key = "set"
			}
			base := "default"
			if p.BaseURLChanged {
				base = "changed"
			}
			if p.BaseURL == "" {
				base = "n/a"
			}
			fmt.Fprintf(os.Stdout, "%s key=%s base_url=%s auth=%s credits=%s\n", p.Provider, key, base, p.AuthStatus, p.CreditsStatus)
		}
	default:
		fmt.Fprintln(os.Stdout, "Provider status:")
		for _, p := range report.Providers {
			key := "missing"
			if p.APIKeySet {
				key = "set"
				if p.APIKeyMasked != "" {
					key += " (" + p.APIKeyMasked + ")"
				}
			}
			base := "not configurable"
			if p.BaseURL != "" {
				base = p.BaseURL
				if p.BaseURLChanged {
					base += " (changed)"
				} else {
					base += " (default)"
				}
			}
			fmt.Fprintf(os.Stdout, "- %s\n", p.Provider)
			fmt.Fprintf(os.Stdout, "    key:      %s\n", key)
			fmt.Fprintf(os.Stdout, "    base URL: %s\n", base)
			fmt.Fprintf(os.Stdout, "    auth:     %s\n", p.AuthStatus)
			if p.AuthMessage != "" && p.AuthStatus != "ok" {
				fmt.Fprintf(os.Stdout, "              %s\n", p.AuthMessage)
			}
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}

func checkProviderStatuses(ctx context.Context) []providerStatus {
	configs := []statusProviderConfig{
		{
			Name:        "google",
			KeyEnvNames: []string{"GEMINI_API_KEY"},
			AuthURL: func(_ string, apiKey string) string {
				return "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
			},
			AuthHeader:     noAuthHeader,
			DefaultBaseURL: "",
			CreditsCheck: func(context.Context) (string, string) {
				return "unsupported", "Gemini API prepay balance is only available in AI Studio Billing; quota requires Google Cloud auth."
			},
		},
		{
			Name:           "xai",
			KeyEnvNames:    []string{"XAI_API_KEY", "GROK_API_KEY"},
			BaseURLEnvName: "XAI_BASE_URL",
			DefaultBaseURL: "https://api.x.ai/v1",
			AuthURL:        func(baseURL, _ string) string { return baseURL + "/image-generation-models" },
			AuthHeader:     bearerAuthHeader,
			CreditsCheck:   checkXAICredits,
		},
		{
			Name:           "openai",
			KeyEnvNames:    []string{"OPENAI_API_KEY"},
			BaseURLEnvName: "OPENAI_BASE_URL",
			DefaultBaseURL: "https://api.openai.com/v1",
			AuthURL:        func(baseURL, _ string) string { return baseURL + "/models" },
			AuthHeader:     bearerAuthHeader,
			CreditsCheck: func(context.Context) (string, string) {
				return "requires_credentials", "OpenAI exposes organization usage/costs via Admin API keys, but not remaining prepaid credit via normal API keys."
			},
		},
		{
			Name:           "codex",
			KeyEnvNames:    []string{"CODEX_ACCESS_TOKEN", "CHATGPT_ACCESS_TOKEN"},
			BaseURLEnvName: "CODEX_BASE_URL",
			DefaultBaseURL: "https://chatgpt.com/backend-api/codex",
			CreditsCheck: func(context.Context) (string, string) {
				return "experimental", "Codex usage is exposed via ChatGPT/Codex usage UI or Codex /status, not a stable public API."
			},
		},
	}

	statuses := make([]providerStatus, 0, len(configs))
	for _, cfg := range configs {
		statuses = append(statuses, checkProviderStatus(ctx, cfg))
	}
	return statuses
}

func checkProviderStatus(ctx context.Context, cfg statusProviderConfig) providerStatus {
	keyEnv, key := firstEnvValue(cfg.KeyEnvNames)
	baseURL := ""
	baseURLChanged := false
	if cfg.DefaultBaseURL != "" {
		baseURL = cfg.DefaultBaseURL
		if v := strings.TrimSpace(os.Getenv(cfg.BaseURLEnvName)); v != "" {
			baseURL = strings.TrimSuffix(v, "/")
			baseURLChanged = baseURL != cfg.DefaultBaseURL
		}
	}

	status := providerStatus{
		Provider:       cfg.Name,
		APIKeySet:      key != "",
		APIKeyEnv:      keyEnv,
		APIKeyMasked:   maskSecret(key),
		BaseURL:        baseURL,
		BaseURLEnv:     cfg.BaseURLEnvName,
		BaseURLChanged: baseURLChanged,
	}
	if cfg.CreditsCheck != nil {
		status.CreditsStatus, status.CreditsMessage = cfg.CreditsCheck(ctx)
	} else {
		status.CreditsStatus = "unknown"
	}

	switch {
	case key == "":
		status.AuthStatus = "skipped"
		status.AuthMessage = "API key is not set in the current process environment."
	case cfg.AuthURL == nil:
		status.AuthStatus = "unsupported"
		status.AuthMessage = "No stable public lightweight auth endpoint is configured for this provider."
	default:
		status.AuthStatus, status.AuthMessage = checkAuth(ctx, cfg, baseURL, key)
	}
	return status
}

func checkAuth(ctx context.Context, cfg statusProviderConfig, baseURL, apiKey string) (string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.AuthURL(baseURL, apiKey), nil)
	if err != nil {
		return "error", err.Error()
	}
	if cfg.AuthHeader != nil {
		name, value := cfg.AuthHeader(apiKey)
		if name != "" {
			req.Header.Set(name, value)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "error", err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok", http.StatusText(resp.StatusCode)
	}
	msg := readSmallBody(resp.Body)
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return "error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, msg)
}

func firstEnvValue(names []string) (string, string) {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return name, v
		}
	}
	return "", ""
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func bearerAuthHeader(apiKey string) (string, string) {
	return "Authorization", "Bearer " + apiKey
}

func noAuthHeader(string) (string, string) {
	return "", ""
}

func describeStatus(status, message string) string {
	if message == "" {
		return status
	}
	return status + " (" + message + ")"
}

func readSmallBody(r io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func checkXAICredits(ctx context.Context) (string, string) {
	managementKey := strings.TrimSpace(os.Getenv("XAI_MANAGEMENT_API_KEY"))
	teamID := strings.TrimSpace(os.Getenv("XAI_TEAM_ID"))
	if managementKey == "" || teamID == "" {
		return "requires_credentials", "requires XAI_MANAGEMENT_API_KEY and XAI_TEAM_ID; inference API keys cannot query prepaid balance."
	}

	url := "https://management-api.x.ai/v1/billing/teams/" + teamID + "/prepaid/balance"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "error", err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+managementKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "error", err.Error()
	}
	defer resp.Body.Close()

	body := readSmallBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if body == "" {
			body = http.StatusText(resp.StatusCode)
		}
		return "error", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, body)
	}
	if body == "" {
		return "ok", "prepaid balance returned"
	}
	return "ok", summarizeJSONField(body, "total")
}

func summarizeJSONField(body, field string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		return "prepaid balance returned"
	}
	v, ok := obj[field]
	if !ok {
		return "prepaid balance returned"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "prepaid balance returned"
	}
	return field + "=" + string(b)
}
