package app

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/ec/paw-layer/internal/cat"
	"github.com/ec/paw-layer/internal/config"
	"github.com/ec/paw-layer/internal/desktop"
	"github.com/ec/paw-layer/internal/physics"
	"github.com/ec/paw-layer/internal/renderer"
)

type App struct {
	cfg      config.Config
	desktop  desktop.Provider
	renderer renderer.Renderer
	log      *slog.Logger
}

type monitorTransition struct {
	active bool
	to     desktop.Monitor
	exit   physics.Vec2
	enter  physics.Vec2
}

func New(cfg config.Config, desktopProvider desktop.Provider, r renderer.Renderer, log *slog.Logger) *App {
	return &App{cfg: cfg, desktop: desktopProvider, renderer: r, log: log}
}

func (a *App) Run(ctx context.Context) error {
	monitor := a.detectInitialMonitor(ctx)
	if err := a.renderer.Init(ctx, renderer.Config{
		Backend:       a.cfg.Renderer.Backend,
		Layer:         a.cfg.Renderer.Layer,
		ClickThrough:  a.cfg.Renderer.ClickThrough,
		Transparent:   a.cfg.Renderer.Transparent,
		AssetsPath:    a.cfg.Assets.Path,
		InitialWidth:  monitor.Width,
		InitialHeight: monitor.Height,
	}); err != nil {
		return err
	}
	defer func() {
		if err := a.renderer.Close(); err != nil {
			a.log.WarnContext(ctx, "renderer.close_failed", "error", err)
		}
	}()

	catCfg := a.cfg.Cats[0]
	miso := cat.New(catCfg.ID, catCfg.Name, catCfg.Speed, catCfg.Scale)
	// Start visible; once GTK reports the real viewport, the pathing code will
	// move the cat to the bottom edge. Hyprland monitor height may differ from
	// the layer-surface logical viewport on scaled/multi-monitor setups.
	miso.Position.Y = 120
	boundsWidth := float64(monitor.Width) - spriteWidth(catCfg.Scale)
	if boundsWidth < 100 {
		boundsWidth = 800
	}

	currentMonitor := monitor
	var activeWindow *desktop.Window
	var cursor *desktop.Cursor
	var lastCursor *desktop.Cursor
	lastCursorMove := time.Now()
	var transition monitorTransition
	var avoidTarget physics.Vec2
	avoidUntil := time.Time{}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	nextMicroAction := time.Now().Add(randomDuration(rng, 4*time.Second, 9*time.Second))
	microActionUntil := time.Time{}
	microAnimation := ""
	wanderActive := false
	wanderTarget := physics.Vec2{}
	nextWanderBreak := time.Now().Add(randomDuration(rng, 12*time.Second, 24*time.Second))
	wakeUntil := time.Time{}
	wasSleeping := false
	sleepRoutineActive := false
	sleepTarget := physics.Vec2{}
	pollInterval := 500 * time.Millisecond
	nextDesktopPoll := time.Time{}

	tickRate := time.Duration(a.cfg.App.TickRate)
	ticker := time.NewTicker(time.Second / tickRate)
	defer ticker.Stop()

	last := time.Now()
	frameIndex := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds()
			last = now
			boundsWidth = a.currentBoundsWidth(boundsWidth, catCfg.Scale)
			viewportHeight := a.currentViewportHeight(currentMonitor.Height)
			bottomInset := a.cfg.Behavior.BottomEdgeInset

			if now.After(nextDesktopPoll) {
				if nextCursor, err := a.desktop.Cursor(ctx); err == nil {
					if lastCursor == nil || nextCursor.X != lastCursor.X || nextCursor.Y != lastCursor.Y {
						lastCursorMove = now
						if wasSleeping {
							wakeUntil = now.Add(800 * time.Millisecond)
							wasSleeping = false
						}
					}
					cursor = nextCursor
					lastCursor = nextCursor
				} else {
					a.log.DebugContext(ctx, "desktop.cursor_unavailable", "error", err)
				}

				window, err := a.desktop.ActiveWindow(ctx)
				if err != nil {
					a.log.DebugContext(ctx, "desktop.active_window_unavailable", "error", err)
				} else {
					activeWindow = window
					if window != nil && !transition.active {
						if nextMonitor, ok := a.monitorForWindow(ctx, *window); ok && nextMonitor.Name != currentMonitor.Name {
							transition = a.newMonitorTransition(currentMonitor, nextMonitor, miso.Position, catCfg.Scale, boundsWidth)
							a.log.InfoContext(ctx, "cat.monitor_transition_started", "from", currentMonitor.Name, "to", nextMonitor.Name, "exit_x", transition.exit.X, "exit_y", transition.exit.Y)
						}
					}
				}
				nextDesktopPoll = now.Add(pollInterval)
			}

			visible := true
			if transition.active {
				if miso.MoveToward(dt, transition.exit) {
					if switcher, ok := a.renderer.(renderer.MonitorSwitcher); ok {
						if err := switcher.SwitchMonitor(transition.to.Name, transition.to.Width, transition.to.Height); err != nil {
							a.log.WarnContext(ctx, "renderer.monitor_switch_failed", "error", err, "monitor", transition.to.Name)
						} else {
							currentMonitor = transition.to
							miso.Position = transition.enter
							boundsWidth = float64(transition.to.Width) - spriteWidth(catCfg.Scale)
							if boundsWidth < 100 {
								boundsWidth = float64(transition.to.Width)
							}
							a.log.InfoContext(ctx, "cat.monitor_transition_finished", "to", currentMonitor.Name, "enter_x", transition.enter.X, "enter_y", transition.enter.Y)
						}
					}
					transition.active = false
				}
			} else if a.shouldSleep(now, lastCursorMove) {
				if !sleepRoutineActive {
					sleepTarget = a.sleepSpot(miso.Position, boundsWidth, viewportHeight, bottomInset, catCfg.Scale)
					sleepRoutineActive = true
					a.log.InfoContext(ctx, "cat.sleep_routine_started", "x", sleepTarget.X, "y", sleepTarget.Y)
				}
				if a.moveAlongScreenFrame(dt, &miso, sleepTarget, currentMonitor, viewportHeight, bottomInset, catCfg.Scale, boundsWidth, catCfg.Speed*0.7, false) {
					miso.Sleep()
					wasSleeping = true
				}
			} else if now.Before(wakeUntil) {
				sleepRoutineActive = false
				miso.Animation = "wake"
			} else if runTarget, ok := a.cursorAvoidTarget(cursor, currentMonitor, miso.Position, catCfg.Scale, boundsWidth); ok {
				sleepRoutineActive = false
				avoidTarget = runTarget
				avoidUntil = now.Add(900 * time.Millisecond)
				miso.MoveTowardWithSpeed(dt, avoidTarget, a.cfg.Behavior.CursorRunSpeed, false)
			} else if now.Before(avoidUntil) {
				if miso.MoveTowardWithSpeed(dt, avoidTarget, a.cfg.Behavior.CursorRunSpeed, false) {
					avoidUntil = time.Time{}
				}
			} else if wanderActive {
				if a.moveAlongScreenFrame(dt, &miso, wanderTarget, currentMonitor, viewportHeight, bottomInset, catCfg.Scale, boundsWidth, catCfg.Speed*0.75, false) {
					wanderActive = false
					nextWanderBreak = now.Add(randomDuration(rng, 12*time.Second, 24*time.Second))
				}
			} else if target, ok, fullscreen := a.windowTarget(activeWindow, currentMonitor, catCfg.Scale, boundsWidth); fullscreen {
				visible = false
				miso.Hide()
			} else if ok {
				a.moveAlongScreenFrame(dt, &miso, target, currentMonitor, viewportHeight, bottomInset, catCfg.Scale, boundsWidth, catCfg.Speed, true)
				if miso.State == cat.StateSit {
					a.lookAtCursor(cursor, currentMonitor, &miso, catCfg.Scale)
					if now.After(nextWanderBreak) {
						wanderTarget = a.wanderBreakTarget(rng, miso.Position, boundsWidth, currentMonitor, catCfg.Scale)
						wanderActive = true
					}
					if now.After(nextMicroAction) {
						microAnimation = randomMicroAnimation(rng)
						microActionUntil = now.Add(700 * time.Millisecond)
						nextMicroAction = now.Add(randomDuration(rng, 4*time.Second, 9*time.Second))
					}
					if now.Before(microActionUntil) {
						miso.Animation = microAnimation
					}
				}
			} else {
				miso.Update(dt, boundsWidth)
			}
			frameIndex++

			if err := a.renderer.Draw(ctx, renderer.Frame{Cats: []renderer.CatRenderState{{
				ID:         miso.ID,
				X:          int(miso.Position.X),
				Y:          int(miso.Position.Y),
				Scale:      miso.Scale,
				SpritePack: catCfg.SpritePack,
				Sprite:     miso.Animation,
				Frame:      frameIndex / 4,
				Direction:  string(miso.Direction),
				Visible:    visible,
			}}}); err != nil {
				return err
			}
		}
	}
}

func (a *App) detectInitialMonitor(ctx context.Context) desktop.Monitor {
	monitors, err := a.desktop.Monitors(ctx)
	if err != nil {
		a.log.WarnContext(ctx, "desktop.monitors_unavailable", "error", err, "fallback_width", 800, "fallback_height", 600)
		return desktop.Monitor{Width: 800, Height: 600}
	}

	for _, monitor := range monitors {
		if monitor.Focused && monitor.Width > 0 && monitor.Height > 0 {
			return monitor
		}
	}
	for _, monitor := range monitors {
		if monitor.X == 0 && monitor.Y == 0 && monitor.Width > 0 && monitor.Height > 0 {
			return monitor
		}
	}
	if len(monitors) > 0 && monitors[0].Width > 0 && monitors[0].Height > 0 {
		return monitors[0]
	}
	return desktop.Monitor{Width: 800, Height: 600}
}

func (a *App) currentBoundsWidth(fallback float64, scale float64) float64 {
	viewport, ok := a.renderer.(renderer.ViewportProvider)
	if !ok {
		return fallback
	}

	width, _, ok := viewport.Viewport()
	if !ok || width <= 0 {
		return fallback
	}

	bounds := float64(width) - spriteWidth(scale)
	if bounds < 100 {
		return float64(width)
	}
	return bounds
}

func (a *App) windowTarget(window *desktop.Window, monitor desktop.Monitor, scale float64, boundsWidth float64) (physics.Vec2, bool, bool) {
	if window == nil || window.Address == "" {
		return physics.Vec2{}, false, false
	}
	// Do not hide on fullscreen for now. Some native backends cannot report
	// fullscreen state reliably yet, and the current product direction is to
	// keep the cat visible while macOS support is being brought up.
	if window.Width <= 0 || window.Height <= 0 {
		return physics.Vec2{}, false, false
	}

	catWidth := spriteWidth(scale)
	catHeight := spriteHeight(scale)
	localX := float64(window.X - monitor.X)
	localY := float64(window.Y - monitor.Y)
	targetX := localX + float64(window.Width)/2 - catWidth/2
	targetY := localY - catHeight + 4

	if math.IsNaN(targetX) || math.IsNaN(targetY) {
		return physics.Vec2{}, false, false
	}
	if targetX < 0 {
		targetX = 0
	}
	if targetX > boundsWidth {
		targetX = boundsWidth
	}
	if targetY < 0 {
		targetY = 0
	}
	return physics.Vec2{X: targetX, Y: targetY}, true, false
}

func spriteWidth(scale float64) float64  { return 32 * scale }
func spriteHeight(scale float64) float64 { return 32 * scale }

func (a *App) monitorForWindow(ctx context.Context, window desktop.Window) (desktop.Monitor, bool) {
	monitors, err := a.desktop.Monitors(ctx)
	if err != nil {
		a.log.DebugContext(ctx, "desktop.monitors_unavailable", "error", err)
		return desktop.Monitor{}, false
	}
	if len(monitors) == 0 {
		return desktop.Monitor{}, false
	}

	windowCenterX := window.X + window.Width/2
	windowCenterY := window.Y + window.Height/2
	for _, monitor := range monitors {
		if windowCenterX >= monitor.X && windowCenterX < monitor.X+monitor.Width && windowCenterY >= monitor.Y && windowCenterY < monitor.Y+monitor.Height {
			return monitor, true
		}
	}

	// Fallback: choose the monitor with the largest intersection area.
	best := monitors[0]
	bestArea := -1
	for _, monitor := range monitors {
		area := intersectionArea(window.X, window.Y, window.Width, window.Height, monitor.X, monitor.Y, monitor.Width, monitor.Height)
		if area > bestArea {
			best = monitor
			bestArea = area
		}
	}
	return best, true
}

func intersectionArea(ax, ay, aw, ah, bx, by, bw, bh int) int {
	x1 := maxInt(ax, bx)
	y1 := maxInt(ay, by)
	x2 := minInt(ax+aw, bx+bw)
	y2 := minInt(ay+ah, by+bh)
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	return (x2 - x1) * (y2 - y1)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *App) newMonitorTransition(from, to desktop.Monitor, position physics.Vec2, scale float64, fromBoundsWidth float64) monitorTransition {
	catW := spriteWidth(scale)
	catH := spriteHeight(scale)
	toBoundsWidth := float64(to.Width) - catW
	if toBoundsWidth < 0 {
		toBoundsWidth = float64(to.Width)
	}
	toBoundsHeight := float64(to.Height) - catH
	if toBoundsHeight < 0 {
		toBoundsHeight = float64(to.Height)
	}

	exit := position
	enter := position

	fromCenterX := from.X + from.Width/2
	fromCenterY := from.Y + from.Height/2
	toCenterX := to.X + to.Width/2
	toCenterY := to.Y + to.Height/2
	dx := toCenterX - fromCenterX
	dy := toCenterY - fromCenterY

	if absInt(dx) >= absInt(dy) {
		if dx >= 0 {
			exit.X = fromBoundsWidth
			enter.X = 0
		} else {
			exit.X = 0
			enter.X = toBoundsWidth
		}
		exit.Y = clampFloat(position.Y, 0, float64(from.Height)-catH)
		enter.Y = clampFloat(position.Y+float64(from.Y-to.Y), 0, toBoundsHeight)
	} else {
		if dy >= 0 {
			exit.Y = float64(from.Height) - catH
			enter.Y = 0
		} else {
			exit.Y = 0
			enter.Y = toBoundsHeight
		}
		exit.X = clampFloat(position.X, 0, fromBoundsWidth)
		enter.X = clampFloat(position.X+float64(from.X-to.X), 0, toBoundsWidth)
	}

	return monitorTransition{active: true, to: to, exit: exit, enter: enter}
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (a *App) cursorAvoidTarget(cursor *desktop.Cursor, monitor desktop.Monitor, catPosition physics.Vec2, scale float64, boundsWidth float64) (physics.Vec2, bool) {
	if cursor == nil || a.cfg.Behavior.CursorAvoidRadius <= 0 {
		return physics.Vec2{}, false
	}

	catW := spriteWidth(scale)
	catH := spriteHeight(scale)
	catCenter := physics.Vec2{X: catPosition.X + catW/2, Y: catPosition.Y + catH/2}
	cursorLocal := physics.Vec2{X: float64(cursor.X - monitor.X), Y: float64(cursor.Y - monitor.Y)}
	if cursorLocal.X < 0 || cursorLocal.Y < 0 || cursorLocal.X > float64(monitor.Width) || cursorLocal.Y > float64(monitor.Height) {
		return physics.Vec2{}, false
	}

	dx := catCenter.X - cursorLocal.X
	dy := catCenter.Y - cursorLocal.Y
	distance := math.Hypot(dx, dy)
	if distance <= 0 || distance > float64(a.cfg.Behavior.CursorAvoidRadius) {
		return physics.Vec2{}, false
	}

	if distance < 1 {
		dx = 1
		dy = 0
		distance = 1
	}
	runDistance := float64(a.cfg.Behavior.CursorAvoidRadius) * 1.5
	target := physics.Vec2{
		X: catPosition.X + dx/distance*runDistance,
		Y: catPosition.Y + dy/distance*runDistance,
	}
	maxY := float64(monitor.Height) - catH
	target.X = clampFloat(target.X, 0, boundsWidth)
	target.Y = clampFloat(target.Y, 0, maxY)
	return target, true
}

func (a *App) shouldSleep(now time.Time, lastCursorMove time.Time) bool {
	if a.cfg.Behavior.SleepAfterSec <= 0 {
		return false
	}
	return now.Sub(lastCursorMove) >= time.Duration(a.cfg.Behavior.SleepAfterSec)*time.Second
}

func (a *App) lookAtCursor(cursor *desktop.Cursor, monitor desktop.Monitor, c *cat.Cat, scale float64) {
	if cursor == nil {
		return
	}
	cursorLocalX := float64(cursor.X - monitor.X)
	cursorLocalY := float64(cursor.Y - monitor.Y)
	if cursorLocalX < 0 || cursorLocalY < 0 || cursorLocalX > float64(monitor.Width) || cursorLocalY > float64(monitor.Height) {
		return
	}
	catCenterX := c.Position.X + spriteWidth(scale)/2
	if math.Abs(cursorLocalX-catCenterX) > 360 {
		return
	}
	if cursorLocalX < catCenterX {
		c.Direction = cat.DirectionLeft
	} else {
		c.Direction = cat.DirectionRight
	}
}

func (a *App) wanderBreakTarget(rng *rand.Rand, position physics.Vec2, boundsWidth float64, monitor desktop.Monitor, scale float64) physics.Vec2 {
	distance := 50 + rng.Float64()*120
	if rng.Intn(2) == 0 {
		distance = -distance
	}
	maxY := float64(monitor.Height) - spriteHeight(scale)
	return physics.Vec2{
		X: clampFloat(position.X+distance, 0, boundsWidth),
		Y: clampFloat(position.Y+rng.Float64()*24-12, 0, maxY),
	}
}

func randomMicroAnimation(rng *rand.Rand) string {
	if rng.Intn(3) == 0 {
		return "tail_flick"
	}
	return "blink"
}

func randomDuration(rng *rand.Rand, minValue time.Duration, maxValue time.Duration) time.Duration {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + time.Duration(rng.Int63n(int64(maxValue-minValue)))
}

func (a *App) moveAlongScreenFrame(dt float64, c *cat.Cat, target physics.Vec2, monitor desktop.Monitor, viewportHeight int, bottomInset float64, scale float64, boundsWidth float64, speed float64, sitOnArrival bool) bool {
	ground := groundY(viewportHeight, scale, bottomInset)
	climbX := clampFloat(target.X, 0, boundsWidth)
	climbThreshold := 5.0
	groundThreshold := 5.0

	// If the cat is away from the bottom and not yet at the climb column, climb
	// down first. This prevents diagonal air-walking between window perches.
	if math.Abs(c.Position.Y-ground) > groundThreshold && math.Abs(c.Position.X-climbX) > climbThreshold {
		waypoint := physics.Vec2{X: c.Position.X, Y: ground}
		c.MoveTowardWithSpeed(dt, waypoint, speed, false)
		return false
	}

	// Walk along the bottom edge until below the target X.
	if math.Abs(c.Position.X-climbX) > climbThreshold {
		waypoint := physics.Vec2{X: climbX, Y: ground}
		c.MoveTowardWithSpeed(dt, waypoint, speed, false)
		return false
	}

	// Climb vertically to the target. Only final arrival uses sit animation.
	arrived := c.MoveTowardWithSpeed(dt, target, speed*0.8, sitOnArrival)
	if !arrived {
		c.Animation = "climb"
	}
	return arrived
}

func (a *App) currentViewportHeight(fallback int) int {
	viewport, ok := a.renderer.(renderer.ViewportProvider)
	if !ok {
		return fallback
	}
	_, height, ok := viewport.Viewport()
	if !ok || height <= 0 {
		return fallback
	}
	return height
}

func groundY(viewportHeight int, scale float64, bottomInset float64) float64 {
	// Keep the cat on the visible bottom frame, not exactly at the layer-surface
	// edge. Some Wayland/Hyprland scale/output combinations report a viewport
	// whose last logical pixels are effectively below the visible desktop edge.
	if bottomInset < 0 {
		bottomInset = 0
	}
	return math.Max(0, float64(viewportHeight)-spriteHeight(scale)-bottomInset)
}

func (a *App) sleepSpot(position physics.Vec2, boundsWidth float64, viewportHeight int, bottomInset float64, scale float64) physics.Vec2 {
	ground := groundY(viewportHeight, scale, bottomInset)
	left := 24.0
	right := math.Max(left, boundsWidth-24)
	if math.Abs(position.X-left) <= math.Abs(position.X-right) {
		return physics.Vec2{X: left, Y: ground}
	}
	return physics.Vec2{X: right, Y: ground}
}
