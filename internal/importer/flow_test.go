package importer

import (
	"os"
	"sync"
	"testing"
	"time"
)

type mockDB struct {
	mu       sync.Mutex
	imported map[string]int64
}

func (m *mockDB) IsImported(filename string, size int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.imported[filename]
	return ok && s == size
}

func (m *mockDB) Insert(filename string, size int64, importedAt, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func TestDeviceFlowStopsAfterMaxFiles(t *testing.T) {
	files := []FileItem{
		{Handle: "h1", Name: "IMG_0001.JPG", Size: 1000, Created: "2026-06-27T10:00:00Z"},
		{Handle: "h2", Name: "IMG_0002.JPG", Size: 1001, Created: "2026-06-27T10:01:00Z"},
		{Handle: "h3", Name: "IMG_0003.JPG", Size: 1002, Created: "2026-06-27T10:02:00Z"},
		{Handle: "h4", Name: "IMG_0004.JPG", Size: 1003, Created: "2026-06-27T10:03:00Z"},
		{Handle: "h5", Name: "IMG_0005.JPG", Size: 1004, Created: "2026-06-27T10:04:00Z"},
	}
	db := &mockDB{imported: map[string]int64{}}

	var progress []string
	stats := DeviceFlow(DeviceFlowInput{
		DeviceUUID: "uuid",
		TargetRoot: t.TempDir(),
		Files:      files,
		DB:         db,
		Helper:     &mockHelper{downloadContent: []byte("bytes")},
		EXIF:       &mockEXIFReader{info: &EXIFInfo{Make: "Apple", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter:  &mockConverter{},
		MaxFiles:   3,
		Progress: func(current, total int, name string) {
			progress = append(progress, name)
		},
	})

	if stats.Imported != 3 {
		t.Fatalf("imported = %d, want 3", stats.Imported)
	}
	if len(progress) != 3 {
		t.Fatalf("progress calls = %d, want 3", len(progress))
	}
	if len(db.imported) != 3 {
		t.Fatalf("db entries = %d, want 3", len(db.imported))
	}
	if progress[2] != "IMG_0003.JPG" {
		t.Errorf("last processed = %q, want IMG_0003.JPG", progress[2])
	}
}

type concurrentHelper struct {
	mu        sync.Mutex
	active    int
	maxActive int
}

func (m *concurrentHelper) Download(deviceUUID, handle, targetPath string) error {
	m.mu.Lock()
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	m.mu.Unlock()

	time.Sleep(50 * time.Millisecond)
	err := os.WriteFile(targetPath, []byte("bytes"), 0644)

	m.mu.Lock()
	m.active--
	m.mu.Unlock()

	return err
}

func (m *concurrentHelper) MaxActive() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxActive
}

func TestDeviceFlowRunsDownloadsInParallel(t *testing.T) {
	files := []FileItem{
		{Handle: "h1", Name: "IMG_0001.JPG", Size: 1000, Created: "2026-06-27T10:00:00Z"},
		{Handle: "h2", Name: "IMG_0002.JPG", Size: 1001, Created: "2026-06-27T10:01:00Z"},
		{Handle: "h3", Name: "IMG_0003.JPG", Size: 1002, Created: "2026-06-27T10:02:00Z"},
		{Handle: "h4", Name: "IMG_0004.JPG", Size: 1003, Created: "2026-06-27T10:03:00Z"},
	}
	helper := &concurrentHelper{}

	stats := DeviceFlow(DeviceFlowInput{
		DeviceUUID:  "uuid",
		TargetRoot:  t.TempDir(),
		Files:       files,
		DB:          &mockDB{imported: map[string]int64{}},
		Helper:      helper,
		EXIF:        &mockEXIFReader{info: &EXIFInfo{Make: "Apple", DateTimeOriginal: "2026:06:27 10:30:00"}},
		Converter:   &mockConverter{},
		MaxParallel: 3,
		Progress:    func(current, total int, name string) {},
	})

	if stats.Imported != 4 {
		t.Fatalf("imported = %d, want 4", stats.Imported)
	}
	if helper.MaxActive() < 2 {
		t.Fatalf("max concurrent downloads = %d, want at least 2", helper.MaxActive())
	}
}
