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

var validSizes = map[string]bool{"512": true, "1K": true, "2K": true, "4K": true}
var ValidAspects = map[string]bool{
	"1:1": true, "16:9": true, "9:16": true, "4:3": true, "3:4": true,
	"3:2": true, "2:3": true, "4:5": true, "5:4": true, "21:9": true,
	"1:4": true, "1:8": true, "4:1": true, "8:1": true,
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
	ShowCosts        bool
	OutputMode       OutputMode
	JQ               string
	Count            int
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
		case "--costs":
			opts.ShowCosts = true
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
			case "minimal", "high":
				opts.ThinkingLevel = strings.ToLower(args[i])
			default:
				return opts, fmt.Errorf("invalid --thinking value %q (use minimal, high)", args[i])
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
		default:
			if strings.HasPrefix(a, "-") {
				return opts, fmt.Errorf("unknown option: %s", a)
			}
			promptParts = append(promptParts, a)
		}
	}

	if !opts.ShowCosts {
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
	fmt.Print(`imagen – generate images via Gemini or Grok

Usage:
  imagen [options] <prompt...>

Options:
  -h, --help            Show this help message
  -o, --output NAME     Output file base name (default: imagen-<timestamp>)
  -s, --size SIZE       Image size: 512, 1K, 2K, 4K (default: 1K)
  -a, --aspect RATIO    Aspect ratio (e.g. 16:9, 4:3, 1:1)
  -m, --model MODEL     Model or alias: flash, pro, grok (default: google/flash)
  -n, --count N         Number of images to generate (default: 1)
  -d, --dir DIR         Output directory (default: current directory)
  -r, --ref FILE        Reference image (can be repeated)
  -t, --transparent     Remove background (pure Go, no external tools)
      --seed N          Random seed for reproducible generation
      --person MODE     Person generation: ALL, ADULT, NONE
      --thinking LEVEL  Thinking level: minimal, high (flash only)
      --api-key KEY     API key (or set GEMINI_API_KEY / XAI_API_KEY)
      --costs           Show accumulated cost summary
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
  imagen -r style.png "apply this style to a forest"
  imagen --json "logo for a coffee shop" | jq .files
  imagen --costs
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
