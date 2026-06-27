package importer

import (
	"fmt"
	"sort"
	"time"

	"github.com/sebastianleuoth/iphone-foto-loader/internal/routing"
)

type DBClient interface {
	IsImported(filename string, size int64) bool
	Insert(filename string, size int64, importedAt, targetPath string) error
}

type DeviceStats struct {
	Imported int
	Skipped  int
	Failed   int
	Errors   []string
}

type DeviceFlowInput struct {
	DeviceUUID string
	TargetRoot string
	Files      []FileItem
	DB         DBClient
	Helper     HelperClient
	EXIF       EXIFReader
	Converter  Converter
	MaxFiles   int
	Progress   func(current, total int, name string)
}

func DeviceFlow(input DeviceFlowInput) DeviceStats {
	stats := DeviceStats{}

	sort.Slice(input.Files, func(i, j int) bool {
		return input.Files[i].Created < input.Files[j].Created
	})

	processed := make(map[string]bool)
	total := len(input.Files)
	processedCount := 0

	for i, file := range input.Files {
		if processed[file.Name] {
			continue
		}
		if input.MaxFiles > 0 && processedCount >= input.MaxFiles {
			break
		}
		processedCount++

		if input.Progress != nil {
			input.Progress(i+1, total, file.Name)
		}

		if file.LivePhotoPair != nil {
			pairFile := findFileByName(input.Files, *file.LivePhotoPair)
			if pairFile != nil && !processed[*file.LivePhotoPair] {
				result, _ := (&Importer{
					Helper:    input.Helper,
					EXIF:      input.EXIF,
					Converter: input.Converter,
				}).ImportLivePhotoPair(file, *pairFile, input.DeviceUUID, input.TargetRoot)

			if result.Success {
				for _, p := range result.TargetPaths {
					prefixName := routing.PrefixFilename(file.Name, parseFileCreated(file.Created))
					input.DB.Insert(prefixName, file.Size, time.Now().Format(time.RFC3339), p)
				}
				processed[file.Name] = true
				processed[pairFile.Name] = true
				stats.Imported++
				continue
			}

				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", file.Name, result.Error))
				processed[file.Name] = true
				processed[pairFile.Name] = true
				continue
			}
		}

		prefixName := routing.PrefixFilename(file.Name, parseFileCreated(file.Created))

		if input.DB.IsImported(prefixName, file.Size) {
			stats.Skipped++
			processed[file.Name] = true
			continue
		}

		result, _ := (&Importer{
			Helper:    input.Helper,
			EXIF:      input.EXIF,
			Converter: input.Converter,
		}).ImportFile(file, input.DeviceUUID, input.TargetRoot)

		if result.Success {
			for _, p := range result.TargetPaths {
				input.DB.Insert(prefixName, file.Size, time.Now().Format(time.RFC3339), p)
			}
			stats.Imported++
		} else {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", file.Name, result.Error))
		}
		processed[file.Name] = true
	}

	return stats
}

func findFileByName(files []FileItem, name string) *FileItem {
	for i := range files {
		if files[i].Name == name {
			return &files[i]
		}
	}
	return nil
}

func parseFileCreated(created string) time.Time {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return time.Now()
	}
	return t
}
