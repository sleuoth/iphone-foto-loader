package routing

import (
	"path/filepath"
	"strings"
	"time"
)

func PrefixFilename(originalName string, date time.Time) string {
	base := originalName
	ext := filepath.Ext(originalName)
	if ext != "" {
		base = strings.TrimSuffix(originalName, ext)
		ext = strings.ToLower(ext)
	}

	if strings.HasPrefix(base, "IMG_") {
		base = strings.TrimPrefix(base, "IMG_")
	}

	prefix := date.Format("01_02_")
	return prefix + base + ext
}

type RouteDecision struct {
	IsCamera     bool
	IsHEIC       bool
	OriginalName string
	CaptureDate  time.Time
}

func ComputeTargetPaths(d RouteDecision) []string {
	year := d.CaptureDate.Format("2006")

	if !d.IsCamera {
		name := PrefixFilename(d.OriginalName, d.CaptureDate)
		return []string{filepath.Join(year, "sonstiges", name)}
	}

	if d.IsHEIC {
		jpegName := PrefixFilename(d.OriginalName, d.CaptureDate)
		jpegName = strings.TrimSuffix(jpegName, filepath.Ext(jpegName)) + ".jpg"
		heicName := PrefixFilename(d.OriginalName, d.CaptureDate)
		return []string{
			filepath.Join(year, jpegName),
			filepath.Join(year, "heic", heicName),
		}
	}

	name := PrefixFilename(d.OriginalName, d.CaptureDate)
	return []string{filepath.Join(year, name)}
}
