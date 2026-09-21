# Spec 025: decode a picture once

**Issue**: [#24](https://github.com/ushineko/fynedesygn/issues/24)

## Status: COMPLETE

## Context

hotaru's window was reported as reaching a 948 MB high-water mark -- "larger
than an electron app" -- and as climbing without plateauing while sections
were switched.

Measured, with this module's own `profiling` package, sampling the live heap
after a forced collection every five seconds:

	secs   live_MB   sys_MB   RSS_MB
	   0        94      167      388
	  30       227      336      593
	  60       405      530      752
	  95       444      579      764

Then, once the switching stopped and the window was touched again, 158 MB. It
is a **bounded working set**, not a leak: Fyne expires a cached object a
minute after it was last touched, and only sweeps on a canvas refresh, so an
idle window holds everything it built and a busy one holds a minute of it.

The bound is visits per minute times decoded bytes per visit, and the second
number was the problem:

	128.70 MB  image.NewNRGBA    ~12 copies of one 1246x2186 diagram
	 73.49 MB  image.NewPaletted ~179 GIF frames
	 40.73 MB  font tables

`canvas.Image` decodes from its resource, and decodes again on every refresh.
A `markdown.Pane` measures its blocks by rendering them, and a shell section
is rebuilt when somebody navigates to it -- so one diagram was decoded at
10.4 MB a time, over and over, and twelve of those were alive at once.

The GIF number is worse in kind: handing an animation to `canvas.Image`
decodes **every frame**, at the file's own size, to draw a square ninety-six
pixels across. One sixty-frame slideshow is 25 MB per visit.

### Why a cache rather than smaller pictures

Both, but the cache is the one that generalises -- and it is what makes the
size question go away. One copy of a 10 MB diagram is a reasonable thing for
a program to hold. Twelve are not. With a cache the picture's size is a
one-off rather than a per-visit cost, so a document can carry a diagram at the
resolution it deserves.

### Keys are the caller's

Only the caller knows what makes two requests the same picture: a content
hash, a path and a modification time, a name and a size. A key that does not
change when the picture does serves a stale image, which is the one way to use
this badly -- so the package says so and takes the key rather than guessing at
one.

mermaid is the good case: a diagram's resource name is already a hash of the
text it was drawn from, so a changed diagram is a different key.

## Requirements

**R1. A shared, bounded cache of decoded images.** Least-recently-used
eviction by decoded bytes, because an unbounded cache is the leak it was
written to prevent.

**R2. A budget that can be set or turned off**, including to nothing, which
disables the cache without changing calling code.

**R3. The decode runs outside the lock.** A slow decode must not stop another
goroutine reading a picture that is already there; two goroutines racing on a
missing key must still come away with the same image.

**R4. A failed decode is not cached.** A picture that could not be read this
time may be readable next time.

**R5. `mermaid` and `markdown` use it**, so a document pane stops decoding its
diagrams per measure.

**R6. The guidance says what to do**, including decoding to the size you draw
at, which is the other half and is the caller's.

## Acceptance Criteria

- [x] AC1. The second request for a key does not decode, and returns the same
      image.
- [x] AC2. The cache evicts the least recently used when the budget is
      reached, and using a picture keeps it.
- [x] AC3. A budget of zero holds nothing.
- [x] AC4. A failed decode is not remembered.
- [x] AC5. Eight goroutines asking at once all get the same image.
- [x] AC6. `Size` counts the pixel buffer, so a paletted frame is not counted
      as an RGBA.
- [x] AC7. Two diagrams from one resource decode once, asserted through the
      cache's own hit count.
- [x] AC8. `docs/performance.md` carries the finding and the pattern, and the
      package is in the README table.

## Risks & Assumptions

- **A wrong key serves a wrong picture.** That is the failure this design
  accepts in exchange for not hashing megabytes on every call, and it is why
  `Forget` exists and why the doc comment leads with it.
- **64 MB is a guess informed by one program.** It is several large pictures
  or a great many thumbnails, and an order of magnitude below what hotaru was
  holding by accident. A program that needs more should say so rather than
  discover the eviction.
- **A cached image is shared**, so a caller that draws into one draws into
  everybody's. Nothing in this module does; an app that wants to modify a
  picture should copy it.
- **Rollback** is a package nothing has to import, plus reverting two call
  sites.

## Alternatives Considered

Considered keying on a hash of the bytes, which removes the stale-key failure;
rejected because the callers that need this most are handing over megabytes,
and hashing them on every request is the cost the cache exists to avoid.

Considered caching `fyne.Resource` rather than `image.Image`; rejected because
the expensive part is the decode, and a resource is what Fyne decodes *from*.
