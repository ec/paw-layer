package hyprland

import (
	"context"

	"github.com/ec/paw-layer/internal/desktop"
)

type Event struct {
	Name string
	Data string
}

type Client interface {
	Clients(ctx context.Context) ([]desktop.Window, error)
	Monitors(ctx context.Context) ([]desktop.Monitor, error)
	ActiveWindow(ctx context.Context) (*desktop.Window, error)
	Cursor(ctx context.Context) (*desktop.Cursor, error)
	Subscribe(ctx context.Context) (<-chan Event, error)
}
