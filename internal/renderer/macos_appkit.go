//go:build darwin && cgo

package renderer

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "macos_appkit_bridge.h"
*/
import "C"

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
)

type MacOSAppKit struct {
	log *slog.Logger

	ready chan error
	done  chan struct{}
	once  sync.Once

	handle uintptr

	mu             sync.RWMutex
	latest         Frame
	viewportWidth  int
	viewportHeight int
}

var (
	macOSHandleSeq atomic.Uintptr
	macOSRegistry  sync.Map // map[uintptr]*MacOSAppKit
)

func NewMacOSAppKit(log *slog.Logger) *MacOSAppKit {
	handle := macOSHandleSeq.Add(1)
	r := &MacOSAppKit{
		log:    log,
		ready:  make(chan error, 1),
		done:   make(chan struct{}),
		handle: handle,
	}
	macOSRegistry.Store(handle, r)
	return r
}

func (r *MacOSAppKit) Init(ctx context.Context, cfg Config) error {
	r.once.Do(func() {
		go r.runAppKit(cfg.InitialWidth, cfg.InitialHeight)
	})

	select {
	case err := <-r.ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *MacOSAppKit) Draw(ctx context.Context, frame Frame) error {
	_ = ctx
	r.mu.Lock()
	r.latest = frame
	r.mu.Unlock()

	if len(frame.Cats) == 0 {
		C.pawlayer_macos_set_cat(C.uintptr_t(r.handle), 0, 0, 1, 1, 0)
		return nil
	}
	cat := frame.Cats[0]
	directionRight := 1
	if cat.Direction == "left" {
		directionRight = 0
	}
	visible := 0
	if cat.Visible {
		visible = 1
	}
	C.pawlayer_macos_set_cat(C.uintptr_t(r.handle), C.double(cat.X), C.double(cat.Y), C.double(cat.Scale), C.int(directionRight), C.int(visible))
	return nil
}

func (r *MacOSAppKit) Viewport() (width int, height int, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.viewportWidth <= 0 || r.viewportHeight <= 0 {
		return 0, 0, false
	}
	return r.viewportWidth, r.viewportHeight, true
}

func (r *MacOSAppKit) Close() error {
	C.pawlayer_macos_quit(C.uintptr_t(r.handle))
	<-r.done
	macOSRegistry.Delete(r.handle)
	return nil
}

func (r *MacOSAppKit) runAppKit(initialWidth int, initialHeight int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.done)

	code := int(C.pawlayer_macos_run(C.uintptr_t(r.handle), C.int(initialWidth), C.int(initialHeight)))
	if code != 0 {
		select {
		case r.ready <- &macOSExitError{Code: code}:
		default:
		}
	}
}

//export pawlayerMacOSReady
func pawlayerMacOSReady(handle C.uintptr_t, width C.int, height C.int) {
	value, ok := macOSRegistry.Load(uintptr(handle))
	if !ok {
		return
	}
	r := value.(*MacOSAppKit)
	r.setViewport(int(width), int(height))
	r.log.Info("renderer.macos_appkit_ready", "click_through", true, "transparent", true)
	r.ready <- nil
}

//export pawlayerMacOSViewportChanged
func pawlayerMacOSViewportChanged(handle C.uintptr_t, width C.int, height C.int) {
	value, ok := macOSRegistry.Load(uintptr(handle))
	if !ok {
		return
	}
	value.(*MacOSAppKit).setViewport(int(width), int(height))
}

func (r *MacOSAppKit) setViewport(width, height int) {
	r.mu.Lock()
	changed := r.viewportWidth != width || r.viewportHeight != height
	r.viewportWidth = width
	r.viewportHeight = height
	r.mu.Unlock()
	if changed {
		r.log.Info("renderer.viewport_changed", "width", width, "height", height)
	}
}

type macOSExitError struct {
	Code int
}

func (e *macOSExitError) Error() string {
	return fmt.Sprintf("macOS application exited with status %d", e.Code)
}
