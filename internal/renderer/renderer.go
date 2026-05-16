package renderer

import "context"

type Config struct {
	Backend       string
	Layer         string
	ClickThrough  bool
	Transparent   bool
	AssetsPath    string
	InitialWidth  int
	InitialHeight int
}

type Renderer interface {
	Init(ctx context.Context, cfg Config) error
	Draw(ctx context.Context, frame Frame) error
	Close() error
}

type MonitorSwitcher interface {
	SwitchMonitor(name string, width int, height int) error
}

type ViewportProvider interface {
	Viewport() (width int, height int, ok bool)
}

type Frame struct {
	Cats []CatRenderState
}

type CatRenderState struct {
	ID         string
	X          int
	Y          int
	Scale      float64
	SpritePack string
	Sprite     string
	Frame      int
	Direction  string
	Visible    bool
}
