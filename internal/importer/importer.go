package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebastianleuoth/iphone-foto-loader/internal/routing"
)

type FileItem struct {
	Handle        string
	Name          string
	Size          int64
	Created       string
	MimeType      string
	LivePhotoPair *string
}

type ImportResult struct {
	Success     bool
	TargetPaths []string
	Error       error
}

type HelperClient interface {
	Download(deviceUUID, handle, targetPath string) error
}

type EXIFInfo struct {
	Make             string
	Model            string
	DateTimeOriginal string
}

type EXIFReader interface {
	Read(filePath string) (*EXIFInfo, error)
}

type Converter interface {
	HEICToJPEG(heicPath, jpegPath string) error
}

type Importer struct {
	Helper    HelperClient
	EXIF      EXIFReader
	Converter Converter
}

func (imp *Importer) ImportFile(file FileItem, deviceUUID, targetRoot string) (ImportResult, error) {
	stagingDir := filepath.Join(targetRoot, ".staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return ImportResult{}, fmt.Errorf("create staging dir: %w", err)
	}

	stagingPath := filepath.Join(stagingDir, file.Name)
	if err := imp.Helper.Download(deviceUUID, file.Handle, stagingPath); err != nil {
		return ImportResult{Success: false, Error: err}, nil
	}
	defer os.Remove(stagingPath)

	info, err := imp.EXIF.Read(stagingPath)
	if err != nil {
		return ImportResult{Success: false, Error: fmt.Errorf("exif read: %w", err)}, nil
	}

	var captureDate time.Time
	if info != nil && info.DateTimeOriginal != "" {
		captureDate, err = parseExifDate(info.DateTimeOriginal)
		if err != nil {
			captureDate = time.Now()
		}
	} else {
		captureDate = time.Now()
	}

	isCamera := info != nil && info.Make != ""
	isHEIC := strings.EqualFold(filepath.Ext(file.Name), ".heic")

	decision := routing.RouteDecision{
		IsCamera:     isCamera,
		IsHEIC:       isHEIC,
		OriginalName: file.Name,
		CaptureDate:  captureDate,
	}

	targetPaths := routing.ComputeTargetPaths(decision)
	fullPaths := make([]string, len(targetPaths))
	for i, rel := range targetPaths {
		fullPaths[i] = filepath.Join(targetRoot, rel)
	}

	if isCamera && isHEIC {
		jpegPath := fullPaths[0]
		heicPath := fullPaths[1]
		if err := os.MkdirAll(filepath.Dir(jpegPath), 0755); err != nil {
			return ImportResult{Success: false, Error: err}, nil
		}
		if err := os.MkdirAll(filepath.Dir(heicPath), 0755); err != nil {
			return ImportResult{Success: false, Error: err}, nil
		}
		if err := imp.Converter.HEICToJPEG(stagingPath, jpegPath); err != nil {
			return ImportResult{Success: false, Error: fmt.Errorf("convert: %w", err)}, nil
		}
		if err := copyFile(stagingPath, heicPath); err != nil {
			return ImportResult{Success: false, Error: fmt.Errorf("copy heic: %w", err)}, nil
		}
	} else {
		dst := fullPaths[0]
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return ImportResult{Success: false, Error: err}, nil
		}
		if err := copyFile(stagingPath, dst); err != nil {
			return ImportResult{Success: false, Error: fmt.Errorf("move to target: %w", err)}, nil
		}
	}

	return ImportResult{Success: true, TargetPaths: fullPaths}, nil
}

func parseExifDate(s string) (time.Time, error) {
	return time.Parse("2006:01:02 15:04:05", s)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func (imp *Importer) ImportLivePhotoPair(heicFile, movFile FileItem, deviceUUID, targetRoot string) (ImportResult, error) {
	heicResult, err := imp.ImportFile(heicFile, deviceUUID, targetRoot)
	if err != nil || !heicResult.Success {
		e := err
		if e == nil {
			e = heicResult.Error
		}
		return ImportResult{Success: false, Error: fmt.Errorf("heic part failed: %w", e)}, nil
	}

	movResult, err := imp.ImportFile(movFile, deviceUUID, targetRoot)
	if err != nil || !movResult.Success {
		for _, p := range heicResult.TargetPaths {
			os.Remove(p)
		}
		e := err
		if e == nil {
			e = movResult.Error
		}
		return ImportResult{Success: false, Error: fmt.Errorf("mov part failed: %w", e)}, nil
	}

	allPaths := append(heicResult.TargetPaths, movResult.TargetPaths...)
	return ImportResult{Success: true, TargetPaths: allPaths}, nil
}
