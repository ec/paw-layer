//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>

static uint32_t pawlayer_active_display_count(void) {
	uint32_t count = 0;
	CGGetActiveDisplayList(0, NULL, &count);
	return count;
}

static uint32_t pawlayer_active_displays(CGDirectDisplayID *displays, uint32_t max) {
	uint32_t count = 0;
	CGGetActiveDisplayList(max, displays, &count);
	return count;
}

static CGRect pawlayer_display_bounds(CGDirectDisplayID display) {
	return CGDisplayBounds(display);
}

static int pawlayer_display_is_main(CGDirectDisplayID display) {
	return display == CGMainDisplayID();
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/ec/paw-layer/internal/desktop"
)

func nativeMacOSMonitors() ([]desktop.Monitor, error) {
	count := int(C.pawlayer_active_display_count())
	if count <= 0 {
		return nil, fmt.Errorf("no active displays")
	}
	displays := make([]C.CGDirectDisplayID, count)
	actual := int(C.pawlayer_active_displays((*C.CGDirectDisplayID)(unsafe.Pointer(&displays[0])), C.uint32_t(count)))
	if actual <= 0 {
		return nil, fmt.Errorf("no active displays")
	}

	monitors := make([]desktop.Monitor, 0, actual)
	for i := 0; i < actual; i++ {
		bounds := C.pawlayer_display_bounds(displays[i])
		focused := C.pawlayer_display_is_main(displays[i]) != 0
		monitors = append(monitors, desktop.Monitor{
			ID:      int(displays[i]),
			Name:    fmt.Sprintf("display-%d", uint32(displays[i])),
			X:       int(bounds.origin.x),
			Y:       int(bounds.origin.y),
			Width:   int(bounds.size.width),
			Height:  int(bounds.size.height),
			Scale:   1,
			Focused: focused,
		})
	}
	return monitors, nil
}
