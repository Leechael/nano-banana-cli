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

    flash    gemini-3.1-flash-image-preview
    pro      gemini-3-pro-image-preview

## Supported Parameters

    --size SIZE
        512, 1K, 2K, 4K  (default: 1K)
        512 is remapped to 1K by the API.

    --aspect RATIO
        All models:  1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 4:5, 5:4, 21:9
        Flash only:  1:4, 4:1, 1:8, 8:1

    --seed N
        Integer seed for reproducible generation.

    --person MODE
        ALL    allow adults and children
        ADULT  allow adults only (default)
        NONE   block all people

    --thinking LEVEL
        minimal, low, medium, high   (flash only)

    --ref FILE
        Reference image. Can be repeated; up to 10 images.

    -n, --count N
        Number of images to generate.

## Examples

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

    --ref FILE
        Reference image (image editing mode). Can be repeated; up to 5 images.

    -n, --count N
        Number of images to generate (max 10 per API call).

## API Parameters (not yet exposed as CLI flags)

    quality    Controls output quality: low, medium, high
               low = fast generation; high = better detail, lighting, text.

## Unsupported Parameters

The following CLI flags are accepted but ignored by the Grok provider:

    --seed, --person, --thinking

## Examples

    imagen -m xai/grok "a futuristic city"
    imagen -m grok -s 2K -a 16:9 "sunset over mountains"
    imagen -m grok -a 2:1 "cinematic landscape"
    imagen -m grok -r photo.jpg "turn this into a watercolor painting"
`)
}
