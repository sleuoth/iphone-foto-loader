package exif

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Info struct {
	Make             string `json:"Make"`
	Model            string `json:"Model"`
	DateTimeOriginal string `json:"DateTimeOriginal"`
}

type Reader interface {
	Read(filePath string) (*Info, error)
}

type SubprocessReader struct {
	exiftoolPath string
}

func NewSubprocessReader(exiftoolPath string) *SubprocessReader {
	return &SubprocessReader{exiftoolPath: exiftoolPath}
}

func (r *SubprocessReader) Read(filePath string) (*Info, error) {
	cmd := exec.Command(r.exiftoolPath, "-j", "-Make", "-Model", "-DateTimeOriginal", filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exiftool: %w (stderr: %s)", err, stderr.String())
	}
	return parseExiftoolOutput(stdout.String())
}

func parseExiftoolOutput(raw string) (*Info, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var infos []Info
	if err := json.Unmarshal([]byte(raw), &infos); err != nil {
		return nil, fmt.Errorf("parse exiftool JSON: %w", err)
	}
	if len(infos) == 0 {
		return nil, nil
	}
	return &infos[0], nil
}

func IsCameraPhoto(info *Info) bool {
	if info == nil {
		return false
	}
	return info.Make != ""
}
