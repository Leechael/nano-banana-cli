# imagen — Multi-Provider AI Image Generation CLI & Go Library

**imagen** is a fast, lightweight command-line tool for generating and editing images with multiple AI providers. Use it from your terminal to create images with [Google Gemini](https://ai.google.dev/), [xAI Grok](https://x.ai/), [OpenAI GPT Image 2](https://platform.openai.com/), or [ChatGPT/Codex](https://chatgpt.com/) — no OpenAI org certification required for Codex.

- **Generate images** from text prompts in one command
- **Edit images** with reference images (style transfer, inpainting)
- **Transparent backgrounds** — native API support for OpenAI/Codex, green-screen fallback for others
- **Batch generation** across all supported providers
- **Built-in cost tracking** per generation with per-provider breakdowns
- **Go library** for embedding image generation into your own applications

```bash
# Generate an image with OpenAI GPT Image 2
imagen -m openai/gpt-image-2 "a serene mountain lake at dawn"

# Generate via ChatGPT/Codex (no API key needed)
imagen -m codex-2 "a cute baby sea otter"

# Create transparent PNGs for design work
imagen -m codex-2 -t "logo for a coffee shop"
```

<!-- omit in toc -->
## Table of Contents

- [Features](#features)
- [Supported Providers & Models](#supported-providers--models)
- [Installation](#installation)
- [Authentication](#authentication)
- [Usage Examples](#usage-examples)
- [CLI Options](#cli-options)
- [Provider Capabilities](#provider-capabilities)
- [Transparent Background](#transparent-background)
- [Cost Tracking](#cost-tracking)
- [Go Library](#go-library)
- [Development](#development)

## Features

| Feature | Description |
|---|---|
| **Multi-provider** | Switch between Google Gemini, xAI Grok, OpenAI, and ChatGPT/Codex without changing your workflow |
| **Transparent backgrounds** | Native `background=transparent` for OpenAI GPT Image 2 and Codex; pure-Go green-screen keying for Google and xAI |
| **Image editing** | Upload reference images for style transfer and inpainting (up to 16 images with OpenAI GPT Image 2) |
| **Batch generation** | Generate multiple variations in a single command |
| **Cost tracking** | Every generation logged to `~/.imagen/costs.json` with per-provider summaries |
| **Flexible sizing** | Abstract sizes (`512`, `1K`, `2K`, `4K`) automatically mapped to each provider's native resolutions |
| **Go library** | Import as a package and register providers via blank imports |

## Supported Providers & Models

Specify models as `provider/model` or use bare aliases:

```bash
imagen -m google/flash "..."          # default
imagen -m openai/gpt-image-2 "..."
imagen -m codex-2 "..."
```

### Provider Aliases

| Alias | Provider | Model ID |
|---|---|---|
| `flash`, `nb2` | Google | `gemini-3.1-flash-image-preview` |
| `pro`, `nb-pro` | Google | `gemini-3-pro-image-preview` |
| `grok`, `grok-imagine` | xAI | `grok-imagine-image` |
| `oai-2` | OpenAI | `gpt-image-2-2026-04-21` |
| `oai-15` | OpenAI | `gpt-image-1.5` |
| `codex-2` | Codex | `gpt-image-2-medium` |
| `codex-2-low` | Codex | `gpt-image-2-low` |
| `codex-2-high` | Codex | `gpt-image-2-high` |

### Full Model List

- **Google**: `gemini-2.5-flash-image`, `gemini-3.1-flash-image-preview`, `gemini-3-pro-image-preview`
- **xAI**: `grok-imagine-image`
- **OpenAI**: `gpt-image-2`, `gpt-image-1.5`, `gpt-image-1`, `gpt-image-1-mini`, `dall-e-3`, `dall-e-2`
- **Codex**: `gpt-image-2` (via ChatGPT/Codex OAuth, quality tiers: `low`, `medium`, `high`)

## Installation

Requires Go 1.25+.

```bash
# Install the CLI
go install github.com/leechael/imagen/cmd/imagen@latest

# Or clone and build from source
git clone https://github.com/leechael/imagen.git
cd imagen
make build   # outputs bin/imagen
```

## Authentication

imagen resolves API keys in this order:

1. `--api-key` flag
2. Environment variable
3. `.env` in the current directory
4. `.env` next to the executable (`<exe-dir>/../.env`)
5. `~/.imagen/.env`

### Required Environment Variables

| Provider | Variable |
|---|---|
| Google | `GEMINI_API_KEY` |
| xAI | `XAI_API_KEY` or `GROK_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Codex | `CODEX_ACCESS_TOKEN` or `CHATGPT_ACCESS_TOKEN` |

```bash
export GEMINI_API_KEY=your_key_here
export OPENAI_API_KEY=your_key_here
export CODEX_ACCESS_TOKEN=your_chatgpt_token
```

### Custom Base URLs

Override the default API endpoint for any provider:

| Provider | Variable | Default |
|---|---|---|
| xAI | `XAI_BASE_URL` | `https://api.x.ai/v1` |
| OpenAI | `OPENAI_BASE_URL` | `https://api.openai.com/v1` |
| Codex | `CODEX_BASE_URL` | `https://chatgpt.com/backend-api/codex` |

### Getting a ChatGPT/Codex Access Token

The Codex provider uses your existing ChatGPT browser session — no OpenAI API key or org certification required.

1. Log in to [chatgpt.com](https://chatgpt.com)
2. Open DevTools (F12) → **Network** tab
3. Filter for `/backend-api/` requests
4. Copy the `Authorization: Bearer <token>` header value
5. Set it as `CODEX_ACCESS_TOKEN`

## Usage Examples

### Basic Image Generation

```bash
# Default provider (google/flash, 1K)
imagen "a serene mountain lake at dawn"

# OpenAI GPT Image 2
imagen -m openai/gpt-image-2 "a cute baby sea otter"

# Codex via ChatGPT — no API key needed
imagen -m codex-2 "a futuristic cityscape"
imagen -m codex-2-high "highly detailed portrait"

# xAI Grok
imagen -m xai/grok "a futuristic cityscape"
```

### Size and Aspect Ratio

```bash
imagen -s 2K -a 16:9 "cinematic sunset over mountains"
imagen -s 4K "ultra-detailed macro photography of a bee"
```

### Transparent Background

```bash
# Native API transparency (OpenAI / Codex)
imagen -m openai/gpt-image-2 -t "logo for a coffee shop"
imagen -m codex-2 -t "mascot with transparent background"

# Green-screen fallback (Google / xAI)
imagen -t -o mascot "robot mascot"
```

### Image Editing & Style Transfer

```bash
# Single reference image
imagen -m openai/gpt-image-2 -r style.png "apply this art style to a forest scene"

# Multiple references
imagen -r base.png -r style.png "blend these two styles"
```

### Batch Generation

```bash
imagen -n 4 "logo concept for a coffee shop"
```

### JSON Output & Pipelines

```bash
imagen --json "product mockup" | jq '.files'

# Read prompt from stdin
echo "a cat in a spacesuit" | imagen -s 2K -o result
pbpaste | imagen -a 4:3    # macOS only
```

### Provider-Specific Help

```bash
imagen help openai
imagen help codex
imagen help grok
imagen help google
```

## CLI Options

| Option | Default | Description |
|---|---|---|
| `-m, --model` | `google/flash` | Model: `provider/model`, `provider/alias`, or bare alias |
| `-s, --size` | `1K` | Image size: `512`, `1K`, `2K`, `4K` |
| `-a, --aspect` | model default | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `4:5`, `5:4`, `21:9`, `1:4`, `1:8`, `4:1`, `8:1`, `2:1`, `1:2`, `20:9`, `19.5:9` |
| `-o, --output` | `imagen-<timestamp>` | Output filename (no extension) |
| `-d, --dir` | current directory | Output directory |
| `-n, --count` | `1` | Number of images to generate |
| `-r, --ref` | — | Reference image path (repeatable) |
| `-t, --transparent` | — | Transparent background (native API or green screen fallback) |
| `--seed` | random | Fixed seed for reproducible generation |
| `--person` | model default | Person generation: `ALL`, `ADULT`, `NONE` |
| `--thinking` | model default | Thinking level: `minimal`, `low`, `medium`, `high` (flash only) |
| `--quality` | model default | Output quality: `low`, `medium`, `high` (grok, codex) |
| `--api-key` | — | API key override |
| `--costs` | — | Show accumulated cost summary |
| `--json` | — | JSON output to stdout |
| `--plain` | — | Plain output — filenames only |
| `--jq EXPR` | — | Filter JSON output (requires `--json`) |

## Provider Capabilities

Unsupported options emit a warning and are ignored (or fall back for size).

| Option | Google | xAI | OpenAI | Codex |
|---|---|---|---|---|
| `--aspect` | yes | yes | no | no |
| `--seed` | yes | no | no | no |
| `--person` | yes | no | no | no |
| `--thinking` | flash only | no | no | no |
| `--quality` | no | yes | no | yes |
| `--transparent` | green screen | green screen | native API | native API |
| `-r / --ref` (max) | 10 | 5 | 16 (GPT image) / 1 (dall-e-2) | no |
| `-s 4K` | yes | fallback to 2K | fallback to 1536×1024 | fallback to 1536×1024 |
| `-n` (batch) | yes (sequential) | yes (native, max 10) | yes (dall-e-3: 1) | no |

### Size Mapping

| CLI size | Google | xAI | OpenAI / Codex |
|---|---|---|---|
| `512` | 1K (remapped) | 1K (remapped) | 1024×1024 (remapped) |
| `1K` | 1024×1024 | 1024×1024 | 1024×1024 / aspect-based |
| `2K` | 2048×2048 | 1024×1024 (fallback) | 1024×1024 or 1536×1024 |
| `4K` | 4096×4096 | 1024×1024 (fallback) | 1536×1024 (fallback) |

## Transparent Background

imagen handles transparency differently depending on the provider:

**Native API support** (OpenAI GPT Image 2, Codex):
- Sets `background=transparent` directly via the API
- Output is PNG with a real alpha channel
- No post-processing

**Green screen fallback** (Google, xAI, OpenAI DALL-E):
- Appends a green screen instruction to the prompt
- Post-processes with a pure-Go colorkey + despill + trim pipeline
- No external tools (ffmpeg, ImageMagick) required

## Cost Tracking

Every generation is logged to `~/.imagen/costs.json`.

```bash
imagen --costs                 # human-readable summary
imagen --costs --json          # machine-readable, per-provider breakdown
```

On first run, if `~/.nano-banana/costs.json` exists and `~/.imagen/costs.json` does not, the log is migrated automatically.

## Go Library

Import the provider package directly to build image generation into your own Go applications.

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

Each provider package registers itself via `init()`. You only need a blank import.

### Provider Interface

```go
type Provider interface {
    Name() string
    Capabilities(model string) Capability
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}
```

- `GenerateRequest` — prompt, model, size, aspect ratio, references, seed, person, thinking level, quality, background, and count (N).
- `GenerateResult` — generated images (raw bytes + MIME type), model ID, cost estimate, token counts, and runtime warnings.
- `Capability` — flags for supported request fields, sizes, max references, and aspect ratios.

Use `provider.ParseModel("provider/alias")` to resolve a user-facing model spec to `(providerName, canonicalModelID)` before calling `provider.Get`.

## Development

```bash
make test    # run tests
make build   # build binary to bin/imagen
make lint    # vet + format check
make ci      # run full CI pipeline (requires pre-commit)
```
