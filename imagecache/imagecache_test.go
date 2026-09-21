package imagecache_test

import (
	"errors"
	"image"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/imagecache"
)

// picture is a decoded image of a known size: w*h*4 bytes.
func picture(w, h int) image.Image { return image.NewRGBA(image.Rect(0, 0, w, h)) }

func decodes(t *testing.T, w, h int, count *int) func() (image.Image, error) {
	t.Helper()
	return func() (image.Image, error) {
		*count++
		return picture(w, h), nil
	}
}

func TestTheSecondAskDoesNotDecode(t *testing.T) {
	/*
		The whole point. A shell section is rebuilt when somebody navigates
		to it, so a picture in one is decoded once per visit -- twelve copies
		of one diagram were measured alive in hotaru after a minute of moving
		between sections.
	*/
	store := imagecache.New(imagecache.DefaultBudget)
	decoded := 0

	first, err := store.Get("k", decodes(t, 10, 10, &decoded))
	require.NoError(t, err)
	second, err := store.Get("k", decodes(t, 10, 10, &decoded))
	require.NoError(t, err)

	require.Equal(t, 1, decoded, "the same key decoded twice")
	require.Same(t, first, second, "the same key gave two images")
}

func TestTheOldestGoesWhenTheBudgetIsReached(t *testing.T) {
	// Bounded, or it is the leak it was written to fix.
	const one = 100 * 100 * 4
	store := imagecache.New(2 * one)
	count := 0

	_, _ = store.Get("a", decodes(t, 100, 100, &count))
	_, _ = store.Get("b", decodes(t, 100, 100, &count))
	require.Equal(t, int64(2*one), store.Held())

	_, _ = store.Get("c", decodes(t, 100, 100, &count))
	require.Equal(t, int64(2*one), store.Held(), "the cache grew past its budget")

	// "a" was the oldest, so it is the one that went.
	_, _ = store.Get("a", decodes(t, 100, 100, &count))
	require.Equal(t, 4, count)
}

func TestUsingAPictureKeepsIt(t *testing.T) {
	// Least *recently used*, not oldest: a picture somebody keeps looking at
	// should not be evicted by one they looked at once.
	const one = 100 * 100 * 4
	store := imagecache.New(2 * one)
	count := 0

	_, _ = store.Get("a", decodes(t, 100, 100, &count)) // decodes
	_, _ = store.Get("b", decodes(t, 100, 100, &count)) // decodes
	_, _ = store.Get("a", decodes(t, 100, 100, &count)) // a hit, and touches a
	_, _ = store.Get("c", decodes(t, 100, 100, &count)) // decodes, evicting b
	require.Equal(t, 3, count)

	// a was touched and survived; b was not and did not.
	_, _ = store.Get("a", decodes(t, 100, 100, &count))
	require.Equal(t, 3, count, "the touched picture was decoded again")
	_, _ = store.Get("b", decodes(t, 100, 100, &count))
	require.Equal(t, 4, count, "the untouched picture was kept")
}

func TestAFailedDecodeIsNotRemembered(t *testing.T) {
	// A picture that could not be read this time may be readable next time,
	// and remembering the failure would be a cache of bad news.
	store := imagecache.New(imagecache.DefaultBudget)
	tries := 0

	_, err := store.Get("k", func() (image.Image, error) {
		tries++
		return nil, errors.New("no")
	})
	require.Error(t, err)

	got, err := store.Get("k", decodes(t, 10, 10, &tries))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, 2, tries)
}

func TestABudgetOfNothingHoldsNothing(t *testing.T) {
	// A way to turn the cache off without changing any calling code.
	store := imagecache.New(0)
	count := 0

	first, err := store.Get("k", decodes(t, 10, 10, &count))
	require.NoError(t, err)
	second, err := store.Get("k", decodes(t, 10, 10, &count))
	require.NoError(t, err)

	require.Equal(t, 2, count)
	require.NotSame(t, first, second)
	require.Zero(t, store.Held())
}

func TestForgettingAPictureFreesIt(t *testing.T) {
	store := imagecache.New(imagecache.DefaultBudget)
	count := 0

	_, _ = store.Get("k", decodes(t, 100, 100, &count))
	require.Positive(t, store.Held())

	store.Forget("k")
	require.Zero(t, store.Held())

	_, _ = store.Get("k", decodes(t, 100, 100, &count))
	require.Equal(t, 2, count)
}

func TestShrinkingTheBudgetEvictsAtOnce(t *testing.T) {
	const one = 100 * 100 * 4
	store := imagecache.New(4 * one)
	count := 0
	for _, key := range []string{"a", "b", "c", "d"} {
		_, _ = store.Get(key, decodes(t, 100, 100, &count))
	}
	require.Equal(t, int64(4*one), store.Held())

	store.SetBudget(one)
	require.Equal(t, int64(one), store.Held())
}

func TestTwoGoroutinesAskingAtOnceGetOnePicture(t *testing.T) {
	/*
		The decode runs outside the lock, so both may run it -- but both must
		come away with the same image, or two halves of a program draw two
		copies and the cache has bought nothing.
	*/
	store := imagecache.New(imagecache.DefaultBudget)

	var wg sync.WaitGroup
	got := make([]image.Image, 8)
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			img, err := store.Get("k", func() (image.Image, error) { return picture(50, 50), nil })
			require.NoError(t, err)
			got[i] = img
		}()
	}
	wg.Wait()

	for i := 1; i < len(got); i++ {
		require.Same(t, got[0], got[i], "goroutine %d got a different picture", i)
	}
}

func TestSizeIsTheBufferWhereThereIsOne(t *testing.T) {
	// A budget in bytes is only as good as the count, and a paletted frame
	// is one byte a pixel where an RGBA is four.
	require.Equal(t, int64(100*100*4), imagecache.Size(image.NewRGBA(image.Rect(0, 0, 100, 100))))
	require.Equal(t, int64(100*100), imagecache.Size(
		image.NewPaletted(image.Rect(0, 0, 100, 100), nil)))
	require.Zero(t, imagecache.Size(nil))
}

func TestTheCacheSaysWhetherTheKeysWork(t *testing.T) {
	store := imagecache.New(imagecache.DefaultBudget)
	count := 0

	_, _ = store.Get("a", decodes(t, 10, 10, &count))
	_, _ = store.Get("a", decodes(t, 10, 10, &count))
	_, _ = store.Get("b", decodes(t, 10, 10, &count))

	hits, misses := store.Counts()
	require.Equal(t, int64(1), hits)
	require.Equal(t, int64(2), misses)
}
