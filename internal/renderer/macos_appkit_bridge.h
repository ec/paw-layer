//go:build darwin && cgo

#pragma once

#include <stdint.h>

int pawlayer_macos_run(uintptr_t handle, int initial_width, int initial_height);
void pawlayer_macos_set_cat(uintptr_t handle, double x, double y, double scale, int direction_right, int visible);
void pawlayer_macos_set_sprite(uintptr_t handle, const char *path, double x, double y, double scale, int tile_width, int tile_height, int frame, int direction_right, int visible);
void pawlayer_macos_quit(uintptr_t handle);

extern void pawlayerMacOSReady(uintptr_t handle, int width, int height);
extern void pawlayerMacOSViewportChanged(uintptr_t handle, int width, int height);
