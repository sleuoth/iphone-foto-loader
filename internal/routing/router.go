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
