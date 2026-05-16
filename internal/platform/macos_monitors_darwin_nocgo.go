//go:build darwin && !cgo

package platform

import (
	"fmt"

	"github.com/ec/paw-layer/internal/desktop"
)

func nativeMacOSMonitors() ([]desktop.Monitor, error) {
	return nil, fmt.Errorf("native macOS monitors require cgo")
}
