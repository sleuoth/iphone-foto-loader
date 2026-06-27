package routing

import (
	"testing"
	"time"
)

func TestPrefixFilenameRemovesIMG(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	got := PrefixFilename("IMG_1234.HEIC", date)
	want := "06_27_1234.heic"
	if got != want {
		t.Errorf("PrefixFilename(IMG_1234.HEIC) = %q, want %q", got, want)
	}
}

func TestPrefixFilenameKeepsNonIMG(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	got := PrefixFilename("whatsapp-image.jpg", date)
	want := "06_27_whatsapp-image.jpg"
	if got != want {
		t.Errorf("PrefixFilename(whatsapp-image.jpg) = %q, want %q", got, want)
	}
}

func TestPrefixFilenamePreservesExtensionCase(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	got := PrefixFilename("IMG_1234.HEIC", date)
	if got != "06_27_1234.heic" {
		t.Errorf("extension should be lowercased: got %q", got)
	}
}

func TestPrefixFilenameMOV(t *testing.T) {
	date := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	got := PrefixFilename("IMG_9999.MOV", date)
	want := "01_05_9999.mov"
	if got != want {
		t.Errorf("PrefixFilename(IMG_9999.MOV) = %q, want %q", got, want)
	}
}

func TestPrefixFilenameNoExtension(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	got := PrefixFilename("IMG_1234", date)
	want := "06_27_1234"
	if got != want {
		t.Errorf("PrefixFilename(IMG_1234) = %q, want %q", got, want)
	}
}

func TestComputeTargetPathCameraHEIC(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	decision := RouteDecision{
		IsCamera:     true,
		IsHEIC:       true,
		OriginalName: "IMG_1234.HEIC",
		CaptureDate:  date,
	}
	paths := ComputeTargetPaths(decision)
	if len(paths) != 2 {
		t.Fatalf("path count = %d, want 2", len(paths))
	}
	expectedJPEG := "2026/06_27_1234.jpg"
	expectedHEIC := "2026/heic/06_27_1234.heic"
	if paths[0] != expectedJPEG {
		t.Errorf("path[0] = %q, want %q", paths[0], expectedJPEG)
	}
	if paths[1] != expectedHEIC {
		t.Errorf("path[1] = %q, want %q", paths[1], expectedHEIC)
	}
}

func TestComputeTargetPathCameraJPEG(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	decision := RouteDecision{
		IsCamera:     true,
		IsHEIC:       false,
		OriginalName: "IMG_1234.JPG",
		CaptureDate:  date,
	}
	paths := ComputeTargetPaths(decision)
	if len(paths) != 1 {
		t.Fatalf("path count = %d, want 1", len(paths))
	}
	expected := "2026/06_27_1234.jpg"
	if paths[0] != expected {
		t.Errorf("path[0] = %q, want %q", paths[0], expected)
	}
}

func TestComputeTargetPathCameraMOV(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	decision := RouteDecision{
		IsCamera:     true,
		IsHEIC:       false,
		OriginalName: "IMG_1234.MOV",
		CaptureDate:  date,
	}
	paths := ComputeTargetPaths(decision)
	if len(paths) != 1 {
		t.Fatalf("path count = %d, want 1", len(paths))
	}
	expected := "2026/06_27_1234.mov"
	if paths[0] != expected {
		t.Errorf("path[0] = %q, want %q", paths[0], expected)
	}
}

func TestComputeTargetPathSonstiges(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	decision := RouteDecision{
		IsCamera:     false,
		IsHEIC:       false,
		OriginalName: "whatsapp-image.jpg",
		CaptureDate:  date,
	}
	paths := ComputeTargetPaths(decision)
	if len(paths) != 1 {
		t.Fatalf("path count = %d, want 1", len(paths))
	}
	expected := "2026/sonstiges/06_27_whatsapp-image.jpg"
	if paths[0] != expected {
		t.Errorf("path[0] = %q, want %q", paths[0], expected)
	}
}

func TestComputeTargetPathSonstigesHEIC(t *testing.T) {
	date := time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)
	decision := RouteDecision{
		IsCamera:     false,
		IsHEIC:       true,
		OriginalName: "saved.heic",
		CaptureDate:  date,
	}
	paths := ComputeTargetPaths(decision)
	if len(paths) != 1 {
		t.Fatalf("path count = %d, want 1 (no conversion for sonstiges)", len(paths))
	}
	expected := "2026/sonstiges/06_27_saved.heic"
	if paths[0] != expected {
		t.Errorf("path[0] = %q, want %q", paths[0], expected)
	}
}
