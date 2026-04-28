package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var validSizes = map[string]bool{"512": true, "1K": true, "2K": true, "4K": true}
var ValidAspects = map[string]bool{
	// Google Gemini
	"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true,
	"3:2": true, "2:3": true, "4:5": true, "5:4": true, "21:9": true,
	"1:4": true, "1:8": true, "4:1": true, "8:1": true,
	// xAI Grok (additional ratios not shared with Google)
	"2:1": true, "1:2": true, "20:9": true, "19.5:9": true, "auto": true,
}

type Options struct {
	Prompt           string
	Output           string
	Size             string
	OutputDir        string
	References       []string
	Transparent      bool
	APIKey           string
	Model            string
	AspectRatio      string
	Seed             *int32
	PersonGeneration string
	ThinkingLevel    string
	Quality          string
	ShowCosts        bool
	ShowStatus       bool
	OutputMode       OutputMode
	JQ               string
	Count            int
	ShowVersion      bool
	HelpTopic        string
}

func ParseArgs(args []string) (Options, error) {
	opts := Options{
		Output:     fmt.Sprintf("imagen-%d", time.Now().UnixMilli()),
		Size:       "1K",
		OutputDir:  mustWd(),
		Model:      "",
		OutputMode: ModeHuman,
		Count:      1,
	}
	if len(args) == 0 {
		if p, err := ReadStdinPipe(); err == nil && p != "" {
			opts.Prompt = p
			return opts, nil
		}
		PrintHelp()
		os.Exit(0)
	}

	promptParts := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			PrintHelp()
			os.Exit(0)
		case "-v", "--version":
			opts.ShowVersion = true
		case "--costs":
			opts.ShowCosts = true
		case "--status":
			opts.ShowStatus = true
		case "--json":
			opts.OutputMode = ModeJSON
		case "--plain":
			opts.OutputMode = ModePlain
		case "--jq":
			i++
			if i >= len(args) {
				return opts, errors.New("--jq 缺少表达式")
			}
			opts.JQ = args[i]
		case "-o", "--output":
			i++
			if i >= len(args) {
				return opts, errors.New("--output 缺少值")
			}
			opts.Output = args[i]
		case "-s", "--size":
			i++
			if i >= len(args) {
				return opts, errors.New("--size 缺少值")
			}
			if !validSizes[args[i]] {
				return opts, fmt.Errorf("invalid size %q", args[i])
			}
			opts.Size = args[i]
		case "-a", "--aspect":
			i++
			if i >= len(args) {
				return opts, errors.New("--aspect 缺少值")
			}
			if !ValidAspects[args[i]] {
				return opts, fmt.Errorf("invalid aspect %q", args[i])
			}
			opts.AspectRatio = args[i]
		case "-m", "--model":
			i++
			if i >= len(args) {
				return opts, errors.New("--model 缺少值")
			}
			opts.Model = args[i]
		case "-d", "--dir":
			i++
			if i >= len(args) {
				return opts, errors.New("--dir 缺少值")
			}
			opts.OutputDir = args[i]
		case "-r", "--ref":
			i++
			if i >= len(args) {
				return opts, errors.New("--ref 缺少值")
			}
			opts.References = append(opts.References, args[i])
		case "-t", "--transparent":
			opts.Transparent = true
		case "--seed":
			i++
			if i >= len(args) {
				return opts, errors.New("--seed 缺少值")
			}
			n, err := strconv.ParseInt(args[i], 10, 32)
			if err != nil {
				return opts, fmt.Errorf("invalid seed %q: %w", args[i], err)
			}
			v := int32(n)
			opts.Seed = &v
		case "--person":
			i++
			if i >= len(args) {
				return opts, errors.New("--person 缺少值")
			}
			switch strings.ToUpper(args[i]) {
			case "ALL", "ALLOW_ALL":
				opts.PersonGeneration = "ALLOW_ALL"
			case "ADULT", "ALLOW_ADULT":
				opts.PersonGeneration = "ALLOW_ADULT"
			case "NONE", "ALLOW_NONE":
				opts.PersonGeneration = "ALLOW_NONE"
			default:
				return opts, fmt.Errorf("invalid --person value %q (use ALL, ADULT, NONE)", args[i])
			}
		case "--thinking":
			i++
			if i >= len(args) {
				return opts, errors.New("--thinking 缺少值")
			}
			switch strings.ToLower(args[i]) {
			case "minimal", "low", "medium", "high":
				opts.ThinkingLevel = strings.ToLower(args[i])
			default:
				return opts, fmt.Errorf("invalid --thinking value %q (use minimal, low, medium, high)", args[i])
			}
		case "--quality":
			i++
			if i >= len(args) {
				return opts, errors.New("--quality 缺少值")
			}
			switch strings.ToLower(args[i]) {
			case "low", "medium", "high":
				opts.Quality = strings.ToLower(args[i])
			default:
				return opts, fmt.Errorf("invalid --quality value %q (use low, medium, high)", args[i])
			}
		case "--api-key":
			i++
			if i >= len(args) {
				return opts, errors.New("--api-key 缺少值")
			}
			opts.APIKey = args[i]
		case "-n", "--count":
			i++
			if i >= len(args) {
				return opts, errors.New("--count 缺少值")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				return opts, fmt.Errorf("invalid --count value %q (must be >= 1)", args[i])
			}
			opts.Count = n
		case "help":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				opts.HelpTopic = args[i+1]
				i++
			} else {
				PrintHelp()
				os.Exit(0)
			}
		case "version":
			opts.ShowVersion = true
		case "status":
			opts.ShowStatus = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, fmt.Errorf("unknown option: %s", a)
			}
			promptParts = append(promptParts, a)
		}
	}

	if opts.ShowVersion {
		return opts, nil
	}

	if opts.HelpTopic != "" {
		return opts, nil
	}

	if !opts.ShowCosts && !opts.ShowStatus {
		opts.Prompt = strings.TrimSpace(strings.Join(promptParts, " "))
		if opts.Prompt == "" {
			if stdinPrompt, err := ReadStdinPipe(); err == nil && stdinPrompt != "" {
				opts.Prompt = stdinPrompt
			} else {
				return opts, errors.New("no prompt provided")
			}
		}
	}
	return opts, nil
}

func PrintHelp() {
	fmt.Print(`imagen – generate images via Gemini, Grok, or OpenAI

Usage:
  imagen [options] <prompt...>
  imagen help <topic>
  imagen status

Options:
  -h, --help            Show this help message
  help <topic>          Show provider-specific help (google, grok, openai, codex)
  -o, --output NAME     Output file base name (default: imagen-<timestamp>)
  -s, --size SIZE       Image size: 512, 1K, 2K, 4K (default: 1K)
  -a, --aspect RATIO    Aspect ratio (e.g. 16:9, 4:3, 1:1)
  -m, --model MODEL     Model or alias: flash, pro, grok, oai-2, codex-2 (default: google/flash)
  -n, --count N         Number of images to generate (default: 1)
  -d, --dir DIR         Output directory (default: current directory)
  -r, --ref FILE        Reference image (can be repeated)
  -t, --transparent     Remove background (pure Go, no external tools)
      --seed N          Random seed for reproducible generation
      --person MODE     Person generation: ALL, ADULT, NONE (google only)
      --thinking LEVEL  Thinking level: minimal, low, medium, high (flash only)
      --quality LEVEL   Output quality: low, medium, high (grok, codex)
      --api-key KEY     API key (or set GEMINI_API_KEY / XAI_API_KEY / OPENAI_API_KEY / CODEX_ACCESS_TOKEN)
      --costs           Show accumulated cost summary
      --status          Check provider API keys, base URLs, and lightweight auth
      --json            JSON output mode
      --plain           Plain output mode (filenames only)
      --jq EXPR         Filter JSON output with jq expression

Stdin:
  Prompt can be piped via stdin when no positional prompt is given.
  echo "a cat in a spacesuit" | imagen -s 2K

Examples:
  imagen "a cat in a spacesuit"
  imagen -s 2K -a 16:9 "sunset over mountains"
  imagen -m xai/grok "a futuristic city"
  imagen -m openai/gpt-image-2 "a cute baby sea otter"
  imagen -m codex-2 "a futuristic city"
  imagen -r style.png "apply this style to a forest"
  imagen --json "logo for a coffee shop" | jq .files
  imagen --costs

Envs:
  GEMINI_API_KEY        API key for Google Gemini models
  XAI_API_KEY           API key for xAI Grok models
  OPENAI_API_KEY        API key for OpenAI image models
  CODEX_ACCESS_TOKEN    ChatGPT/Codex access token (for codex provider)

Supported Models:
  google/gemini-2.5-flash-image            aliases: nb
  google/gemini-3.1-flash-image-preview    aliases: flash, nb2
  google/gemini-3-pro-image-preview        aliases: pro, nb-pro
  xai/grok-imagine-image                   aliases: grok, grok-imagine
  openai/gpt-image-2                       gpt-image-2-2026-04-21 (alias: oai-2)
  openai/gpt-image-1.5                     gpt-image-1.5 (alias: oai-15)
  codex/codex-2                            gpt-image-2-medium (default)
  codex/codex-2-low                        gpt-image-2-low
  codex/codex-2-high                       gpt-image-2-high

Help Topics:
  google, gemini           Google Gemini provider details
  grok, xai                xAI Grok provider details
  openai, gpt-image        OpenAI image generation provider details
  codex, chatgpt           ChatGPT/Codex image generation provider details
`)
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func ReadStdinPipe() (string, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func Nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
