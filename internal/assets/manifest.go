package assets

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Manifest struct {
	Name       string
	TileWidth  int
	TileHeight int
	Animations map[string]Animation
}

type Animation struct {
	Name   string
	File   string
	Frames int
	FPS    int
	Loop   bool
}

func LoadManifest(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()

	manifest := Manifest{Animations: make(map[string]Animation)}
	var current *Animation

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasSuffix(line, ":") {
			name := strings.TrimSuffix(line, ":")
			if name != "animations" && strings.HasPrefix(raw, "  ") {
				anim := Animation{Name: name, Frames: 1, FPS: 1, Loop: true}
				manifest.Animations[name] = anim
				current = &anim
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"")

		if current != nil && strings.HasPrefix(raw, "    ") {
			switch key {
			case "file":
				current.File = value
			case "frames":
				current.Frames = atoi(value, current.Frames)
			case "fps":
				current.FPS = atoi(value, current.FPS)
			case "loop":
				current.Loop = parseBool(value, current.Loop)
			}
			manifest.Animations[current.Name] = *current
			continue
		}

		current = nil
		switch key {
		case "name":
			manifest.Name = value
		case "tile_width":
			manifest.TileWidth = atoi(value, manifest.TileWidth)
		case "tile_height":
			manifest.TileHeight = atoi(value, manifest.TileHeight)
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, err
	}

	if manifest.Name == "" {
		manifest.Name = filepath.Base(filepath.Dir(path))
	}
	if manifest.TileWidth <= 0 || manifest.TileHeight <= 0 {
		return Manifest{}, fmt.Errorf("manifest %s must define positive tile_width/tile_height", path)
	}
	for name, anim := range manifest.Animations {
		if anim.File == "" {
			return Manifest{}, fmt.Errorf("animation %q must define file", name)
		}
		if anim.Frames <= 0 || anim.FPS <= 0 {
			return Manifest{}, fmt.Errorf("animation %q must define positive frames/fps", name)
		}
	}
	return manifest, nil
}

func atoi(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
