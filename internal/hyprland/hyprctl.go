package hyprland

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ec/paw-layer/internal/desktop"
)

type Hyprctl struct{}

func NewHyprctl() Hyprctl { return Hyprctl{} }

func (Hyprctl) Monitors(ctx context.Context) ([]desktop.Monitor, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "-j", "monitors").Output()
	if err != nil {
		return nil, err
	}

	var raw []rawMonitor
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	monitors := make([]desktop.Monitor, 0, len(raw))
	for _, item := range raw {
		monitors = append(monitors, item.monitor())
	}
	return monitors, nil
}

func (Hyprctl) Clients(ctx context.Context) ([]desktop.Window, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "-j", "clients").Output()
	if err != nil {
		return nil, err
	}

	var raw []rawWindow
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	windows := make([]desktop.Window, 0, len(raw))
	for _, item := range raw {
		windows = append(windows, item.window())
	}
	return windows, nil
}

func (Hyprctl) ActiveWindow(ctx context.Context) (*desktop.Window, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "-j", "activewindow").Output()
	if err != nil {
		return nil, err
	}

	var raw rawWindow
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	if raw.Address == "" {
		return nil, nil
	}
	window := raw.window()
	window.Focused = true
	return &window, nil
}

func (Hyprctl) Cursor(ctx context.Context) (*desktop.Cursor, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "-j", "cursorpos").Output()
	if err != nil {
		return nil, err
	}
	return parseCursor(out)
}

func (Hyprctl) Subscribe(ctx context.Context) (<-chan Event, error) {
	return nil, fmt.Errorf("hyprland event subscription is not implemented yet")
}

type rawMonitor struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Scale   float64 `json:"scale"`
	Focused bool    `json:"focused"`
}

func (m rawMonitor) monitor() desktop.Monitor {
	return desktop.Monitor{ID: m.ID, Name: m.Name, X: m.X, Y: m.Y, Width: m.Width, Height: m.Height, Scale: m.Scale, Focused: m.Focused}
}

type rawWindow struct {
	Address    string          `json:"address"`
	Class      string          `json:"class"`
	Title      string          `json:"title"`
	At         [2]int          `json:"at"`
	Size       [2]int          `json:"size"`
	Workspace  rawWorkspace    `json:"workspace"`
	Floating   bool            `json:"floating"`
	Fullscreen json.RawMessage `json:"fullscreen"`
}

type rawWorkspace struct {
	ID int `json:"id"`
}

func (w rawWindow) window() desktop.Window {
	return desktop.Window{
		Address:    w.Address,
		Class:      w.Class,
		Title:      w.Title,
		X:          w.At[0],
		Y:          w.At[1],
		Width:      w.Size[0],
		Height:     w.Size[1],
		Workspace:  w.Workspace.ID,
		Floating:   w.Floating,
		Fullscreen: parseFullscreen(w.Fullscreen),
	}
}

func parseFullscreen(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n != 0
	}
	return false
}

var cursorNumberPattern = regexp.MustCompile(`-?\d+`)

func parseCursor(out []byte) (*desktop.Cursor, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, fmt.Errorf("empty cursor position")
	}

	var object struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := json.Unmarshal(out, &object); err == nil {
		return &desktop.Cursor{X: object.X, Y: object.Y}, nil
	}

	numbers := cursorNumberPattern.FindAllString(trimmed, 2)
	if len(numbers) < 2 {
		return nil, fmt.Errorf("parse cursor position %q", trimmed)
	}
	x, err := strconv.Atoi(numbers[0])
	if err != nil {
		return nil, err
	}
	y, err := strconv.Atoi(numbers[1])
	if err != nil {
		return nil, err
	}
	return &desktop.Cursor{X: x, Y: y}, nil
}
