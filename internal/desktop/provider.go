package desktop

import "context"

// Provider exposes native desktop state used by the shared cat behavior loop.
// Implementations are platform-specific: Hyprland on Linux, AppKit/CoreGraphics
// on macOS, and potentially other compositors/desktops later.
type Provider interface {
	Monitors(ctx context.Context) ([]Monitor, error)
	Clients(ctx context.Context) ([]Window, error)
	ActiveWindow(ctx context.Context) (*Window, error)
	Cursor(ctx context.Context) (*Cursor, error)
}
