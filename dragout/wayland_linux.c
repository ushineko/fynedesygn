//go:build cgo && linux && !android && !test_web_driver && (wayland || !x11)

// The Wayland drag source (quirk 44). It shares GLFW's wl_display and binds
// its own wl_seat, wl_pointer and wl_data_device on it.
//
// The pointer exists for one number: wl_data_device.start_drag needs the
// serial of the button press that began the drag, and GLFW keeps that serial
// to itself. A compositor sends button events to every wl_pointer the client
// holds for the seat, so a pointer of our own sees the same serial.
//
// Everything runs on the UI thread. The globals are bound on a private queue
// so the roundtrip dispatches nothing of GLFW's; the objects are then moved
// to the default queue, which GLFW dispatches, and their listeners are plain
// C that never call back into Go.

#include <errno.h>
#include <linux/input-event-codes.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <wayland-client.h>

#include "wayland_linux.h"

static struct wl_seat *seat;
static struct wl_pointer *pointer;
static struct wl_data_device_manager *manager;
static struct wl_data_device *device;
static uint32_t press_serial;
static int pressed;

// --- pointer: remember the serial of the left button's press -------------

static void pointer_enter(void *d, struct wl_pointer *p, uint32_t serial,
	struct wl_surface *s, wl_fixed_t x, wl_fixed_t y) {}
static void pointer_leave(void *d, struct wl_pointer *p, uint32_t serial,
	struct wl_surface *s) {
	pressed = 0;
}
static void pointer_motion(void *d, struct wl_pointer *p, uint32_t t,
	wl_fixed_t x, wl_fixed_t y) {}
static void pointer_button(void *d, struct wl_pointer *p, uint32_t serial,
	uint32_t t, uint32_t button, uint32_t state) {
	if (button != BTN_LEFT) {
		return;
	}
	if (state == WL_POINTER_BUTTON_STATE_PRESSED) {
		press_serial = serial;
		pressed = 1;
	} else {
		pressed = 0;
	}
}
static void pointer_axis(void *d, struct wl_pointer *p, uint32_t t,
	uint32_t axis, wl_fixed_t v) {}
static void pointer_frame(void *d, struct wl_pointer *p) {}
static void pointer_axis_source(void *d, struct wl_pointer *p, uint32_t s) {}
static void pointer_axis_stop(void *d, struct wl_pointer *p, uint32_t t,
	uint32_t axis) {}
static void pointer_axis_discrete(void *d, struct wl_pointer *p, uint32_t axis,
	int32_t v) {}

// Version 5 of wl_seat, so these nine are the whole of wl_pointer's events.
static const struct wl_pointer_listener pointer_listener = {
	pointer_enter, pointer_leave, pointer_motion, pointer_button, pointer_axis,
	pointer_frame, pointer_axis_source, pointer_axis_stop, pointer_axis_discrete,
};

static void seat_capabilities(void *d, struct wl_seat *s, uint32_t caps) {
	if ((caps & WL_SEAT_CAPABILITY_POINTER) && pointer == NULL) {
		pointer = wl_seat_get_pointer(s);
		wl_pointer_add_listener(pointer, &pointer_listener, NULL);
	} else if (!(caps & WL_SEAT_CAPABILITY_POINTER) && pointer != NULL) {
		wl_pointer_release(pointer);
		pointer = NULL;
		pressed = 0;
	}
}
static void seat_name(void *d, struct wl_seat *s, const char *name) {}

static const struct wl_seat_listener seat_listener = {
	seat_capabilities, seat_name,
};

// --- data device: we only send, so every offer is destroyed on arrival -----

static void device_data_offer(void *d, struct wl_data_device *dev,
	struct wl_data_offer *offer) {
	wl_data_offer_destroy(offer);
}
static void device_enter(void *d, struct wl_data_device *dev, uint32_t serial,
	struct wl_surface *s, wl_fixed_t x, wl_fixed_t y, struct wl_data_offer *o) {}
static void device_leave(void *d, struct wl_data_device *dev) {}
static void device_motion(void *d, struct wl_data_device *dev, uint32_t t,
	wl_fixed_t x, wl_fixed_t y) {}
static void device_drop(void *d, struct wl_data_device *dev) {}
static void device_selection(void *d, struct wl_data_device *dev,
	struct wl_data_offer *o) {}

static const struct wl_data_device_listener device_listener = {
	device_data_offer, device_enter, device_leave, device_motion, device_drop,
	device_selection,
};

// --- data source: one per drag, freed when the drag ends -------------------

struct drag {
	char *data;
	size_t len;
};

static void source_target(void *d, struct wl_data_source *src, const char *mime) {}

static void source_send(void *d, struct wl_data_source *src, const char *mime,
	int32_t fd) {
	struct drag *drag = d;
	size_t off = 0;
	while (off < drag->len) {
		ssize_t n = write(fd, drag->data + off, drag->len - off);
		if (n < 0) {
			if (errno == EINTR) {
				continue;
			}
			break;
		}
		off += (size_t)n;
	}
	close(fd);
}

static void source_end(void *d, struct wl_data_source *src) {
	struct drag *drag = d;
	wl_data_source_destroy(src);
	free(drag->data);
	free(drag);
}

static void source_dnd_drop_performed(void *d, struct wl_data_source *src) {}
static void source_action(void *d, struct wl_data_source *src, uint32_t a) {}

static const struct wl_data_source_listener source_listener = {
	source_target, source_send, source_end, source_dnd_drop_performed,
	source_end, source_action,
};

// --- binding -------------------------------------------------------------

static void registry_global(void *d, struct wl_registry *r, uint32_t name,
	const char *iface, uint32_t version) {
	if (strcmp(iface, wl_seat_interface.name) == 0 && seat == NULL) {
		seat = wl_registry_bind(r, name, &wl_seat_interface,
			version < 5 ? version : 5);
		wl_seat_add_listener(seat, &seat_listener, NULL);
	} else if (strcmp(iface, wl_data_device_manager_interface.name) == 0 &&
		version >= 3 && manager == NULL) {
		manager = wl_registry_bind(r, name, &wl_data_device_manager_interface, 3);
	}
}
static void registry_remove(void *d, struct wl_registry *r, uint32_t name) {}

static const struct wl_registry_listener registry_listener = {
	registry_global, registry_remove,
};

int dragout_wl_prepare(struct wl_display *display) {
	if (device != NULL) {
		return DRAGOUT_OK;
	}
	if (seat != NULL || manager != NULL) {
		return DRAGOUT_NO_SEAT; // tried before and the compositor lacks one
	}

	struct wl_event_queue *queue = wl_display_create_queue(display);
	struct wl_display *wrapper = wl_proxy_create_wrapper(display);
	wl_proxy_set_queue((struct wl_proxy *)wrapper, queue);
	struct wl_registry *registry = wl_display_get_registry(wrapper);
	wl_proxy_wrapper_destroy(wrapper);
	wl_registry_add_listener(registry, &registry_listener, NULL);

	// The first roundtrip binds the globals, the second delivers the seat's
	// capabilities and so creates the pointer.
	wl_display_roundtrip_queue(display, queue);
	wl_display_roundtrip_queue(display, queue);
	wl_registry_destroy(registry);

	if (seat == NULL || manager == NULL) {
		wl_event_queue_destroy(queue);
		return DRAGOUT_NO_SEAT;
	}
	device = wl_data_device_manager_get_data_device(manager, seat);
	wl_data_device_add_listener(device, &device_listener, NULL);

	wl_proxy_set_queue((struct wl_proxy *)seat, NULL);
	wl_proxy_set_queue((struct wl_proxy *)manager, NULL);
	wl_proxy_set_queue((struct wl_proxy *)device, NULL);
	if (pointer != NULL) {
		wl_proxy_set_queue((struct wl_proxy *)pointer, NULL);
	}
	wl_event_queue_destroy(queue);
	return DRAGOUT_OK;
}

int dragout_wl_start(struct wl_display *display, struct wl_surface *origin,
	const char *data, size_t len) {
	if (device == NULL) {
		return DRAGOUT_NO_SEAT;
	}
	if (!pressed) {
		return DRAGOUT_NO_SERIAL;
	}

	struct drag *drag = malloc(sizeof *drag);
	if (drag == NULL) {
		return DRAGOUT_NO_MEMORY;
	}
	drag->data = malloc(len);
	if (drag->data == NULL) {
		free(drag);
		return DRAGOUT_NO_MEMORY;
	}
	memcpy(drag->data, data, len);
	drag->len = len;

	struct wl_data_source *src = wl_data_device_manager_create_data_source(manager);
	wl_data_source_add_listener(src, &source_listener, drag);
	wl_data_source_offer(src, "text/uri-list");
	wl_data_source_set_actions(src, WL_DATA_DEVICE_MANAGER_DND_ACTION_COPY);
	wl_data_device_start_drag(device, src, origin, NULL, press_serial);
	pressed = 0;
	wl_display_flush(display);
	return DRAGOUT_OK;
}
