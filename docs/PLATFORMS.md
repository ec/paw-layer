# Platform support

`paw-layer` keeps cat behavior platform-neutral and isolates OS/compositor integration behind small native backends.

## Shared core

The shared Go core owns:

- config loading and validation
- cat state and movement
- behavior priority
- sprite frame selection
- command-line entrypoints

The core talks to desktop state through `desktop.Provider`:

```go
type Provider interface {
    Monitors(ctx context.Context) ([]Monitor, error)
    Clients(ctx context.Context) ([]Window, error)
    ActiveWindow(ctx context.Context) (*Window, error)
    Cursor(ctx context.Context) (*Cursor, error)
}
```

Rendering stays behind `renderer.Renderer`.

## Linux / Hyprland

Current native backend:

- desktop provider: `hyprctl` JSON commands
- renderer: GTK4 + `gtk4-layer-shell`
- overlay behavior: transparent, always-on-top layer-shell surface
- input behavior: click-through input region

Linux backend files are selected with `//go:build linux` where native GTK/layer-shell code is required.

## macOS target

The macOS backend should behave analogously to Linux, but use native Apple APIs:

- renderer: AppKit `NSPanel`/`NSWindow`
- overlay behavior: borderless transparent always-on-top panel
- input behavior: `ignoresMouseEvents = true`
- monitor geometry: AppKit/CoreGraphics screen APIs
- cursor position: CoreGraphics/AppKit
- active/frontmost window geometry: Accessibility API and/or `CGWindowList`
- fullscreen detection: window/screen state from Accessibility/CoreGraphics where available

Expected user-facing behavior should match Linux:

- cat walks on the screen frame
- cat sits on the active window
- cat avoids the cursor
- cat sleeps/wakes based on cursor inactivity
- cat hides for fullscreen windows
- sprite packs/config stay shared

## Development rule

New behavior should land in the shared core whenever possible. Platform-specific code should only translate native desktop/rendering APIs into the shared `desktop` and `renderer` models.

This keeps feature work applying to both Linux and macOS instead of creating two separate apps.
