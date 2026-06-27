package importer

import (
	"testing"
)

type mockDB struct {
	imported map[string]int64
}

func (m *mockDB) IsImported(filename string, size int64) bool {
	s, ok := m.imported[filename]
	return ok && s == size
}

func (m *mockDB) Insert(filename string, size int64, importedAt, targetPath string) error {
	m.imported[filename] = size
	return nil
}

func TestDeviceFlowSkipsImported(t *testing.T) {
	files := []FileItem{
		{Handle: "h1", Name: "IMG_1234.JPG", Size: 1000, Created: "2026-06-27T10:30:00Z"},
		{Handle: "h2", Name: "IMG_1235.JPG", Size: 2000, Created: "2026-06-27T10:31:00Z"},
	}
	// Prefix for IMG_1234.JPG with date 2026-06-27 = "06_27_1234.jpg"
	db := &mockDB{imported: map[string]int64{"06_27_1234.jpg": 1000}}

	stats := DeviceFlow(DeviceFlowInput{
		DeviceUUID: "uuid",
		TargetRoot: t.TempDir(),
		Files:      files,
		DB:         db,
		Helper:     &mockHelper{downloadContent: []byte("bytes")},
		EXIF:       &mockEXIFReader{info: &EXIFInfo{Make: "Apple", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter:  &mockConverter{},
		Progress:   func(current, total int, name string) {},
	})

	if stats.Imported != 1 {
		t.Errorf("imported = %d, want 1", stats.Imported)
	}
	if stats.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", stats.Skipped)
	}
}

func TestDeviceFlowSortsByDate(t *testing.T) {
	files := []FileItem{
		{Handle: "h2", Name: "IMG_1235.JPG", Size: 2000, Created: "2026-06-28T10:00:00Z"},
		{Handle: "h1", Name: "IMG_1234.JPG", Size: 1000, Created: "2026-06-27T10:00:00Z"},
	}
	db := &mockDB{imported: map[string]int64{}}

	var order []string
	DeviceFlow(DeviceFlowInput{
		DeviceUUID: "uuid",
		TargetRoot: t.TempDir(),
		Files:      files,
		DB:         db,
		Helper:     &mockHelper{downloadContent: []byte("bytes")},
		EXIF:       &mockEXIFReader{info: &EXIFInfo{Make: "Apple", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter:  &mockConverter{},
		Progress: func(current, total int, name string) {
			order = append(order, name)
		},
	})

	if len(order) != 2 {
		t.Fatalf("order length = %d, want 2", len(order))
	}
	if order[0] != "IMG_1234.JPG" {
		t.Errorf("first processed = %q, want IMG_1234.JPG (older)", order[0])
	}
}
