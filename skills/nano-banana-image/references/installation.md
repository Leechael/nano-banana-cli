# Installation & Configuration

## Install nano-banana

Preferred: download pre-built binary from GitHub Releases.

```bash
# 1) Download latest release
gh release download -R Leechael/nano-banana-image-skill --pattern 'nano-banana-*.tar.gz'

# 2) Extract and install
tar -xzf nano-banana-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz
install -m 0755 nano-banana ~/.local/bin/nano-banana
```

Make sure `~/.local/bin` is in your PATH.

Alternative: build from source (requires Go 1.25+):

```bash
git clone https://github.com/Leechael/nano-banana-image-skill.git
cd nano-banana-image-skill
go build -o ~/.local/bin/nano-banana ./cmd/nano-banana
```

Quick check:

```bash
nano-banana --help
```

## Configure credentials

Required:

```bash
export GEMINI_API_KEY="<your-key>"
```

Get a Gemini API key at: https://aistudio.google.com/apikey

## Recommended: 1Password CLI for credentials

Prefer running `nano-banana` with the 1Password CLI (`op`) so the API key is injected at runtime instead of stored in shell profiles.

Reference: https://developer.1password.com/docs/service-accounts/use-with-1password-cli

```bash
op run --env-file=.env -- nano-banana "a cat in a spacesuit"
op run --env-file=.env -- nano-banana "robot mascot" -t -o mascot
```

Your `.env` should use a 1Password secret reference:

```
GEMINI_API_KEY=op://vault/gemini/api-key
```
