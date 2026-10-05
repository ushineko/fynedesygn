//go:build cgo && linux && !android && !test_web_driver && (x11 || !wayland)

// The X11 drag source (quirk 44): XDND version 5 on a Display connection of
// its own, so the XDND and selection traffic never reaches GLFW's event loop.
// GLFW answers SelectionRequest for the clipboard and would answer
// XdndSelection wrongly.
//
// dragout_x11_begin runs on the UI thread and takes the pointer;
// dragout_x11_run then runs the drag to its end on a thread of its own. The
// Display is used by one thread at a time, never two.

#include <X11/Xatom.h>
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/cursorfont.h>
#include <X11/keysym.h>
#include <poll.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include "x11_linux.h"

#define XDND_VERSION 5

struct dragout_x11 {
	Display *dpy;
	Window root, src;
	Cursor cursor;
	char *data;
	size_t len;

	Atom aware, proxy, selection, enter, position, status, leave, drop,
		finished, copy, urilist, targets;

	Window target;    // the XdndAware window under the pointer, or None
	Window deliver;   // where its messages go: it, or its XdndProxy
	int version;      // the XDND version agreed with target
	int accepted;     // its last XdndStatus accepted the drop
	int waiting;      // a position was sent and its status has not come
	int pending;      // a motion arrived while waiting
	int px, py;       // the pointer, root coordinates
	Time time;        // the time of the last event that had one
};

// --- errors --------------------------------------------------------------

// A target window can be destroyed between finding it and asking it a
// question, which is BadWindow, and Xlib's default handler exits the
// process. Errors on our own connections are ignored; anything else goes to
// whatever handler was there before. Installed once, on the UI thread, and
// never removed: GLFW saves and restores it around its own grabs.
static XErrorHandler previous_handler;
static int handler_installed;
static Display *ours[8];

static int on_error(Display *dpy, XErrorEvent *e) {
	for (size_t i = 0; i < sizeof ours / sizeof ours[0]; i++) {
		if (ours[i] == dpy) {
			return 0;
		}
	}
	return previous_handler ? previous_handler(dpy, e) : 0;
}

static void claim(Display *dpy) {
	for (size_t i = 0; i < sizeof ours / sizeof ours[0]; i++) {
		if (ours[i] == NULL) {
			ours[i] = dpy;
			return;
		}
	}
}

static void unclaim(Display *dpy) {
	for (size_t i = 0; i < sizeof ours / sizeof ours[0]; i++) {
		if (ours[i] == dpy) {
			ours[i] = NULL;
		}
	}
}

// --- protocol ------------------------------------------------------------

static void send_msg(struct dragout_x11 *d, Atom type, long l1, long l2,
	long l3, long l4) {
	XEvent e;
	memset(&e, 0, sizeof e);
	e.xclient.type = ClientMessage;
	e.xclient.window = d->target;
	e.xclient.message_type = type;
	e.xclient.format = 32;
	e.xclient.data.l[0] = (long)d->src;
	e.xclient.data.l[1] = l1;
	e.xclient.data.l[2] = l2;
	e.xclient.data.l[3] = l3;
	e.xclient.data.l[4] = l4;
	XSendEvent(d->dpy, d->deliver, False, NoEventMask, &e);
}

// window_property reads one 32-bit item of a property, or returns 0.
static int window_property(Display *dpy, Window w, Atom prop, Atom type,
	unsigned long *out) {
	Atom actual;
	int format;
	unsigned long n, after;
	unsigned char *buf = NULL;
	if (XGetWindowProperty(dpy, w, prop, 0, 1, False, type, &actual, &format,
			&n, &after, &buf) != Success || buf == NULL) {
		return 0;
	}
	int ok = actual == type && format == 32 && n == 1;
	if (ok) {
		*out = ((unsigned long *)buf)[0];
	}
	XFree(buf);
	return ok;
}

// find_target is the topmost XdndAware window under the root point (x, y),
// descending from the root through the window manager's frames.
static Window find_target(struct dragout_x11 *d, int x, int y, int *version,
	Window *deliver) {
	Window w = d->root;
	for (int depth = 0; depth < 64; depth++) {
		Window child = None;
		int tx, ty;
		if (!XTranslateCoordinates(d->dpy, d->root, w, x, y, &tx, &ty, &child) ||
			child == None) {
			return None;
		}
		w = child;

		unsigned long v = 0, proxy = 0;
		Window via = w;
		if (window_property(d->dpy, w, d->proxy, XA_WINDOW, &proxy) && proxy != 0) {
			via = (Window)proxy;
		}
		if (window_property(d->dpy, via, d->aware, XA_ATOM, &v) && v >= 3) {
			*version = v < XDND_VERSION ? (int)v : XDND_VERSION;
			*deliver = via;
			return w;
		}
	}
	return None;
}

static void send_position(struct dragout_x11 *d) {
	send_msg(d, d->position, 0, ((long)d->px << 16) | (d->py & 0xffff),
		(long)d->time, (long)d->copy);
	d->waiting = 1;
	d->pending = 0;
}

static void leave_target(struct dragout_x11 *d) {
	if (d->target != None) {
		send_msg(d, d->leave, 0, 0, 0, 0);
	}
	d->target = None;
	d->accepted = d->waiting = d->pending = 0;
}

static void moved(struct dragout_x11 *d) {
	int version = 0;
	Window deliver = None;
	Window t = find_target(d, d->px, d->py, &version, &deliver);
	if (t != d->target) {
		leave_target(d);
		if (t == None) {
			return;
		}
		d->target = t;
		d->deliver = deliver;
		d->version = version;
		send_msg(d, d->enter, (long)version << 24, (long)d->urilist, None, None);
	}
	if (d->waiting) {
		d->pending = 1;
	} else {
		send_position(d);
	}
}

static void answer_selection(struct dragout_x11 *d, XSelectionRequestEvent *r) {
	XEvent reply;
	memset(&reply, 0, sizeof reply);
	reply.xselection.type = SelectionNotify;
	reply.xselection.requestor = r->requestor;
	reply.xselection.selection = r->selection;
	reply.xselection.target = r->target;
	reply.xselection.time = r->time;
	reply.xselection.property = None;

	Atom property = r->property != None ? r->property : r->target;
	if (r->selection == d->selection && r->target == d->urilist) {
		XChangeProperty(d->dpy, r->requestor, property, d->urilist, 8,
			PropModeReplace, (unsigned char *)d->data, (int)d->len);
		reply.xselection.property = property;
	} else if (r->selection == d->selection && r->target == d->targets) {
		Atom list[2] = {d->targets, d->urilist};
		XChangeProperty(d->dpy, r->requestor, property, XA_ATOM, 32,
			PropModeReplace, (unsigned char *)list, 2);
		reply.xselection.property = property;
	}
	XSendEvent(d->dpy, r->requestor, False, NoEventMask, &reply);
}

// next_event waits for an event until deadline (CLOCK_MONOTONIC
// milliseconds; 0 waits for ever) and reports whether one came.
static long long now_ms(void) {
	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);
	return (long long)ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
}

static int next_event(Display *dpy, XEvent *e, long long deadline) {
	while (!XPending(dpy)) {
		int timeout = -1;
		if (deadline != 0) {
			long long left = deadline - now_ms();
			if (left <= 0) {
				return 0;
			}
			timeout = (int)left;
		}
		struct pollfd p = {ConnectionNumber(dpy), POLLIN, 0};
		if (poll(&p, 1, timeout) == 0) {
			return 0;
		}
	}
	XNextEvent(dpy, e);
	return 1;
}

// --- the drag ------------------------------------------------------------

struct dragout_x11 *dragout_x11_begin(const char *data, size_t len, int *err) {
	Display *dpy = XOpenDisplay(NULL);
	if (dpy == NULL) {
		*err = DRAGOUT_X11_NO_DISPLAY;
		return NULL;
	}
	if (!handler_installed) {
		previous_handler = XSetErrorHandler(on_error);
		handler_installed = 1;
	}
	claim(dpy);

	struct dragout_x11 *d = calloc(1, sizeof *d);
	char *copy = malloc(len);
	if (d == NULL || copy == NULL) {
		free(d);
		free(copy);
		unclaim(dpy);
		XCloseDisplay(dpy);
		*err = DRAGOUT_X11_NO_MEMORY;
		return NULL;
	}
	d->dpy = dpy;
	d->root = DefaultRootWindow(dpy);
	d->data = copy;
	memcpy(d->data, data, len);
	d->len = len;

	d->aware = XInternAtom(dpy, "XdndAware", False);
	d->proxy = XInternAtom(dpy, "XdndProxy", False);
	d->selection = XInternAtom(dpy, "XdndSelection", False);
	d->enter = XInternAtom(dpy, "XdndEnter", False);
	d->position = XInternAtom(dpy, "XdndPosition", False);
	d->status = XInternAtom(dpy, "XdndStatus", False);
	d->leave = XInternAtom(dpy, "XdndLeave", False);
	d->drop = XInternAtom(dpy, "XdndDrop", False);
	d->finished = XInternAtom(dpy, "XdndFinished", False);
	d->copy = XInternAtom(dpy, "XdndActionCopy", False);
	d->urilist = XInternAtom(dpy, "text/uri-list", False);
	d->targets = XInternAtom(dpy, "TARGETS", False);

	// An unmapped window owns the selection and receives the replies.
	d->src = XCreateSimpleWindow(dpy, d->root, -1, -1, 1, 1, 0, 0, 0);
	d->cursor = XCreateFontCursor(dpy, XC_hand2);

	// GLFW let go of its implicit grab just before this, on its own
	// connection; the server may not have processed that yet.
	int grabbed = GrabNotViewable;
	for (int i = 0; i < 50; i++) {
		grabbed = XGrabPointer(dpy, d->root, False,
			ButtonReleaseMask | PointerMotionMask, GrabModeAsync, GrabModeAsync,
			None, d->cursor, CurrentTime);
		if (grabbed == GrabSuccess) {
			break;
		}
		usleep(10000);
	}
	if (grabbed != GrabSuccess) {
		dragout_x11_free(d);
		*err = DRAGOUT_X11_NO_GRAB;
		return NULL;
	}
	XGrabKeyboard(dpy, d->root, False, GrabModeAsync, GrabModeAsync, CurrentTime);
	XSetSelectionOwner(dpy, d->selection, d->src, CurrentTime);

	Window r, c;
	int rx, ry, wx, wy;
	unsigned int mask;
	XQueryPointer(dpy, d->root, &r, &c, &rx, &ry, &wx, &wy, &mask);
	if (!(mask & (Button1Mask | Button2Mask))) {
		// Released before the grab: there is no drag left to run.
		dragout_x11_free(d);
		*err = DRAGOUT_X11_NO_BUTTON;
		return NULL;
	}
	d->px = rx;
	d->py = ry;
	XFlush(dpy);
	*err = DRAGOUT_X11_OK;
	return d;
}

void dragout_x11_run(struct dragout_x11 *d) {
	XEvent e;
	int released = 0, dropped = 0;
	long long deadline = 0;

	d->time = CurrentTime;
	moved(d);
	XFlush(d->dpy);

	for (;;) {
		if (!next_event(d->dpy, &e, deadline)) {
			leave_target(d); // a target that never answered
			break;
		}
		if (e.type == MotionNotify && !released) {
			d->px = e.xmotion.x_root;
			d->py = e.xmotion.y_root;
			d->time = e.xmotion.time;
			moved(d);
		} else if (e.type == ButtonRelease && !released) {
			released = 1;
			d->time = e.xbutton.time;
			XUngrabPointer(d->dpy, CurrentTime);
			XUngrabKeyboard(d->dpy, CurrentTime);
			if (d->target == None) {
				break;
			}
			if (!d->waiting) {
				if (!d->accepted) {
					leave_target(d);
					break;
				}
				send_msg(d, d->drop, 0, (long)d->time, 0, 0);
				dropped = 1;
			}
			deadline = now_ms() + 5000;
		} else if (e.type == KeyPress && !released &&
			XLookupKeysym(&e.xkey, 0) == XK_Escape) {
			leave_target(d);
			break;
		} else if (e.type == ClientMessage && e.xclient.message_type == d->status &&
			(Window)e.xclient.data.l[0] == d->target) {
			d->waiting = 0;
			d->accepted = e.xclient.data.l[1] & 1;
			if (released && !dropped) {
				if (!d->accepted) {
					leave_target(d);
					break;
				}
				send_msg(d, d->drop, 0, (long)d->time, 0, 0);
				dropped = 1;
			} else if (!released && d->pending) {
				send_position(d);
			}
		} else if (e.type == ClientMessage && e.xclient.message_type == d->finished) {
			break;
		} else if (e.type == SelectionRequest) {
			answer_selection(d, &e.xselectionrequest);
		}
		XFlush(d->dpy);
	}
	XFlush(d->dpy);
	dragout_x11_free(d);
}

void dragout_x11_free(struct dragout_x11 *d) {
	XUngrabPointer(d->dpy, CurrentTime);
	XUngrabKeyboard(d->dpy, CurrentTime);
	XFreeCursor(d->dpy, d->cursor);
	XDestroyWindow(d->dpy, d->src);
	unclaim(d->dpy);
	XCloseDisplay(d->dpy);
	free(d->data);
	free(d);
}

void dragout_x11_release_grab(Display *glfw) {
	XUngrabPointer(glfw, CurrentTime);
	XFlush(glfw);
}
