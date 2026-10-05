#ifndef DRAGOUT_WAYLAND_LINUX_H
#define DRAGOUT_WAYLAND_LINUX_H

#include <stddef.h>

struct wl_display;
struct wl_surface;

enum {
	DRAGOUT_OK = 0,
	DRAGOUT_NO_SEAT = 1,
	DRAGOUT_NO_SERIAL = 2,
	DRAGOUT_NO_MEMORY = 3,
};

int dragout_wl_prepare(struct wl_display *display);
int dragout_wl_start(struct wl_display *display, struct wl_surface *origin,
	const char *data, size_t len);

#endif
