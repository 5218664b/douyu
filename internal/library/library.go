package library

import (
	"errors"
	"io/fs"
	"net/url"
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
	if strings.TrimSpace(sourceDir) == "" {
		return nil, nil
	}

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

func FromURLs(urls []string) []Item {
	items := make([]Item, 0, len(urls))
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		items = append(items, Item{
			ID:     raw,
			Path:   raw,
			Name:   urlDisplayName(raw),
			Format: strings.ToLower(filepath.Ext(raw)),
		})
	}

	for i := range items {
		items[i].Position = i
	}

	return items
}

func Merge(groups ...[]Item) []Item {
	var merged []Item
	position := 0
	for _, group := range groups {
		for _, item := range group {
			item.Position = position
			merged = append(merged, item)
			position++
		}
	}
	return merged
}

func urlDisplayName(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	base := filepath.Base(parsed.Path)
	if base == "." || base == "/" || base == "" {
		if parsed.Host != "" {
			return raw
		}
		return parsed.Path
	}

	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
