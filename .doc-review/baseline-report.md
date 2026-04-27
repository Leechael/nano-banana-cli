# Documentation Baseline Report

Generated: 2026-04-27
Repository: imagen
Total files: 4
Total lines: 456

## Health Summary

| Health    | Count | Percentage |
|-----------|-------|------------|
| good      | 3     | 75%        |
| fair      | 1     | 25%        |
| needs work| 0     | 0%         |

## File Inventory

| File | Lines | Words | Type | Last Modified |
|------|-------|-------|------|---------------|
| README.md | 292 | 1343 | landing page / reference | 2026-04-27 |
| skills/nano-banana-image/SKILL.md | 71 | ~300 | skill reference | 2026-03-05 |
| skills/nano-banana-image/references/installation.md | 81 | ~350 | how-to | 2026-03-05 |
| skills/README.md | 12 | ~50 | landing page | 2026-03-05 |

## Broken Links

None. `skills/nano-banana-image/SKILL.md` references `references/installation.md` which exists at the correct relative path.

## Orphaned Files

- `skills/README.md` — not referenced from main README.md
- `skills/nano-banana-image/SKILL.md` — not referenced from main README.md
- `skills/nano-banana-image/references/installation.md` — not referenced from main README.md

These are skill subsystem docs, not user-facing documentation. They may be intentionally separate.

## Terminology (pending confirmation)

| Term | Variants Found | Files | Recommendation |
|------|---------------|-------|----------------|
| imagen | — | 4 | Product name, always lowercase |
| GPT Image 2 | gpt-image-2, GPT Image 2 | 1 | Use "GPT Image 2" in prose; `gpt-image-2` for CLI/model IDs |
| DALL-E | DALL-E, dall-e-2, dall-e-3 | 1 | Use "DALL-E" in prose; lowercase with hyphens for model IDs |
| green-screen | green screen | 1 | Pick one: "green-screen" (adjective) or "green screen" (noun) |
| Codex | ChatGPT/Codex, Codex | 1 | Standardize: "Codex" or "ChatGPT/Codex OAuth" on first use |

## Audience

**Primary:** Developers who want a CLI or Go library for multi-provider image generation.
**Prerequisites:** Familiarity with Go, command-line tools, and API key management.

## Per-File Detail

### README.md
- Type: landing page / reference
- Lines: 292
- Words: 1343
- Reading time: ~6.7 min (detail), ~2.7 min (skim)
- Issues: 4
  - `pbpaste | imagen -a 4:3` assumes macOS without note
  - `make ci` in Development section — non-standard Go command, may confuse users
  - "Bare aliases are accepted for convenience" — awkward phrasing
  - Inconsistent casing: "google" (lowercase in tables) vs "Google Gemini" (title case in intro)
- Health: fair

### skills/nano-banana-image/SKILL.md
- Type: skill reference
- Lines: 71
- Issues: 0
- Health: good

### skills/nano-banana-image/references/installation.md
- Type: how-to
- Lines: 81
- Issues: 0
- Health: good

### skills/README.md
- Type: landing page
- Lines: 12
- Issues: 0
- Health: good

## Recommended Next Steps

1. Confirm terminology table (especially green-screen vs green screen)
2. Run `/doc-review copy` on README.md to fix prose issues
3. Consider adding a note about `pbpaste` being macOS-specific
