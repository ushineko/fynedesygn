/*
Command master-detail is a fynedesygn example: a listing with one status per
row, a detail card for the selected row, a reload through Perform with a
simulated delay, and a status-bar segment that shows the count from every
section.

The program's state carries the loaded-flag pattern: "loaded and empty" and
"not loaded" are different states, and the section asks for its data when it
has none and rebuilds itself when the load lands.
*/
package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/table"
	"github.com/ushineko/fynedesygn/widgets"
)

func main() {
	shell.Run(options(newState(), 400*time.Millisecond))
}

// item is one row of the listing.
type item struct {
	Name, Kind string
	Size       int64
	Status     fd.Status
	Verdict    string
}

// state is the program's data with its loaded flag.
type state struct {
	items    []item
	itemsOK  bool
	selected int // -1 when nothing is selected
}

func newState() *state { return &state{selected: -1} }

// fetch is the pretend core call: it sleeps to show the busy popup and
// returns a fixed listing.
func fetch(ctx context.Context, delay time.Duration) ([]item, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fetch cancelled: %w", ctx.Err())
	case <-time.After(delay):
	}
	return []item{
		{"alpha.txt", "text", 1200, fd.StatusGood, "ok"},
		{"beta.bin", "binary", 4 << 20, fd.StatusInfo, "unchecked"},
		{"gamma.cfg", "config", 300, fd.StatusWarn, "deprecated key"},
		{"delta.so", "library", 9 << 20, fd.StatusBad, "failed to load"},
	}, nil
}

func options(st *state, delay time.Duration) shell.Options {
	return shell.Options{
		AppID:    "io.ushineko.fynedesygn.example.masterdetail",
		Name:     "master-detail",
		Sections: []shell.Section{shell.NewSection("Items", fynetheme.ListIcon, func(s *shell.Shell) fyne.CanvasObject { return buildItems(s, st, delay) })},
		StatusBar: func(*shell.Shell) []fyne.CanvasObject {
			count := "loading..."
			if st.itemsOK {
				count = fmt.Sprintf("%d", len(st.items))
			}
			return []fyne.CanvasObject{widgets.Dim("items"), widget.NewLabel(count)}
		},
		OnInvalidate: func(*shell.Shell) { st.itemsOK = false; st.selected = -1 },
	}
}

// load fetches the listing when it is not loaded. The flag is set before the
// goroutine starts so a rebuild mid-load does not start a second fetch.
func load(s *shell.Shell, st *state, delay time.Duration) {
	if st.itemsOK {
		return
	}
	st.itemsOK = true
	s.Perform("Loading the items...", func(ctx context.Context) error {
		items, err := fetch(ctx, delay)
		if err != nil {
			fyne.Do(func() { st.itemsOK = false })
			return err
		}
		fyne.Do(func() {
			st.items = items
			s.Rebuild()
		})
		return nil
	})
}

func buildItems(s *shell.Shell, st *state, delay time.Duration) fyne.CanvasObject {
	load(s, st, delay)

	t := table.New()
	t.Header("Name", "Kind", "Size", "Verdict")
	for _, it := range st.items {
		t.Row(it.Status, it.Name, it.Kind, widgets.HumanSize(it.Size), it.Verdict)
	}
	tbl := t.Widget()
	tbl.OnSelected = func(id widget.TableCellID) {
		st.selected = id.Row
		s.Refresh()
	}

	detail := widgets.Card("Detail", widgets.Dim("Select a row."))
	remove := widget.NewButton("Remove...", func() {})
	remove.Importance = widget.DangerImportance
	remove.Disable()
	if st.selected >= 0 && st.selected < len(st.items) {
		it := st.items[st.selected]
		remove.Enable()
		detail = widgets.Card(it.Name,
			widgets.FactRow("Kind", it.Kind, fd.StatusInfo),
			widgets.PlainRow("Size", widgets.HumanSize(it.Size)),
			widgets.FactRow("Verdict", it.Verdict, it.Status),
			widgets.RowWithAction(widgets.PlainRow("Actions", "Row actions start disabled and are enabled by the selection."), remove),
		)
	}
	s.Gate(remove)

	return container.NewBorder(
		widgets.Heading("Items", "A read-only listing with one status per row. Refresh (F5) reloads through Perform; the count in the status bar comes from the same state."),
		detail, nil, nil,
		widgets.FixedHeight(tbl, 260),
	)
}
