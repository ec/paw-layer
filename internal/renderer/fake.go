package renderer

import (
	"context"
	"log/slog"
)

type Fake struct {
	log *slog.Logger
}

func NewFake(log *slog.Logger) *Fake {
	return &Fake{log: log}
}

func (r *Fake) Init(ctx context.Context, cfg Config) error {
	r.log.InfoContext(ctx, "renderer.initialized", "backend", cfg.Backend, "click_through", cfg.ClickThrough, "transparent", cfg.Transparent)
	return nil
}

func (r *Fake) Draw(ctx context.Context, frame Frame) error {
	for _, cat := range frame.Cats {
		r.log.DebugContext(ctx, "renderer.draw_cat", "id", cat.ID, "x", cat.X, "y", cat.Y, "sprite", cat.Sprite, "direction", cat.Direction)
	}
	return nil
}

func (r *Fake) Close() error { return nil }
