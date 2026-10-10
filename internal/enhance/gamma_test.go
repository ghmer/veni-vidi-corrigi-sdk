package enhance

import (
	"testing"
)

func TestGammaIdentityAtOne(t *testing.T) {
	src := rampImg(8, 255)
	out, err := Gamma(src, 1.0)
	if err != nil {
		t.Fatalf("Gamma: %v", err)
	}
	for y := 0; y <= 255; y += 17 {
		if got := valueAt(out, 4, y); got != valueAt(src, 4, y) {
			t.Fatalf("row %d: %d, want identity %d", y, got, valueAt(src, 4, y))
		}
	}
}

func TestGammaTwoDarkensMidtones(t *testing.T) {
	src := solidImg(4, 4, grayColor(128))
	out, err := Gamma(src, 2.0)
	if err != nil {
		t.Fatalf("Gamma: %v", err)
	}
	// 255 * (128/255)^2 = 64.25 -> 64
	if got := valueAt(out, 2, 2); got != 64 {
		t.Fatalf("gamma 2 on 128 = %d, want 64", got)
	}
}

func TestGammaHalfBrightensMidtones(t *testing.T) {
	src := solidImg(4, 4, grayColor(64))
	out, err := Gamma(src, 0.5)
	if err != nil {
		t.Fatalf("Gamma: %v", err)
	}
	// 255 * (64/255)^0.5 = 127.6 -> 128
	if got := valueAt(out, 1, 1); got != 128 {
		t.Fatalf("gamma 0.5 on 64 = %d, want 128", got)
	}
}

func TestGammaErrors(t *testing.T) {
	for _, g := range []float64{0, -1} {
		if _, err := Gamma(rampImg(4, 100), g); err == nil {
			t.Fatalf("Gamma(%v) accepted", g)
		}
	}
}
