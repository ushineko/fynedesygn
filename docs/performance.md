# Performance

This page tells you how to make a program on this module fast, and how to
find out why it is not. It is the second half of the `profiling` package and
the more useful half. The endpoint needs a minute to add, and the mistakes are
in how people read what it reports.

## Measure. Do not read the code and decide.

Somebody reported that the window of clockwork-orange used too much memory.
Three people read the source and gave three answers, and one answer was wrong.
The fourth examination was a heap profile. It needed a minute, and it reported
what the three answers did not: the live heap was correct, the caches were
bounded and worked as designed, and the apparent leak was allocation churn and
the Go arena.

This is why `profiling` exists. **A performance change that you accept from an
argument, and not from a profile, is a guess with code in it.** hotaru has the
same rule for its hardware: accept no change to the write path from its own
telemetry, because a person must look at the machine. Accept no change to a
hot path from a reading of the code.

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

`?gc=1` on the heap is the necessary flag. Without it, the profile contains
memory that the collector has not yet taken, and everything looks like a
leak.

## Reading a Go heap profile

People read three numbers as one number. The three mean different things.

| | what it is | what it means |
|---|---|---|
| `HeapAlloc` after a forced GC | the live heap | the only one that can leak |
| `TotalAlloc` | everything ever allocated | churn: the GC's workload, not memory held |
| `Sys` / `VmHWM` | the arena, and its high-water mark | a burst the runtime grew to fit and kept |
| `VmRSS` | resident pages | overstates both: Go releases with `MADV_FREE`, so released pages stay counted until the kernel wants them |

clockwork-orange measured 228 MB live, 17.7 GB of churn, an arena of 1.9 GB,
and 1.59 GB resident. It had already returned 455 MB of the resident memory.
Only the first number was a question about the design of the program. The
others were questions about the runtime.

**An RSS that increases is not a leak.** A live heap that increases across
forced collections is a leak.

## What actually costs, in a Fyne program

### Rebuilding a widget tree is not free, and may not be collected

Fyne keeps the renderer of a widget in a cache, and [it has not always
destroyed a renderer when the widget goes
away](https://github.com/fyne-io/fyne/issues/4903). A program that rebuilds a
section on a timer makes widgets at that rate.

**Build the tree one time and change it in position.** Replace the text of a
label, the `Resource` of a picture, or a progress value. Rebuild when the
*shape* changes, such as when a control appears or a list receives a row. Do
not rebuild when a number changes.

This is a rule about correctness and not only about speed. A rebuild destroys
the widget that a person is typing into, closes the menu they held open, and
loses their scroll position. The scene list of hotaru, its picture queue and
its dashboard preview each needed a correction for this.

### `Refresh` repaints the whole widget

`RichText` puts every segment it holds in position and paints it again at each
refresh, and a scroller refreshes its content while it moves. One `RichText`
over a README of 221 lines was approximately 250 segments, and needed
approximately 51 ms for a frame during a scroll in the software painter.
`markdown.Pane` draws one block at a time and only near the viewport, and
needed approximately 33 ms for the same document. Use the pane for a document
longer than a few paragraphs.

### Resize arrives hundreds of times, from inside the event poll

Fyne lays out again inside `glfwPollEvents`, and it does this synchronously.
Work in `Resize` therefore stops the event queue, and the window moves
unevenly. A drag of the About section of clockwork-orange put
`markdown.(*Pane).Resize` at 41.9% of all samples, and `measure` at 31.6%.

**Do the work one time.** `markdown.Options.SettleResize` waits until the
width stops changing and then measures one time. A change of a quarter or more
is not a drag: it is a section that appears, a window that becomes full size,
or a first layout. The pane measures those at once, because a delay would show
the wrong heights and gain nothing. A component whose layout does real work
should do the same.

**Do not make the work shorter by changing the answer.** The same
specification removed one of two `MinSize` calls as unnecessary. It was
necessary, because the height of a paragraph is a property of the paragraph
*and* of the width it receives. The document then measured two thirds of the
height it drew at; see specification 021.

### Caches are the live heap, so size them from the profile

Two caches of sixteen RGBA frames at 1600x900 are 176 MB, for panes a few
hundred pixels wide. They were bounded and worked as designed, and they were
much larger than necessary. At 1200x675 and twelve frames they are 74 MB, and
they still hold more than one pixel for each pixel drawn at a desktop scale of
1.5.

Decode to the size that you draw at, and not to the size of the file.

### A control that is not a control is cheaper as a `Swatch`

`widget.Button` measures itself with a theme lookup, a padding calculation and
a `RichText.MinSize` for its label. That is the correct quantity of work for a
button, and too much for a coloured square that a person can click. A resize
profile of the scene editor of hotaru put `buttonRenderer.MinSize` at 14% of
all samples.

`widgets.Swatch` is one widget, and its minimum size is the number you gave
it. Apply the same reasoning to anything that you build as an invisible
control above a drawing. Count the objects, and ask what each object
measures.

### Fyne's caches are keyed by interface, and that is not free

`runtime.mapaccess2` was 28% of that same profile. The renderer cache of Fyne
was 61% of that, and its text-size cache was 26%, with `nilinterhash` and
`nilinterequal` below them, because the keys are interface values. From
outside Fyne you can only ask for fewer lookups, which means fewer widgets.

### Decode a picture once, through the shared cache

`canvas.Image` decodes from its resource, and decodes again at each refresh.
A section that the shell rebuilds when a person navigates to it therefore
decodes its pictures one time for each visit. Each copy stays in memory until
the caches of Fyne remove it a minute later. hotaru was measured with 128 MB
of `image.NewNRGBA`, which was twelve copies of one diagram at 1246x2186, and
73 MB of paletted frames. This is because `canvas.Image` decodes a whole GIF
animation to draw a square ninety-six pixels wide.

	img, err := imagecache.Shared.Get(key, func() (image.Image, error) {
		return gif.Decode(bytes.NewReader(body))
	})
	picture := canvas.NewImageFromImage(img)

`markdown` and `mermaid` do this for you. For the pictures of your own
program, the key is whatever makes two requests the same request: a hash of
the content, or a path and a modification time. A key that does not change
when the picture changes is the one way to use the cache incorrectly.

**The cache also makes a large picture affordable.** One copy of a diagram of
10 MB is reasonable to hold, and twelve copies are not. Make the picture the
size that draws it, and let the cache make that work happen one time.

### Decode to the size you draw at

`canvas.Image` decodes a whole GIF: every frame, at its own resolution. For a
thumbnail, decode the first frame yourself and change its size one time. Use
`gif.Decode` and not `gif.DecodeAll`, then `x/image/draw` to the size that the
list draws. Sixty frames at 640x640 are 25 MB, and a thumbnail of 192 pixels
is 147 KB.

### Give a replaced image resource a name of its own

Fyne keys a decoded image by the name of its resource. If you give a
`canvas.Image` a second `fyne.NewStaticResource("preview.gif", ...)`, it can
draw the first image again.

### Nothing transient may reflow

This is a rule of the design system (`docs/design-system.md`), and it is also
a rule about performance. A banner that moves the page down makes Fyne lay out
everything below it. A popup, or a height that you keep, costs one composite
instead.

## What a task manager shows, and whether to chase it

This part is optional. Read it when a person looks at the process in a system
monitor and objects to the number. That is a different question from whether
the program uses memory well.

These are the measurements of a Fyne program on one machine with a GPU, when
the program had started and was idle:

	Go live heap        24 MB     after a forced collection
	Go arena            57 MB
	RSS                291 MB     what a task manager shows
	PSS                131 MB     the honest share
	Private_Dirty       85 MB     what actually dies with the process

Where the RSS goes:

	77 MB  libnvidia-gpucomp.so      the driver's shader compiler
	24 MB  libLLVM.so                mesa's shader compiler
	24 MB  the program's own binary   fonts, embedded assets, Fyne
	21 MB  libnvidia-eglcore.so
	21 MB  libgallium.so
	 6 MB  libgtk-3.so

**Approximately 145 MB of this is the graphics stack.** The process mapped it
because it opened a GL context. 176 MB of the RSS is `Shared_Clean`, which the
machine shares with every other GL program and counts again in the RSS of each
one. Every Fyne program on that machine has the same minimum, and so does an
Electron program, which also has a browser engine.

	$ grep -E '^(Rss|Pss|Private_Dirty|Shared_Clean)' /proc/$PID/smaps_rollup
	$ awk '/^[0-9a-f]/{n=$6} /^Rss:/{r[n]+=$2} END{for(k in r) print r[k], k}' \
	    /proc/$PID/smaps | sort -rn | head

**Quote the PSS**, and make the `Private_Dirty` smaller. The RSS counts the
pages of a library against every process that maps that library.

### Two levers, if the visible number matters

Both levers change real behaviour and both are measurable. Set them, then
measure again. Do not assume the result.

**A memory ceiling below the maximum.** `profiling.Limit` makes the collector
work during a burst. Without it, the arena grows to hold the burst and keeps
that size. Choose the value from a profile: above the measured maximum live
heap, and not above the largest RSS that somebody saw.

**`GODEBUG=madvdontneed=1`.** Go releases pages with `MADV_FREE`, so the RSS
counts a released page until the kernel needs it. This is how a process with a
live heap of 270 MB shows a maximum of 774 MB. `madvdontneed` returns the
pages at once, so the RSS agrees with the live heap. It costs page faults when
the heap grows again.

### And the honest answer

A program with a live heap of 24 MB, in a proportional footprint of 131 MB
that belongs mostly to the graphics driver, does not use memory badly. This is
true whatever the number in a monitor shows. Correct what a profile reports as
waste, and explain the rest.

## The reusable pieces

Use these rather than write the same thing again in each program. That is
what this module is for, and a component is complete when a program can delete
its own copy.

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

- Take a profile before and a profile after, on the same machine, and do the
  same thing both times.
- Put the number in the commit message, with the machine it came from.
- Write a test that fails if the hot path returns, where a test is possible.
  `markdown` asserts that a new pane measures the same height as one that is
  laid out, and that a drag measures one time.
- Change nothing in the visible behaviour, or write in the specification what
  you changed.
