package renderer

import "context"

// MainThreadRunner is implemented by renderers whose native event loop must run
// on the process main thread, such as AppKit on macOS.
type MainThreadRunner interface {
	RunMain(ctx context.Context, cfg Config, runApp func(context.Context) error) error
}
