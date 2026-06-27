package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sebastianleuoth/iphone-foto-loader/internal/config"
	"github.com/sebastianleuoth/iphone-foto-loader/internal/convert"
	"github.com/sebastianleuoth/iphone-foto-loader/internal/db"
	"github.com/sebastianleuoth/iphone-foto-loader/internal/exif"
	"github.com/sebastianleuoth/iphone-foto-loader/internal/helper"
	"github.com/sebastianleuoth/iphone-foto-loader/internal/importer"
)

type App struct {
	ConfigPath     string
	TargetOverride string
}

func ParseFlags() *App {
	app := &App{}
	flag.StringVar(&app.ConfigPath, "config", defaultConfigPath(), "path to config.toml")
	flag.StringVar(&app.TargetOverride, "target", "", "override target folder for single device")
	flag.Parse()
	return app
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(home, ".config", "iphone-loader", "config.toml")
}

func (a *App) Run() int {
	cfg, err := config.Load(a.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(a.ConfigPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "error: create config dir: %v\n", err)
				return 1
			}
			cfg = &config.Config{Devices: make(map[string]config.DeviceConfig)}
			if err := cfg.Save(a.ConfigPath); err != nil {
				fmt.Fprintf(os.Stderr, "warn: could not create default config: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "error: load config: %v\n", err)
			return 1
		}
	}

	helperPath := a.resolveHelperPath(cfg)
	client := helper.NewSubprocessClient(helperPath)

	devices, err := client.Identify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: identify devices: %v\n", err)
		return 1
	}

	if len(devices) == 0 {
		fmt.Println("Kein iPhone gefunden. iPhone per USB anschließen und erneut versuchen.")
		return 0
	}

	for _, dev := range devices {
		if !dev.IsTrusted {
			fmt.Printf("iPhone \"%s\" nicht vertraut. Auf dem Gerät „Vertrauen\" antippen.\n", dev.DeviceName)
			continue
		}

		target := a.resolveTarget(cfg, dev)
		if target == "" {
			target = a.promptForTarget(dev)
			if target == "" {
				fmt.Printf("Überspringe %s (kein Zielordner)\n", dev.DeviceName)
				continue
			}
			cfg.Devices[dev.UUID] = config.DeviceConfig{Name: dev.DeviceName, Target: target}
			if err := cfg.Save(a.ConfigPath); err != nil {
				fmt.Fprintf(os.Stderr, "warn: could not save config: %v\n", err)
			}
		}

		if err := a.processDevice(client, dev, target); err != nil {
			fmt.Fprintf(os.Stderr, "error: device %s: %v\n", dev.DeviceName, err)
		}
	}

	return 0
}

func (a *App) resolveHelperPath(cfg *config.Config) string {
	if p := os.Getenv("IPHONE_IC_HELPER"); p != "" {
		return p
	}
	if cfg.Helper.Path != "" {
		return cfg.Helper.Path
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "iphone-ic-helper")
	}
	return "iphone-ic-helper"
}

func (a *App) resolveTarget(cfg *config.Config, dev helper.Device) string {
	if a.TargetOverride != "" {
		return a.TargetOverride
	}
	if d, ok := cfg.Devices[dev.UUID]; ok {
		return d.Target
	}
	return ""
}

func (a *App) promptForTarget(dev helper.Device) string {
	fmt.Printf("Neues iPhone erkannt: \"%s\" (UUID %s)\n", dev.DeviceName, dev.UUID)
	fmt.Print("Zielordner für dieses Gerät: ")
	var target string
	fmt.Scanln(&target)
	if target == "" {
		return ""
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid path: %v\n", err)
		return ""
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: create target dir: %v\n", err)
		return ""
	}
	return abs
}

func (a *App) processDevice(client *helper.SubprocessClient, dev helper.Device, target string) error {
	fmt.Printf("Verarbeite \"%s\"...\n", dev.DeviceName)

	files, err := client.List(dev.UUID)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	dbPath := filepath.Join(target, dev.DeviceName, ".iphone-loader.sqlite")
	deviceDir := filepath.Join(target, dev.DeviceName)
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return fmt.Errorf("create device dir: %w", err)
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	exifReader := &exifAdapter{reader: exif.NewSubprocessReader(lookupExiftool())}
	conv := convert.NewSubprocessConverter(lookupSips(), lookupExiftool())

	fileItems := make([]importer.FileItem, len(files))
	for i, f := range files {
		fileItems[i] = importer.FileItem{
			Handle:        f.Handle,
			Name:          f.Name,
			Size:          f.Size,
			Created:       f.Created,
			MimeType:      f.MimeType,
			LivePhotoPair: f.LivePhotoPair,
		}
	}

	stats := importer.DeviceFlow(importer.DeviceFlowInput{
		DeviceUUID: dev.UUID,
		TargetRoot: deviceDir,
		Files:      fileItems,
		DB:         store,
		Helper:     client,
		EXIF:       exifReader,
		Converter:  conv,
		Progress: func(current, total int, name string) {
			fmt.Printf("[%d/%d] %s\n", current, total, name)
		},
	})

	fmt.Printf("Fertig: %d neu, %d übersprungen, %d Fehler\n", stats.Imported, stats.Skipped, stats.Failed)
	for _, e := range stats.Errors {
		fmt.Printf("  Fehler: %s\n", e)
	}

	return nil
}

func lookupExiftool() string {
	for _, p := range []string{"/opt/homebrew/bin/exiftool", "/usr/local/bin/exiftool", "/usr/bin/exiftool"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if path, err := exec.LookPath("exiftool"); err == nil {
		return path
	}
	return ""
}

func lookupSips() string {
	if path, err := exec.LookPath("sips"); err == nil {
		return path
	}
	for _, p := range []string{"/usr/bin/sips", "/usr/local/bin/sips"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "sips"
}

type exifAdapter struct {
	reader *exif.SubprocessReader
}

func (a *exifAdapter) Read(filePath string) (*importer.EXIFInfo, error) {
	info, err := a.reader.Read(filePath)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &importer.EXIFInfo{
		Make:             info.Make,
		Model:            info.Model,
		DateTimeOriginal: info.DateTimeOriginal,
	}, nil
}
