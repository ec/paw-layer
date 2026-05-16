# paw-layer

Pixel desktop cats for Hyprland / Wayland.

`paw-layer` is a small Go desktop companion app. A pixel cat lives in a transparent click-through layer-shell overlay, follows the active Hyprland window, sits on it, reacts to the cursor, sleeps after inactivity, and can move between monitors.

> Status: early MVP / technical playground. Linux + Hyprland only for now.

## Features

- Transparent GTK4 + `gtk4-layer-shell` overlay
- Click-through input region, including after monitor switches
- PNG spritesheet renderer with multiple sprite packs
- Active-window tracking via `hyprctl`
- Cat sits on the active window top edge
- Cat hides when the active window is fullscreen
- Cursor avoidance with hysteresis
- Sleep routine after cursor inactivity
- Wake/startle, blink, tail-flick, and short wander micro-actions
- Screen-frame movement: walk along the bottom edge, climb vertically to windows
- Multi-monitor follow with edge-transition behavior
- CLI commands for config validation and Hyprland inspection

## Requirements

- Go 1.25+
- Hyprland
- `hyprctl`
- GTK4 development files
- `gtk4-layer-shell` development files
- `pkg-config`

On Arch Linux / Omarchy-like systems:

```bash
sudo pacman -S go gtk4 gtk4-layer-shell pkgconf
```

## Quick start

```bash
git clone https://github.com/ec/paw-layer.git
cd paw-layer
go run ./cmd/paw-layer run --config configs/default.yaml
```

Stop with `Ctrl-C`.

Build a local binary:

```bash
go build -o paw-layer ./cmd/paw-layer
./paw-layer run --config configs/default.yaml
```

## Commands

```bash
# Run the cat
go run ./cmd/paw-layer run --config configs/default.yaml

# Validate config
go run ./cmd/paw-layer validate-config --config configs/default.yaml

# Inspect Hyprland state
go run ./cmd/paw-layer list-monitors
go run ./cmd/paw-layer list-windows
```

`debug-overlay` is reserved but not implemented yet.

## Configuration

Default config: [`configs/default.yaml`](configs/default.yaml)

Important options:

```yaml
app:
  tick_rate: 30
  debug: true

renderer:
  backend: gtk4-layer-shell
  click_through: true
  transparent: true

cats:
  - name: Miso
    sprite_pack: black
    scale: 3
    speed: 180

behavior:
  sleep_after_sec: 5
  cursor_avoid_radius: 140
  cursor_run_speed: 320
  bottom_edge_inset: 80
```

Notes:

- `sleep_after_sec: 5` is intentionally low for testing. Raise it for normal daily use.
- `bottom_edge_inset` controls how far above the physical bottom edge the cat walks.
- The config parser is intentionally minimal and supports the current config shape only.
- `cats[].behaviors` exists in the config but is not yet a real toggle system.

## Sprite packs

Current packs:

```text
assets/cats/black/       black cat
assets/cats/seal-point/  light seal-point style cat
assets/cats/default/     current default asset copy
```

Each pack contains:

```text
manifest.yaml
idle.png
walk.png
sit.png
sleep.png
climb.png
blink.png
tail_flick.png
wake.png
peek.png
```

Switch pack in config:

```yaml
cats:
  - sprite_pack: seal-point
```

`peek.png` exists for future use, but runtime peek behavior is currently disabled.

## Architecture

```text
cmd/paw-layer/            CLI entrypoint
internal/app/             main loop and behavior orchestration
internal/cat/             cat state and movement primitives
internal/hyprland/        hyprctl-backed desktop state
internal/desktop/         monitor/window/cursor models
internal/renderer/        renderer protocol + GTK layer-shell backend
internal/assets/          sprite manifest loading
internal/physics/         vector helpers
assets/cats/              sprite packs
configs/                  sample config
docs/                     specs and reference notes
```

Renderer design:

- Go core owns behavior and state.
- GTK renderer is intentionally dumb: draw current frame, switch monitor, report viewport.
- GTK integration uses a small direct C bridge instead of gotk4 to keep generated cgo noise and dependencies low.

## Behavior priority

Per tick, behavior roughly resolves as:

1. monitor transition
2. sleep routine / wake animation
3. cursor avoidance
4. random wander break
5. active-window fullscreen hide
6. frame-path movement to active window and sit
7. bottom-edge wander fallback

## Known limitations

- Multi-monitor movement is not truly continuous across outputs. The cat walks to the edge, the layer surface switches monitor, then the cat enters from the corresponding edge.
- True seamless cross-monitor walking likely needs one layer-shell surface per monitor or a different renderer strategy.
- Hyprland state currently uses polling + `hyprctl`; direct IPC/event sockets are future work.
- The config parser is hand-rolled and minimal.
- Sprite art is prototype-quality pixel art.
- No packaging yet.
- No tray or control socket yet.

## Development

Run checks:

```bash
go test ./...
go run ./cmd/paw-layer validate-config --config configs/default.yaml
go build ./cmd/paw-layer
```

Before changing behavior, prefer small iterations: run the app, observe motion, then tune constants/config.

## Roadmap

Near-term:

- behavior toggles from config
- stable tracking when windows move/resize
- animation timing tied to movement speed
- better hand-drawn sprite packs
- normal-use config defaults instead of test tuning

Later:

- direct Hyprland IPC/events
- multiple cats
- hot config reload
- persisted position/monitor state
- tray / `paw-layerctl`
- packaging / AUR

## Contributing

This project is still early, but contributions are welcome.

Good first contributions:

- better sprite packs
- config parser cleanup
- Hyprland IPC/event support
- packaging
- behavior tuning

Please keep changes small and test with:

```bash
go test ./...
go run ./cmd/paw-layer validate-config --config configs/default.yaml
```

## Security / privacy

`paw-layer` reads local Hyprland state via `hyprctl` and renders a local overlay. It does not require network access at runtime.

## License

No license has been selected yet. Until a license is added, all rights are reserved by default.

## References

See [`docs/REFERENCES.md`](docs/REFERENCES.md).
