#ifndef DRAGOUT_X11_LINUX_H
#define DRAGOUT_X11_LINUX_H

#include <X11/Xlib.h>
#include <stddef.h>

enum {
	DRAGOUT_X11_OK = 0,
	DRAGOUT_X11_NO_DISPLAY = 1,
	DRAGOUT_X11_NO_GRAB = 2,
	DRAGOUT_X11_NO_BUTTON = 3,
	DRAGOUT_X11_NO_MEMORY = 4,
};

struct dragout_x11;

struct dragout_x11 *dragout_x11_begin(const char *data, size_t len, int *err);
void dragout_x11_run(struct dragout_x11 *d);
void dragout_x11_free(struct dragout_x11 *d);
void dragout_x11_release_grab(Display *glfw);

#endif
