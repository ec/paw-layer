//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

static CGPoint pawlayer_cursor_location(void) {
	CGEventRef event = CGEventCreate(NULL);
	if (event == NULL) {
		return CGPointMake(0, 0);
	}
	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);
	return point;
}
*/
import "C"

import "github.com/ec/paw-layer/internal/desktop"

func nativeMacOSCursor() (*desktop.Cursor, error) {
	point := C.pawlayer_cursor_location()
	return &desktop.Cursor{X: int(point.x), Y: int(point.y)}, nil
}
