package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/docs"
	"github.com/ushineko/fynedesygn/markdown"
	"github.com/ushineko/fynedesygn/mermaid"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/table"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// sections lists the pages in navigation order, with a fresh job demo. Titles
// are readable before a Fyne app exists; icons are deferred by the shell.
func sections() []shell.Section { return sectionsWith(newJobDemo()) }

func sectionsWith(demo *jobDemo) []shell.Section {
	readme := &aboutDoc{}
	logs := newLogDemo()
	form := newFormDemo()
	return []shell.Section{
		shell.AppearanceSection("2026-09-17 16:20:01 INFO  rendered 44 blocks in 3.1ms"),
		shell.NewSection("Widgets", fynetheme.ListIcon, buildWidgets),
		shell.NewSection("Table", fynetheme.StorageIcon, buildTable),
		shell.NewSection("Fonts", fynetheme.DocumentIcon, buildFonts),
		shell.NewSection("Shell", fynetheme.MediaPlayIcon, demo.build).OnArrive(demo.arrive),
		shell.NewSection("Dialogs", fynetheme.QuestionIcon, buildDialogs),
		shell.NewSection("Log", fynetheme.ListIcon, logs.build).OnDetach(logs.detach),
		shell.NewSection("Forms", fynetheme.SettingsIcon, form.build),
		markdown.Section("Documents", fynetheme.DocumentCreateIcon, mustDoc("markdown.md"), docOptions(), func(*shell.Shell) fyne.CanvasObject {
			return widgets.Heading("Documents",
				"markdown.Pane showing docs/markdown.md from the embedded docs package: per-block rendering near "+
					"the viewport, code panels that wrap, a pipe table drawn as a grid, and a mermaid diagram "+
					"rendered at development time and picked for the active scheme.")
		}),
		shell.AboutSection(shell.About{
			Name:    "fynedesygn gallery",
			Version: version,
			Blurb: "The reference program for the fynedesygn design system: every component in every " +
				"colour scheme, in one window, so a change is judged here before it is judged in a program.",
			URL: "https://github.com/ushineko/fynedesygn",
			Notes: []shell.Note{
				{Title: "Appearance", Detail: "The standard picker every program gets from shell.AppearanceSection."},
				{Title: "The navigation's shape", Detail: "The button beside Refresh: icons and " +
					"labels, icons alone or nothing, down the left or along the top. Ctrl+B hides " +
					"it and brings it back. A program declares which shapes it allows, and one " +
					"that declares nothing keeps the window it has."},
				{Title: "Widgets, Table, Fonts", Detail: "Each component drawn with its Go name as the caption."},
				{Title: "Shell", Detail: "Perform, cancellation, banners and gating, driven by a fake job."},
			},
			Facts: []shell.Fact{
				{Label: "Licence", Value: "MIT"},
				{Label: "Fyne", Value: "v2.8.1"},
				{Label: "Preferences", Value: "Fyne store for " + appID},
			},
			Extra: readme.extra,
		}).OnDetach(readme.detach),
	}
}

// docOptions point a document at the embedded docs and their diagrams.
func docOptions() markdown.Options {
	return markdown.Options{FS: docs.FS, Diagrams: mermaid.NewSet(docs.FS, "diagrams")}
}

// mustDoc reads one embedded document; the names are fixed at build time.
func mustDoc(name string) string {
	b, err := docs.FS.ReadFile(name)
	if err != nil {
		return "# " + name + "\n\nNot embedded: " + err.Error()
	}
	return string(b)
}

// aboutDoc is the About section's Extra: the module README following the
// shell's scroller, released when the section is replaced.
type aboutDoc struct{ pane *markdown.Pane }

func (d *aboutDoc) extra(s *shell.Shell) fyne.CanvasObject {
	d.detach()
	d.pane = markdown.New(fd.README(), docOptions())
	d.pane.Follow(s.Scroller())
	return container.NewVBox(
		widget.NewLabelWithStyle("README", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		d.pane,
	)
}

func (d *aboutDoc) detach() {
	if d.pane != nil {
		d.pane.Detach()
		d.pane = nil
	}
}

// --- Widgets -----------------------------------------------------------------

func buildWidgets(_ *shell.Shell) fyne.CanvasObject {
	actionRow, _ := widgets.Action("Action", "A titled block with its consequences stated, and one button.", "Do it", false, func() {})
	dangerRow, _ := widgets.Action("Action, danger", "The same shape for something irreversible. The button is red; the dialog it opens says what is not touched.", "Remove...", true, func() {})
	open := widget.NewButton("Open", func() {})

	demo := func(name string, o fyne.CanvasObject) fyne.CanvasObject {
		return container.NewVBox(widgets.Dim("widgets."+name), o, widget.NewSeparator())
	}

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Widgets", "The small shared vocabulary. Reach for one of these before arranging labels by hand; a shape that appears in a second section belongs here."),
		demo("Heading", widgets.Heading("A heading", "With the blurb that says what the section is for, in one sentence.")),
		demo("Card", widgets.Card("A card",
			widgets.FactRow("Status", "running", fd.StatusGood),
			widgets.FactRow("Last run", widgets.HumanAgoAt(time.Now().Add(-90*time.Minute), time.Now()), fd.StatusInfo),
			widgets.FactRow("Warnings", "2", fd.StatusWarn),
			widgets.FactRow("Failures", "1", fd.StatusBad),
			widgets.PlainRow("Size", widgets.HumanSize(1536*1024)),
			widgets.RowWithAction(widgets.PlainRow("Path", "~/.local/share/example"), open),
			widgets.Note("A note under a row, for the things a value cannot say on its own.", fd.StatusInfo),
			widgets.Note("A warning note.", fd.StatusWarn),
		)),
		demo("StatusText / Marker", container.NewHBox(
			widgets.Marker(fd.StatusInfo), widgets.StatusText("info", fd.StatusInfo), widgets.Sep(),
			widgets.Marker(fd.StatusGood), widgets.StatusText("good", fd.StatusGood), widgets.Sep(),
			widgets.Marker(fd.StatusWarn), widgets.StatusText("warn", fd.StatusWarn), widgets.Sep(),
			widgets.Marker(fd.StatusBad), widgets.StatusText("bad", fd.StatusBad),
		)),
		demo("Action", container.NewVBox(actionRow, dangerRow)),
		demo("AboutNote", widgets.AboutNote("What it does", "A bold title over a dim, wrapped paragraph. The About section is a column of these.")),
		demo("Wrapped / Dim", container.NewVBox(
			widgets.Wrapped("A paragraph that reflows rather than running off the edge. Used for the sentences in dialogs, which are the ones that state consequences, and for any prose longer than a label."),
			widgets.Dim("Secondary text at low importance."),
		)),
		demo("FixedHeight", widgets.FixedHeight(widget.NewLabel("This label sits in a 60 px tall slot whatever its own height."), 60)),
		demo("HumanSize / HumanAgo / OrNone", widgets.Wrapped(strings.Join([]string{
			widgets.HumanSize(512), widgets.HumanSize(3 << 20), widgets.HumanSize(7 << 30),
			widgets.HumanAgoAt(time.Time{}, time.Now()), widgets.HumanAgoAt(time.Now().Add(-3*24*time.Hour), time.Now()),
			widgets.OrNone("", "(none)"),
		}, "   "))),
	))
}

// --- Table -------------------------------------------------------------------

func buildTable(_ *shell.Shell) fyne.CanvasObject {
	plain := table.New()
	plain.Header("Name", "Kind", "Size", "Verdict")
	plain.Row(fd.StatusGood, "alpha.txt", "text", widgets.HumanSize(1200), "ok")
	plain.Row(fd.StatusInfo, "beta.bin", "binary", widgets.HumanSize(4<<20), "unchecked")
	plain.Row(fd.StatusWarn, "gamma.cfg", "config", widgets.HumanSize(300), "deprecated key")
	plain.Row(fd.StatusBad, "delta.so", "library", widgets.HumanSize(9<<20), "failed to load")
	for i := range 20 {
		plain.Row(fd.StatusInfo, fmt.Sprintf("row-%02d", i), "filler", widgets.HumanSize(int64(i)*1000), "scrolls")
	}

	thumbs := table.New()
	thumbs.Header("", "Colour", "Note")
	tints := []color.NRGBA{{R: 220, G: 70, B: 80, A: 255}, {R: 40, G: 170, B: 100, A: 255}, {R: 60, G: 140, B: 230, A: 255}, {R: 240, G: 160, B: 30, A: 255}}
	images := make([][]byte, 0, len(tints))
	for i, c := range tints {
		thumbs.Row(fd.StatusInfo, "", fmt.Sprintf("tint %d", i+1), "a 64 px image cell decoded once and cached")
		images = append(images, squarePNG(c))
	}
	thumbs.SetThumbnails(0, images)

	return container.NewVBox(
		widgets.Heading("Table", "A read-only detail table: strings in, a widget.Table out, one status per row colouring every cell. Measured column widths; fixed height so the section stays put."),
		widgets.Dim("table.Detail"),
		widgets.FixedHeight(plain.Widget(), 300),
		widget.NewSeparator(),
		widgets.Dim("table.Detail with SetThumbnails"),
		widgets.FixedHeight(thumbs.Widget(), 320),
		widget.NewSeparator(),
		widgets.Dim("table.SortHeader"),
		container.NewHBox(
			table.SortHeader("Name", table.Ascending, func() {}),
			table.SortHeader("Size", table.Unsorted, func() {}),
			table.SortHeader("Modified", table.Descending, func() {}),
		),
	)
}

// squarePNG encodes a 64x64 square of one colour.
func squarePNG(c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// --- Fonts -------------------------------------------------------------------

func buildFonts(_ *shell.Shell) fyne.CanvasObject {
	names := fdtheme.FontNames()
	t := table.New()
	t.Header("Family", "Regular", "Bold", "Italic", "Bold italic")
	for _, n := range names {
		if n == fdtheme.DefaultFontName {
			t.Row(fd.StatusInfo, n, "bundled", "bundled", "bundled", "bundled")
			continue
		}
		f := fdtheme.LoadFont(n)
		if f == nil {
			t.Row(fd.StatusWarn, n, "unreadable", "", "", "")
			continue
		}
		reg := f.Face(fyne.TextStyle{}).Name()
		face := func(s fyne.TextStyle) string {
			r := f.Face(s)
			if r == nil || r.Name() == reg {
				return "falls back to regular"
			}
			return r.Name()
		}
		t.Row(fd.StatusGood, n, reg, face(fyne.TextStyle{Bold: true}), face(fyne.TextStyle{Italic: true}), face(fyne.TextStyle{Bold: true, Italic: true}))
	}
	return container.NewBorder(
		container.NewVBox(
			widgets.Heading("Fonts", "What the scanner found in this machine's font directories. Fyne does not consult fontconfig, so this is what is on disk, not what the desktop is configured to use."),
			widgets.DimWrapped(fmt.Sprintf("%d families offered. Variable fonts are skipped; families without a regular face are not offered.", len(names)-1)),
		),
		nil, nil, nil,
		t.Widget(),
	)
}

// --- Shell -------------------------------------------------------------------

/*
settingsCard demonstrates the program's own settings file.

The counter is the point: it is written to the file a second after it changes
and read back when the gallery next starts, without the gallery holding a
document of its own.
*/
func (d *jobDemo) settingsCard(s *shell.Shell) fyne.CanvasObject {
	type visits struct {
		Count int    `json:"count"`
		Last  string `json:"last"`
	}
	var v visits
	s.Settings().Get("gallery.visits", &v)

	say := func(v visits) string {
		return fmt.Sprintf("%d visit(s), last %s", v.Count, widgets.OrNone(v.Last, "never"))
	}
	count := widget.NewLabel(say(v))
	note := func() {
		v.Count++
		v.Last = time.Now().Format("15:04:05")
		if err := s.Settings().Set("gallery.visits", v); err != nil {
			s.Report("Saving", err)
			return
		}
		count.SetText(say(v))
	}

	return container.NewVBox(
		widgets.Wrapped("One file per program, one section per key, decoded into the caller's own type."),
		container.NewHBox(widget.NewButton("Count this visit", note), count),
		widgets.DimWrapped("Written to "+s.Settings().Path()+" a second after the last change, "+
			"and whenever the window closes. Sections beginning \"fynedesygn.\" are the "+
			"library's -- the appearance picker's choices are one of them. A section this "+
			"build does not know is kept rather than dropped, so an older binary cannot "+
			"quietly delete a setting a newer one wrote."),
		widgets.DimWrapped("The extension chooses the format: settings.yaml is YAML once "+
			"settings/yamlcodec is imported for its effect."),
	)
}

// jobDemo is the fake job the Shell section runs: it drives Perform, the
// cancellable form, the banners and the status bar, and shows what gating does
// to buttons while it runs.
type jobDemo struct {
	runs      int
	lastState string
	// arrivals and builds count the two reasons a section is drawn, so the
	// card can show that they are not the same number.
	arrivals, builds int
}

func newJobDemo() *jobDemo { return &jobDemo{lastState: "idle"} }

// arrive counts a navigation to this section. A section that refetches does it
// here rather than in build, which the shell calls again for every rebuild.
func (d *jobDemo) arrive() { d.arrivals++ }

// statusBar is the segment the shell draws from every section.
func (d *jobDemo) statusBar(_ *shell.Shell) []fyne.CanvasObject {
	st := fd.StatusInfo
	switch d.lastState {
	case "finished":
		st = fd.StatusGood
	case "cancelled":
		st = fd.StatusWarn
	case "failed":
		st = fd.StatusBad
	}
	return []fyne.CanvasObject{
		widgets.Dim("job"), widgets.StatusText(d.lastState, st), widgets.Sep(),
		widgets.Dim("runs"), widget.NewLabel(fmt.Sprintf("%d", d.runs)),
	}
}

// swatch draws one palette role as a filled square with its name on it in
// the role's foreground colour, so a scheme can be read at a glance.
func swatch(name string, bg, fg color.Color) fyne.CanvasObject {
	rect := canvas.NewRectangle(bg)
	rect.SetMinSize(fyne.NewSize(120, 48))
	rect.CornerRadius = 4
	text := canvas.NewText(name, fg)
	text.Alignment = fyne.TextAlignCenter
	return container.NewStack(rect, container.NewCenter(text))
}

func (d *jobDemo) build(s *shell.Shell) fyne.CanvasObject {
	d.builds++
	job := func(ctx context.Context, seconds int, fail bool) error {
		for i := 0; i < seconds*10; i++ {
			select {
			case <-ctx.Done():
				fyne.Do(func() { d.lastState = "cancelled"; s.RedrawStatus() })
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		if fail {
			fyne.Do(func() { d.lastState = "failed"; s.RedrawStatus() })
			return fmt.Errorf("the job was asked to fail")
		}
		fyne.Do(func() {
			d.runs++
			d.lastState = "finished"
			s.OK("The job finished.")
		})
		return nil
	}
	start := func(seconds int, cancellable, fail bool) func() {
		return func() {
			d.lastState = "running"
			s.RedrawStatus()
			fn := func(ctx context.Context) error { return job(ctx, seconds, fail) }
			if cancellable {
				s.PerformCancellable("Running the demo job...", fn)
			} else {
				s.Perform("Running the demo job...", fn)
			}
		}
	}

	quick := widget.NewButton("Run (3 s)", start(3, false, false))
	long := widget.NewButton("Run cancellable (30 s)", start(30, true, false))
	failing := widget.NewButton("Run and fail (2 s)", start(2, false, true))
	failing.Importance = widget.DangerImportance

	// The third shape: a job that holds the indicator itself rather than
	// running through Perform, because it pumps a log around the call and
	// reports its own outcome. It hands its cancel to BusyCancellable, so the
	// Cancel is on the popup where it can actually be clicked.
	held := widget.NewButton("Run holding Busy (30 s)", func() {
		if s.Working() {
			s.Flash("Something is already running.", fd.StatusWarn)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		d.lastState = "running"
		s.RedrawStatus()
		go func() {
			defer cancel()
			done := s.BusyCancellable("Running the held job...", cancel)
			defer done()
			_ = job(ctx, 30, false)
		}()
	})
	s.Gate(quick, long, failing, held)

	banners := container.NewHBox(
		widget.NewButton("Info", func() { s.Flash("An informational banner. It fades after six seconds.", fd.StatusInfo) }),
		widget.NewButton("Good", func() { s.Flash("The operation succeeded.", fd.StatusGood) }),
		widget.NewButton("Warn", func() { s.Flash("Something deserves attention. Warnings stay twelve seconds.", fd.StatusWarn) }),
		widget.NewButton("Bad", func() { s.Flash("A failure. It stays until dismissed.", fd.StatusBad) }),
	)

	p := s.Appearance().Theme().Palette()
	swatches := container.NewGridWithColumns(6,
		swatch("Window", p.WindowBG, p.WindowFG),
		swatch("View", p.ViewBG, p.ViewFG),
		swatch("Button", p.ButtonBG, p.ButtonFG),
		swatch("Selection", p.SelectionBG, p.SelectionFG),
		swatch("Header", p.ViewAltBG, p.ViewFG),
		swatch("Tooltip", p.TooltipBG, p.WindowFG),
	)

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Shell", "The window skeleton and its runner. Every core call goes through Perform, which puts the busy popup up after 300 ms, refuses a second call, and reports failures as banners that stay."),
		widgets.Card("shell.Perform / PerformCancellable / BusyCancellable",
			widgets.Wrapped("The buttons below are gated: they disable while the job runs and come back when it stops, because the section is rebuilt from state at both transitions."),
			container.NewHBox(quick, long, failing, held),
			widgets.DimWrapped("The busy popup is modal, so a Cancel left enabled in a toolbar behind it cannot be clicked. A cancellable job puts its Cancel on the popup instead: through PerformCancellable when the runner owns the job, through BusyCancellable when the job holds the indicator itself."),
			widgets.DimWrapped(fmt.Sprintf("State: %s. Runs: %d. The status bar at the bottom shows the same from every section.", d.lastState, d.runs)),
		),
		widgets.Card("shell.Arriver",
			widgets.DimWrapped(fmt.Sprintf("Arrived at %d time(s); built %d time(s). Navigate away and back to raise the first; run a job, or press F5, to raise only the second. A section that refetches on arrival hooks OnArrive, because refetching in the builder would rebuild itself forever: the fetch finishing is one of the things that rebuilds it.", d.arrivals, d.builds)),
		),
		widgets.Card("widgets.WithTip",
			widgets.Wrapped("Rest the pointer on either of these for half a second."),
			container.NewHBox(
				widgets.WithTip(widget.NewButton("Hover me", func() {}),
					"The note a control cannot fit in its label. This one also still works: "+
						"the tip takes the hover and the tap walks past it."),
				widgets.WithTip(widget.NewCheck("And me", func(bool) {}),
					"Wrapping a control does not change it. A check still checks."),
			),
			widgets.DimWrapped("Fyne has no tooltip. This stacks a transparent catcher over the "+
				"control, which wins the hover because Fyne gives a pointer event to the last "+
				"match in the tree walk."),
		),
		widgets.Card("shell.Settings", d.settingsCard(s)),
		widgets.Card("shell.Flash", widgets.Wrapped("One banner at a time, floated over the content, never inserted into it."), banners),
		widgets.Card("Scheme roles",
			swatches,
			widgets.DimWrapped(fmt.Sprintf("%s: dark=%v, corner radius %g, padding %g.", p.Name, p.Dark, p.Radius, p.Padding)),
		),
	))
}
