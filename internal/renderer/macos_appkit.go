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
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ec/paw-layer/internal/assets"
)

type MacOSAppKit struct {
	log *slog.Logger

	ready chan error
	done  chan struct{}

	handle uintptr

	mu             sync.RWMutex
	latest         Frame
	assets         *macOSSpriteStore
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
	return fmt.Errorf("macOS AppKit renderer must run through RunMain")
}

func (r *MacOSAppKit) RunMain(ctx context.Context, cfg Config, runApp func(context.Context) error) error {
	store, err := loadMacOSSpriteStore(cfg.AssetsPath)
	if err != nil {
		r.log.Warn("renderer.macos_sprite_store_unavailable", "error", err)
	} else {
		r.assets = store
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.done)
	defer macOSRegistry.Delete(r.handle)

	appCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		select {
		case err := <-r.ready:
			if err != nil {
				errCh <- err
				cancel()
				C.pawlayer_macos_quit(C.uintptr_t(r.handle))
				return
			}
		case <-appCtx.Done():
			errCh <- appCtx.Err()
			return
		}

		if err := runApp(appCtx); err != nil {
			errCh <- err
		} else {
			errCh <- nil
		}
		cancel()
		C.pawlayer_macos_quit(C.uintptr_t(r.handle))
	}()

	go func() {
		<-appCtx.Done()
		C.pawlayer_macos_quit(C.uintptr_t(r.handle))
	}()

	r.log.Info("renderer.macos_appkit_run_main", "width", cfg.InitialWidth, "height", cfg.InitialHeight)
	code := int(C.pawlayer_macos_run(C.uintptr_t(r.handle), C.int(cfg.InitialWidth), C.int(cfg.InitialHeight)))
	if code != 0 {
		return &macOSExitError{Code: code}
	}

	select {
	case err := <-errCh:
		if err == context.Canceled {
			return nil
		}
		return err
	default:
		return nil
	}
}

func (r *MacOSAppKit) Draw(ctx context.Context, frame Frame) error {
	_ = ctx
	r.mu.Lock()
	r.latest = frame
	r.mu.Unlock()

	if len(frame.Cats) == 0 {
		C.pawlayer_macos_set_cat(C.uintptr_t(r.handle), 0, 0, 1, 1, 0, nil)
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
	var speech *C.char
	if cat.Speech != "" {
		speech = C.CString(cat.Speech)
		defer C.free(unsafe.Pointer(speech))
	}
	if sprite, ok := r.spriteFor(cat); ok {
		path := C.CString(sprite.path)
		defer C.free(unsafe.Pointer(path))
		C.pawlayer_macos_set_sprite(C.uintptr_t(r.handle), path, C.double(cat.X), C.double(cat.Y), C.double(cat.Scale), C.int(sprite.tileWidth), C.int(sprite.tileHeight), C.int(sprite.frame), C.int(directionRight), C.int(visible), speech)
		return nil
	}

	C.pawlayer_macos_set_cat(C.uintptr_t(r.handle), C.double(cat.X), C.double(cat.Y), C.double(cat.Scale), C.int(directionRight), C.int(visible), speech)
	return nil
}

type macOSSpriteStore struct {
	packs map[string]macOSSpritePack
}

type macOSSpritePack struct {
	manifest assets.Manifest
	paths    map[string]string
}

type macOSSprite struct {
	path       string
	tileWidth  int
	tileHeight int
	frame      int
}

func loadMacOSSpriteStore(root string) (*macOSSpriteStore, error) {
	if root == "" {
		root = "./assets/cats"
	}
	entries, err := filepath.Glob(filepath.Join(root, "*", "manifest.yaml"))
	if err != nil {
		return nil, err
	}
	store := &macOSSpriteStore{packs: make(map[string]macOSSpritePack)}
	for _, manifestPath := range entries {
		manifest, err := assets.LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		packDir := filepath.Dir(manifestPath)
		pack := macOSSpritePack{manifest: manifest, paths: make(map[string]string)}
		for name, anim := range manifest.Animations {
			pack.paths[name] = filepath.Join(packDir, anim.File)
		}
		store.packs[filepath.Base(packDir)] = pack
		store.packs[manifest.Name] = pack
	}
	if len(store.packs) == 0 {
		return nil, fmt.Errorf("no sprite packs found in %s", root)
	}
	return store, nil
}

func (r *MacOSAppKit) spriteFor(cat CatRenderState) (macOSSprite, bool) {
	r.mu.RLock()
	store := r.assets
	r.mu.RUnlock()
	if store == nil {
		return macOSSprite{}, false
	}
	pack, ok := store.packs[cat.SpritePack]
	if !ok {
		return macOSSprite{}, false
	}
	anim, ok := pack.manifest.Animations[cat.Sprite]
	if !ok {
		return macOSSprite{}, false
	}
	path, ok := pack.paths[cat.Sprite]
	if !ok || path == "" {
		return macOSSprite{}, false
	}
	frame := cat.Frame
	if anim.Frames > 0 {
		frame %= anim.Frames
	}
	if frame < 0 {
		frame = 0
	}
	return macOSSprite{path: path, tileWidth: pack.manifest.TileWidth, tileHeight: pack.manifest.TileHeight, frame: frame}, true
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
	return nil
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
	select {
	case r.ready <- nil:
	default:
	}
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
