//go:build darwin

package platform

import (
	"context"
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
	switch cfg.Renderer.Backend {
	case "fake":
		return renderer.NewFake(log), nil
	case "macos-appkit", "native":
		return nil, fmt.Errorf("macOS AppKit renderer is planned but not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported renderer backend %q on macos", cfg.Renderer.Backend)
	}
}

type macOSDesktopProvider struct{}

func (macOSDesktopProvider) Monitors(ctx context.Context) ([]desktop.Monitor, error) {
	_ = ctx
	return nil, fmt.Errorf("macOS monitor provider is planned but not implemented yet")
}

func (macOSDesktopProvider) Clients(ctx context.Context) ([]desktop.Window, error) {
	_ = ctx
	return nil, fmt.Errorf("macOS window provider is planned but not implemented yet")
}

func (macOSDesktopProvider) ActiveWindow(ctx context.Context) (*desktop.Window, error) {
	_ = ctx
	return nil, fmt.Errorf("macOS active-window provider is planned but not implemented yet")
}

func (macOSDesktopProvider) Cursor(ctx context.Context) (*desktop.Cursor, error) {
	_ = ctx
	return nil, fmt.Errorf("macOS cursor provider is planned but not implemented yet")
}
