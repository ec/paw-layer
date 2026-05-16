//go:build darwin && !cgo

package platform

import (
	"fmt"
	"log/slog"

	"github.com/ec/paw-layer/internal/config"
	"github.com/ec/paw-layer/internal/desktop"
	"github.com/ec/paw-layer/internal/renderer"
)

func NewDesktopProvider(cfg config.Config, log *slog.Logger) (desktop.Provider, error) {
	_ = cfg
	_ = log
	return macOSDesktopProvider{}, nil
}

func NewRenderer(cfg config.Config, log *slog.Logger) (renderer.Renderer, error) {
	if cfg.Renderer.Backend == "fake" {
		return renderer.NewFake(log), nil
	}
	return nil, fmt.Errorf("macOS renderer %q requires cgo/AppKit", cfg.Renderer.Backend)
}
