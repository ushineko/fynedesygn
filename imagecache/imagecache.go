/*
Package imagecache holds decoded images so nothing decodes one twice.

Fyne's `canvas.Image` decodes from its resource, and it decodes again on every
refresh. A program that rebuilds a section when the user navigates to it --
which is what a shell section does -- therefore decodes the same picture once
per visit, and each copy stays alive until Fyne's own caches expire it a
minute later.

Measured in hotaru: 128 MB of `image.NewNRGBA` after a minute of moving
between sections, which is twelve copies of one 1246x2186 diagram; and 73 MB
of `image.NewPaletted`, because handing a GIF to `canvas.Image` decodes the
whole animation -- sixty frames, at 640x640, to draw a square ninety-six
pixels across.

A cache answers both: the twelfth visit gets the first visit's image, and the
size of that image stops being a per-visit cost. **Which is what makes a big
picture affordable**: one copy of a 10 MB diagram is a reasonable thing to
hold, and twelve of them is not.

	img, err := imagecache.Shared.Get(key, func() (image.Image, error) {
		return gif.Decode(bytes.NewReader(body))
	})

The key is the caller's, because only the caller knows what makes two
requests the same thing: a content hash, a path and a modification time, a
name and a size. A key that does not change when the picture does will serve
a stale image, which is the one way to use this badly.
*/
package imagecache

import (
	"image"
	"sync"
)

/*
DefaultBudget is how much decoded image a shared cache holds before it starts
forgetting the least recently used.

64 MB is several large pictures or a great many thumbnails, and an order of
magnitude below what an uncached program was measured holding by accident.
A program that draws more than this at once should say so with SetBudget
rather than discovering the eviction.
*/
const DefaultBudget int64 = 64 << 20

// Shared is the cache a program uses unless it has a reason not to. One per
// program: two caches of the same pictures is the problem this solves, again.
var Shared = New(DefaultBudget)

// Store is a bounded cache of decoded images, least-recently-used first out.
type Store struct {
	mu     sync.Mutex
	budget int64
	held   int64
	// order is keys oldest-first. A slice rather than a list: a cache this
	// size is tens of entries, and a linear scan of tens is faster than the
	// pointers a list would need.
	order []string
	items map[string]image.Image
	sizes map[string]int64
	// hits and misses are for a program that wants to know whether its keys
	// are doing what it thinks.
	hits, misses int64
}

// New is a cache holding up to budget bytes of decoded image. A budget of
// zero or less holds nothing, which is a way to turn the cache off without
// changing any calling code.
func New(budget int64) *Store {
	return &Store{
		budget: budget,
		items:  map[string]image.Image{},
		sizes:  map[string]int64{},
	}
}

/*
Get returns the image for a key, decoding it only if this cache does not have
it.

decode runs outside the lock, so a slow decode does not stop another
goroutine reading a picture it already has. Two goroutines asking for the same
missing key both decode; the second one's result is dropped in favour of the
first, which costs one wasted decode and avoids holding a lock across an
arbitrary amount of work.

An error from decode is returned and nothing is cached: a picture that could
not be read this time may be readable next time, and remembering the failure
would be a cache of bad news.
*/
func (s *Store) Get(key string, decode func() (image.Image, error)) (image.Image, error) {
	if img, ok := s.lookup(key); ok {
		return img, nil
	}

	img, err := decode()
	if err != nil {
		return nil, err
	}
	return s.keep(key, img), nil
}

func (s *Store) lookup(key string) (image.Image, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	img, ok := s.items[key]
	if !ok {
		s.misses++
		return nil, false
	}
	s.hits++
	s.touch(key)
	return img, true
}

// keep stores an image and returns the one to use, which is the one already
// there when another goroutine got in first.
func (s *Store) keep(key string, img image.Image) image.Image {
	s.mu.Lock()
	defer s.mu.Unlock()

	if already, ok := s.items[key]; ok {
		return already
	}
	if s.budget <= 0 {
		return img
	}

	size := Size(img)
	s.items[key] = img
	s.sizes[key] = size
	s.order = append(s.order, key)
	s.held += size
	s.evict()
	return img
}

// evict drops the oldest entries until the cache is inside its budget. The
// caller holds the lock.
func (s *Store) evict() {
	for s.held > s.budget && len(s.order) > 0 {
		oldest := s.order[0]
		s.order = s.order[1:]
		s.held -= s.sizes[oldest]
		delete(s.items, oldest)
		delete(s.sizes, oldest)
	}
}

// touch moves a key to the newest end. The caller holds the lock.
func (s *Store) touch(key string) {
	for i, held := range s.order {
		if held != key {
			continue
		}
		s.order = append(s.order[:i], s.order[i+1:]...)
		s.order = append(s.order, key)
		return
	}
}

/*
Forget drops one key.

For the program that knows a picture has changed: a file replaced, a
conversion redone. The alternative -- keys that carry a version -- is better
where a caller can manage it, because a forgotten key is a decode and a stale
key is a wrong picture.
*/
func (s *Store) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if size, ok := s.sizes[key]; ok {
		s.held -= size
		delete(s.items, key)
		delete(s.sizes, key)
		s.touch(key)
		s.order = s.order[:len(s.order)-1]
	}
}

// Clear drops everything.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = map[string]image.Image{}
	s.sizes = map[string]int64{}
	s.order = nil
	s.held = 0
}

// Held is how much decoded image the cache is holding, in bytes.
func (s *Store) Held() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held
}

// Counts are the hits and misses since the cache was made, for a program
// that wants to know whether its keys are doing what it thinks.
func (s *Store) Counts() (hits, misses int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits, s.misses
}

// SetBudget changes how much the cache holds, evicting at once if the new
// budget is smaller.
func (s *Store) SetBudget(bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.budget = bytes
	s.evict()
}

/*
Size is how much memory a decoded image occupies, near enough.

From the pixel buffer where the type has one, because that is where the
memory is: an RGBA is four bytes a pixel and a paletted animation frame is
one. An image type this does not know is counted at four bytes a pixel, which
is the worst case among the standard ones and the right way to be wrong in a
budget.
*/
func Size(img image.Image) int64 {
	if img == nil {
		return 0
	}
	switch it := img.(type) {
	case *image.RGBA:
		return int64(len(it.Pix))
	case *image.NRGBA:
		return int64(len(it.Pix))
	case *image.RGBA64:
		return int64(len(it.Pix))
	case *image.NRGBA64:
		return int64(len(it.Pix))
	case *image.Gray:
		return int64(len(it.Pix))
	case *image.Paletted:
		return int64(len(it.Pix))
	case *image.YCbCr:
		return int64(len(it.Y) + len(it.Cb) + len(it.Cr))
	}
	b := img.Bounds()
	return int64(b.Dx()) * int64(b.Dy()) * 4
}
