#include "gtk_layer_shell_bridge.h"

static void pawlayer_make_click_through(GtkWindow *window) {
	GdkSurface *surface = gtk_native_get_surface(GTK_NATIVE(window));
	if (surface == NULL) {
		return;
	}

	cairo_region_t *empty = cairo_region_create();
	gdk_surface_set_input_region(surface, empty);
	cairo_region_destroy(empty);
}

typedef struct {
	uintptr_t handle;
	int initial_width;
	int initial_height;
} PawLayerGTKState;

static void pawlayer_draw_func(GtkDrawingArea *area, cairo_t *cr, int width, int height, gpointer data) {
	(void)area;
	PawLayerGTKState *state = data;
	pawlayerGTKDraw(state->handle, cr, width, height);
}

static void pawlayer_install_transparent_css(void) {
	GtkCssProvider *provider = gtk_css_provider_new();
	gtk_css_provider_load_from_string(provider,
		"window, window.background, drawingarea { background: transparent; }"
	);

	GdkDisplay *display = gdk_display_get_default();
	if (display != NULL) {
		gtk_style_context_add_provider_for_display(
			display,
			GTK_STYLE_PROVIDER(provider),
			GTK_STYLE_PROVIDER_PRIORITY_APPLICATION
		);
	}
	g_object_unref(provider);
}

static void pawlayer_activate(GtkApplication *app, gpointer data) {
	PawLayerGTKState *state = data;
	pawlayer_install_transparent_css();

	GtkWidget *window = gtk_application_window_new(app);
	gtk_window_set_title(GTK_WINDOW(window), "paw-layer");
	gtk_window_set_decorated(GTK_WINDOW(window), FALSE);
	gtk_window_set_resizable(GTK_WINDOW(window), FALSE);
	gtk_widget_set_can_target(window, FALSE);
	gtk_window_set_default_size(GTK_WINDOW(window), state->initial_width, state->initial_height);

	gtk_layer_init_for_window(GTK_WINDOW(window));
	gtk_layer_set_namespace(GTK_WINDOW(window), "paw-layer");
	gtk_layer_set_layer(GTK_WINDOW(window), GTK_LAYER_SHELL_LAYER_OVERLAY);
	gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_TOP, TRUE);
	gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_RIGHT, TRUE);
	gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_BOTTOM, TRUE);
	gtk_layer_set_anchor(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_LEFT, TRUE);
	gtk_layer_set_margin(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_TOP, 0);
	gtk_layer_set_margin(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_RIGHT, 0);
	gtk_layer_set_margin(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_BOTTOM, 0);
	gtk_layer_set_margin(GTK_WINDOW(window), GTK_LAYER_SHELL_EDGE_LEFT, 0);
	gtk_layer_set_exclusive_zone(GTK_WINDOW(window), -1);
	gtk_layer_set_keyboard_mode(GTK_WINDOW(window), GTK_LAYER_SHELL_KEYBOARD_MODE_NONE);

	GtkWidget *drawing_area = gtk_drawing_area_new();
	gtk_widget_set_can_target(drawing_area, FALSE);
	gtk_widget_set_hexpand(drawing_area, TRUE);
	gtk_widget_set_vexpand(drawing_area, TRUE);
	gtk_widget_set_size_request(drawing_area, state->initial_width, state->initial_height);
	gtk_drawing_area_set_draw_func(GTK_DRAWING_AREA(drawing_area), pawlayer_draw_func, state, NULL);
	gtk_window_set_child(GTK_WINDOW(window), drawing_area);

	gtk_window_present(GTK_WINDOW(window));
	pawlayer_make_click_through(GTK_WINDOW(window));

	pawlayerGTKReady(state->handle, app, window, drawing_area);
}

int pawlayer_gtk_run(uintptr_t handle, int initial_width, int initial_height) {
	PawLayerGTKState state = {
		.handle = handle,
		.initial_width = initial_width > 0 ? initial_width : 800,
		.initial_height = initial_height > 0 ? initial_height : 600,
	};
	GtkApplication *app = gtk_application_new("dev.openclaw.PawLayer", G_APPLICATION_DEFAULT_FLAGS);
	g_signal_connect(app, "activate", G_CALLBACK(pawlayer_activate), &state);
	int status = g_application_run(G_APPLICATION(app), 0, NULL);
	g_object_unref(app);
	return status;
}

static gboolean pawlayer_queue_draw_idle(gpointer data) {
	gtk_widget_queue_draw(GTK_WIDGET(data));
	return G_SOURCE_REMOVE;
}

void pawlayer_gtk_queue_draw(GtkWidget *widget) {
	if (widget == NULL) {
		return;
	}
	g_idle_add(pawlayer_queue_draw_idle, widget);
}

static gboolean pawlayer_quit_idle(gpointer data) {
	g_application_quit(G_APPLICATION(data));
	return G_SOURCE_REMOVE;
}

void pawlayer_gtk_quit(GtkApplication *app) {
	if (app == NULL) {
		return;
	}
	g_idle_add(pawlayer_quit_idle, app);
}

typedef struct {
	GtkWidget *window;
	GtkWidget *drawing_area;
	char *connector;
	int width;
	int height;
} PawLayerSwitchMonitorRequest;

static gboolean pawlayer_switch_monitor_idle(gpointer data) {
	PawLayerSwitchMonitorRequest *request = data;
	if (request->window == NULL || request->drawing_area == NULL || request->connector == NULL) {
		goto cleanup;
	}

	GdkDisplay *display = gdk_display_get_default();
	if (display == NULL) {
		goto cleanup;
	}

	GListModel *monitors = gdk_display_get_monitors(display);
	guint count = g_list_model_get_n_items(monitors);
	for (guint i = 0; i < count; i++) {
		GdkMonitor *monitor = g_list_model_get_item(monitors, i);
		const char *connector = gdk_monitor_get_connector(monitor);
		if (connector != NULL && g_strcmp0(connector, request->connector) == 0) {
			gtk_layer_set_monitor(GTK_WINDOW(request->window), monitor);
			gtk_window_set_default_size(GTK_WINDOW(request->window), request->width, request->height);
			gtk_widget_set_size_request(request->drawing_area, request->width, request->height);
			pawlayer_make_click_through(GTK_WINDOW(request->window));
			gtk_widget_queue_draw(request->drawing_area);
			g_object_unref(monitor);
			goto cleanup;
		}
		g_object_unref(monitor);
	}

cleanup:
	g_free(request->connector);
	g_free(request);
	return G_SOURCE_REMOVE;
}

void pawlayer_gtk_switch_monitor(GtkWidget *window, GtkWidget *drawing_area, const char *connector, int width, int height) {
	if (window == NULL || drawing_area == NULL || connector == NULL) {
		return;
	}
	PawLayerSwitchMonitorRequest *request = g_new0(PawLayerSwitchMonitorRequest, 1);
	request->window = window;
	request->drawing_area = drawing_area;
	request->connector = g_strdup(connector);
	request->width = width > 0 ? width : 800;
	request->height = height > 0 ? height : 600;
	g_idle_add(pawlayer_switch_monitor_idle, request);
}
