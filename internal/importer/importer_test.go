package importer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockHelper struct {
	downloadContent []byte
	downloadCalled  bool
}

func (m *mockHelper) Download(deviceUUID, handle, targetPath string) error {
	m.downloadCalled = true
	return os.WriteFile(targetPath, m.downloadContent, 0644)
}

type mockEXIFReader struct {
	info *EXIFInfo
}

func (m *mockEXIFReader) Read(filePath string) (*EXIFInfo, error) {
	return m.info, nil
}

type mockConverter struct{}

func (m *mockConverter) HEICToJPEG(heicPath, jpegPath string) error {
	return os.WriteFile(jpegPath, []byte("fake-jpeg"), 0644)
}

func TestImportCameraJPEG(t *testing.T) {
	dir := t.TempDir()
	targetRoot := filepath.Join(dir, "archive")

	imp := &Importer{
		Helper:    &mockHelper{downloadContent: []byte("jpeg-bytes")},
		EXIF:      &mockEXIFReader{info: &EXIFInfo{Make: "Apple", Model: "iPhone 15 Pro", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter: &mockConverter{},
	}

	file := FileItem{
		Handle: "h1",
		Name:   "IMG_1234.JPG",
		Size:   1000,
	}
	deviceUUID := "test-uuid"

	result, err := imp.ImportFile(file, deviceUUID, targetRoot)
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.TargetPaths) != 1 {
		t.Fatalf("target path count = %d, want 1", len(result.TargetPaths))
	}
	expectedPath := filepath.Join(targetRoot, "2026", "06_27_1234.jpg")
	if result.TargetPaths[0] != expectedPath {
		t.Errorf("target path = %q, want %q", result.TargetPaths[0], expectedPath)
	}
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("file not written to expected path")
	}
}

func TestImportCameraHEIC(t *testing.T) {
	dir := t.TempDir()
	targetRoot := filepath.Join(dir, "archive")

	imp := &Importer{
		Helper:    &mockHelper{downloadContent: []byte("heic-bytes")},
		EXIF:      &mockEXIFReader{info: &EXIFInfo{Make: "Apple", Model: "iPhone 15 Pro", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter: &mockConverter{},
	}

	file := FileItem{
		Handle: "h1",
		Name:   "IMG_1234.HEIC",
		Size:   2000,
	}

	result, err := imp.ImportFile(file, "test-uuid", targetRoot)
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.TargetPaths) != 2 {
		t.Fatalf("target path count = %d, want 2 (jpeg + heic)", len(result.TargetPaths))
	}
	jpegPath := filepath.Join(targetRoot, "2026", "06_27_1234.jpg")
	heicPath := filepath.Join(targetRoot, "2026", "heic", "06_27_1234.heic")
	if result.TargetPaths[0] != jpegPath {
		t.Errorf("jpeg path = %q, want %q", result.TargetPaths[0], jpegPath)
	}
	if result.TargetPaths[1] != heicPath {
		t.Errorf("heic path = %q, want %q", result.TargetPaths[1], heicPath)
	}
	if _, err := os.Stat(jpegPath); os.IsNotExist(err) {
		t.Error("JPEG not created")
	}
	if _, err := os.Stat(heicPath); os.IsNotExist(err) {
		t.Error("HEIC not moved")
	}
}

func TestImportSonstiges(t *testing.T) {
	dir := t.TempDir()
	targetRoot := filepath.Join(dir, "archive")

	imp := &Importer{
		Helper:    &mockHelper{downloadContent: []byte("whatsapp-bytes")},
		EXIF:      &mockEXIFReader{info: nil},
		Converter: &mockConverter{},
	}

	file := FileItem{
		Handle: "h2",
		Name:   "whatsapp-image.jpg",
		Size:   500,
	}

	result, err := imp.ImportFile(file, "test-uuid", targetRoot)
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(result.TargetPaths) != 1 {
		t.Fatalf("target path count = %d, want 1", len(result.TargetPaths))
	}
	expectedPath := filepath.Join(targetRoot, "2026", "sonstiges", "06_27_whatsapp-image.jpg")
	if result.TargetPaths[0] != expectedPath {
		t.Errorf("target path = %q, want %q", result.TargetPaths[0], expectedPath)
	}
}

func TestParseExifDate(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"2026:06:27 10:30:00", time.Date(2026, 6, 27, 10, 30, 0, 0, time.UTC)},
		{"2026:01:05 08:00:00", time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		got, err := parseExifDate(tt.input)
		if err != nil {
			t.Fatalf("parseExifDate(%q) error: %v", tt.input, err)
		}
		if !got.Equal(tt.want) {
			t.Errorf("parseExifDate(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseExifDateInvalid(t *testing.T) {
	_, err := parseExifDate("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}
