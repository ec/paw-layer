//go:build linux

package platform

import (
	"fmt"
	"log/slog"

	"github.com/ec/paw-layer/internal/config"
	"github.com/ec/paw-layer/internal/desktop"
	"github.com/ec/paw-layer/internal/hyprland"
	"github.com/ec/paw-layer/internal/renderer"
)

func NewDesktopProvider(cfg config.Config, log *slog.Logger) (desktop.Provider, error) {
	_ = cfg
	_ = log
	return hyprland.NewHyprctl(), nil
}

func NewRenderer(cfg config.Config, log *slog.Logger) (renderer.Renderer, error) {
	switch cfg.Renderer.Backend {
	case "fake":
		return renderer.NewFake(log), nil
	case "gtk4-layer-shell":
		return renderer.NewGTKLayerShell(log), nil
	default:
		return nil, fmt.Errorf("unsupported renderer backend %q on linux", cfg.Renderer.Backend)
	}
}
