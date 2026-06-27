package exif

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExiftoolJSON(t *testing.T) {
	raw := `[{"Make":"Apple","Model":"iPhone 15 Pro","DateTimeOriginal":"2026:06:27 10:30:00"}]`
	info, err := parseExiftoolOutput(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if info.Make != "Apple" {
		t.Errorf("Make = %q, want Apple", info.Make)
	}
	if info.Model != "iPhone 15 Pro" {
		t.Errorf("Model = %q, want iPhone 15 Pro", info.Model)
	}
	if info.DateTimeOriginal != "2026:06:27 10:30:00" {
		t.Errorf("DateTimeOriginal = %q", info.DateTimeOriginal)
	}
}

func TestParseExiftoolEmptyJSON(t *testing.T) {
	raw := `[]`
	info, err := parseExiftoolOutput(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if info != nil {
		t.Error("expected nil for empty exiftool output")
	}
}

func TestIsCameraPhoto(t *testing.T) {
	tests := []struct {
		name string
		info *Info
		want bool
	}{
		{"apple make", &Info{Make: "Apple", Model: "iPhone 15 Pro"}, true},
		{"canon make", &Info{Make: "Canon", Model: "EOS R5"}, true},
		{"empty make", &Info{Make: "", Model: ""}, false},
		{"nil info", nil, false},
		{"whatsapp no make", &Info{Make: "", Model: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCameraPhoto(tt.info); got != tt.want {
				t.Errorf("IsCameraPhoto() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubprocessReaderRead(t *testing.T) {
	dir := t.TempDir()
	stubPath := filepath.Join(dir, "fake-exiftool")
	stubContent := `#!/bin/bash
echo '[{"Make":"Apple","Model":"iPhone 15 Pro","DateTimeOriginal":"2026:06:27 10:30:00"}]'
`
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}

	reader := NewSubprocessReader(stubPath)
	info, err := reader.Read("/tmp/fake.heic")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if info.Make != "Apple" {
		t.Errorf("Make = %q, want Apple", info.Make)
	}
}

func TestSubprocessReaderNotFound(t *testing.T) {
	reader := NewSubprocessReader("/nonexistent/exiftool")
	_, err := reader.Read("/tmp/fake.heic")
	if err == nil {
		t.Fatal("expected error for missing exiftool, got nil")
	}
}
