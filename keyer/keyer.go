package keyer

import (
	"image"
	"image/color"
	"math"
)

// DefaultSimilarity is the ffmpeg colorkey default similarity threshold.
const DefaultSimilarity = 0.25

// DefaultBlend is the ffmpeg colorkey default blend value.
const DefaultBlend = 0.08

// DetectKeyColor samples the top-left 4×4 pixels and returns the most common color.
func DetectKeyColor(img image.Image) color.RGBA {
	counts := map[color.RGBA]int{}
	bounds := img.Bounds()
	maxX := bounds.Min.X + 4
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	maxY := bounds.Min.Y + 4
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}
	for y := bounds.Min.Y; y < maxY; y++ {
		for x := bounds.Min.X; x < maxX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
			counts[c]++
		}
	}
	best, bestCount := color.RGBA{G: 255, A: 255}, 0
	for c, n := range counts {
		if n > bestCount {
			best, bestCount = c, n
		}
	}
	return best
}

// ColorKey applies chroma key removal, replicating ffmpeg's colorkey filter.
// Pixels close to keyColor become transparent; a blend zone provides soft edges.
func ColorKey(img image.Image, keyColor color.RGBA, similarity, blend float64) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	kr, kg, kb := float64(keyColor.R), float64(keyColor.G), float64(keyColor.B)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r32, g32, b32, a32 := img.At(x, y).RGBA()
			pr, pg, pb := float64(r32>>8), float64(g32>>8), float64(b32>>8)
			origAlpha := float64(a32 >> 8)

			dr := pr - kr
			dg := pg - kg
			db := pb - kb
			diff := math.Sqrt((dr*dr + dg*dg + db*db) / (3.0 * 255.0 * 255.0))

			var alpha float64
			switch {
			case diff <= similarity:
				alpha = 0
			case blend > 0 && diff < similarity+blend:
				alpha = (diff - similarity) / blend
			default:
				alpha = 1
			}

			finalAlpha := alpha * origAlpha
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8(pr),
				G: uint8(pg),
				B: uint8(pb),
				A: uint8(math.Round(finalAlpha)),
			})
		}
	}
	return out
}

// Despill removes green spill from edge pixels, replicating ffmpeg's despill=green.
// If G > average(R, B), G is clamped to average(R, B).
func Despill(img *image.NRGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		off := (y - bounds.Min.Y) * img.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			i := off + (x-bounds.Min.X)*4
			r := img.Pix[i]
			g := img.Pix[i+1]
			b := img.Pix[i+2]
			avg := (uint16(r) + uint16(b)) / 2
			if uint16(g) > avg {
				img.Pix[i+1] = uint8(avg)
			}
		}
	}
}

// Trim crops the image to the bounding box of non-transparent pixels (alpha > 0).
func Trim(img *image.NRGBA) *image.NRGBA {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		off := (y - bounds.Min.Y) * img.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a := img.Pix[off+(x-bounds.Min.X)*4+3]
			if a > 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if maxX < minX || maxY < minY {
		// Fully transparent; return 1×1 transparent image.
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}

	rect := image.Rect(0, 0, maxX-minX+1, maxY-minY+1)
	trimmed := image.NewNRGBA(rect)
	for y := 0; y < rect.Dy(); y++ {
		srcOff := (minY - bounds.Min.Y + y) * img.Stride
		dstOff := y * trimmed.Stride
		srcStart := srcOff + (minX-bounds.Min.X)*4
		copy(trimmed.Pix[dstOff:dstOff+rect.Dx()*4], img.Pix[srcStart:srcStart+rect.Dx()*4])
	}
	return trimmed
}

// RemoveBackground is the full pipeline: detect key color, color key, despill, trim.
func RemoveBackground(img image.Image) *image.NRGBA {
	key := DetectKeyColor(img)
	keyed := ColorKey(img, key, DefaultSimilarity, DefaultBlend)
	Despill(keyed)
	return Trim(keyed)
}
