package keyer

import (
	"image"
	"image/color"
	"testing"
)

func TestColorKey_PureGreen(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	out := ColorKey(img, key, DefaultSimilarity, DefaultBlend)
	a := out.NRGBAAt(0, 0).A
	if a != 0 {
		t.Errorf("pure green alpha = %d, want 0", a)
	}
}

func TestColorKey_NonGreen(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	out := ColorKey(img, key, DefaultSimilarity, DefaultBlend)
	a := out.NRGBAAt(0, 0).A
	if a != 255 {
		t.Errorf("red pixel alpha = %d, want 255", a)
	}
}

func TestColorKey_NearGreen(t *testing.T) {
	// A color in the blend zone: diff ≈ 0.286, between similarity (0.25) and similarity+blend (0.33).
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 90, G: 190, B: 60, A: 255})
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	out := ColorKey(img, key, DefaultSimilarity, DefaultBlend)
	a := out.NRGBAAt(0, 0).A
	if a == 0 || a == 255 {
		t.Errorf("near-green alpha = %d, want 0 < alpha < 255", a)
	}
}

func TestColorKey_PreservesRGB(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 50, B: 100, A: 255})
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	out := ColorKey(img, key, DefaultSimilarity, DefaultBlend)
	c := out.NRGBAAt(0, 0)
	if c.R != 200 || c.G != 50 || c.B != 100 {
		t.Errorf("RGB changed: got (%d,%d,%d), want (200,50,100)", c.R, c.G, c.B)
	}
}

func TestDespill_GreenTint(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 200, B: 80, A: 255})
	Despill(img)
	c := img.NRGBAAt(0, 0)
	// avg(100, 80) = 90, G should be clamped to 90
	if c.G != 90 {
		t.Errorf("G = %d, want 90", c.G)
	}
	if c.R != 100 || c.B != 80 {
		t.Errorf("R/B changed: (%d,%d), want (100,80)", c.R, c.B)
	}
}

func TestDespill_NoChange(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 50, B: 100, A: 255})
	Despill(img)
	c := img.NRGBAAt(0, 0)
	if c.R != 200 || c.G != 50 || c.B != 100 {
		t.Errorf("pixel changed: (%d,%d,%d), want (200,50,100)", c.R, c.G, c.B)
	}
}

func TestTrim_TransparentBorders(t *testing.T) {
	// 5×5 image with opaque pixels only at (1,1)-(3,3).
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	trimmed := Trim(img)
	b := trimmed.Bounds()
	if b.Dx() != 3 || b.Dy() != 3 {
		t.Errorf("trimmed size = %dx%d, want 3x3", b.Dx(), b.Dy())
	}
}

func TestTrim_FullyTransparent(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	trimmed := Trim(img)
	b := trimmed.Bounds()
	if b.Dx() != 1 || b.Dy() != 1 {
		t.Errorf("fully transparent trim = %dx%d, want 1x1", b.Dx(), b.Dy())
	}
}

func TestTrim_NoTrimNeeded(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	trimmed := Trim(img)
	b := trimmed.Bounds()
	if b.Dx() != 3 || b.Dy() != 3 {
		t.Errorf("no-trim size = %dx%d, want 3x3", b.Dx(), b.Dy())
	}
}

func TestDetectKeyColor_MostCommon(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	green := color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	// Fill with green except one red pixel.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, green)
		}
	}
	img.SetNRGBA(0, 0, red)
	got := DetectKeyColor(img)
	want := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	if got != want {
		t.Errorf("DetectKeyColor = %v, want %v", got, want)
	}
}

func TestDetectKeyColor_SmallImage(t *testing.T) {
	// Image smaller than 4×4.
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	c := color.NRGBA{R: 0, G: 200, B: 0, A: 255}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	got := DetectKeyColor(img)
	want := color.RGBA{R: 0, G: 200, B: 0, A: 255}
	if got != want {
		t.Errorf("DetectKeyColor = %v, want %v", got, want)
	}
}

func TestRemoveBackground_Synthetic(t *testing.T) {
	// 10×10 green image with a 4×4 red square at center (3,3)-(6,6).
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	for y := 3; y < 7; y++ {
		for x := 3; x < 7; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	out := RemoveBackground(img)
	b := out.Bounds()
	// Should be trimmed to roughly 4×4.
	if b.Dx() != 4 || b.Dy() != 4 {
		t.Errorf("RemoveBackground size = %dx%d, want 4x4", b.Dx(), b.Dy())
	}
	// All remaining pixels should be opaque red.
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := out.NRGBAAt(x, y)
			if c.A != 255 {
				t.Errorf("pixel (%d,%d) alpha = %d, want 255", x, y, c.A)
			}
			if c.R != 255 || c.G != 0 || c.B != 0 {
				t.Errorf("pixel (%d,%d) = (%d,%d,%d), want (255,0,0)", x, y, c.R, c.G, c.B)
			}
		}
	}
}
