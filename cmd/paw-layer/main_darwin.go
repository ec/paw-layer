//go:build darwin

package main

import "runtime"

func init() {
	// Cocoa/AppKit must be initialized and run on the process initial thread.
	// Locking in init keeps Go's main goroutine on that thread before main starts.
	runtime.LockOSThread()
}
