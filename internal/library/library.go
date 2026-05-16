package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Item struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Format   string `json:"format"`
	Position int    `json:"position"`
}

func Scan(sourceDir string, formats []string) ([]Item, error) {
	info, err := os.Stat(sourceDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("video source path is not a directory")
	}

	allowed := make(map[string]struct{}, len(formats))
	for _, format := range formats {
		allowed[strings.ToLower(strings.TrimSpace(format))] = struct{}{}
	}

	var items []Item
	err = filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := allowed[ext]; !ok {
			return nil
		}

		items = append(items, Item{
			ID:     path,
			Path:   path,
			Name:   strings.TrimSuffix(filepath.Base(path), ext),
			Format: ext,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	for i := range items {
		items[i].Position = i
	}

	return items, nil
}
