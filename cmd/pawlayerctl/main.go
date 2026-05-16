package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const appName = "paw-layer"

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
	case "start":
		fs := flag.NewFlagSet("start", flag.ExitOnError)
		configPath := fs.String("config", "configs/default.yaml", "path to config file")
		binaryPath := fs.String("binary", "", "path to paw-layer binary")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		return start(*configPath, *binaryPath)
	case "stop":
		return stop()
	case "restart":
		fs := flag.NewFlagSet("restart", flag.ExitOnError)
		configPath := fs.String("config", "configs/default.yaml", "path to config file")
		binaryPath := fs.String("binary", "", "path to paw-layer binary")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if err := stop(); err != nil {
			return err
		}
		return start(*configPath, *binaryPath)
	case "status":
		return status()
	case "logs":
		fs := flag.NewFlagSet("logs", flag.ExitOnError)
		follow := fs.Bool("f", false, "follow log output")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		return logs(*follow)
	default:
		return usage()
	}
}

type paths struct {
	runDir  string
	pidFile string
	logFile string
}

func runtimePaths() (paths, error) {
	runDir := os.Getenv("XDG_RUNTIME_DIR")
	if runDir == "" {
		runDir = filepath.Join(os.TempDir(), fmt.Sprintf("paw-layer-%d", os.Getuid()))
	}
	runDir = filepath.Join(runDir, "paw-layer")
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return paths{}, err
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	logDir := filepath.Join(stateDir, "paw-layer")
	if err := ensureUserDir(runDir); err != nil {
		return paths{}, err
	}
	if err := ensureUserDir(logDir); err != nil {
		return paths{}, err
	}
	return paths{
		runDir:  runDir,
		pidFile: filepath.Join(runDir, "paw-layer.pid"),
		logFile: filepath.Join(logDir, "paw-layer.log"),
	}, nil
}

func ensureUserDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" && info.Sys() != nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Geteuid()) {
			return fmt.Errorf("%s is owned by uid %d, not current uid %d; fix with: sudo chown -R $(id -u):$(id -g) %q", path, stat.Uid, os.Geteuid(), path)
		}
	}
	return nil
}

func start(configPath string, binaryPath string) error {
	if runtime.GOOS == "darwin" && os.Geteuid() == 0 {
		return fmt.Errorf("do not run paw-layer with sudo on macOS: GUI overlays must run in the logged-in user's WindowServer session")
	}

	p, err := runtimePaths()
	if err != nil {
		return err
	}
	if pid, ok := readLivePID(p.pidFile); ok {
		fmt.Printf("paw-layer already running (pid %d)\n", pid)
		return nil
	}
	_ = os.Remove(p.pidFile)

	cmd, err := commandForStart(configPath, binaryPath)
	if err != nil {
		return err
	}
	logFile, err := os.OpenFile(p.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if cwd, err := os.Getwd(); err == nil {
		_, _ = fmt.Fprintf(logFile, "\n--- pawlayerctl start %s ---\ncwd=%s\ncommand=%s %s\n", time.Now().Format(time.RFC3339), cwd, cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	if !processAlive(cmd.Process.Pid) {
		return fmt.Errorf("paw-layer exited immediately; see log: %s", p.logFile)
	}
	if err := os.WriteFile(p.pidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o600); err != nil {
		_ = killProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		return err
	}
	fmt.Printf("started paw-layer (pid %d)\nlog: %s\n", cmd.Process.Pid, p.logFile)
	return nil
}

func commandForStart(configPath string, binaryPath string) (*exec.Cmd, error) {
	if binaryPath != "" {
		return exec.Command(binaryPath, "run", "--config", configPath), nil
	}
	if path, err := findPawLayerBinary(); err == nil {
		return exec.Command(path, "run", "--config", configPath), nil
	}
	if _, err := os.Stat(filepath.Join("cmd", "paw-layer")); err == nil {
		return exec.Command("go", "run", "./cmd/paw-layer", "run", "--config", configPath), nil
	}
	return nil, fmt.Errorf("could not find paw-layer binary; pass --binary /path/to/paw-layer")
}

func findPawLayerBinary() (string, error) {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), appName)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath(appName); err == nil {
		return path, nil
	}
	if isExecutable("./paw-layer") {
		return "./paw-layer", nil
	}
	return "", os.ErrNotExist
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func stop() error {
	p, err := runtimePaths()
	if err != nil {
		return err
	}
	pid, ok := readLivePID(p.pidFile)
	if !ok {
		_ = os.Remove(p.pidFile)
		fmt.Println("paw-layer already stopped")
		return nil
	}
	if err := killProcessGroup(pid, syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			_ = os.Remove(p.pidFile)
			fmt.Println("stopped paw-layer")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := killProcessGroup(pid, syscall.SIGKILL); err != nil {
		return err
	}
	_ = os.Remove(p.pidFile)
	fmt.Println("stopped paw-layer")
	return nil
}

func status() error {
	p, err := runtimePaths()
	if err != nil {
		return err
	}
	if pid, ok := readLivePID(p.pidFile); ok {
		fmt.Printf("paw-layer running (pid %d)\nlog: %s\n", pid, p.logFile)
		return nil
	}
	fmt.Println("paw-layer stopped")
	return nil
}

func logs(follow bool) error {
	p, err := runtimePaths()
	if err != nil {
		return err
	}
	if !follow {
		file, err := os.Open(p.logFile)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(os.Stdout, file)
		return err
	}
	cmd := exec.Command("tail", "-f", p.logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func readLivePID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 || !processAlive(pid) {
		return 0, false
	}
	return pid, true
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func killProcessGroup(pid int, signal syscall.Signal) error {
	if err := syscall.Kill(-pid, signal); err == nil {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(signal)
}

func usage() error {
	fmt.Fprintln(os.Stderr, "usage: pawlayerctl <start|stop|restart|status|logs>")
	return errors.New("invalid command")
}
