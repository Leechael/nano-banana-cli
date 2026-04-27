//go:build regression

package keyer

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestRegressionVsFFmpeg(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Skip("no test images in testdata/")
	}

	for _, input := range inputs {
		base := filepath.Base(input)
		golden := filepath.Join("testdata", "golden", base)
		if _, err := os.Stat(golden); os.IsNotExist(err) {
			t.Logf("skipping %s: no golden file", base)
			continue
		}

		t.Run(base, func(t *testing.T) {
			srcFile, err := os.Open(input)
			if err != nil {
				t.Fatal(err)
			}
			defer srcFile.Close()
			srcImg, _, err := image.Decode(srcFile)
			if err != nil {
				t.Fatal(err)
			}

			got := RemoveBackground(srcImg)

			goldenFile, err := os.Open(golden)
			if err != nil {
				t.Fatal(err)
			}
			defer goldenFile.Close()
			wantImg, err := png.Decode(goldenFile)
			if err != nil {
				t.Fatal(err)
			}

			gotBounds := got.Bounds()
			wantBounds := wantImg.Bounds()

			t.Logf("got: %dx%d, golden: %dx%d", gotBounds.Dx(), gotBounds.Dy(), wantBounds.Dx(), wantBounds.Dy())

			// Compare using the overlapping region.
			w := min(gotBounds.Dx(), wantBounds.Dx())
			h := min(gotBounds.Dy(), wantBounds.Dy())
			if w == 0 || h == 0 {
				t.Fatal("zero-size comparison region")
			}

			var sumSq float64
			var maxDiff float64
			n := float64(w * h * 4) // R, G, B, A channels
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
					wr, wg, wb, wa := wantImg.At(wantBounds.Min.X+x, wantBounds.Min.Y+y).RGBA()
					for _, pair := range [][2]uint32{{gr, wr}, {gg, wg}, {gb, wb}, {ga, wa}} {
						d := math.Abs(float64(pair[0]>>8) - float64(pair[1]>>8))
						sumSq += d * d
						if d > maxDiff {
							maxDiff = d
						}
					}
				}
			}

			rmse := math.Sqrt(sumSq / n)
			t.Logf("RMSE: %.4f, max channel diff: %.0f", rmse, maxDiff)

			// Allow some tolerance for implementation differences.
			const rmseThreshold = 10.0
			if rmse > rmseThreshold {
				t.Errorf("RMSE %.4f exceeds threshold %.1f", rmse, rmseThreshold)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
