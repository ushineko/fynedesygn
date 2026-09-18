package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/table"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// section is one page of the gallery.
type section struct {
	title string
	build func(*gallery) fyne.CanvasObject
}

// sections lists the pages in nav order. It is a function so the titles can
// be read before a Fyne app exists (for --help) without touching any theme
// resource, which would log a warning about a missing app.
func sections() []section {
	return []section{
		{"Appearance", buildAppearance},
		{"Widgets", buildWidgets},
		{"Table", buildTable},
		{"Fonts", buildFonts},
	}
}

// sectionNames are the titles, for flag help and the --section check.
func sectionNames() []string {
	all := sections()
	names := make([]string, 0, len(all))
	for _, s := range all {
		names = append(names, s.title)
	}
	return names
}

// sectionIndex finds a section by title, case-insensitively; -1 when absent.
func sectionIndex(name string) int {
	for i, s := range sections() {
		if strings.EqualFold(s.title, name) {
			return i
		}
	}
	return -1
}

// --- Appearance --------------------------------------------------------------

func buildAppearance(g *gallery) fyne.CanvasObject {
	a := &g.appearance

	scheme := widget.NewSelect(fdtheme.SchemeNames(), func(name string) {
		a.Scheme = name
		g.applyAppearance()
	})
	scheme.SetSelected(a.Scheme)

	font := widget.NewSelect(fdtheme.FontNames(), func(name string) {
		a.Font = name
		g.applyAppearance()
	})
	font.SetSelected(a.Font)

	mono := widget.NewSelect(fdtheme.FontNames(), func(name string) {
		a.Mono = name
		g.applyAppearance()
	})
	mono.SetSelected(a.Mono)

	sizes := make([]string, 0, len(fdtheme.TextSizes()))
	for _, s := range fdtheme.TextSizes() {
		sizes = append(sizes, fmt.Sprintf("%g", s))
	}
	size := widget.NewSelect(sizes, func(v string) {
		for _, s := range fdtheme.TextSizes() {
			if fmt.Sprintf("%g", s) == v {
				a.TextSize = s
				g.applyAppearance()
				return
			}
		}
	})
	size.SetSelected(fmt.Sprintf("%g", a.TextSize))

	scales := make([]string, 0, len(fdtheme.ScaleChoices()))
	for _, s := range fdtheme.ScaleChoices() {
		scales = append(scales, fdtheme.ScaleLabel(s))
	}
	scale := widget.NewSelect(scales, func(v string) {
		a.Scale = fdtheme.ScaleValue(v)
		g.applyAppearance()
	})
	scale.SetSelected(fdtheme.ScaleLabel(a.Scale))

	reset := widget.NewButton("Reset to defaults", func() {
		*a = fdtheme.DefaultAppearance()
		scheme.SetSelected(a.Scheme)
		font.SetSelected(a.Font)
		mono.SetSelected(a.Mono)
		size.SetSelected(fmt.Sprintf("%g", a.TextSize))
		scale.SetSelected(fdtheme.ScaleLabel(a.Scale))
		g.applyAppearance()
	})

	form := widget.NewForm(
		widget.NewFormItem("Colour scheme", scheme),
		widget.NewFormItem("Font", font),
		widget.NewFormItem("Monospace font", mono),
		widget.NewFormItem("Text size", size),
		widget.NewFormItem("Interface scale", scale),
	)

	p := fdtheme.SchemeByName(a.Scheme)
	swatches := container.NewGridWithColumns(6,
		swatch("Window", p.WindowBG, p.WindowFG),
		swatch("View", p.ViewBG, p.ViewFG),
		swatch("Button", p.ButtonBG, p.ButtonFG),
		swatch("Selection", p.SelectionBG, p.SelectionFG),
		swatch("Header", p.ViewAltBG, p.ViewFG),
		swatch("Tooltip", p.TooltipBG, p.WindowFG),
	)

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Appearance",
			"Every setting about how a window built on fynedesygn looks. Fyne draws its own widgets, "+
				"so these choices are the whole of what makes it sit well next to the rest of the desktop."),
		form,
		container.NewHBox(reset),
		widgets.Dim("The scheme, fonts, text size and scale are kept in Fyne's preference store under the "+
			"appearance.* keys and nowhere else. The scale applies when the window next opens."),
		widget.NewSeparator(),
		fdtheme.Sample("2026-09-17 16:20:01 INFO  rendered 44 blocks in 3.1ms"),
		widget.NewSeparator(),
		widgets.Card("Scheme roles", swatches),
		widgets.Dim(fmt.Sprintf("%s: dark=%v, corner radius %g, padding %g.", p.Name, p.Dark, p.Radius, p.Padding)),
	))
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

// --- Widgets -----------------------------------------------------------------

func buildWidgets(_ *gallery) fyne.CanvasObject {
	actionRow, actionBtn := widgets.Action("Action", "A titled block with its consequences stated, and one button.", "Do it", false, func() {})
	dangerRow, _ := widgets.Action("Action, danger", "The same shape for something irreversible. The button is red; the dialog it opens says what is not touched.", "Remove...", true, func() {})
	open := widget.NewButton("Open", func() {})
	_ = actionBtn

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

func buildTable(_ *gallery) fyne.CanvasObject {
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

func buildFonts(_ *gallery) fyne.CanvasObject {
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
			widgets.Dim(fmt.Sprintf("%d families offered. Variable fonts are skipped; families without a regular face are not offered.", len(names)-1)),
		),
		nil, nil, nil,
		t.Widget(),
	)
}
