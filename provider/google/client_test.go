package google

import (
	"testing"
)

func TestCalculateCost_Flash(t *testing.T) {
	cost := CalculateCost("gemini-3.1-flash-image-preview", 1000, 1000)
	// (1000/1e6)*0.25 + (1000/1e6)*60 = 0.00025 + 0.06 = 0.06025
	expected := 0.06025
	if diff := cost - expected; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cost = %.10f, want %.10f", cost, expected)
	}
}

func TestCalculateCost_Pro(t *testing.T) {
	cost := CalculateCost("gemini-3-pro-image-preview", 1000, 1000)
	// (1000/1e6)*2.0 + (1000/1e6)*120 = 0.002 + 0.12 = 0.122
	expected := 0.122
	if diff := cost - expected; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cost = %.10f, want %.10f", cost, expected)
	}
}

func TestCalculateCost_UnknownModel(t *testing.T) {
	cost := CalculateCost("unknown-model", 1000, 1000)
	// Falls back to flash rates
	expected := 0.06025
	if diff := cost - expected; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cost = %.10f, want %.10f (fallback to default)", cost, expected)
	}
}

func TestImageSize_512(t *testing.T) {
	if got := imageSize("512"); got != "1K" {
		t.Errorf("imageSize(512) = %q, want 1K", got)
	}
}

func TestImageSize_Passthrough(t *testing.T) {
	for _, s := range []string{"1K", "2K", "4K"} {
		if got := imageSize(s); got != s {
			t.Errorf("imageSize(%q) = %q, want %q", s, got, s)
		}
	}
}

func TestExtFromMime_Known(t *testing.T) {
	got := ExtFromMime("image/png")
	if got != ".png" {
		t.Errorf("ExtFromMime(image/png) = %q, want .png", got)
	}
}

func TestExtFromMime_JPEG(t *testing.T) {
	got := ExtFromMime("image/jpeg")
	if got != ".jpg" {
		t.Errorf("ExtFromMime(image/jpeg) = %q, want .jpg", got)
	}
}

func TestExtFromMime_Webp(t *testing.T) {
	got := ExtFromMime("image/webp")
	if got == "" {
		t.Error("ExtFromMime(image/webp) returned empty")
	}
}

func TestExtFromMime_NoSlash(t *testing.T) {
	got := ExtFromMime("bogus")
	if got != ".png" {
		t.Errorf("ExtFromMime(bogus) = %q, want .png", got)
	}
}

func TestExtFromMime_UnknownSubtype(t *testing.T) {
	got := ExtFromMime("image/xyzzy")
	if got != ".xyzzy" {
		t.Errorf("ExtFromMime(image/xyzzy) = %q, want .xyzzy", got)
	}
}

func TestCapabilities_Flash(t *testing.T) {
	c := New("key")
	cap := c.Capabilities("gemini-3.1-flash-image-preview")
	if !cap.SupportsThinking {
		t.Error("flash should support thinking")
	}
	if !cap.SupportsSeed {
		t.Error("should support seed")
	}
	if !cap.SupportsPerson {
		t.Error("should support person")
	}
	if !cap.SupportsReferences {
		t.Error("should support references")
	}
	if !cap.SupportsBatch {
		t.Error("should support batch")
	}
	if cap.MaxReferences != 10 {
		t.Errorf("MaxReferences = %d, want 10", cap.MaxReferences)
	}
}

func TestCapabilities_Pro(t *testing.T) {
	c := New("key")
	cap := c.Capabilities("gemini-3-pro-image-preview")
	if cap.SupportsThinking {
		t.Error("pro should not support thinking")
	}
}

func TestName(t *testing.T) {
	c := New("key")
	if c.Name() != "google" {
		t.Errorf("Name() = %q, want google", c.Name())
	}
}
