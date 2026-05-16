package renderer

/*
#cgo pkg-config: gtk4 gtk4-layer-shell-0
#include <stdlib.h>
#include "gtk_layer_shell_bridge.h"
*/
import "C"

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ec/paw-layer/internal/assets"
)

type GTKLayerShell struct {
	log *slog.Logger

	ready chan error
	done  chan struct{}
	once  sync.Once

	handle        uintptr
	app           *C.GtkApplication
	area          *C.GtkWidget
	window        *C.GtkWidget
	initialWidth  int
	initialHeight int
	monitorName   string

	mu             sync.RWMutex
	latest         Frame
	assets         *spriteStore
	started        bool
	viewportWidth  int
	viewportHeight int
}

var (
	gtkHandleSeq atomic.Uintptr
	gtkRegistry  sync.Map // map[uintptr]*GTKLayerShell
)

func NewGTKLayerShell(log *slog.Logger) *GTKLayerShell {
	handle := gtkHandleSeq.Add(1)
	r := &GTKLayerShell{
		log:    log,
		ready:  make(chan error, 1),
		done:   make(chan struct{}),
		handle: handle,
	}
	gtkRegistry.Store(handle, r)
	return r
}

func (r *GTKLayerShell) Init(ctx context.Context, cfg Config) error {
	r.initialWidth = cfg.InitialWidth
	r.initialHeight = cfg.InitialHeight

	store, err := loadSpriteStore(cfg.AssetsPath)
	if err != nil {
		r.log.Warn("renderer.sprite_store_unavailable", "error", err)
	} else {
		r.assets = store
	}

	r.once.Do(func() {
		go r.runGTK()
	})

	select {
	case err := <-r.ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *GTKLayerShell) Draw(ctx context.Context, frame Frame) error {
	r.mu.Lock()
	r.latest = frame
	area := r.area
	r.mu.Unlock()

	if area != nil {
		C.pawlayer_gtk_queue_draw(area)
	}
	return nil
}

func (r *GTKLayerShell) SwitchMonitor(name string, width int, height int) error {
	if name == "" {
		return nil
	}
	r.mu.RLock()
	window := r.window
	area := r.area
	currentName := r.monitorName
	r.mu.RUnlock()
	if window == nil || area == nil {
		return nil
	}
	r.mu.RLock()
	currentWidth := r.initialWidth
	currentHeight := r.initialHeight
	r.mu.RUnlock()
	if currentName == name && width == currentWidth && height == currentHeight {
		return nil
	}

	cname := C.CString(name)
	C.pawlayer_gtk_switch_monitor(window, area, cname, C.int(width), C.int(height))
	C.free(unsafe.Pointer(cname))

	r.mu.Lock()
	r.monitorName = name
	r.initialWidth = width
	r.initialHeight = height
	r.viewportWidth = 0
	r.viewportHeight = 0
	r.mu.Unlock()

	r.log.Info("renderer.monitor_switch_requested", "monitor", name, "width", width, "height", height)
	return nil
}

func (r *GTKLayerShell) Viewport() (width int, height int, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.viewportWidth <= 0 || r.viewportHeight <= 0 {
		return 0, 0, false
	}
	return r.viewportWidth, r.viewportHeight, true
}

func (r *GTKLayerShell) Close() error {
	r.mu.RLock()
	app := r.app
	r.mu.RUnlock()

	if app != nil {
		C.pawlayer_gtk_quit(app)
	}

	<-r.done
	gtkRegistry.Delete(r.handle)
	return nil
}

func (r *GTKLayerShell) runGTK() {
	// GTK4 may default to Vulkan on some setups, which can emit noisy
	// gdk_vulkan_debug_report lines for a tiny transparent overlay. Cairo is
	// deterministic and sufficient for pixel-art sprites.
	if os.Getenv("GSK_RENDERER") == "" {
		_ = os.Setenv("GSK_RENDERER", "cairo")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(r.done)

	code := int(C.pawlayer_gtk_run(C.uintptr_t(r.handle), C.int(r.initialWidth), C.int(r.initialHeight)))
	if code != 0 {
		select {
		case r.ready <- &ExitError{Code: code}:
		default:
		}
	}
}

//export pawlayerGTKReady
func pawlayerGTKReady(handle C.uintptr_t, app *C.GtkApplication, window *C.GtkWidget, drawingArea *C.GtkWidget) {
	value, ok := gtkRegistry.Load(uintptr(handle))
	if !ok {
		return
	}
	r := value.(*GTKLayerShell)

	r.mu.Lock()
	r.app = app
	r.window = window
	r.area = drawingArea
	r.started = true
	r.mu.Unlock()

	r.log.Info("renderer.gtk_layer_shell_ready", "click_through", true, "transparent", true)
	r.ready <- nil
}

//export pawlayerGTKDraw
func pawlayerGTKDraw(handle C.uintptr_t, cr *C.cairo_t, width C.int, height C.int) {
	value, ok := gtkRegistry.Load(uintptr(handle))
	if !ok {
		return
	}
	r := value.(*GTKLayerShell)
	r.setViewport(int(width), int(height))
	r.draw(cr)
}

func (r *GTKLayerShell) setViewport(width, height int) {
	r.mu.Lock()
	changed := r.viewportWidth != width || r.viewportHeight != height
	r.viewportWidth = width
	r.viewportHeight = height
	r.mu.Unlock()
	if changed {
		r.log.Info("renderer.viewport_changed", "width", width, "height", height)
	}
}

func (r *GTKLayerShell) draw(cr *C.cairo_t) {
	C.cairo_save(cr)
	C.cairo_set_operator(cr, C.CAIRO_OPERATOR_CLEAR)
	C.cairo_paint(cr)
	C.cairo_restore(cr)
	C.cairo_set_operator(cr, C.CAIRO_OPERATOR_OVER)

	r.mu.RLock()
	frame := r.latest
	store := r.assets
	r.mu.RUnlock()

	for _, cat := range frame.Cats {
		if !cat.Visible {
			continue
		}
		if store != nil && store.Draw(cr, cat) {
			continue
		}
		drawPixelCat(cr, float64(cat.X), float64(cat.Y), cat.Scale, cat.Direction)
	}
}

type spriteStore struct {
	packs map[string]spritePack
}

type spritePack struct {
	manifest assets.Manifest
	surfaces map[string]*C.cairo_surface_t
}

func loadSpriteStore(root string) (*spriteStore, error) {
	if root == "" {
		root = "./assets/cats"
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	store := &spriteStore{packs: make(map[string]spritePack)}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packDir := filepath.Join(root, entry.Name())
		manifest, err := assets.LoadManifest(filepath.Join(packDir, "manifest.yaml"))
		if err != nil {
			continue
		}

		pack := spritePack{manifest: manifest, surfaces: make(map[string]*C.cairo_surface_t)}
		for name, anim := range manifest.Animations {
			path := C.CString(filepath.Join(packDir, anim.File))
			surface := C.cairo_image_surface_create_from_png(path)
			C.free(unsafe.Pointer(path))
			if status := C.cairo_surface_status(surface); status != C.CAIRO_STATUS_SUCCESS {
				C.cairo_surface_destroy(surface)
				return nil, fmt.Errorf("load sprite pack %s animation %s: cairo status %d", entry.Name(), anim.File, int(status))
			}
			pack.surfaces[name] = surface
		}
		store.packs[entry.Name()] = pack
		store.packs[manifest.Name] = pack
	}

	if len(store.packs) == 0 {
		return nil, fmt.Errorf("no sprite packs found in %s", root)
	}
	return store, nil
}

func (s *spriteStore) Draw(cr *C.cairo_t, cat CatRenderState) bool {
	pack, ok := s.packs[cat.SpritePack]
	if !ok {
		return false
	}
	anim, ok := pack.manifest.Animations[cat.Sprite]
	if !ok {
		return false
	}
	surface := pack.surfaces[cat.Sprite]
	if surface == nil {
		return false
	}

	frame := cat.Frame
	if anim.Frames > 0 {
		frame %= anim.Frames
	}
	if frame < 0 {
		frame = 0
	}

	C.cairo_save(cr)
	defer C.cairo_restore(cr)
	C.cairo_translate(cr, C.double(cat.X), C.double(cat.Y))
	if cat.Direction == "left" {
		C.cairo_translate(cr, C.double(float64(pack.manifest.TileWidth)*cat.Scale), 0)
		C.cairo_scale(cr, C.double(-cat.Scale), C.double(cat.Scale))
	} else {
		C.cairo_scale(cr, C.double(cat.Scale), C.double(cat.Scale))
	}
	C.cairo_rectangle(cr, 0, 0, C.double(pack.manifest.TileWidth), C.double(pack.manifest.TileHeight))
	C.cairo_clip(cr)
	C.cairo_set_source_surface(cr, surface, C.double(-frame*pack.manifest.TileWidth), 0)
	C.cairo_paint(cr)
	return true
}

func drawPixelCat(cr *C.cairo_t, x, y, scale float64, direction string) {
	if scale <= 0 {
		scale = 1
	}

	color := func(r, g, b, a float64) {
		C.cairo_set_source_rgba(cr, C.double(r), C.double(g), C.double(b), C.double(a))
	}
	px := func(cx, cy, w, h float64) {
		C.cairo_rectangle(cr, C.double(x+cx*scale), C.double(y+cy*scale), C.double(w*scale), C.double(h*scale))
		C.cairo_fill(cr)
	}

	color(0.93, 0.60, 0.24, 1)
	px(4, 8, 15, 8)
	px(7, 4, 8, 6)
	px(7, 2, 2, 3)
	px(13, 2, 2, 3)

	if direction == "left" {
		px(1, 7, 4, 2)
		px(0, 5, 2, 2)
	} else {
		px(18, 7, 4, 2)
		px(21, 5, 2, 2)
	}

	color(0.35, 0.20, 0.12, 1)
	px(6, 16, 3, 2)
	px(15, 16, 3, 2)
	px(9, 7, 1, 1)
	px(13, 7, 1, 1)
	px(11, 9, 1, 1)
}

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("gtk application exited with status %d", e.Code)
}
