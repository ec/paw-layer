package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App      AppConfig
	Renderer RendererConfig
	Cats     []CatConfig
	Behavior BehaviorConfig
	Assets   AssetsConfig
}

type AppConfig struct {
	TickRate int
	Debug    bool
}

type RendererConfig struct {
	Backend      string
	Layer        string
	ClickThrough bool
	Transparent  bool
}

type CatConfig struct {
	ID         string
	Name       string
	SpritePack string
	Scale      float64
	Speed      float64
	Behaviors  []string
}

type BehaviorConfig struct {
	CursorAvoidRadius int
	CursorRunSpeed    float64
	SleepAfterSec     int
	BottomEdgeInset   float64
}

type AssetsConfig struct {
	Path string
}

func Default() Config {
	return Config{
		App:      AppConfig{TickRate: 30, Debug: false},
		Renderer: RendererConfig{Backend: "fake", Layer: "overlay", ClickThrough: true, Transparent: true},
		Cats:     []CatConfig{{ID: "default", Name: "Miso", SpritePack: "default", Scale: 3, Speed: 180}},
		Behavior: BehaviorConfig{CursorAvoidRadius: 120, CursorRunSpeed: 280, SleepAfterSec: 180, BottomEdgeInset: 80},
		Assets:   AssetsConfig{Path: "./assets/cats"},
	}
}

// Load reads the small default YAML shape used by this project without pulling dependencies yet.
// It is intentionally conservative and should be replaced by a real YAML parser once dependencies are allowed.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	section := ""
	inFirstCat := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(raw, " ") && strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			inFirstCat = false
			continue
		}
		if section == "cats" && strings.HasPrefix(line, "-") {
			inFirstCat = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"")

		switch section + "." + key {
		case "app.tick_rate":
			cfg.App.TickRate = atoi(value, cfg.App.TickRate)
		case "app.debug":
			cfg.App.Debug = parseBool(value, cfg.App.Debug)
		case "renderer.backend":
			cfg.Renderer.Backend = value
		case "renderer.layer":
			cfg.Renderer.Layer = value
		case "renderer.click_through":
			cfg.Renderer.ClickThrough = parseBool(value, cfg.Renderer.ClickThrough)
		case "renderer.transparent":
			cfg.Renderer.Transparent = parseBool(value, cfg.Renderer.Transparent)
		case "cats.id":
			if inFirstCat {
				cfg.Cats[0].ID = value
			}
		case "cats.name":
			if inFirstCat {
				cfg.Cats[0].Name = value
			}
		case "cats.sprite_pack":
			if inFirstCat {
				cfg.Cats[0].SpritePack = value
			}
		case "cats.scale":
			if inFirstCat {
				cfg.Cats[0].Scale = atof(value, cfg.Cats[0].Scale)
			}
		case "cats.speed":
			if inFirstCat {
				cfg.Cats[0].Speed = atof(value, cfg.Cats[0].Speed)
			}
		case "behavior.cursor_avoid_radius":
			cfg.Behavior.CursorAvoidRadius = atoi(value, cfg.Behavior.CursorAvoidRadius)
		case "behavior.cursor_run_speed":
			cfg.Behavior.CursorRunSpeed = atof(value, cfg.Behavior.CursorRunSpeed)
		case "behavior.sleep_after_sec":
			cfg.Behavior.SleepAfterSec = atoi(value, cfg.Behavior.SleepAfterSec)
		case "behavior.bottom_edge_inset":
			cfg.Behavior.BottomEdgeInset = atof(value, cfg.Behavior.BottomEdgeInset)
		case "assets.path":
			cfg.Assets.Path = value
		}
	}

	return cfg, Validate(cfg)
}

func Validate(cfg Config) error {
	if cfg.App.TickRate <= 0 || cfg.App.TickRate > 240 {
		return fmt.Errorf("app.tick_rate must be between 1 and 240")
	}
	if len(cfg.Cats) == 0 {
		return fmt.Errorf("at least one cat must be configured")
	}
	for _, cat := range cfg.Cats {
		if cat.ID == "" {
			return fmt.Errorf("cat.id is required")
		}
		if cat.Speed <= 0 {
			return fmt.Errorf("cat %q speed must be positive", cat.ID)
		}
		if cat.Scale <= 0 {
			return fmt.Errorf("cat %q scale must be positive", cat.ID)
		}
	}
	return nil
}

func atoi(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func atof(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
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
