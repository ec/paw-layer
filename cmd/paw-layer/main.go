package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ec/paw-layer/internal/app"
	"github.com/ec/paw-layer/internal/config"
	"github.com/ec/paw-layer/internal/hyprland"
	"github.com/ec/paw-layer/internal/renderer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usage()
	}

	switch os.Args[1] {
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := fs.String("config", "configs/default.yaml", "path to config file")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		return runApp(*configPath)
	case "validate-config":
		fs := flag.NewFlagSet("validate-config", flag.ExitOnError)
		configPath := fs.String("config", "configs/default.yaml", "path to config file")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		return config.Validate(cfg)
	case "list-windows":
		return listWindows()
	case "list-monitors":
		return listMonitors()
	case "debug-overlay":
		return errors.New("command is planned but not implemented yet")
	default:
		return usage()
	}
}

func runApp(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if cfg.App.Debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var r renderer.Renderer
	switch cfg.Renderer.Backend {
	case "fake":
		r = renderer.NewFake(log)
	case "gtk4-layer-shell":
		r = renderer.NewGTKLayerShell(log)
	default:
		return fmt.Errorf("unsupported renderer backend %q", cfg.Renderer.Backend)
	}

	if err := app.New(cfg, r, log).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func listWindows() error {
	windows, err := hyprland.NewHyprctl().Clients(context.Background())
	if err != nil {
		return err
	}
	for _, window := range windows {
		fmt.Printf("%s %dx%d+%d+%d fullscreen=%t floating=%t class=%q title=%q\n", window.Address, window.Width, window.Height, window.X, window.Y, window.Fullscreen, window.Floating, window.Class, window.Title)
	}
	return nil
}

func listMonitors() error {
	monitors, err := hyprland.NewHyprctl().Monitors(context.Background())
	if err != nil {
		return err
	}
	for _, monitor := range monitors {
		fmt.Printf("%s %dx%d+%d+%d scale=%.2f focused=%t\n", monitor.Name, monitor.Width, monitor.Height, monitor.X, monitor.Y, monitor.Scale, monitor.Focused)
	}
	return nil
}

func usage() error {
	fmt.Fprintln(os.Stderr, "usage: paw-layer <run|validate-config|list-windows|list-monitors|debug-overlay>")
	return errors.New("invalid command")
}
