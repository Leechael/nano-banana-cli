package cli

import (
	"fmt"
	"os"
	"strings"
)

// PrintHelpTopic prints provider-specific help documentation.
func PrintHelpTopic(topic string) {
	switch strings.ToLower(topic) {
	case "google", "gemini":
		printGoogleHelp()
	case "grok", "xai":
		printXAIHelp()
	case "openai", "gpt-image":
		printOpenAIHelp()
	case "codex", "chatgpt":
		printCodexHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown help topic: %q\n\n", topic)
		PrintHelp()
		os.Exit(1)
	}
}

func printGoogleHelp() {
	fmt.Print(`# Google Gemini Image Generation

## Authentication

Environment variable:

    GEMINI_API_KEY    API key for Google Gemini

Also reads from .env files in the current directory, the binary directory,
or ~/.imagen/.env.

## Models

    nb       gemini-2.5-flash-image           (Nano Banana)
    flash    gemini-3.1-flash-image-preview   (Nano Banana 2)
    nb2      gemini-3.1-flash-image-preview   (Nano Banana 2, same as flash)
    pro      gemini-3-pro-image-preview       (Nano Banana Pro)
    nb-pro   gemini-3-pro-image-preview       (Nano Banana Pro, same as pro)

## Supported Parameters

    --size SIZE
        nb:           1K only
        flash, pro:   512, 1K, 2K, 4K  (512 is remapped to 1K by the API)

    --aspect RATIO
        All models:   1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 4:5, 5:4, 21:9
        flash only:   1:4, 4:1, 1:8, 8:1

    --seed N
        Integer seed for reproducible generation.
        Not supported by nb (gemini-2.5-flash-image).

    --person MODE
        Controls whether generated images may include people.
        ALL    people of any age allowed
        ADULT  only adults allowed (API default)
        NONE   no people generated

    --thinking LEVEL
        minimal, low, medium, high   (flash / nb2 only)

    --ref FILE
        Reference image. Can be repeated.
        nb:          up to 3 images
        flash, pro:  up to 10 images

    -n, --count N
        Number of images to generate.

## Examples

    imagen -m nb "a cat in a spacesuit"
    imagen -m google/flash "a cat in a spacesuit"
    imagen -m pro --seed 42 "futuristic city"
    imagen -m flash --thinking high "complex architectural diagram"
    imagen -s 2K -a 16:9 --person ADULT "portrait of an astronaut"
`)
}

func printXAIHelp() {
	fmt.Print(`# xAI Grok Image Generation

## Authentication

Environment variables (checked in order):

    XAI_API_KEY     Primary API key for xAI Grok
    GROK_API_KEY    Fallback API key (also accepted)

Also reads from .env files in the current directory, the binary directory,
or ~/.imagen/.env.

## Models

    grok    grok-imagine-image

## Supported Parameters

    --size SIZE
        1K, 2K  (default: 1K)
        512 falls back to 1K, 4K falls back to 2K.

    --aspect RATIO
        Standard:  1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 2:1, 1:2
        Wide/tall: 20:9, 19.5:9
        Auto:      auto  (match aspect ratio of the first reference image)

    --quality LEVEL
        low     fast generation, lower latency
        medium  balanced (API default)
        high    better detail, lighting, shadows, and text rendering

    --ref FILE
        Reference image (image editing mode). Can be repeated; up to 5 images.

    -n, --count N
        Number of images to generate (max 10 per API call).

## Examples

    imagen -m xai/grok "a futuristic city"
    imagen -m grok -s 2K -a 16:9 "sunset over mountains"
    imagen -m grok -a 2:1 "cinematic landscape"
    imagen -m grok -r photo.jpg "turn this into a watercolor painting"
`)
}

func printCodexHelp() {
	fmt.Print(`# OpenAI Codex / ChatGPT Image Generation

## Authentication

Environment variables (checked in order):

    CODEX_ACCESS_TOKEN     ChatGPT/Codex access token (from browser devtools)
    CHATGPT_ACCESS_TOKEN   Fallback token name

Also reads from .env files in the current directory, the binary directory,
or ~/.imagen/.env.

To obtain a token:
  1. Log in to https://chatgpt.com in your browser
  2. Open browser DevTools (F12) → Application/Storage → Cookies
  3. Find the __Secure-next-auth.session-token or similar auth cookie
  4. Or use the Network tab, find a request to /backend-api/, copy the
     Authorization: Bearer <token> header value

## Models

    codex-2        gpt-image-2-medium   (default, balanced)
    codex-2-low    gpt-image-2-low      (fast, lowest cost)
    codex-2-medium gpt-image-2-medium   (balanced)
    codex-2-high   gpt-image-2-high     (highest fidelity)

All tiers use gpt-image-2 under the hood; the alias selects quality level.

## Supported Parameters

    --size SIZE
        512, 1K, 2K, 4K  (default: 1K)
        Native sizes: 1024x1024, 1536x1024, 1024x1536.
        Abstract sizes are mapped to the closest native size.

    --aspect RATIO
        Used to select landscape (1536x1024) or portrait (1024x1536).
        Not a native parameter; ignored if a specific size is given.

    -t, --transparent
        Sets background=transparent.

    --quality LEVEL
        low, medium, high
        Each tier alias already sets a quality; this flag overrides it.

    -n, --count N
        Only 1 image per request is supported by the Codex API.

## Unsupported Parameters

The following CLI flags are accepted but ignored:

    --seed, --person, --thinking, --ref

## Examples

    imagen -m codex-2 "a cute baby sea otter"
    imagen -m codex/codex-2-high "a futuristic city"
    imagen -m codex-2 -s 1K -a 16:9 "sunset over mountains"
    imagen -m codex-2 -t "logo for a coffee shop"
`)
}

func printOpenAIHelp() {
	fmt.Print(`# OpenAI Image Generation

## Authentication

Environment variable:

    OPENAI_API_KEY    API key for OpenAI

Also reads from .env files in the current directory, the binary directory,
or ~/.imagen/.env.

## Models

    gpt-image-2              gpt-image-2-2026-04-21  (alias: oai-2)
    gpt-image-1.5            gpt-image-1.5           (alias: oai-15)
    gpt-image-1              gpt-image-1
    gpt-image-1-mini         gpt-image-1-mini
    chatgpt-image-latest     chatgpt-image-latest
    dall-e-3                 dall-e-3
    dall-e-2                 dall-e-2

## Supported Parameters

    --size SIZE
        512, 1K, 2K, 4K  (default: 1K)
        OpenAI native sizes are auto, 1024x1024, 1536x1024, 1024x1536.
        Abstract sizes are mapped to the closest native size:
          512 -> 1024x1024
          1K  -> 1024x1024 (or landscape/portrait based on --aspect)
          2K  -> 1024x1024 (or landscape/portrait)
          4K  -> 1536x1024

    --aspect RATIO
        Used to select landscape (1536x1024) or portrait (1024x1536).
        Not a native OpenAI parameter; ignored if a specific size is given.

    --ref FILE
        Reference image for editing. Can be repeated.
        GPT image models: up to 16 images.
        dall-e-2: up to 1 image.

    -n, --count N
        Number of images to generate.
        dall-e-3 only supports n=1; higher counts loop.

    -t, --transparent
        Native transparency support for GPT image models.
        Sets background=transparent and output_format=png.
        No external keying tools needed.

## Unsupported Parameters

The following CLI flags are accepted but ignored by OpenAI providers:

    --seed, --person, --thinking

## Examples

    imagen -m openai/gpt-image-2 "a cute baby sea otter"
    imagen -m oai-2 "a futuristic city"
    imagen -m openai/gpt-image-1.5 -s 1K -a 16:9 "sunset over mountains"
    imagen -m openai/gpt-image-2 -t "logo for a coffee shop"
    imagen -m openai/gpt-image-2 -r photo.jpg "turn this into a watercolor painting"
    imagen -m openai/dall-e-3 "a highly detailed digital art of a dragon"
`)
}
