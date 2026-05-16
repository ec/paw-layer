# hyprcats

Pixel desktop cats for Hyprland / Wayland.

`hyprcats` is a small Go desktop companion: a pixel cat lives in a transparent click-through layer-shell overlay, follows the active Hyprland window, sits on it, reacts to the cursor, sleeps after inactivity, and can move between monitors.

## Status

Early MVP / technical playground.

Working:

- GTK4 + `gtk4-layer-shell` transparent overlay
- click-through input region
- PNG spritesheet rendering
- active-window tracking via `hyprctl`
- sit on active window top center
- hide on fullscreen active window
- cursor avoidance with hysteresis
- sleep after cursor inactivity
- multi-monitor follow with edge-transition behavior
- basic CLI config validation and Hyprland inspection commands

## Requirements

Runtime/build dependencies:

- Go 1.25+
- Hyprland
- `hyprctl`
- GTK4 development package
- `gtk4-layer-shell` development package
- pkg-config

On Arch-like systems, the relevant packages are typically:

```bash
sudo pacman -S go gtk4 gtk4-layer-shell pkgconf
```

## Run

From the repo root:

```bash
go run ./cmd/hyprcats run --config configs/default.yaml
```

Stop with `Ctrl-C`.

Build a local binary:

```bash
go build -o hyprcats ./cmd/hyprcats
./hyprcats run --config configs/default.yaml
```

## Commands

```bash
go run ./cmd/hyprcats run --config configs/default.yaml

go run ./cmd/hyprcats validate-config --config configs/default.yaml

go run ./cmd/hyprcats list-monitors

go run ./cmd/hyprcats list-windows
```

`debug-overlay` is reserved but not implemented yet.

## Configuration

Default config: [`configs/default.yaml`](configs/default.yaml)

Important knobs:

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
    sprite_pack: default
    scale: 3
    speed: 180

behavior:
  sleep_after_sec: 5
  cursor_avoid_radius: 140
  cursor_run_speed: 320
```

Notes:

- `sleep_after_sec: 5` is intentionally low for testing. Raise it later for normal use.
- The config parser is intentionally minimal right now and supports only the current shape.
- `cats[].behaviors` is present in config but not yet used as a real behavior toggle system.

## Assets

Default sprite pack:

```text
assets/cats/default/
  manifest.yaml
  idle.png
  walk.png
  sit.png
  sleep.png
  peek.png
```

The current default style is a black pixel cat with yellow eyes, pink ears, and a red collar.

`peek.png` exists but runtime peek behavior is disabled for now.

## Architecture

High-level packages:

```text
cmd/hyprcats/          CLI entrypoint
internal/app/          main loop and behavior orchestration
internal/cat/          cat state and movement primitives
internal/hyprland/     hyprctl-backed desktop state
internal/desktop/      monitor/window/cursor models
internal/renderer/     renderer protocol + GTK layer-shell backend
internal/assets/       sprite manifest loading
internal/physics/      vector helpers
```

Renderer design:

- Go core owns behavior and state.
- GTK renderer is intentionally dumb: draw current frame, switch monitor, report viewport.
- GTK integration is a small direct C bridge, not gotk4, to avoid noisy generated cgo warnings and keep the renderer narrow.

## Current behavior priority

Per tick, behavior roughly resolves as:

1. monitor transition
2. sleep after cursor inactivity
3. cursor avoidance
4. active-window fullscreen hide
5. sit on active window
6. wander fallback

## Known issues / limitations

- Multi-monitor movement is not truly continuous across outputs. The cat walks to the edge, the layer surface switches monitor, then the cat enters from the corresponding edge.
- Real seamless cross-monitor walking likely needs one layer-shell surface per monitor or a different renderer strategy.
- Hyprland state currently uses polling + `hyprctl`; direct IPC/event socket is still future work.
- The config parser is hand-rolled and minimal.
- Sprite art is generated/prototype quality, not final hand-drawn art.
- No packaging yet.
- No tray or control socket yet.

## Roadmap

Near-term:

- better behavior toggles from config
- stable window tracking when windows move/resize
- cleaner animation timing tied to movement speed
- more polished sprite pack
- config defaults for real use, not test tuning

Later:

- direct Hyprland IPC/events
- multiple cats
- hot config reload
- proper theme/sprite pack format
- tray / `hyprcatsctl`
- packaging / AUR

## References

See [`docs/REFERENCES.md`](docs/REFERENCES.md).
# paw-layer
