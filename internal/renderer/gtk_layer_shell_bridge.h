//go:build linux

#pragma once

#include <stdint.h>
#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <cairo.h>
#include <gtk4-layer-shell.h>

int pawlayer_gtk_run(uintptr_t handle, int initial_width, int initial_height);
void pawlayer_gtk_queue_draw(GtkWidget *widget);
void pawlayer_gtk_quit(GtkApplication *app);
void pawlayer_gtk_switch_monitor(GtkWidget *window, GtkWidget *drawing_area, const char *connector, int width, int height);

extern void pawlayerGTKReady(uintptr_t handle, GtkApplication *app, GtkWidget *window, GtkWidget *drawing_area);
extern void pawlayerGTKDraw(uintptr_t handle, cairo_t *cr, int width, int height);
