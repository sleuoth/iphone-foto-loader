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
