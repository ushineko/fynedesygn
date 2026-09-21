# Spec 023: a profile rather than a reading

**Issue**: [#24](https://github.com/ushineko/fynedesygn/issues/24)

## Status: COMPLETE

## Context

clockwork-orange has an opt-in pprof endpoint (its spec 017) and a soft memory
ceiling sized from a heap profile (its spec 018). terrariabonker has a
`--pprof` flag of its own. hotaru has neither and is about to be profiled.

That is the copy-carrying problem this module exists to end, and the code is
the smaller half of it. The larger half is what reading the profile taught,
which currently lives in one app's spec files where nobody using this module
would find it:

- clockwork-orange's memory was answered three times from reading the source,
  and one of those answers was wrong. The heap profile took a minute.
- What looked like a leak was 228 MB of live heap, 17.7 GB of churn, an arena
  grown to 1.9 GB to fit a burst, and an RSS overstating even that because Go
  releases with `MADV_FREE`.
- The window moved in bursts during a drag because `Resize` arrives hundreds
  of times from inside the event poll, and this module's own document pane
  measured on every one of them (spec 015).
- And the fix for that went one step too far and changed an answer, which
  spec 021 had to undo.

None of that is discoverable from the README.

### Why the endpoint is loopback whatever it is told

`:6060` means every interface to `net.Listen`. What it serves is the
program's memory and goroutine state. The app is a desktop program on a
machine with a browser; the address is for the person sitting at it, and the
package forces the host rather than trusting the caller to.

### Why the ceiling defers to GOMEMLIMIT

It is the runtime's own switch, read before the program runs. Overriding it
from inside would make a documented variable a lie.

## Requirements

**R1. A profiling endpoint any app can turn on**, off by default, from an
environment variable, returning a stop function that is a no-op when nothing
was started.

**R2. Loopback, whatever address it is given.** A bare port is honoured as a
port and never as every interface.

**R3. Failing to bind is not fatal.** It is reported and the window opens.

**R4. A soft memory ceiling** that `GOMEMLIMIT` overrides, that an app's own
variable can set or switch off, and that an unreadable value leaves alone.

**R5. `docs/performance.md`**, linked from the README's Documentation list,
with the package in the package table: how to measure, how to read a heap
profile, and what costs in a Fyne program.

**R6. A standing rule in the project guidelines**, because a document is only
read by somebody who already knows it exists: performance is measured, a
component that would be rebuilt per app belongs here, and a widget tree is
built once and updated in place.

## Acceptance Criteria

- [x] AC1. Nothing is served until an address asks for one.
- [x] AC2. A served profile answers `/debug/pprof/heap`, and stops when the
      returned function is called.
- [x] AC3. `:6060`, `6060`, `0.0.0.0:6060` and a LAN address all listen on
      127.0.0.1.
- [x] AC4. An address that cannot bind reports and does not panic.
- [x] AC5. The ceiling is set; `GOMEMLIMIT` wins; an app's own variable sets
      or disables it; an unreadable value changes nothing.
- [x] AC6. The memory tests restore the process-wide limit, which outlives
      the test that set it.
- [x] AC7. `docs/performance.md` exists, is linked from the README, and
      `profiling` is in the package table.
- [x] AC8. The project guidelines carry the rule.

## Risks & Assumptions

- **A profiling endpoint in a shipped binary** is dead code until an
  environment variable names an address, and serves only to loopback when it
  does. The alternative -- a build tag -- means the binary somebody has
  trouble with is not the binary that can be profiled.
- **The guidance is one program's experience.** clockwork-orange's numbers are
  from image previews and hotaru's will be from something else; what
  generalises is the method and the distinction between live heap, churn and
  RSS, which is why those are what the document leads with.
- **Rollback** is removing a package nothing else imports.

## Alternatives Considered

Considered importing `net/http/pprof` for its side effect on
`DefaultServeMux`, which is the usual one-line version; rejected because an
app should not be one import away from serving profiles from a mux something
else might also use.
