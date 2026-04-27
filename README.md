# imagen

Multi-provider image generation CLI + Go library.

Supports Google Gemini, xAI Grok, OpenAI (gpt-image-2 / gpt-image-1.5 / DALL-E), and ChatGPT/Codex OAuth.

## Install

Requires Go 1.25+.

```bash
go install github.com/leechael/imagen/cmd/imagen@latest
```

Or build from source:

```bash
make build   # produces bin/imagen
```

## API Keys

Key resolution per provider (checked in order):

1. `--api-key` flag
2. Environment variable
3. `.env` in the current directory
4. `.env` next to the executable's parent directory (`<exe-dir>/../.env`)
5. `~/.imagen/.env`

| Provider | Env Var(s) |
|---|---|
| **google** | `GEMINI_API_KEY` |
| **xai** | `XAI_API_KEY`, `GROK_API_KEY` |
| **openai** | `OPENAI_API_KEY` |
| **codex** | `CODEX_ACCESS_TOKEN`, `CHATGPT_ACCESS_TOKEN` |

```bash
export GEMINI_API_KEY=your_key_here
export XAI_API_KEY=your_key_here
export OPENAI_API_KEY=your_key_here
export CODEX_ACCESS_TOKEN=your_chatgpt_token

# Or store in a dotenv file
mkdir -p ~/.imagen
cat >> ~/.imagen/.env <<EOF
GEMINI_API_KEY=your_key_here
XAI_API_KEY=your_key_here
OPENAI_API_KEY=your_key_here
CODEX_ACCESS_TOKEN=your_chatgpt_token
EOF
```

### Custom Base URL

All providers support overriding the API endpoint via environment variable:

| Provider | Env Var | Default |
|---|---|---|
| **xai** | `XAI_BASE_URL` | `https://api.x.ai/v1` |
| **openai** | `OPENAI_BASE_URL` | `https://api.openai.com/v1` |
| **codex** | `CODEX_BASE_URL` | `https://chatgpt.com/backend-api/codex` |

### Obtaining a Codex Access Token

The Codex provider uses your ChatGPT session — no OpenAI API key or org certification required.

1. Log in to https://chatgpt.com in your browser
2. Open DevTools (F12) → Network tab
3. Filter for `/backend-api/` requests
4. Copy the `Authorization: Bearer <token>` header value
5. Set as `CODEX_ACCESS_TOKEN`

## Model Selection

Models are specified as `provider/model`. Bare aliases are accepted for convenience.

```bash
imagen -m google/flash "..."        # default
imagen -m google/pro "..."
imagen -m xai/grok "..."
imagen -m openai/gpt-image-2 "..."
imagen -m codex-2 "..."
```

### Aliases

| Alias | Provider | Resolves to |
|---|---|---|
| `flash`, `nb2` | google | `gemini-3.1-flash-image-preview` |
| `pro`, `nb-pro` | google | `gemini-3-pro-image-preview` |
| `grok`, `grok-imagine` | xai | `grok-imagine-image` |
| `oai-2` | openai | `gpt-image-2-2026-04-21` |
| `oai-15` | openai | `gpt-image-1.5` |
| `codex-2` | codex | `gpt-image-2-medium` |
| `codex-2-low` | codex | `gpt-image-2-low` |
| `codex-2-high` | codex | `gpt-image-2-high` |

## Usage Examples

```bash
# Basic prompt (defaults to google/flash at 1K)
imagen "a serene mountain lake at dawn"

# Size and aspect ratio
imagen -s 2K -a 16:9 "cinematic sunset over mountains"

# OpenAI GPT Image 2
imagen -m openai/gpt-image-2 "a cute baby sea otter"

# Codex via ChatGPT (no API key needed)
imagen -m codex-2 "a futuristic cityscape"
imagen -m codex-2-high "highly detailed portrait"

# Transparent background (native for OpenAI/Codex GPT image models)
imagen -m openai/gpt-image-2 -t "logo for a coffee shop"
imagen -m codex-2 -t "mascot with transparent background"

# Reference images (style transfer / inpainting)
imagen -m openai/gpt-image-2 -r style.png "apply this art style to a forest scene"
imagen -r base.png -r style.png "blend these two styles"

# Batch generation
imagen -n 4 "logo concept for a coffee shop"

# JSON output piped to jq
imagen --json "product mockup" | jq '.files'

# Read prompt from stdin
echo "a cat in a spacesuit" | imagen -s 2K -o result
pbpaste | imagen -a 4:3

# Provider-specific help
imagen help openai
imagen help codex
imagen help grok
```

## Options

| Option | Default | Description |
|---|---|---|
| `-o, --output` | `imagen-<timestamp>` | Output filename (no extension) |
| `-s, --size` | `1K` | Image size: `512`, `1K`, `2K`, `4K` |
| `-a, --aspect` | model default | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `4:5`, `5:4`, `21:9`, `1:4`, `1:8`, `4:1`, `8:1`, `2:1`, `1:2`, `20:9`, `19.5:9` |
| `-m, --model` | `google/flash` | Model spec: `provider/model`, `provider/alias`, or bare alias |
| `-n, --count` | `1` | Number of images to generate |
| `-d, --dir` | current directory | Output directory |
| `-r, --ref` | - | Reference image path (can be repeated) |
| `-t, --transparent` | - | Transparent background (native API support or green-screen fallback) |
| `--seed` | random | Fixed seed for reproducible generation |
| `--person` | model default | Person generation: `ALL`, `ADULT`, `NONE` |
| `--thinking` | model default | Thinking level: `minimal`, `low`, `medium`, `high` (flash model only) |
| `--quality` | model default | Output quality: `low`, `medium`, `high` (grok, codex) |
| `--api-key` | - | API key override |
| `--costs` | - | Show accumulated cost summary |
| `--json` | - | JSON output to stdout |
| `--plain` | - | Plain output — filenames only |
| `--jq EXPR` | - | Filter JSON output (requires `--json`) |

## Provider Capability Matrix

Unsupported options emit a warning and are ignored (or fall back, for size).

| Option | google | xai | openai | codex |
|---|---|---|---|---|
| `--aspect` | yes | yes | no | no |
| `--seed` | yes | no | no | no |
| `--person` | yes | no | no | no |
| `--thinking` | flash only | no | no | no |
| `--quality` | no | yes | no | yes |
| `--transparent` | green-screen | green-screen | native API | native API |
| `-r / --ref` (max) | 10 | 5 | 16 (GPT image) / 1 (dall-e-2) | no |
| `-s 4K` | yes | fallback to 2K | fallback to 1536x1024 | fallback to 1536x1024 |
| `-n` (batch) | yes (sequential) | yes (native, max 10) | yes (dall-e-3: 1) | no |

### Size Mapping

Not all providers support the abstract sizes directly. Mappings:

| CLI size | google | xai | openai / codex |
|---|---|---|---|
| `512` | 1K (remapped) | 1K (remapped) | 1024x1024 (remapped) |
| `1K` | 1024x1024 | 1024x1024 | 1024x1024 / aspect-based |
| `2K` | 2048x2048 | 1024x1024 (fallback) | 1024x1024 or 1536x1024 |
| `4K` | 4096x4096 | 1024x1024 (fallback) | 1536x1024 (fallback) |

## Transparent Mode

Behavior varies by provider:

**Native API support** (OpenAI GPT image models, Codex):
- Sets `background=transparent` directly via API
- Output is PNG with alpha channel
- No post-processing needed

**Green-screen fallback** (Google, xAI, OpenAI DALL-E):
- Appends a green-screen instruction to the prompt
- Post-processes each output with a pure-Go colorkey + despill + trim pipeline
- No external tools (ffmpeg, imagemagick) required

```bash
# Native transparency (OpenAI / Codex)
imagen -m openai/gpt-image-2 -t "logo for a coffee shop"

# Green-screen fallback (Google / xAI)
imagen -t -o mascot "robot mascot"
```

## Cost Tracking

Each generation is logged to `~/.imagen/costs.json`.

```bash
imagen --costs                 # human-readable summary
imagen --costs --json          # machine-readable, per-provider breakdown
```

On first run, if `~/.nano-banana/costs.json` exists and `~/.imagen/costs.json` does not, the log is migrated automatically.

## Library Usage

Import the provider package directly without using the CLI.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/leechael/imagen/provider"
    _ "github.com/leechael/imagen/provider/codex"
    _ "github.com/leechael/imagen/provider/google"
    _ "github.com/leechael/imagen/provider/openai"
    _ "github.com/leechael/imagen/provider/xai"
)

func main() {
    providerName, modelID, err := provider.ParseModel("openai/gpt-image-2")
    if err != nil {
        panic(err)
    }

    p, err := provider.Get(providerName, os.Getenv("OPENAI_API_KEY"))
    if err != nil {
        panic(err)
    }

    res, err := p.Generate(context.Background(), provider.GenerateRequest{
        Model:  modelID,
        Prompt: "a serene mountain lake at dawn",
        Size:   "2K",
        N:      1,
    })
    if err != nil {
        panic(err)
    }

    for i, img := range res.Images {
        os.WriteFile(fmt.Sprintf("out-%d.png", i), img.Data, 0o644)
    }
}
```

Each provider package registers itself in `init()`. The blank import is the only requirement.

### Provider interface

```go
type Provider interface {
    Name() string
    Capabilities(model string) Capability
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}
```

- `GenerateRequest` — prompt, model, size, aspect ratio, references, seed, person, thinking level, quality, background, and count (N).
- `GenerateResult` — generated images (raw bytes + MIME type), model ID, cost estimate, token counts, and any runtime warnings.
- `Capability` — per-model flags describing which request fields the provider accepts, supported sizes, max references, and valid aspect ratios.

Use `provider.ParseModel("provider/alias")` to resolve a user-facing model spec to `(providerName, canonicalModelID)` before calling `provider.Get`.

## Development

```bash
make test    # run tests
make build   # build binary to bin/imagen
make lint    # vet + format check
make ci      # full CI pipeline
```
