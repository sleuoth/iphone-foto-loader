package convert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubprocessConverterConvertHEICToJPEG(t *testing.T) {
	dir := t.TempDir()
	stubSips := filepath.Join(dir, "fake-sips")
	stubContent := `#!/bin/bash
# sips -s format jpeg <input> --out <output>
# last arg is output path
OUT="${@: -1}"
echo "fake-jpeg" > "$OUT"
`
	if err := os.WriteFile(stubSips, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}

	stubExiftool := filepath.Join(dir, "fake-exiftool")
	exifContent := `#!/bin/bash
# exiftool -tagsFromFile <src> -all:all <dst>
exit 0
`
	if err := os.WriteFile(stubExiftool, []byte(exifContent), 0755); err != nil {
		t.Fatal(err)
	}

	inputPath := filepath.Join(dir, "input.heic")
	if err := os.WriteFile(inputPath, []byte("fake-heic"), 0644); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(dir, "output.jpg")

	conv := NewSubprocessConverter(stubSips, stubExiftool)
	err := conv.HEICToJPEG(inputPath, outputPath)
	if err != nil {
		t.Fatalf("HEICToJPEG failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output JPEG was not created")
	}
}

func TestSubprocessConverterSipsMissing(t *testing.T) {
	conv := NewSubprocessConverter("/nonexistent/sips", "/nonexistent/exiftool")
	err := conv.HEICToJPEG("/tmp/in.heic", "/tmp/out.jpg")
	if err == nil {
		t.Fatal("expected error for missing sips, got nil")
	}
}

func TestSubprocessConverterNoExiftool(t *testing.T) {
	dir := t.TempDir()
	stubSips := filepath.Join(dir, "fake-sips")
	stubContent := `#!/bin/bash
OUT="${@: -1}"
echo "fake-jpeg" > "$OUT"
`
	if err := os.WriteFile(stubSips, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}

	inputPath := filepath.Join(dir, "input.heic")
	if err := os.WriteFile(inputPath, []byte("fake-heic"), 0644); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(dir, "output.jpg")

	conv := NewSubprocessConverter(stubSips, "")
	err := conv.HEICToJPEG(inputPath, outputPath)
	if err != nil {
		t.Fatalf("HEICToJPEG without exiftool should not fail: %v", err)
	}
	if conv.ExiftoolWarning == "" {
		t.Error("expected exiftool warning to be set")
	}
}
