//go:build darwin && !cgo

package platform

import (
	"fmt"

	"github.com/ec/paw-layer/internal/desktop"
)

func nativeMacOSCursor() (*desktop.Cursor, error) {
	return nil, fmt.Errorf("native macOS cursor requires cgo")
}
