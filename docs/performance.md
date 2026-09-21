# Performance

The standard guidance for making a program on this module fast, and for
finding out why it is not. It is the second half of the `profiling` package
and the more valuable one: the endpoint takes a minute to add, and reading
what it says is where the mistakes are.

## Measure. Do not read the code and decide.

clockwork-orange's window was reported as using too much memory. It was
answered three times from reading the source, and one of those answers was
wrong. The fourth look was a heap profile, took a minute, and said something
none of the three had: the live heap was fine, the caches were bounded and
working, and what looked like a leak was allocation churn plus Go's arena.

That is why `profiling` exists. **A performance change accepted on reasoning
rather than on a profile is a guess that now has code in it.** The same rule
hotaru writes for its hardware -- no change to the write path accepted on its
own telemetry, somebody looks at the machine -- applies here: no change to a
hot path accepted on a reading of the code.

## Turning it on

```go
defer profiling.FromEnv("MYAPP_PPROF", report)()
profiling.Limit(768<<20, "MYAPP_MEMLIMIT", report)
```

```console
$ MYAPP_PPROF=:6060 myapp-gui
profiling on http://127.0.0.1:6060/debug/pprof/

$ go tool pprof -top -inuse_space 'http://127.0.0.1:6060/debug/pprof/heap?gc=1'
$ go tool pprof -top 'http://127.0.0.1:6060/debug/pprof/profile?seconds=20'
$ curl -s 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=1' | head -40
```

`?gc=1` on the heap is the important flag: without it the profile includes
garbage that has not been collected yet, and everything looks like a leak.

## Reading a Go heap profile

Three numbers get read as one, and they mean different things.

| | what it is | what it means |
|---|---|---|
| `HeapAlloc` after a forced GC | the live heap | the only one that can leak |
| `TotalAlloc` | everything ever allocated | churn: the GC's workload, not memory held |
| `Sys` / `VmHWM` | the arena, and its high-water mark | a burst the runtime grew to fit and kept |
| `VmRSS` | resident pages | overstates both: Go releases with `MADV_FREE`, so released pages stay counted until the kernel wants them |

clockwork-orange measured 228 MB live, 17.7 GB churn, a 1.9 GB arena and
1.59 GB resident, of which 455 MB had already been handed back. Only the first
number was a question about the program's design; the rest were questions
about the runtime.

**A climbing RSS is not a leak.** A live heap that climbs across forced
collections is.

## What actually costs, in a Fyne program

### Rebuilding a widget tree is not free, and may not be collected

Fyne caches a widget's renderer, and [renderers have not always been
destroyed when the widget goes away](https://github.com/fyne-io/fyne/issues/4903).
A program that rebuilds a section on a timer is creating widgets at that rate.

So: **build the tree once and update it in place.** Swap a label's text, a
picture's `Resource`, a progress value. Rebuild when the *shape* changes --
a control appears, a list gains a row -- and not when a number does.

This is also a correctness rule, not only a speed one. A rebuild destroys the
widget somebody is typing into, closes the dropdown they were holding open,
and loses their scroll position. hotaru's scene list, its picture drop queue
and its dashboard preview each had to be fixed for exactly that.

### `Refresh` repaints the whole widget

`RichText` lays out and repaints every segment it holds on each refresh, and a
scroller refreshes its content as it moves. One `RichText` over a 221-line
README was about 250 segments and cost ~51 ms a frame during a scroll in the
software painter; `markdown.Pane` renders per block and only near the
viewport, at ~33 ms for the same document. Use the pane for anything longer
than a few paragraphs.

### Resize arrives hundreds of times, from inside the event poll

Fyne relays out synchronously inside `glfwPollEvents`, so work done in
`Resize` blocks the event queue and the window moves in bursts. A drag of
clockwork-orange's About section put `markdown.(*Pane).Resize` at 41.9% of all
samples and `measure` at 31.6%.

**Coalesce it.** `markdown.Options.SettleResize` waits for the width to stop
changing and measures once; a change of a quarter or more is a jump rather
than a drag -- a section being shown, a window maximised, a first layout --
and is measured at once, because delaying that would show the wrong heights
for no gain. Any component whose layout costs real work should do the same.

**And do not shorten the work by changing the answer.** The same spec dropped
one of two `MinSize` calls as redundant; it was not, because a paragraph's
height is a property of the paragraph *and* the width it is given, and the
document came out two thirds of its drawn height (spec 021).

### Caches are the live heap, so size them from the profile

Two sixteen-frame caches of 1600x900 RGBA are 176 MB for panes a few hundred
pixels across -- bounded, working as designed, and far larger than they needed
to be. At 1200x675 and twelve frames they are 74 MB, and still over a pixel
per pixel at a 1.5x desktop scale.

Decode to the size you draw at, not the size the file happens to be.

### A control that is not a control is cheaper as a `Swatch`

`widget.Button` measures itself with a theme lookup, a padding calculation and
a `RichText.MinSize` for its label -- which is the right amount of work for a
button and the wrong amount for a coloured square somebody can click. A resize
profile of hotaru's scene editor put `buttonRenderer.MinSize` at 14% of all
samples.

`widgets.Swatch` is one widget whose minimum size is the number it was given.
The same reasoning applies to anything built as "an invisible control stacked
over a drawing": count the objects, and ask what each of them measures.

### Fyne's caches are keyed by interface, and that is not free

28% of that same profile was `runtime.mapaccess2`, 61% of it from Fyne's
renderer cache and 26% from its text-size cache -- with `nilinterhash` and
`nilinterequal` underneath, because the keys are interface values. There is no
lever on this from outside Fyne except asking for fewer lookups, which means
fewer widgets.

### Decode a picture once, through the shared cache

`canvas.Image` decodes from its resource, and decodes again on every refresh.
A section rebuilt when somebody navigates to it therefore decodes its pictures
once per visit, and each copy stays alive until Fyne's caches expire it a
minute later. hotaru was measured holding 128 MB of `image.NewNRGBA` -- twelve
copies of one 1246x2186 diagram -- and 73 MB of paletted frames, because
handing a GIF to `canvas.Image` decodes the whole animation to draw a square
ninety-six pixels across.

	img, err := imagecache.Shared.Get(key, func() (image.Image, error) {
		return gif.Decode(bytes.NewReader(body))
	})
	picture := canvas.NewImageFromImage(img)

`markdown` and `mermaid` do this for you. For an app's own pictures, the key
is whatever makes two requests the same thing -- a content hash, a path and a
modification time -- and a key that does not change when the picture does is
the one way to use it badly.

**It is also what makes a large picture affordable.** One copy of a 10 MB
diagram is a reasonable thing to hold; twelve are not. Size the picture for
what draws it, and then let the cache make the size a one-off.

### Decode to the size you draw at

A GIF handed to `canvas.Image` is decoded in full: every frame, at its own
resolution. For a thumbnail, decode the first frame yourself and scale it once
-- `gif.Decode` rather than `gif.DecodeAll`, and `x/image/draw` to the size the
list actually draws. Sixty frames at 640x640 is 25 MB; a 192-pixel thumbnail
is 147 KB.

### Give a replaced image resource a name of its own

Fyne caches a decoded image against its resource's name. Handing a
`canvas.Image` a second `fyne.NewStaticResource("preview.gif", ...)` can draw
the first one back.

### Nothing transient may reflow

Already the design system's rule (`docs/design-system.md`), and it is a
performance rule too: a banner that pushes the page down forces a layout of
everything below it. Popups and reserved heights cost one composite instead.

## The reusable pieces

Prefer these over hand-rolling the same thing per app. That is what this
module is for, and a component is done when an app can delete its copy.

| For | Use |
|---|---|
| A profiling endpoint, a memory ceiling | `profiling` |
| Long documents | `markdown.Pane`, `markdown.Section` |
| Diagrams | `mermaid` -- rendered at build time, never at runtime |
| Log output that would otherwise refresh per line | `logpane` |
| Tabular data | `table` -- not a Markdown table, not a `RichText` |
| Asserting nothing scrolls inside a section | `fynetest.ScrollableIn` |
| Asserting what a headless tree says | `fynetest.Text`, `fynetest.TextIn` |

## Before claiming a speed-up

- A profile before and a profile after, on the same machine, doing the same
  thing.
- The number in the commit message, with what it was taken on.
- A test that fails if the hot path grows back, where one can be written:
  `markdown` pins that a fresh pane measures the same height as a laid-out
  one, and that a drag measures once.
- Nothing in the visible behaviour changed, or the spec says what did.
