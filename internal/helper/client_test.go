package helper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSubprocessClientIdentify(t *testing.T) {
	stub := writeStubHelper(t, "identify", `{"devices":[{"uuid":"test-uuid","productName":"iPhone 15 Pro","deviceName":"Test iPhone","isTrusted":true}]}`)
	client := NewSubprocessClient(stub)

	devices, err := client.Identify()
	if err != nil {
		t.Fatalf("Identify failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(devices))
	}
	if devices[0].UUID != "test-uuid" {
		t.Errorf("uuid = %q, want test-uuid", devices[0].UUID)
	}
}

func TestSubprocessClientList(t *testing.T) {
	stub := writeStubHelper(t, "list", `{"files":[{"handle":"h1","name":"IMG_1234.HEIC","size":1000,"created":"2026-06-27T10:00:00Z","mimeType":"image/heic","livePhotoPair":null}]}`)
	client := NewSubprocessClient(stub)

	files, err := client.List("test-uuid")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("file count = %d, want 1", len(files))
	}
	if files[0].Name != "IMG_1234.HEIC" {
		t.Errorf("name = %q, want IMG_1234.HEIC", files[0].Name)
	}
}

func TestSubprocessClientIdentifyNoDevices(t *testing.T) {
	stub := writeStubHelper(t, "identify", `{"devices":[]}`)
	client := NewSubprocessClient(stub)

	devices, err := client.Identify()
	if err != nil {
		t.Fatalf("Identify failed: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("device count = %d, want 0", len(devices))
	}
}

func TestSubprocessClientHelperNotFound(t *testing.T) {
	client := NewSubprocessClient("/nonexistent/helper-binary")
	_, err := client.Identify()
	if err == nil {
		t.Fatal("expected error for missing helper, got nil")
	}
}

func writeStubHelper(t *testing.T, expectedSubcmd string, jsonResponse string) string {
	t.Helper()
	dir := t.TempDir()
	stubPath := filepath.Join(dir, "stub-helper")
	stubContent := `#!/bin/bash
echo '` + jsonResponse + `'
`
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}
	return stubPath
}

func TestDownloadWritesToTarget(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "downloaded.heic")
	stubContent := `#!/bin/bash
# args: download --device <uuid> --handle <h> --to <path>
# extract --to value
TO=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --to) TO="$2"; shift 2 ;;
    *) shift ;;
  esac
done
echo "fake-bytes" > "$TO"
`
	stubPath := filepath.Join(dir, "stub-helper")
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}

	client := NewSubprocessClient(stubPath)
	err := client.Download("test-uuid", "handle-1", targetPath)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "fake-bytes\n" {
		t.Errorf("target content = %q, want 'fake-bytes\\n'", string(data))
	}
}

var _ = json.Marshal // keep import
