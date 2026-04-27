package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	icli "github.com/leechael/imagen/internal/cli"
	"github.com/leechael/imagen/keyer"
	"github.com/leechael/imagen/provider"

	// Register providers via init().
	_ "github.com/leechael/imagen/provider/google"
	_ "github.com/leechael/imagen/provider/xai"
)

func main() {
	opts, err := icli.ParseArgs(os.Args[1:])
	if err != nil {
		icli.ExitError(err, icli.ModeHuman)
	}

	icli.MigrateCostLog()

	if opts.ShowCosts {
		if err := icli.PrintCosts(opts.OutputMode); err != nil {
			icli.ExitError(err, opts.OutputMode)
		}
		return
	}

	if opts.JQ != "" && opts.OutputMode != icli.ModeJSON {
		icli.ExitError(errors.New("--jq 只能和 --json 一起使用"), opts.OutputMode)
	}

	modelSpec := opts.Model
	if modelSpec == "" {
		modelSpec = "google/flash"
	}
	providerName, modelID, err := provider.ParseModel(modelSpec)
	if err != nil {
		icli.ExitError(err, opts.OutputMode)
	}

	apiKey, err := resolveAPIKey(opts.APIKey, providerName)
	if err != nil {
		icli.ExitError(err, opts.OutputMode)
	}

	prov, err := provider.Get(providerName, apiKey)
	if err != nil {
		icli.ExitError(err, opts.OutputMode)
	}

	cap := prov.Capabilities(modelID)
	var warnings []string

	genReq := provider.GenerateRequest{
		Model:  modelID,
		Prompt: opts.Prompt,
		Size:   opts.Size,
		N:      opts.Count,
	}

	if opts.AspectRatio != "" {
		if cap.SupportsAspectRatio {
			genReq.AspectRatio = opts.AspectRatio
		} else {
			warnings = append(warnings, fmt.Sprintf("provider %s does not support aspect ratio, ignoring", providerName))
		}
	}
	if opts.Seed != nil {
		if cap.SupportsSeed {
			genReq.Seed = opts.Seed
		} else {
			warnings = append(warnings, fmt.Sprintf("provider %s does not support --seed, ignoring", providerName))
		}
	}
	if opts.PersonGeneration != "" {
		if cap.SupportsPerson {
			genReq.Person = opts.PersonGeneration
		} else {
			warnings = append(warnings, fmt.Sprintf("provider %s does not support --person, ignoring", providerName))
		}
	}
	if opts.ThinkingLevel != "" {
		if cap.SupportsThinking {
			genReq.Thinking = opts.ThinkingLevel
		} else {
			warnings = append(warnings, fmt.Sprintf("provider %s/%s does not support --thinking, ignoring", providerName, modelID))
		}
	}
	if opts.Count > 1 && !cap.SupportsBatch {
		warnings = append(warnings, fmt.Sprintf("provider %s does not support batch generation, using count=1", providerName))
		genReq.N = 1
	}

	if len(opts.References) > 0 {
		if !cap.SupportsReferences {
			warnings = append(warnings, fmt.Sprintf("provider %s does not support references, ignoring", providerName))
		} else {
			refs, err := loadReferences(opts.References, cap.MaxReferences)
			if err != nil {
				icli.ExitError(err, opts.OutputMode)
			}
			genReq.References = refs
		}
	}

	if opts.Transparent {
		genReq.Prompt += ". Place the subject on a solid bright green background (#00FF00). The background must be a single flat green color with no gradients, shadows, or variation."
	}

	for _, w := range warnings {
		icli.LogLine(opts.OutputMode, "warn", "%s", w)
	}

	icli.LogLine(opts.OutputMode, "info", "generating image...")
	res, err := prov.Generate(context.Background(), genReq)
	if err != nil {
		icli.ExitError(err, opts.OutputMode)
	}

	for _, w := range res.Warnings {
		icli.LogLine(opts.OutputMode, "warn", "%s", w)
		warnings = append(warnings, w)
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		icli.ExitError(err, opts.OutputMode)
	}

	files := []string{}
	for idx, img := range res.Images {
		ext := extFromMime(img.MIMEType)
		name := opts.Output
		if idx > 0 {
			name = fmt.Sprintf("%s_%d", opts.Output, idx)
		}
		out := filepath.Join(opts.OutputDir, name+ext)
		if err := os.WriteFile(out, img.Data, 0o644); err != nil {
			icli.ExitError(err, opts.OutputMode)
		}
		files = append(files, out)
	}

	if len(files) == 0 {
		icli.ExitError(errors.New("no images generated"), opts.OutputMode)
	}

	if opts.Transparent {
		processed := make([]string, 0, len(files))
		for _, file := range files {
			out, rerr := removeBackground(file)
			if rerr != nil {
				icli.LogLine(opts.OutputMode, "warn", "transparent failed: %s (%v)", file, rerr)
				processed = append(processed, file)
				continue
			}
			processed = append(processed, out)
		}
		files = processed
	}

	if res.PromptTokens > 0 || res.OutputTokens > 0 {
		_ = icli.LogCost(icli.CostEntry{
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			Model:        res.Model,
			Provider:     providerName,
			Size:         opts.Size,
			Aspect:       icli.Nullable(opts.AspectRatio),
			PromptTokens: res.PromptTokens,
			OutputTokens: res.OutputTokens,
			Estimated:    res.Cost,
			OutputFile:   files[0],
		})
	} else if res.Cost > 0 {
		_ = icli.LogCost(icli.CostEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Model:      res.Model,
			Provider:   providerName,
			Size:       opts.Size,
			Aspect:     icli.Nullable(opts.AspectRatio),
			Estimated:  res.Cost,
			OutputFile: files[0],
		})
	}

	result := icli.Result{
		OK:       true,
		Model:    res.Model,
		Prompt:   opts.Prompt,
		Size:     opts.Size,
		Aspect:   opts.AspectRatio,
		Files:    files,
		Cost:     res.Cost,
		Warnings: warnings,
		At:       time.Now().UTC(),
	}

	switch opts.OutputMode {
	case icli.ModeJSON:
		_ = json.NewEncoder(os.Stdout).Encode(result)
	case icli.ModePlain:
		for _, f := range files {
			fmt.Fprintln(os.Stdout, f)
		}
	default:
		icli.LogLine(opts.OutputMode, "info", "generated %d image(s)", len(files))
		for _, f := range files {
			fmt.Fprintf(os.Stderr, "  + %s\n", f)
		}
	}
}

func resolveAPIKey(flagKey, providerName string) (string, error) {
	if flagKey != "" {
		return flagKey, nil
	}
	switch providerName {
	case "google":
		if v := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); v != "" {
			return v, nil
		}
		paths := dotEnvPaths()
		for _, p := range paths {
			if v := icli.ReadDotEnvValue(p, "GEMINI_API_KEY"); v != "" {
				return v, nil
			}
		}
		return "", errors.New("GEMINI_API_KEY is required")
	case "xai":
		if v := strings.TrimSpace(os.Getenv("XAI_API_KEY")); v != "" {
			return v, nil
		}
		if v := strings.TrimSpace(os.Getenv("GROK_API_KEY")); v != "" {
			return v, nil
		}
		paths := dotEnvPaths()
		for _, p := range paths {
			if v := icli.ReadDotEnvValue(p, "XAI_API_KEY"); v != "" {
				return v, nil
			}
			if v := icli.ReadDotEnvValue(p, "GROK_API_KEY"); v != "" {
				return v, nil
			}
		}
		return "", errors.New("XAI_API_KEY is required")
	default:
		return "", fmt.Errorf("no API key resolution defined for provider %q", providerName)
	}
}

func dotEnvPaths() []string {
	paths := []string{}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "..", ".env"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".imagen", ".env"))
	}
	return paths
}

func loadReferences(paths []string, maxRefs int) ([]provider.Reference, error) {
	if maxRefs > 0 && len(paths) > maxRefs {
		return nil, fmt.Errorf("too many references: provider supports at most %d, got %d", maxRefs, len(paths))
	}
	refs := make([]provider.Reference, 0, len(paths))
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			wd, _ := os.Getwd()
			p = filepath.Join(wd, p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("reference not found: %s", p)
		}
		ext := strings.ToLower(filepath.Ext(p))
		m := mime.TypeByExtension(ext)
		if m == "" {
			switch ext {
			case ".jpg", ".jpeg":
				m = "image/jpeg"
			case ".webp":
				m = "image/webp"
			case ".gif":
				m = "image/gif"
			default:
				m = "image/png"
			}
		}
		refs = append(refs, provider.Reference{Data: b, MIMEType: m})
	}
	return refs, nil
}

func removeBackground(input string) (string, error) {
	f, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode %s: %w", input, err)
	}
	result := keyer.RemoveBackground(img)
	dir := filepath.Dir(input)
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	out := filepath.Join(dir, base+".png")
	w, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer w.Close()
	if err := png.Encode(w, result); err != nil {
		return "", err
	}
	return out, nil
}

func extFromMime(mt string) string {
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
