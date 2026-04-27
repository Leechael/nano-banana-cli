# imagen

Multi-provider image generation — CLI + Go library. Supports Google Gemini and xAI Grok image models.

## Install (CLI)

Requires Go 1.25+.

```bash
go install github.com/leechael/imagen/cmd/imagen@latest
```

Or build from source:

```bash
make build   # produces bin/imagen
```

## API Keys

Key resolution per provider:

**google** models — checked in order:
1. `--api-key` flag
2. `GEMINI_API_KEY` environment variable
3. `.env` in the current directory
4. `.env` next to the executable's parent directory (`<exe-dir>/../.env`)
5. `~/.imagen/.env`

**xai** models — same order, but checks `XAI_API_KEY` then `GROK_API_KEY` at each dotenv path.

```bash
export GEMINI_API_KEY=your_key_here
export XAI_API_KEY=your_key_here

# Or store in a dotenv file
mkdir -p ~/.imagen
echo "GEMINI_API_KEY=your_key_here" >> ~/.imagen/.env
echo "XAI_API_KEY=your_key_here"    >> ~/.imagen/.env
```

## Model Selection

Models are specified as `provider/model`. Bare aliases are accepted for back-compat.

```bash
imagen --model google/flash "..."        # default
imagen --model google/pro "..."
imagen --model xai/grok "..."
imagen --model xai/grok-imagine-image "..."   # full model ID
```

### Aliases

| Alias | Resolves to |
|---|---|
| `google/flash`, `flash`, `nb2` | `gemini-3.1-flash-image-preview` |
| `google/pro`, `pro`, `nb-pro` | `gemini-3-pro-image-preview` |
| `xai/grok`, `grok`, `grok-imagine` | `grok-imagine-image` |

## Usage Examples

```bash
# Basic prompt (defaults to google/flash at 1K)
imagen "a serene mountain lake at dawn"

# Size and aspect ratio
imagen -s 2K -a 16:9 "cinematic sunset over mountains"

# Reference images (style transfer or inpainting)
imagen -r style.png "apply this art style to a forest scene"
imagen -r base.png -r style.png "blend these two styles"

# Transparent background (green-screen removal)
imagen -t -o mascot "robot mascot on green background"

# Batch — generate 4 variations
imagen -n 4 "logo concept for a coffee shop"

# JSON output piped to jq
imagen --json "product mockup" | jq '.files'

# xAI Grok
imagen --model xai/grok -s 2K "futuristic cityscape"

# Read prompt from stdin
echo "a cat in a spacesuit" | imagen -s 2K -o result
pbpaste | imagen -a 4:3
```

## Options

| Option | Default | Description |
|---|---|---|
| `-o, --output` | `imagen-<timestamp>` | Output filename (no extension) |
| `-s, --size` | `1K` | Image size: `512`, `1K`, `2K`, `4K` |
| `-a, --aspect` | model default | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `4:5`, `5:4`, `21:9`, `1:4`, `1:8`, `4:1`, `8:1` |
| `-m, --model` | `google/flash` | Model spec: `provider/model`, `provider/alias`, or bare alias |
| `-n, --count` | `1` | Number of images to generate |
| `-d, --dir` | current directory | Output directory |
| `-r, --ref` | - | Reference image path (can be repeated) |
| `-t, --transparent` | - | Remove background via green-screen keyer (pure Go) |
| `--seed` | random | Fixed seed for reproducible generation |
| `--person` | model default | Person generation: `ALL`, `ADULT`, `NONE` |
| `--thinking` | model default | Thinking level: `minimal`, `high` (flash model only) |
| `--api-key` | - | API key override (see API Keys section for env vars) |
| `--costs` | - | Show accumulated cost summary |
| `--json` | - | JSON output to stdout |
| `--plain` | - | Plain output — filenames only |
| `--jq EXPR` | - | Filter JSON output (requires `--json`) |

## Provider Capability Matrix

Unsupported options emit a warning and are ignored (or fall back, for size).

| Option | google | xai |
|---|---|---|
| `--aspect` | yes | yes |
| `--seed` | yes | warn, ignored |
| `--person` | yes | warn, ignored |
| `--thinking` | flash model only | warn, ignored |
| `-r / --ref` (max) | 10 | 5 |
| `-s 4K` | yes | warn, falls back to `2K` |
| `-n` (batch) | yes (sequential) | yes (native) |

## Transparent Mode

`-t` appends a green-screen instruction to the prompt, then post-processes each output image with a pure-Go colorkey + despill + trim pipeline. No external tools (ffmpeg, imagemagick, etc.) are required. Output is written as PNG with an alpha channel. Works with any provider.

```bash
imagen -t -o mascot "robot mascot"
```

## Cost Tracking

Each generation is logged to `~/.imagen/costs.json`.

```bash
imagen --costs                 # human-readable summary
imagen --costs --json          # machine-readable, per-provider breakdown
```

On first run, if `~/.nano-banana/costs.json` exists and `~/.imagen/costs.json` does not, the log is migrated automatically and a notice is printed to stderr.

## Library Usage

Another project can import the provider package directly without using the CLI.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/leechael/imagen/provider"
    _ "github.com/leechael/imagen/provider/google" // register google via init()
    _ "github.com/leechael/imagen/provider/xai"    // register xai via init()
)

func main() {
    providerName, modelID, err := provider.ParseModel("xai/grok")
    if err != nil {
        panic(err)
    }

    p, err := provider.Get(providerName, os.Getenv("XAI_API_KEY"))
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
        os.WriteFile(fmt.Sprintf("out-%d.jpg", i), img.Data, 0o644)
    }
}
```

Each provider package registers itself in `init()`. The blank import is the only requirement; no manual factory calls are needed.

### Provider interface

```go
type Provider interface {
    Name() string
    Capabilities(model string) Capability
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
}
```

- `GenerateRequest` — prompt, model, size, aspect ratio, references, seed, person, thinking level, and count (N).
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
