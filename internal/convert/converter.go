package convert

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Converter interface {
	HEICToJPEG(heicPath, jpegPath string) error
}

type SubprocessConverter struct {
	sipsPath        string
	exiftoolPath    string
	ExiftoolWarning string
}

func NewSubprocessConverter(sipsPath, exiftoolPath string) *SubprocessConverter {
	return &SubprocessConverter{
		sipsPath:     sipsPath,
		exiftoolPath: exiftoolPath,
	}
}

func (c *SubprocessConverter) HEICToJPEG(heicPath, jpegPath string) error {
	cmd := exec.Command(c.sipsPath, "-s", "format", "jpeg", heicPath, "--out", jpegPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sips HEIC→JPEG: %w (stderr: %s)", err, stderr.String())
	}

	if c.exiftoolPath != "" {
		cmd := exec.Command(c.exiftoolPath, "-tagsFromFile", heicPath, "-all:all", jpegPath)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			c.ExiftoolWarning = fmt.Sprintf("exiftool metadata transfer failed: %v (stderr: %s) — JPEG has incomplete metadata, HEIC original is preserved with full metadata", err, stderr.String())
		}
	} else {
		c.ExiftoolWarning = "exiftool not configured — JPEG has only basic EXIF from sips, HEIC original is preserved with full metadata"
	}

	return nil
}
