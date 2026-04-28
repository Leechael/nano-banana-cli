package main

import (
	"os"
	"path/filepath"
	"testing"

	icli "github.com/leechael/imagen/internal/cli"
)

// --- ParseArgs ---

func TestParseArgs_BasicPrompt(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"a", "cat"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Prompt != "a cat" {
		t.Errorf("prompt = %q, want %q", opts.Prompt, "a cat")
	}
	if opts.Size != "1K" {
		t.Errorf("size = %q, want %q", opts.Size, "1K")
	}
	if opts.OutputMode != icli.ModeHuman {
		t.Errorf("mode = %q, want %q", opts.OutputMode, icli.ModeHuman)
	}
}

func TestParseArgs_AllFlags(t *testing.T) {
	opts, err := icli.ParseArgs([]string{
		"-o", "out", "-s", "2K", "-a", "16:9",
		"-m", "google/pro", "-d", "/tmp", "-r", "ref.png",
		"-t", "--api-key", "key123", "--json",
		"hello", "world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Output != "out" {
		t.Errorf("output = %q", opts.Output)
	}
	if opts.Size != "2K" {
		t.Errorf("size = %q", opts.Size)
	}
	if opts.AspectRatio != "16:9" {
		t.Errorf("aspect = %q", opts.AspectRatio)
	}
	if opts.Model != "google/pro" {
		t.Errorf("model = %q", opts.Model)
	}
	if opts.OutputDir != "/tmp" {
		t.Errorf("dir = %q", opts.OutputDir)
	}
	if len(opts.References) != 1 || opts.References[0] != "ref.png" {
		t.Errorf("refs = %v", opts.References)
	}
	if !opts.Transparent {
		t.Error("transparent should be true")
	}
	if opts.APIKey != "key123" {
		t.Errorf("apiKey = %q", opts.APIKey)
	}
	if opts.OutputMode != icli.ModeJSON {
		t.Errorf("mode = %q", opts.OutputMode)
	}
	if opts.Prompt != "hello world" {
		t.Errorf("prompt = %q", opts.Prompt)
	}
}

func TestParseArgs_PlainMode(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"--plain", "test"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.OutputMode != icli.ModePlain {
		t.Errorf("mode = %q, want %q", opts.OutputMode, icli.ModePlain)
	}
}

func TestParseArgs_CostsMode(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"--costs"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShowCosts {
		t.Error("ShowCosts should be true")
	}
}

func TestParseArgs_StatusCommand(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"status"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShowStatus {
		t.Error("ShowStatus should be true")
	}
	if opts.Prompt != "" {
		t.Errorf("prompt = %q, want empty", opts.Prompt)
	}
}

func TestParseArgs_StatusFlag(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"--status", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.ShowStatus {
		t.Error("ShowStatus should be true")
	}
	if opts.OutputMode != icli.ModeJSON {
		t.Errorf("mode = %q, want %q", opts.OutputMode, icli.ModeJSON)
	}
}

func TestParseArgs_JQ(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"--jq", ".files", "test"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.JQ != ".files" {
		t.Errorf("jq = %q", opts.JQ)
	}
}

func TestParseArgs_NoPrompt(t *testing.T) {
	_, err := icli.ParseArgs([]string{"-s", "1K"})
	if err == nil {
		t.Error("expected error for no prompt")
	}
}

func TestParseArgs_UnknownOption(t *testing.T) {
	_, err := icli.ParseArgs([]string{"--bogus", "test"})
	if err == nil {
		t.Error("expected error for unknown option")
	}
}

func TestParseArgs_MissingValue(t *testing.T) {
	flags := []string{"--output", "--size", "--aspect", "--model", "--dir", "--ref", "--api-key", "--jq"}
	for _, f := range flags {
		_, err := icli.ParseArgs([]string{f})
		if err == nil {
			t.Errorf("expected error for %s without value", f)
		}
	}
}

func TestParseArgs_InvalidSize(t *testing.T) {
	_, err := icli.ParseArgs([]string{"-s", "3K", "test"})
	if err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestParseArgs_InvalidAspect(t *testing.T) {
	_, err := icli.ParseArgs([]string{"-a", "7:3", "test"})
	if err == nil {
		t.Error("expected error for invalid aspect")
	}
}

func TestParseArgs_Count(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"-n", "3", "test"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Count != 3 {
		t.Errorf("count = %d, want 3", opts.Count)
	}
}

func TestParseArgs_CountLongFlag(t *testing.T) {
	opts, err := icli.ParseArgs([]string{"--count", "5", "test"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Count != 5 {
		t.Errorf("count = %d, want 5", opts.Count)
	}
}

func TestParseArgs_InvalidCount(t *testing.T) {
	_, err := icli.ParseArgs([]string{"-n", "0", "test"})
	if err == nil {
		t.Error("expected error for count=0")
	}
}

// --- extFromMime ---

func TestExtFromMime_Known(t *testing.T) {
	got := extFromMime("image/png")
	if got != ".png" {
		t.Errorf("extFromMime(image/png) = %q, want .png", got)
	}
}

func TestExtFromMime_Fallback(t *testing.T) {
	got := extFromMime("image/webp")
	if got == "" {
		t.Error("extFromMime(image/webp) returned empty")
	}
}

func TestExtFromMime_NoSlash(t *testing.T) {
	got := extFromMime("bogus")
	if got != ".png" {
		t.Errorf("extFromMime(bogus) = %q, want .png", got)
	}
}

func TestExtFromMime_UnknownSubtype(t *testing.T) {
	got := extFromMime("image/xyzzy")
	if got != ".xyzzy" {
		t.Errorf("extFromMime(image/xyzzy) = %q, want .xyzzy", got)
	}
}

// --- Nullable ---

func TestNullable_Empty(t *testing.T) {
	if icli.Nullable("") != nil {
		t.Error("Nullable(\"\") should return nil")
	}
}

func TestNullable_NonEmpty(t *testing.T) {
	p := icli.Nullable("hello")
	if p == nil {
		t.Fatal("Nullable(\"hello\") should not return nil")
	}
	if *p != "hello" {
		t.Errorf("Nullable(\"hello\") = %q", *p)
	}
}

// --- ReadDotEnvValue ---

func TestReadDotEnvValue_ReadsKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\n\nGEMINI_API_KEY=test-key-123\nOTHER=val\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := icli.ReadDotEnvValue(path, "GEMINI_API_KEY")
	if got != "test-key-123" {
		t.Errorf("ReadDotEnvValue = %q, want %q", got, "test-key-123")
	}
}

func TestReadDotEnvValue_QuotedValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `GEMINI_API_KEY="quoted-key"` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := icli.ReadDotEnvValue(path, "GEMINI_API_KEY")
	if got != "quoted-key" {
		t.Errorf("ReadDotEnvValue = %q, want %q", got, "quoted-key")
	}
}

func TestReadDotEnvValue_MissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "OTHER=val\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := icli.ReadDotEnvValue(path, "GEMINI_API_KEY")
	if got != "" {
		t.Errorf("ReadDotEnvValue = %q, want empty", got)
	}
}

func TestReadDotEnvValue_MissingFile(t *testing.T) {
	got := icli.ReadDotEnvValue("/nonexistent/path/.env", "GEMINI_API_KEY")
	if got != "" {
		t.Errorf("ReadDotEnvValue = %q, want empty", got)
	}
}

// --- ReadStdinPipe ---

func TestReadStdinPipe_FromPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()

	go func() {
		_, _ = w.Write([]byte("hello from pipe\n"))
		w.Close()
	}()

	got, err := icli.ReadStdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello from pipe" {
		t.Errorf("ReadStdinPipe() = %q, want %q", got, "hello from pipe")
	}
}
