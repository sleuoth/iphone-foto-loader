package db

import (
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	_, err = store.db.Exec("SELECT filename, size, imported_at, target_path FROM imported LIMIT 1")
	if err != nil {
		t.Fatalf("imported table not created: %v", err)
	}
}

func TestOpenFailsOnInvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/dir/db.sqlite")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}
