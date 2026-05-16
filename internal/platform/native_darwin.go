//go:build darwin && cgo

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
	return newMacOSDesktopProvider(log), nil
}

func NewRenderer(cfg config.Config, log *slog.Logger) (renderer.Renderer, error) {
	switch cfg.Renderer.Backend {
	case "fake":
		return renderer.NewFake(log), nil
	case "macos-appkit", "native":
		return renderer.NewMacOSAppKit(log), nil
	default:
		return nil, fmt.Errorf("unsupported renderer backend %q on macos", cfg.Renderer.Backend)
	}
}
