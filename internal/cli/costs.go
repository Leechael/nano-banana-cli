package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type CostEntry struct {
	Timestamp    string  `json:"timestamp"`
	Model        string  `json:"model"`
	Provider     string  `json:"provider,omitempty"`
	Size         string  `json:"size"`
	Aspect       *string `json:"aspect"`
	PromptTokens int32   `json:"prompt_tokens"`
	OutputTokens int32   `json:"output_tokens"`
	Estimated    float64 `json:"estimated_cost"`
	OutputFile   string  `json:"output_file"`
}

func CostLogPath() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".imagen", "costs.json")
}

func MigrateCostLog() {
	dest := CostLogPath()
	if _, err := os.Stat(dest); err == nil {
		return // already exists
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	src := filepath.Join(home, ".nano-banana", "costs.json")
	if _, err := os.Stat(src); err != nil {
		return // source doesn't exist
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0o755)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return
	}
	fmt.Fprintln(os.Stderr, "[imagen] migrated cost log from ~/.nano-banana/ to ~/.imagen/ — old path no longer used")
}

func LogCost(entry CostEntry) error {
	path := CostLogPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	entries := []CostEntry{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &entries)
	}
	entries = append(entries, entry)
	b, _ := json.MarshalIndent(entries, "", "  ")
	return os.WriteFile(path, b, 0o644)
}

func PrintCosts(mode OutputMode) error {
	path := CostLogPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if mode == ModeJSON {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "total": 0, "entries": 0})
			return nil
		}
		fmt.Fprintln(os.Stderr, "No cost data found")
		return nil
	}
	entries := []CostEntry{}
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	total := 0.0
	byModel := map[string]float64{}
	byProvider := map[string]float64{}
	for _, e := range entries {
		total += e.Estimated
		byModel[e.Model] += e.Estimated
		if e.Provider != "" {
			byProvider[e.Provider] += e.Estimated
		}
	}
	if mode == ModeJSON {
		names := make([]string, 0, len(byModel))
		for m := range byModel {
			names = append(names, m)
		}
		sort.Strings(names)
		models := make([]map[string]any, 0, len(names))
		for _, m := range names {
			models = append(models, map[string]any{"model": m, "cost": byModel[m]})
		}
		provNames := make([]string, 0, len(byProvider))
		for p := range byProvider {
			provNames = append(provNames, p)
		}
		sort.Strings(provNames)
		providers := make([]map[string]any, 0, len(provNames))
		for _, p := range provNames {
			providers = append(providers, map[string]any{"provider": p, "cost": byProvider[p]})
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":        true,
			"entries":   len(entries),
			"total":     total,
			"models":    models,
			"providers": providers,
		})
	}
	fmt.Fprintf(os.Stderr, "Total generations: %d\n", len(entries))
	fmt.Fprintf(os.Stderr, "Total cost: $%.4f\n", total)
	return nil
}
