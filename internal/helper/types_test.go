package helper

import (
	"encoding/json"
	"testing"
)

func TestIdentifyResponseUnmarshal(t *testing.T) {
	raw := `{"devices":[{"uuid":"a1b2c3d4","productName":"iPhone 15 Pro","deviceName":"Sebs iPhone","isTrusted":true}]}`
	var resp IdentifyResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(resp.Devices))
	}
	dev := resp.Devices[0]
	if dev.UUID != "a1b2c3d4" {
		t.Errorf("uuid = %q, want a1b2c3d4", dev.UUID)
	}
	if dev.ProductName != "iPhone 15 Pro" {
		t.Errorf("productName = %q, want 'iPhone 15 Pro'", dev.ProductName)
	}
	if dev.DeviceName != "Sebs iPhone" {
		t.Errorf("deviceName = %q, want 'Sebs iPhone'", dev.DeviceName)
	}
	if !dev.IsTrusted {
		t.Error("isTrusted = false, want true")
	}
}

func TestListResponseUnmarshal(t *testing.T) {
	raw := `{"files":[{"handle":"h1","name":"IMG_1234.HEIC","size":1234567,"created":"2026-06-27T10:30:00Z","mimeType":"image/heic","livePhotoPair":"IMG_1234.MOV"},{"handle":"h2","name":"IMG_1235.MOV","size":2345678,"created":"2026-06-27T10:31:00Z","mimeType":"video/quicktime","livePhotoPair":null}]}`
	var resp ListResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Files) != 2 {
		t.Fatalf("file count = %d, want 2", len(resp.Files))
	}
	f := resp.Files[0]
	if f.Handle != "h1" {
		t.Errorf("handle = %q, want h1", f.Handle)
	}
	if f.Size != 1234567 {
		t.Errorf("size = %d, want 1234567", f.Size)
	}
	if f.LivePhotoPair == nil {
		t.Fatal("livePhotoPair = nil, want non-nil")
	}
	if *f.LivePhotoPair != "IMG_1234.MOV" {
		t.Errorf("livePhotoPair = %q, want IMG_1234.MOV", *f.LivePhotoPair)
	}
	if resp.Files[1].LivePhotoPair != nil {
		t.Error("second file livePhotoPair should be nil")
	}
}

func TestListResponseEmptyFiles(t *testing.T) {
	raw := `{"files":[]}`
	var resp ListResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Files) != 0 {
		t.Errorf("file count = %d, want 0", len(resp.Files))
	}
}
