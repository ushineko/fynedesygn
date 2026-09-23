package shell

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/settings"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// Default window geometry: wide enough for a table with five columns beside
// the navigation, tall enough for a heading, a table and an action strip.
const (
	DefaultWidth  float32 = 1180
	DefaultHeight float32 = 760
	// NavOffset is the HSplit position of the section list.
	NavOffset = 0.16
)

// Options describe a program to the shell.
type Options struct {
	// AppID names the preference store and, on Wayland, the window's app_id,
	// which compositors match to a desktop entry of the same basename.
	AppID string
	// Name is the header label and the window title's first word.
	Name string
	// Version is appended to the window title when set.
	Version string
	// Icon is the window and application icon; nil leaves Fyne's.
	Icon fyne.Resource
	// SettingsPath is the program's settings file. Empty means the default:
	// the user's configuration directory, a directory named for AppID, and
	// settings.json in it. A program that already owns a configuration
	// directory points this at it and keeps one directory rather than two.
	//
	// The extension chooses the format. ".yaml" needs the YAML codec imported
	// for its effect; see the settings package.
	SettingsPath string
	// Sections in navigation order. At least one.
	Sections []Section

	// Groups fold runs of sections under headings in the navigation list.
	// Optional, and drawn only in the shape that has a list; see group.go.
	Groups []NavGroup
	// Section is the title to open on; empty or unknown opens the first. A
	// wrong name gives a wrong screenshot, which is obvious, not a dead window.
	Section string
	// Scheme forces a colour scheme for this run without saving it, so a
	// capture does not overwrite whatever the user had chosen.
	Scheme string
	// Size is the initial window size; zero means DefaultWidth x DefaultHeight.
	Size fyne.Size
	// Header returns extra header actions drawn after the Refresh button.
	Header func(s *Shell) []fyne.CanvasObject
	// StatusBar returns the status bar's segments in order. They are laid out
	// sequentially, never in a Border centre: a centre region is sized from
	// what is left over, so a long path pushed two segments into each other.
	StatusBar func(s *Shell) []fyne.CanvasObject
	// OnStart runs after the window exists and before it shows. Program-wide
	// state the status bar shows from every section is loaded here rather
	// than by whichever section happens to be open first.
	OnStart func(s *Shell)
	// OnInvalidate runs at the start of Invalidate: drop loaded flags and start
	// the reloads. The shell rebuilds afterwards when it is on screen.
	OnInvalidate func(s *Shell)
	// OnStop runs when the window closes and before Restart replaces the
	// process: flush a pending save, release a lock. It runs once.
	OnStop func(s *Shell)
	// AlsoWorking reports program work that runs outside Perform (a job with
	// its own step list). Working() includes it.
	AlsoWorking func() bool
	// OnCreate is called with the shell before anything else runs: before the
	// first section is built, the status bar composed or OnStart called. A
	// program that keeps the shell in a field stores it here, so its builders
	// can rely on the field rather than on the argument each is handed.
	OnCreate func(s *Shell)
	// Theme, when set, builds the theme from the appearance wherever the shell
	// would apply Appearance.Theme(): at start and in SetAppearance. A program
	// that keeps a setting the theme needs outside the preference store (a
	// console font in its own configuration file) supplies it here.
	Theme func(a fdtheme.Appearance) fyne.Theme
	// OnTypedKey receives every typed key the shell did not handle itself (it
	// binds F5 to Invalidate). Sections with keyboard navigation use it.
	OnTypedKey func(e *fyne.KeyEvent)
	// NavModes and NavPlacements are the navigation shapes this program
	// allows. Empty means the one it has always had -- icons and labels down
	// the left -- with no control to change it and no shortcut bound.
	//
	// A program that lists more than one shape gets a stock control in the
	// header offering exactly what it listed. See spec 013.
	NavModes      []NavMode
	NavPlacements []NavPlacement
	// Nav and NavPlace are the shape a program starts in before its user has
	// chosen one. A stored choice wins over them.
	Nav      NavMode
	NavPlace NavPlacement
}

// Shell owns the window and everything transient in it.
type Shell struct {
	// App and Window are exported for dialogs, the clipboard and the
	// preference store. Sections write to widgets only on the UI thread.
	App    fyne.App
	Window fyne.Window

	opts       Options
	appearance fdtheme.Appearance
	oneRun     bool // Scheme was forced for this run

	// store is the program's settings file, and settingsErr what went wrong
	// opening it -- reported once there is a window to report in, because
	// newShell runs before there is one.
	store       *settings.Store
	settingsErr error
	stopOnce    sync.Once

	// splitPos is where each named divider was last dragged to, and splits the
	// live one per key. See split.go.
	splitPos map[string]float64
	splits   map[string]*container.Split

	// The navigation's shape, the last non-hidden mode (so Ctrl+B has
	// something to come back to), and the holder the button navigations are
	// redrawn in. See nav.go.
	navMode   NavMode
	navPlace  NavPlacement
	navShown  NavMode
	navHolder *fyne.Container
	navBtn    *widget.Button // the shape control, kept so a test can tap it

	// tips is the layer hover notes are drawn in, over everything else, and
	// floats the one result banners are drawn in, under them.
	tips   *fyne.Container
	floats *fyne.Container

	nav     *widget.List
	content *container.Scroll

	// The navigation list's rows, which are the sections plus a heading per
	// group and minus the members of a closed one, the groups the user has
	// closed, and the flag that restores a selection without swapping the
	// section under it. See group.go.
	rows     []navRow
	closed   map[string]bool
	navQuiet bool
	frame    *fyne.Container // holds the status bar at index frameStatusBar
	current  int

	busyMu    sync.Mutex // guards the busy fields from Busy's goroutine callers
	busyCount int
	busyWhat  string
	busyPop   *widget.PopUp
	busyLabel *widget.Label
	busySeq   int
	// busyCancel is the cancellable operation currently holding the popup, and
	// busyCancelBtn the button it put there. UI thread only.
	busyCancel    *busyJob
	busyCancelBtn *widget.Button

	flashes  *fyne.Container
	flashSeq int
}

const frameStatusBar = 0

// New creates the app and builds the window without showing it. Call
// Window.ShowAndRun, or use Run.
func New(o Options) *Shell {
	// Before the toolkit starts: GLFW reads the cursor theme from the
	// environment at init, and there is no second chance once the window is up.
	fdtheme.ApplyCursorTheme()

	a := app.NewWithID(o.AppID)
	s := newShell(a, o)
	fdtheme.ApplyScale(s.appearance.Scale)
	a.Settings().SetTheme(s.theme())
	if o.Icon != nil {
		a.SetIcon(o.Icon)
	}

	title := o.Name
	if o.Version != "" {
		title += " " + o.Version
	}
	s.Window = a.NewWindow(title)
	if o.Icon != nil {
		s.Window.SetIcon(o.Icon)
	}
	s.buildWindow()
	// Closing the window is a stop like any other: a pending settings write has
	// to reach the disk, and a program with a lock to release gets the same
	// notice it gets before a restart.
	s.Window.SetOnClosed(s.Stop)
	if s.settingsErr != nil {
		s.Report("Reading settings", s.settingsErr)
	}
	if o.OnStart != nil {
		o.OnStart(s)
	}
	return s
}

/*
Stop runs the program's OnStop and writes any pending settings.

Called when the window closes and before Restart replaces the process, and it
runs once however many times it is called: a restart that goes through both
paths must not release a lock twice.
*/
func (s *Shell) Stop() {
	s.stopOnce.Do(func() {
		s.rememberSplits()
		if s.opts.OnStop != nil {
			s.opts.OnStop(s)
		}
		if err := s.store.Flush(); err != nil {
			s.Report("Saving settings", err)
		}
	})
}

// Run is New followed by ShowAndRun; it blocks until the window closes.
func Run(o Options) {
	New(o).Window.ShowAndRun()
}

// Headless builds a shell with no window over an existing app (a test app),
// so sections can be built and Perform runs inline. OnScreen reports false.
// OnStart is not called: a test decides what is loaded.
func Headless(a fyne.App, o Options) *Shell {
	s := newShell(a, o)
	a.Settings().SetTheme(s.theme())
	return s
}

// newShell is the part of construction New and Headless share.
func newShell(a fyne.App, o Options) *Shell {
	s := &Shell{App: a, opts: o, flashes: container.NewVBox()}
	s.openSettings()
	s.loadSplits()
	s.loadNav()
	s.loadGroups()
	s.appearance = fdtheme.LoadAppearanceFrom(s.store, a.Preferences())
	if o.Scheme != "" {
		s.appearance.Scheme = fdtheme.SchemeByName(o.Scheme).Name
		s.oneRun = true
	}
	// The section to open on, or the first: a name that is not there is a
	// program asking for something it no longer has, and the first section
	// is the honest answer to "open somewhere".
	s.current = s.index(o.Section)
	if s.current < 0 {
		s.current = 0
	}
	if o.OnCreate != nil {
		o.OnCreate(s)
	}
	return s
}

/*
openSettings opens the program's settings file.

A store that could not be opened is not a reason to refuse to start: the shell
carries on with one that holds nothing, so the window opens on defaults and says
what happened rather than not opening at all.
*/
func (s *Shell) openSettings() {
	path := s.opts.SettingsPath
	if path == "" {
		got, err := settings.DefaultPath(s.opts.AppID)
		if err != nil {
			s.settingsErr, s.store = err, settings.Memory()
			return
		}
		path = got
	}
	st, err := settings.Open(path)
	s.store, s.settingsErr = st, err
	if st == nil {
		s.store = settings.Memory()
		return
	}
	st.OnError(func(err error) { fyne.Do(func() { s.Report("Saving settings", err) }) })
}

/*
Settings is the program's settings file: one section per key, decoded into
whatever the caller hands it.

	var cfg jobs
	s.Settings().Get("jobs", &cfg)
	s.Settings().Set("jobs", cfg)

Keys beginning settings.Prefix are the library's. Never nil: a store that could
not be opened is an empty one, so a caller needs no branch.
*/
func (s *Shell) Settings() *settings.Store { return s.store }

// theme builds the theme for the current appearance, through the program's
// Theme hook when it has one.
func (s *Shell) theme() fyne.Theme {
	if s.opts.Theme != nil {
		return s.opts.Theme(s.appearance)
	}
	return s.appearance.Theme()
}

// buildWindow assembles the skeleton: header on top, status bar at the
// bottom, the section list and the content scroller in a split between.
func (s *Shell) buildWindow() {
	o := s.opts
	s.content = container.NewScroll(widget.NewLabel(""))

	s.buildRows()
	s.nav = s.newNavList()
	// Result banners and the progress indicator float over the content as
	// popups, so nothing below the header reflows when an operation starts,
	// finishes or reports. The frame holds the status bar alone.
	s.frame = container.NewVBox(s.statusBar())
	s.layout()

	// F5 and Ctrl+R reload, the two bindings people already try.
	s.Window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { s.Invalidate() })
	// Ctrl+B is the binding people already try for a sidebar, and it is bound
	// only where there is a sidebar to hide.
	if s.allowsMode(NavHidden) {
		s.Window.Canvas().AddShortcut(
			&desktop.CustomShortcut{KeyName: fyne.KeyB, Modifier: fyne.KeyModifierControl},
			func(fyne.Shortcut) { s.toggleNav() })
	}
	/*
		Ctrl+1 to Ctrl+9 select a section.

		The third binding people already try, after reload and the sidebar --
		it is what every browser, terminal and editor does with numbered tabs.
		Nine because that is how many keys there are in a row, and a program
		with more sections than that has a bigger problem than its shortcuts.

		Bound by index rather than by title so a program can rename a section
		without renaming a shortcut nobody wrote down.
	*/
	for i := range min(len(o.Sections), 9) {
		s.Window.Canvas().AddShortcut(
			&desktop.CustomShortcut{
				KeyName:  fyne.KeyName(rune('1' + i)),
				Modifier: fyne.KeyModifierControl,
			},
			func(fyne.Shortcut) { s.selectIndex(i) })
	}

	s.Window.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyF5 {
			s.Invalidate()
			return
		}
		if o.OnTypedKey != nil {
			o.OnTypedKey(e)
		}
	})

	size := o.Size
	if size.IsZero() {
		size = fyne.NewSize(DefaultWidth, DefaultHeight)
	}
	s.Window.Resize(size)
	s.Window.SetMaster()
	s.selectIndex(s.current)
	if !s.usesList() {
		s.swap(false) // the list's selection is what swaps in the other shapes
	}
}

/*
layout assembles the window from the navigation's current shape.

Called at build and whenever the shape changes. The content scroller and the
section list survive it: only the region around them is rebuilt, so changing
shape does not rebuild the section the user is looking at.
*/
func (s *Shell) layout() {
	if s.Window == nil {
		return
	}
	// The tip layer goes over everything and is drawn last, so a hover note
	// floats above whatever it covers. It is part of the content rather than an
	// overlay on purpose: Fyne routes pointer events to the top overlay only,
	// so a tip in one would take every click in the window (quirk 26).
	if s.tips == nil {
		s.tips = widgets.NewTipLayer()
	}
	if s.floats == nil {
		s.floats = container.New(flashLayout{}, s.flashes)
	}
	s.Window.SetContent(container.NewStack(
		container.NewBorder(s.header(), s.frame, nil, nil, s.body()),
		s.floats,
		s.tips,
	))
}

// header is the window's title strip: the program name, then Refresh and the
// program's own actions at the trailing edge.
func (s *Shell) header() fyne.CanvasObject {
	actions := []fyne.CanvasObject{}
	if c := s.navControl(); c != nil {
		actions = append(actions, c)
	}
	actions = append(actions,
		widget.NewButtonWithIcon("Refresh", fynetheme.ViewRefreshIcon(), func() { s.Invalidate() }))
	if s.opts.Header != nil {
		actions = append(actions, s.opts.Header(s)...)
	}

	/*
		A Border rather than an HBox and a spacer.

		The two lay the actions out identically — hard against the trailing
		edge — but a Border hands what is left to what it opens with, and
		what it opens with may be the navigation, which is a scroller. A
		scroller in an HBox asks for almost nothing and gets it, so the
		sections would have arrived squeezed into a few pixels at the leading
		edge. In a Border centre it takes the room the actions do not.
	*/
	// The actions at their own height, against the top of the row.
	//
	// A Border stretches what it is given to the height of the row, and the
	// row is two buttons tall whenever the navigation is along the top with a
	// group open -- so Refresh was drawn twice as tall as every other button,
	// which says it is twice the control. The top rather than the middle:
	// these act on the window, and the window's first row is where its
	// controls live however many rows the navigation has grown to.
	bar := container.NewBorder(nil, nil, nil,
		container.NewVBox(container.NewHBox(actions...)), s.headerLead())
	return container.NewVBox(container.NewPadded(bar), widget.NewSeparator())
}

/*
headerLead is what the header opens with: the navigation when it is along the
top, and the program's name otherwise.

A navigation along the top used to be a second strip under the header, which
spent a whole row of the window on a handful of buttons while the row above it
held a program name the title bar was already showing. The shell sets the
window title from Options.Name and Version, so in that shape the label was the
same word twice, one line apart, in exchange for the space the sections could
have been sitting in.

Down the left it stays: there the name sits above the navigation and reads as
what the list belongs to, and there is no row to save.
*/
func (s *Shell) headerLead() fyne.CanvasObject {
	if s.navPlace == NavTop && s.navMode != NavHidden {
		return s.navHolderFor(true)
	}
	return widget.NewLabelWithStyle(s.opts.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// statusBar lays the program's segments out sequentially.
func (s *Shell) statusBar() fyne.CanvasObject {
	var segs []fyne.CanvasObject
	if s.opts.StatusBar != nil {
		segs = s.opts.StatusBar(s)
	}
	segs = append(segs, layout.NewSpacer())
	return container.NewVBox(widget.NewSeparator(), container.NewPadded(container.NewHBox(segs...)))
}

// index resolves a title to a position, case-insensitively; unknown is 0.
func (s *Shell) index(title string) int {
	if title == "" {
		return 0
	}
	for i, sec := range s.opts.Sections {
		if strings.EqualFold(sec.Title(), title) {
			return i
		}
	}
	return -1
}

// Sections are the program's sections in navigation order.
func (s *Shell) Sections() []Section { return s.opts.Sections }

// Current is the section on screen, or the one that would be.
func (s *Shell) Current() Section {
	if s.current < 0 || s.current >= len(s.opts.Sections) {
		return nil
	}
	return s.opts.Sections[s.current]
}

/*
Select moves the navigation to a titled section. Used by buttons that hand the
reader on ("View report" after a job).

**An unknown title is ignored**, which is what this always said and did not do:
it navigated to the first section instead. A program whose sections have been
rearranged -- a section folded into a group, a title reworded -- then jumped to
the front every time it asked for a name that had moved, and nothing said why.
Found in hotaru, where dropping a picture on the window selected "Pictures"
after that section had become part of a "Create" group, and the window went to
the service page instead.
*/
func (s *Shell) Select(title string) {
	i := s.index(title)
	if i < 0 {
		return
	}
	if s.nav == nil {
		s.current = i
		return
	}
	s.selectIndex(i)
}

// Scroller is the content pane's scroller, for components that follow the
// section's scroll (a document pane). Nil when the shell is headless: a
// component with no viewport to be outside of renders everything.
func (s *Shell) Scroller() *container.Scroll { return s.content }

// OnScreen reports whether there is a window to draw into. False in a
// headless test, and in the gap between a load finishing and the process
// exiting. Builders that would start a load for nobody check it.
func (s *Shell) OnScreen() bool { return s.content != nil }

// Appearance is the current look-and-feel choices.
func (s *Shell) Appearance() fdtheme.Appearance { return s.appearance }

// SetAppearance applies choices to the running app (through Options.Theme
// when set) and saves them, except the scheme while a one-run override is in
// force: that run must not overwrite what the user had chosen. A program whose
// theme depends on something outside the appearance calls it again with the
// same appearance when that something changes.
func (s *Shell) SetAppearance(a fdtheme.Appearance) {
	s.appearance = a
	s.App.Settings().SetTheme(s.theme())
	saved := a
	if s.oneRun {
		saved.Scheme = fdtheme.LoadAppearanceFrom(s.store, s.App.Preferences()).Scheme
	}
	saved.SaveTo(s.store)
}

// RedrawStatus repaints the status bar and nothing else. This is the whole
// reason progress lives down there: an indicator inserted above the content
// pushes everything below it down, and the row the pointer is over moves out
// from under it mid-click.
func (s *Shell) RedrawStatus() {
	if s.frame == nil {
		return
	}
	s.frame.Objects[frameStatusBar] = s.statusBar()
	s.frame.Refresh()
}

// Refresh rebuilds the current section in place, keeping its scroll offset, so
// a view picks up what an operation just changed. A no-op without a window:
// building a section for nobody would start the loads that section asks for.
func (s *Shell) Refresh() {
	if s.content == nil {
		return
	}
	s.swap(true)
}

// Rebuild redraws the status bar and the current section.
func (s *Shell) Rebuild() {
	s.RedrawStatus()
	s.Refresh()
}

// Invalidate discards what the program has loaded (through OnInvalidate) and
// rebuilds. Bound to F5, Ctrl+R and the Refresh button. Off screen the hook
// still runs and nothing is drawn, so a headless test sees the flags drop
// without the sections fetching.
func (s *Shell) Invalidate() {
	if s.opts.OnInvalidate != nil {
		s.opts.OnInvalidate(s)
	}
	if !s.OnScreen() {
		return
	}
	s.Rebuild()
}

/*
swap replaces the content pane with a freshly built current section, detaching
the outgoing one first.

keepScroll is also what tells the two reasons for a swap apart. False is
navigation — the list, Select, the section the window opens on — and the
incoming section is told it has been arrived at. True is a rebuild of the
section already on screen, which keeps its scroll offset and arrives at
nothing: a section that refetched on every build would rebuild itself forever,
because the fetch finishing is one of the things that rebuilds it.
*/
func (s *Shell) swap(keepScroll bool) {
	if s.content == nil {
		return
	}
	sec := s.Current()
	if sec == nil {
		return
	}
	for _, other := range s.opts.Sections {
		if d, ok := other.(Detacher); ok {
			d.Detach()
		}
	}
	if !keepScroll {
		if a, ok := sec.(Arriver); ok {
			a.Arrive()
		}
	}
	// The last moment the outgoing section's dividers exist.
	s.rememberSplits()
	offset := s.content.Offset
	s.content.Content = sec.Build(s)
	s.content.Refresh()
	if !keepScroll {
		s.content.ScrollToTop()
		return
	}
	s.content.Offset = offset
	s.content.Refresh()
}
