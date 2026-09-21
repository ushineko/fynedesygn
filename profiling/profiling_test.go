package profiling

import (
	"io"
	"math"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// said collects what a start reported, which is where the address ends up.
func said(t *testing.T) (func(string), *[]string) {
	t.Helper()
	var lines []string
	return func(s string) { lines = append(lines, s) }, &lines
}

func TestNothingIsServedUntilSomethingAsks(t *testing.T) {
	// Off by default. A profiling endpoint exposes a program's memory and
	// goroutine state; it is opt-in for the same reason it is loopback-only.
	report, lines := said(t)
	stop := Serve("", report)
	t.Cleanup(stop)

	require.Empty(t, *lines)
}

func TestAProfileIsServedAndStops(t *testing.T) {
	report, lines := said(t)
	stop := Serve("127.0.0.1:0", report)
	t.Cleanup(stop)

	require.Len(t, *lines, 1)
	addr := strings.TrimSuffix(strings.TrimPrefix((*lines)[0], "profiling on "), "/debug/pprof/")

	res, err := http.Get(addr + "/debug/pprof/heap?debug=1") //nolint:noctx // a test against itself
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Body.Close() })
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "heap profile")

	stop()
	after, err := http.Get(addr + "/debug/pprof/heap") //nolint:noctx // a test against itself
	if err == nil {
		_ = after.Body.Close()
	}
	require.Error(t, err, "the server kept serving after it was stopped")
}

func TestAPortAloneIsStillLoopback(t *testing.T) {
	/*
		The whole reason this is not one line of net/http/pprof. A bare port
		means every interface to net.Listen, which is how a debugging
		endpoint ends up reachable from the network -- and what it serves is
		the program's memory.
	*/
	require.Equal(t, "127.0.0.1:6060", loopback(":6060"))
	require.Equal(t, "127.0.0.1:6060", loopback("6060"))
	require.Equal(t, "127.0.0.1:6060", loopback("0.0.0.0:6060"))
	require.Equal(t, "127.0.0.1:6060", loopback("192.168.1.10:6060"))
}

func TestAnAddressThatWillNotBindIsNotFatal(t *testing.T) {
	// A profiling server that cannot bind is a reason to carry on without
	// one, never a reason for a window not to open.
	report, lines := said(t)
	require.NotPanics(t, func() { Serve("127.0.0.1:1", report)() })
	require.Len(t, *lines, 1)
	require.Contains(t, (*lines)[0], "profiling on")
}

// restore puts the process-wide limit back: SetMemoryLimit outlives the test
// that set it, and every other test in this package runs under whatever the
// last one left.
func restore(t *testing.T) {
	t.Helper()
	was := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(was) })
}

func TestACeilingIsSet(t *testing.T) {
	restore(t)
	Limit(768<<20, "FYNEDESYGN_TEST_MEMLIMIT", nil)
	require.Equal(t, int64(768<<20), debug.SetMemoryLimit(-1))
}

func TestTheRuntimesOwnLimitWins(t *testing.T) {
	/*
		GOMEMLIMIT is read before this program runs, so overriding it from
		inside would make the documented variable a lie.
	*/
	restore(t)
	was := debug.SetMemoryLimit(-1)
	t.Setenv("GOMEMLIMIT", "512MiB")

	Limit(768<<20, "FYNEDESYGN_TEST_MEMLIMIT", nil)
	require.Equal(t, was, debug.SetMemoryLimit(-1), "the runtime's own limit was overridden")
}

func TestTheCeilingCanBeSetOrTurnedOff(t *testing.T) {
	restore(t)
	t.Setenv("FYNEDESYGN_TEST_MEMLIMIT", strconv.FormatInt(256<<20, 10))
	Limit(768<<20, "FYNEDESYGN_TEST_MEMLIMIT", nil)
	require.Equal(t, int64(256<<20), debug.SetMemoryLimit(-1))

	t.Setenv("FYNEDESYGN_TEST_MEMLIMIT", "0")
	debug.SetMemoryLimit(math.MaxInt64)
	Limit(768<<20, "FYNEDESYGN_TEST_MEMLIMIT", nil)
	require.Equal(t, int64(math.MaxInt64), debug.SetMemoryLimit(-1),
		"zero should leave the runtime's own behaviour alone")
}

func TestAnUnreadableCeilingChangesNothing(t *testing.T) {
	// A typo in a desktop entry should not silently change how a window
	// collects.
	restore(t)
	debug.SetMemoryLimit(math.MaxInt64)
	t.Setenv("FYNEDESYGN_TEST_MEMLIMIT", "lots")

	report, lines := said(t)
	Limit(768<<20, "FYNEDESYGN_TEST_MEMLIMIT", report)

	require.Equal(t, int64(math.MaxInt64), debug.SetMemoryLimit(-1))
	require.Len(t, *lines, 1)
	require.Contains(t, (*lines)[0], "not a number of bytes")
}
