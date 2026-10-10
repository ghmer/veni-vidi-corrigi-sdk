package enhance

import (
	"image/color"
	"testing"
)

func grayColor(v uint8) color.NRGBA { return color.NRGBA{v, v, v, 255} }

func TestOtsuSeparatesBimodalInkPaper(t *testing.T) {
	// 40% ink (20) on 60% paper (240): Otsu must separate exactly there;
	// the threshold equals the darkest class representative.
	hist := make([]int, 256)
	hist[20] = 40
	hist[240] = 60
	tt, err := Otsu(hist)
	if err != nil {
		t.Fatalf("Otsu: %v", err)
	}
	if tt != 20 {
		t.Fatalf("threshold = %d, want 20", tt)
	}
}

func TestOtsuThresholdSplitsRealHistogram(t *testing.T) {
	// Ink cluster around 60, paper cluster around 200 with jitter:
	// any threshold inside (60..~200) separates; it must not sit on either
	// cluster boundary value.
	hist := make([]int, 256)
	for v := 55; v <= 65; v++ {
		hist[v] = 10
	}
	for v := 190; v <= 210; v++ {
		hist[v] = 10
	}
	tt, err := Otsu(hist)
	if err != nil {
		t.Fatalf("Otsu: %v", err)
	}
	// Otsu picks the first threshold in the empty gap between the clusters;
	// any value in [65, 189] separates the classes perfectly when binarized
	// with the at-or-below rule.
	if tt < 65 || tt > 189 {
		t.Fatalf("threshold = %d, want inside the gap [65, 189]", tt)
	}
}

func TestOtsuUniformReturnsZero(t *testing.T) {
	hist := make([]int, 256)
	hist[200] = 100
	tt, err := Otsu(hist)
	if err != nil {
		t.Fatalf("Otsu: %v", err)
	}
	if tt != 0 {
		t.Fatalf("threshold = %d, want 0 for uniform histogram", tt)
	}
}

func TestOtsuErrorsOnWrongBucketCount(t *testing.T) {
	if _, err := Otsu(make([]int, 128)); err == nil {
		t.Fatal("accepted 128-bucket histogram")
	}
}

func TestBinarizeClassAndPaper(t *testing.T) {
	gray := []uint8{10, 20, 20, 30}
	out := Binarize(gray, 2, 2, 20) // 20 stays ink (at-or-below rule)
	want := []uint8{0, 0, 0, 255}
	for i, w := range want {
		if out.Pix[i] != w {
			t.Fatalf("bin[%d] = %d, want %d", i, out.Pix[i], w)
		}
	}
}
