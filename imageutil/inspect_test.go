package imageutil

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInfoBasics(t *testing.T) {
	s := newTestService(t)

	info, err := s.Info(context.Background(), pngInput("scan1.png", solidPage(2480, 3508)))
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.ID != "scan1" || info.Width != 2480 || info.Height != 3508 || info.Format != "png" {
		t.Fatalf("info = %+v", info)
	}
	// 3508 exceeds the default MaxSide: the LLM hint must be present.
	if len(info.Notes) == 0 {
		t.Fatal("expected the exceeds-MaxSide hint note")
	}
	// Info is metadata-only: no result bytes are attached.
	if info.Data != nil {
		t.Fatal("Info must not return image bytes")
	}
}

func TestInfoReportsExifOrientation(t *testing.T) {
	s := newTestService(t)
	src := Input{Data: jpegWithExif(t, solidPage(3, 2), 6), Name: "phone"}

	info, err := s.Info(context.Background(), src)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if len(info.Notes) == 0 {
		t.Fatal("expected EXIF orientation note")
	}
	errContains(t, errors.New(info.Notes[0]), "orientation 6")
}

func TestInfoReportsCorrectedDimensionsPerOrientation(t *testing.T) {
	s := newTestService(t)
	// Stored 3x2 JPEG: orientations 5-8 swap the axes to the corrected
	// 2x3 canonical frame, orientations 1-4 keep 3x2.
	for orient := 1; orient <= 8; orient++ {
		src := Input{Data: jpegWithExif(t, solidPage(3, 2), uint16(orient)), Name: "phone"}
		info, err := s.Info(context.Background(), src)
		if err != nil {
			t.Fatalf("orientation %d: Info: %v", orient, err)
		}
		if info.Orientation != orient {
			t.Fatalf("orientation %d: reported %d", orient, info.Orientation)
		}
		wantW, wantH := 3, 2
		if orient >= 5 {
			wantW, wantH = 2, 3
		}
		if info.Width != wantW || info.Height != wantH {
			t.Fatalf("orientation %d: dims %dx%d, want %dx%d",
				orient, info.Width, info.Height, wantW, wantH)
		}
		if info.Data != nil {
			t.Fatalf("orientation %d: Info must not return image bytes", orient)
		}
	}
}

func TestInfoPNGReportsOrientationOneUnswapped(t *testing.T) {
	s := newTestService(t)

	info, err := s.Info(context.Background(), pngInput("scan1.png", solidPage(3, 2)))
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Orientation != 1 {
		t.Fatalf("orientation = %d, want 1 for PNG", info.Orientation)
	}
	if info.Width != 3 || info.Height != 2 {
		t.Fatalf("dims = %dx%d, want unswapped 3x2", info.Width, info.Height)
	}
}

func TestInfoReportsZeroForMalformedExif(t *testing.T) {
	s := newTestService(t)
	// 12 lies outside the defined 1..8 range: the EXIF block exists but
	// is unreadable, so the stored dimensions stay untouched.
	src := Input{Data: jpegWithExif(t, solidPage(3, 2), 12), Name: "broken"}

	info, err := s.Info(context.Background(), src)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Orientation != 0 {
		t.Fatalf("orientation = %d, want 0 for malformed EXIF", info.Orientation)
	}
	if info.Width != 3 || info.Height != 2 {
		t.Fatalf("dims = %dx%d, want stored 3x2", info.Width, info.Height)
	}
	errContains(t, errors.New(joinNotes(info)), "unreadable")
}

func TestInfoOversizedJPEGReportsCorrectedDimsWithoutResize(t *testing.T) {
	s := newTestService(t)
	// 5000x3000 stored, orientation 6: the corrected frame is 3000x5000.
	// hardMaxSide applies to re-encoding operations only — Info reports
	// the metadata without any resize.
	src := Input{Data: jpegWithExif(t, solidPage(5000, 3000), 6), Name: "a4"}

	info, err := s.Info(context.Background(), src)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Width != 3000 || info.Height != 5000 {
		t.Fatalf("dims = %dx%d, want corrected 3000x5000", info.Width, info.Height)
	}
	if info.Orientation != 6 {
		t.Fatalf("orientation = %d, want 6", info.Orientation)
	}
	if info.Data != nil {
		t.Fatal("Info must not return image bytes")
	}
	notes := joinNotes(info)
	if strings.Contains(notes, "capped") || strings.Contains(notes, "resized") {
		t.Fatalf("notes = %q, want no resize or cap hints", notes)
	}
	if !strings.Contains(notes, "exceeds MaxSide") {
		t.Fatalf("notes = %q, want the exceeds-MaxSide hint", notes)
	}
}

func TestInfoRejectsUndecodableData(t *testing.T) {
	s := newTestService(t)

	_, err := s.Info(context.Background(), bytesInput("broken", []byte("not a png at all")))
	if err == nil {
		t.Fatal("expected an error for non-image data")
	}
}

func TestViewScalesToModelSize(t *testing.T) {
	s := newTestService(t)
	src := pngInput("big", solidPage(2480, 3508))

	res, err := s.View(context.Background(), src, 0)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	// 1600/3508 * 2480 = 1131.13 -> 1131
	if res.Width != 1131 || res.Height != 1600 {
		t.Fatalf("dims = %dx%d, want 1131x1600", res.Width, res.Height)
	}
	if res.Format != "png" || res.ID != "big-view01" {
		t.Fatalf("result = %+v", res)
	}
	if len(res.Notes) != 1 {
		t.Fatalf("notes = %v, want a single resize note", res.Notes)
	}
	if len(res.Data) == 0 || res.Size != int64(len(res.Data)) {
		t.Fatalf("result bytes missing: size = %d", res.Size)
	}
}

func TestViewKeepsSmallPages(t *testing.T) {
	s := newTestService(t)
	src := pngInput("small", solidPage(500, 700))

	res, err := s.View(context.Background(), src, 0)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if res.Width != 500 || res.Height != 700 {
		t.Fatalf("dims = %dx%d, want untouched", res.Width, res.Height)
	}
	errContains(t, errors.New(joinNotes(res)), "already within")
}

func TestViewValidatesMaxSide(t *testing.T) {
	s := newTestService(t)
	src := pngInput("page", solidPage(100, 100))

	if _, err := s.View(context.Background(), src, -3); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("error = %v, want ErrInvalidParameter", err)
	}
	res, err := s.View(context.Background(), src, 9000)
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	errContains(t, errors.New(joinNotes(res)), "capped")
	if res.Width != 100 || res.Height != 100 {
		t.Fatalf("small page changed to %dx%d", res.Width, res.Height)
	}
}
