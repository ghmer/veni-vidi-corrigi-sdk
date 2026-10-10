package imageutil

import (
	"context"
	"errors"
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
