//go:build darwin

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ec/paw-layer/internal/desktop"
)

type macOSDesktopProvider struct{}

func (macOSDesktopProvider) Monitors(ctx context.Context) ([]desktop.Monitor, error) {
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

func (macOSDesktopProvider) Clients(ctx context.Context) ([]desktop.Window, error) {
	active, err := macOSDesktopProvider{}.ActiveWindow(ctx)
	if err != nil || active == nil {
		return nil, err
	}
	return []desktop.Window{*active}, nil
}

func (macOSDesktopProvider) ActiveWindow(ctx context.Context) (*desktop.Window, error) {
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

func (macOSDesktopProvider) Cursor(ctx context.Context) (*desktop.Cursor, error) {
	out, err := exec.CommandContext(ctx, "osascript", "-e", cursorPositionScript).Output()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("parse cursor position %q", strings.TrimSpace(string(out)))
	}
	x, err := parseAppleScriptInt(parts[0])
	if err != nil {
		return nil, err
	}
	y, err := parseAppleScriptInt(parts[1])
	if err != nil {
		return nil, err
	}
	return &desktop.Cursor{X: x, Y: y}, nil
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

const cursorPositionScript = `
use framework "Foundation"
use framework "CoreGraphics"
set mouseLoc to current application's NSEvent's mouseLocation()
set mainScreenHeight to ((current application's NSScreen's mainScreen())'s frame())'s size's height
return ((mouseLoc's x) as integer) & "," & ((mainScreenHeight - (mouseLoc's y)) as integer)
`
