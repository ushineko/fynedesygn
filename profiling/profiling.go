/*
Package profiling is how an app on this module finds out why it is slow.

It exists because the alternative is inference. clockwork-orange's memory was
read from the source three times and answered three ways, one of them wrong;
the fourth look was a heap profile and took a minute (its specs 017 and 018).
The lesson generalises, and this package is where it stops being one app's.

Two things, both opt-in and both off by default:

	stop := profiling.FromEnv("HOTARU_PPROF", report)
	defer stop()

	profiling.Limit(768<<20, "HOTARU_MEMLIMIT", report)

See docs/performance.md for what to do with what comes back.
*/
package profiling

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

// Timeouts for the profiling server. A profile can legitimately take thirty
// seconds, hence the generous write timeout; the rest keep a stuck client
// from holding the listener open.
const (
	ReadTimeout   = 10 * time.Second
	WriteTimeout  = 2 * time.Minute
	ListenTimeout = 5 * time.Second
)

/*
FromEnv starts a profiling server when the named variable holds an address,
and returns a function that stops it.

Off unless asked for, and a no-op to stop when it was not, so a caller needs
no branch:

	defer profiling.FromEnv("HOTARU_PPROF", report)()

report is told what happened and may be nil. A profiling server that cannot
bind is a reason to carry on without one, never a reason for a window not to
open.
*/
func FromEnv(name string, report func(string)) func() {
	return Serve(os.Getenv(name), report)
}

/*
Serve starts a profiling server on addr, or does nothing when addr is empty.

**Bound to the loopback interface whatever addr says.** A profiling endpoint
exposes a program's memory and goroutine state and has no business on a
network interface; the app is a desktop program on a machine with a browser,
and the address is for the person sitting at it. A bare port means loopback
here, where to net.Listen it would mean every interface.

	HOTARU_PPROF=:6060 hotaru-gui
	go tool pprof http://127.0.0.1:6060/debug/pprof/heap
*/
func Serve(addr string, report func(string)) func() {
	if report == nil {
		report = func(string) {}
	}
	if addr == "" {
		return func() {}
	}
	addr = loopback(addr)

	/*
		Registered by hand rather than by importing net/http/pprof for its
		side effect on DefaultServeMux: an app should not be one import away
		from serving profiles from a mux something else might also use.
	*/
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// ListenConfig rather than net.Listen: the linter wants a context on the
	// listen, and a server that cannot bind promptly is one to carry on
	// without.
	ctx, cancel := context.WithTimeout(context.Background(), ListenTimeout)
	defer cancel()
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		report(fmt.Sprintf("profiling on %s: %v", addr, err))
		return func() {}
	}

	server := &http.Server{Handler: mux, ReadHeaderTimeout: ReadTimeout, WriteTimeout: WriteTimeout}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			report(fmt.Sprintf("profiling: %v", err))
		}
	}()
	report(fmt.Sprintf("profiling on http://%s/debug/pprof/", listener.Addr()))

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
}

/*
loopback forces addr onto the loopback interface.

A bare port (":6060") means every interface to net.Listen, which is how a
debugging endpoint ends up reachable from the network. Whatever host the
caller gave, the listener is 127.0.0.1; a port alone is honoured, so "6060"
works.
*/
func loopback(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return net.JoinHostPort("127.0.0.1", addr)
	}
	return net.JoinHostPort("127.0.0.1", port)
}

/*
Limit sets a soft memory ceiling, unless the runtime was already told one.

Soft, not a cap: Go runs the collector harder rather than failing an
allocation, so a machine doing something the number did not anticipate
degrades into more GC and not a crash. What it buys is the arena not growing
to fit a burst and then keeping it -- clockwork-orange's window reached a
1.9 GB high-water mark over a live heap of 228 MB, because Go grows to about
twice the live set at the last collection and does not hand the arena back.

`GOMEMLIMIT` wins outright: it is the runtime's own switch, read before the
program runs, and overriding it from inside would make the documented variable
a lie. env names a variable of the app's own, in bytes; zero or negative turns
the ceiling off, and a value that will not parse changes nothing rather than
guessing -- a typo in a desktop entry should not silently change how a window
collects.

Sized from a profile, never from a feeling. See docs/performance.md.
*/
func Limit(bytes int64, env string, report func(string)) {
	if report == nil {
		report = func(string) {}
	}
	if os.Getenv("GOMEMLIMIT") != "" {
		return
	}
	if set := os.Getenv(env); set != "" {
		asked, err := strconv.ParseInt(set, 10, 64)
		if err != nil {
			report(fmt.Sprintf("%s=%q is not a number of bytes; leaving the runtime alone", env, set))
			return
		}
		bytes = asked
	}
	if bytes <= 0 {
		return
	}
	debug.SetMemoryLimit(bytes)
}
