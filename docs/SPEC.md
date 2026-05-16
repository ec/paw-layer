# hyprcats project spec

## Goal

Build a Linux desktop companion app for Omarchy/Hyprland: pixel cats live above the desktop, walk around, sit on windows, peek from window edges, and react to the environment.

## Target platform

- OS: Arch / Omarchy
- Compositor: Hyprland
- Session: Wayland
- Core language: Go
- Initial renderer target: GTK4 + layer-shell, likely via Go bindings if viable

Hyprland is the initial target because it exposes JSON state and IPC through UNIX sockets. Overlay windows on Wayland require `wlr-layer-shell`-style surfaces.

## MVP scope

### v0.1

- 1 cat
- transparent overlay window
- always-on-top layer
- click-through
- cat walks on screen
- idle / walk / sleep animations
- YAML config
- Hyprland-only

### v0.2

- read Hyprland window geometry
- cat can sit on active window top edge
- cat avoids cursor
- cat hides on fullscreen
- multiple cats

### v0.3

- behavior scripting
- custom sprite packs
- config reload without restart
- tray / CLI control

## Non-goals initially

- GNOME/KDE/X11 support
- AI assistant behavior
- real browser tab awareness
- complex physics
- sprite editor

Browser tab interaction is not generally available from the compositor. We simulate this through active-window edges.

## Architecture

Go owns domain state and behavior. Renderer is intentionally dumb.

Core responsibilities:

- read Hyprland state/events
- track monitors/windows/workspaces
- update cat FSM and movement
- choose animation frames
- send render state

Renderer responsibilities:

- create transparent layer-shell surface
- draw sprites
- avoid business logic

Initial process model is a single binary: `hyprcats`. Future split can introduce `hyprcats-daemon`, `hyprcats-renderer`, and `hyprcatsctl`.

## Package layout

```text
cmd/hyprcats/              CLI entrypoint
internal/app/              orchestration and main loop
internal/config/           config loading/validation
internal/hyprland/         Hyprland client and event parsing
internal/desktop/          monitor/window/workspace models
internal/cat/              cat FSM, behavior, animation state
internal/physics/          movement and geometry primitives
internal/renderer/         renderer protocol and implementations
internal/assets/           sprite manifests and asset loading
configs/                   sample configs
docs/                      specs and design notes
assets/cats/default/       default sprite pack placeholder
```

## Main loop

1. Load config
2. Connect to Hyprland state provider
3. Start renderer overlay
4. Start event listener
5. Start simulation loop
6. Every tick:
   - refresh desktop state as needed
   - update cats
   - resolve movement/collisions
   - render frame

Target rates:

- logic: 30 FPS
- render: 60 FPS if feasible

## Hyprland integration

Initial implementation may shell out to:

- `hyprctl -j clients`
- `hyprctl -j monitors`
- `hyprctl -j activewindow`
- `hyprctl -j workspaces`

Later, replace command calls with direct UNIX socket IPC. Hyprland exposes one socket for commands and another for events.

## Core interfaces

```go
type HyprlandClient interface {
    Clients(ctx context.Context) ([]desktop.Window, error)
    Monitors(ctx context.Context) ([]desktop.Monitor, error)
    ActiveWindow(ctx context.Context) (*desktop.Window, error)
    Subscribe(ctx context.Context) (<-chan HyprEvent, error)
}
```

```go
type Renderer interface {
    Init(ctx context.Context, cfg Config) error
    Draw(ctx context.Context, frame Frame) error
    Close() error
}
```

## Acceptance criteria for MVP

MVP is ready when:

- app runs on Omarchy/Hyprland
- a transparent pixel cat appears above the desktop
- clicks pass through the overlay
- cat walks around
- cat changes idle/walk/sleep animation
- cat can sit on active window
- cat hides during fullscreen
- config controls speed/scale/sprite path

## References

See [REFERENCES.md](./REFERENCES.md) for notes from similar projects:

- clawd-on-desk
- CATAI

## Technical spike order

Validate these risks first:

1. transparent layer-shell window on Omarchy
2. click-through input region
3. Go can read Hyprland clients/activewindow reliably

Milestone 0: show a PNG cat moving horizontally every frame in a transparent click-through window. Until layer-shell is proven, use a fake renderer to validate the core loop.
