---
name: nano-banana-image
description: Generate AI images using nano-banana CLI (Gemini Flash/Pro). Use this skill when asked to generate images, create sprites, make assets, produce artwork, or any image generation task. Supports multi-resolution (512–4K), aspect ratios, reference images for style transfer, and transparent background removal.
compatibility: Requires nano-banana binary in PATH and GEMINI_API_KEY.
---

# nano-banana-image

AI image generation CLI powered by Gemini image models.

## Prerequisites

- `nano-banana` binary in PATH
- `GEMINI_API_KEY` environment variable

If not set up, see `references/installation.md`.

## Quick Reference

```bash
# Basic generation
nano-banana "a cat in a spacesuit"

# High-res widescreen
nano-banana "cinematic landscape" -s 2K -a 16:9

# Transparent asset (no external tools needed)
nano-banana "robot mascot" -t -o mascot

# Style transfer from reference image
nano-banana "apply this style to a forest" -r style.png -o forest

# Pro model for highest quality
nano-banana "detailed portrait" -m pro -s 2K
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `-o, --output` | `nano-gen-{ts}` | Output filename (no extension) |
| `-s, --size` | `1K` | Image size: `512`, `1K`, `2K`, `4K` |
| `-a, --aspect` | model default | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, etc. |
| `-m, --model` | `flash` | Model: `flash`, `pro` |
| `-d, --dir` | current directory | Output directory |
| `-r, --ref` | - | Reference image (repeatable) |
| `-t, --transparent` | - | Generate with transparent background |
| `--json` | - | JSON output to stdout |
| `--plain` | - | Plain output (filenames only) |

## Models

| Alias | Model | Use case |
|-------|-------|----------|
| `flash` | Gemini 3.1 Flash | Default. Fast, cheap |
| `pro` | Gemini 3 Pro | Highest quality |

## Transparent Mode

`-t` generates on a green screen, then removes the background using built-in colorkey + despill + trim. Pure Go, no ffmpeg or ImageMagick needed.

## Cost Tracking

```bash
nano-banana --costs
nano-banana --costs --json
```

## Detailed Installation

See `references/installation.md`.
