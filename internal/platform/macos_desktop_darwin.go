//go:build darwin

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ec/paw-layer/internal/desktop"
)

type macOSDesktopProvider struct {
	log *slog.Logger

	mu               sync.RWMutex
	activeWindow     *desktop.Window
	activeWindowErr  error
	activeRefreshAt  time.Time
	activeRefreshing bool
}

func newMacOSDesktopProvider(log *slog.Logger) *macOSDesktopProvider {
	return &macOSDesktopProvider{log: log}
}

func (p *macOSDesktopProvider) Monitors(ctx context.Context) ([]desktop.Monitor, error) {
	out, err := exec.CommandContext(ctx, "osascript", "-e", `tell application "Finder" to get bounds of window of desktop`).Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("parse desktop bounds %q", strings.TrimSpace(string(out)))
	}
	left, err := parseAppleScriptInt(parts[0])
	if err != nil {
		return nil, err
	}
	top, err := parseAppleScriptInt(parts[1])
	if err != nil {
		return nil, err
	}
	right, err := parseAppleScriptInt(parts[2])
	if err != nil {
		return nil, err
	}
	bottom, err := parseAppleScriptInt(parts[3])
	if err != nil {
		return nil, err
	}
	return []desktop.Monitor{{
		ID:      0,
		Name:    "main",
		X:       left,
		Y:       top,
		Width:   right - left,
		Height:  bottom - top,
		Scale:   1,
		Focused: true,
	}}, nil
}

func (p *macOSDesktopProvider) Clients(ctx context.Context) ([]desktop.Window, error) {
	active, err := p.ActiveWindow(ctx)
	if err != nil || active == nil {
		return nil, err
	}
	return []desktop.Window{*active}, nil
}

func (p *macOSDesktopProvider) ActiveWindow(ctx context.Context) (*desktop.Window, error) {
	_ = ctx
	now := time.Now()

	p.mu.Lock()
	if now.After(p.activeRefreshAt) && !p.activeRefreshing {
		p.activeRefreshing = true
		p.activeRefreshAt = now.Add(750 * time.Millisecond)
		go p.refreshActiveWindow()
	}
	window := cloneWindow(p.activeWindow)
	err := p.activeWindowErr
	p.mu.Unlock()

	return window, err
}

func (p *macOSDesktopProvider) refreshActiveWindow() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	window, err := queryMacOSActiveWindow(ctx)
	p.mu.Lock()
	p.activeWindow = cloneWindow(window)
	p.activeWindowErr = err
	p.activeRefreshing = false
	p.mu.Unlock()
	if err != nil && p.log != nil {
		p.log.Debug("macos.active_window_refresh_failed", "error", err)
	}
}

func queryMacOSActiveWindow(ctx context.Context) (*desktop.Window, error) {
	out, err := exec.CommandContext(ctx, "osascript", "-e", activeWindowScript).Output()
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var raw struct {
		App    string `json:"app"`
		Title  string `json:"title"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, err
	}
	if raw.Width <= 0 || raw.Height <= 0 {
		return nil, nil
	}
	return &desktop.Window{
		Address:    raw.App + ":" + raw.Title,
		Class:      raw.App,
		Title:      raw.Title,
		X:          raw.X,
		Y:          raw.Y,
		Width:      raw.Width,
		Height:     raw.Height,
		Workspace:  0,
		Floating:   false,
		Fullscreen: false,
		Focused:    true,
	}, nil
}

func (p *macOSDesktopProvider) Cursor(ctx context.Context) (*desktop.Cursor, error) {
	_ = ctx
	return nativeMacOSCursor()
}

func cloneWindow(window *desktop.Window) *desktop.Window {
	if window == nil {
		return nil
	}
	clone := *window
	return &clone
}

func parseAppleScriptInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}

const activeWindowScript = `
tell application "System Events"
  set frontProc to first application process whose frontmost is true
  set appName to name of frontProc
  if not (exists window 1 of frontProc) then return "null"
  tell window 1 of frontProc
    set windowTitle to name
    set windowPosition to position
    set windowSize to size
  end tell
end tell
return "{\"app\":\"" & appName & "\",\"title\":\"" & windowTitle & "\",\"x\":" & item 1 of windowPosition & ",\"y\":" & item 2 of windowPosition & ",\"width\":" & item 1 of windowSize & ",\"height\":" & item 2 of windowSize & "}"
`
