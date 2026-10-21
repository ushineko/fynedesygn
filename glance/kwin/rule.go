package kwin

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Rule is a KWin window rule for a glance window: the three things the
// compositor grants that Fyne cannot ask for itself.
//
// AppID is how KWin finds the window. On Wayland it matches the `app_id`,
// which Fyne sets from fyne.App.UniqueID, so the app's unique ID, its desktop
// file basename and this field are the same string. On X11 it is the WM_CLASS.
type Rule struct {
	// AppID is the window class to match, exactly.
	AppID string

	// Title, when set, is the window title to match as well, exactly. A
	// program whose main window shares the app ID gives its glance window a
	// title of its own and matches on both, or the rule strips the main
	// window's titlebar too. A rule with a title is a different rule from
	// one without: Install, Lookup and Remove key on the pair.
	Title string

	// Description is what System Settings shows in its rule list. A rule the
	// user cannot identify is a rule they cannot remove.
	Description string

	// AlwaysOnTop keeps the window above others, forced. It duplicates
	// desktop.Window.RequestAlwaysOnTop and is worth setting anyway: the rule
	// survives a compositor restart, and the toolkit's request is one the
	// window manager is allowed to decline.
	AlwaysOnTop bool

	// NoBorder removes the titlebar, forced.
	//
	// This is not a duplicate of what a splash window already is. Fyne asks
	// GLFW for an undecorated window and KWin decorates it anyway, so without
	// this rule a glance window on Plasma has a titlebar and window buttons.
	NoBorder bool

	// Opacity is the window's opacity as a percentage, 1..100. Zero leaves
	// the user's own setting alone.
	//
	// This is the only way a glance window is translucent: Fyne's desktop
	// backend never requests a transparent framebuffer, so nothing in the
	// process can draw through the window (quirk 32). It is applied
	// initially rather than forced, so the user can still override it from
	// the window menu.
	Opacity int

	// SkipTaskbar, SkipSwitcher and SkipPager keep the window out of the
	// taskbar, the window switcher and the pager, forced. An indicator that
	// appears in Alt+Tab is a defect.
	SkipTaskbar  bool
	SkipSwitcher bool
	SkipPager    bool

	// NoFocus stops the window from taking the focus, forced. Measured on
	// Plasma 6: a splash window takes the focus when it is shown (quirk 36),
	// and an indicator that took the focus from a game on every key press
	// would be worse than no indicator.
	NoFocus bool
}

// The rule-type vocabulary KWin uses for the "…rule" companion of each
// property. Only the one this package sets is named.
//
// Opacity was written as "4" until it was measured. KWin's SetRule enum is
// DontAffect 1, Force 2, Apply 3, Remember 4, so "4" asks KWin to *remember*
// whatever opacity the window already has rather than to set one, and the
// rule had no effect at any percentage. "3" (the GUI's "apply initially") was
// measured too and does not apply it either. Only Force does: a window under
// a rule at 100, 95, 80 and 50 % tracked the blend against the desktop behind
// it exactly at "2" and was opaque at every percentage at "3" and "4".
const ruleForce = "2"

// decorationNone is KWin 6.8's decorationpolicy for a window with no border.
const decorationNone = "none"

// generalSection holds the list of rules.
const generalSection = "General"

// The two places the list of rules has lived. KWin 6.8 reads `Order`; `rules`
// and `count` are what it read before, and what KWin 6.7 still reads.
//
// **A legacy list in a 6.8 file deletes rules.** KWin 6.8 takes a non-empty
// `rules` as an old file to migrate and deletes every rule group it does not
// list (RuleBookSettings::usrRead, v6.8.0). This package once wrote `rules=1`
// into a file that listed its rules in `Order`, and on the next reconfigure
// KWin deleted every rule but that one (spec 074).
const (
	orderKey  = "Order"
	legacyKey = "rules"
	countKey  = "count"
)

// staleKeys are keys an earlier version of a rule may have written that this
// one must clear.
//
// Position is deliberately not a rule. Measured on Plasma 6 on Wayland, a
// forced `position` with coordinates does place the window there; this
// package still does not write one, because a position in a rule is fixed to
// one screen, and a window that should appear on the screen the pointer is on
// is moved by the compositor's scripting instead (see CentreOnPointerScript).
// A rule that once set one has it removed rather than left behind.
var staleKeys = []string{
	"position", "positionrule",
	"size", "sizerule",
	"screen", "screenrule",
}

// Path is the rules file this package reads and writes, honouring
// XDG_CONFIG_HOME. It is exported so a program can name the file it is about
// to change when it asks the user.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "kwinrulesrc"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating the home directory: %w", err)
	}
	return filepath.Join(home, ".config", "kwinrulesrc"), nil
}

// Install writes r into the user's KWin rules, replacing the rule for the same
// AppID and Title if one is already there and appending a new one otherwise.
// Every other rule in the file is left as it was found.
//
// **This is an explicit step a program offers, never one it takes on its
// own.** A glance window that silently edited kwinrulesrc on first run would
// be a program that changed the desktop without being asked.
//
// KWin does not notice the change until it is told. See Reconfigure.
func Install(r Rule) error {
	if strings.TrimSpace(r.AppID) == "" {
		return fmt.Errorf("installing a KWin rule: no app ID to match on")
	}
	if r.Opacity < 0 || r.Opacity > 100 {
		return fmt.Errorf("installing a KWin rule: opacity %d is not a percentage", r.Opacity)
	}
	// A value is written as the rest of its line. One carrying a newline would
	// continue into keys of its own — or a whole section — inside a file that
	// holds every window rule the user has. The two fields a caller supplies
	// as free text are checked here rather than escaped, because neither has
	// any business containing a control character.
	for _, f := range []struct{ name, value string }{
		{"app ID", r.AppID},
		{"title", r.Title},
		{"description", r.Description},
	} {
		if strings.ContainsAny(f.value, "\r\n") {
			return fmt.Errorf("installing a KWin rule: the %s contains a line break", f.name)
		}
	}

	path, err := Path()
	if err != nil {
		return err
	}
	file, err := load(path)
	if err != nil {
		return err
	}

	list := readList(file)
	sec := file.section(sectionFor(file, list, r.AppID, r.Title))
	if sec == nil {
		sec = appendRule(file, &list)
	}
	write(sec, r, list.order)
	list.store()

	return save(path, file)
}

// Remove deletes the rule matching appID and no title, and says whether there
// was one. The file is left untouched when there was not.
func Remove(appID string) (bool, error) { return RemoveTitled(appID, "") }

// RemoveTitled deletes the rule matching appID and title, and says whether
// there was one.
func RemoveTitled(appID, title string) (bool, error) {
	path, err := Path()
	if err != nil {
		return false, err
	}
	file, err := load(path)
	if err != nil {
		return false, err
	}

	list := readList(file)
	name := sectionFor(file, list, appID, title)
	if name == "" {
		return false, nil
	}
	file.remove(name)

	kept := list.names[:0]
	for _, n := range list.names {
		if n != name {
			kept = append(kept, n)
		}
	}
	list.names = kept
	list.store()

	return true, save(path, file)
}

// Lookup returns the rule installed for appID, and whether there is one. It is
// what a program's settings section asks before offering to install.
func Lookup(appID string) (Rule, bool, error) { return LookupTitled(appID, "") }

// LookupTitled reads the rule for appID and title, and says whether there is
// one.
func LookupTitled(appID, title string) (Rule, bool, error) {
	path, err := Path()
	if err != nil {
		return Rule{}, false, err
	}
	file, err := load(path)
	if err != nil {
		return Rule{}, false, err
	}
	name := sectionFor(file, readList(file), appID, title)
	if name == "" {
		return Rule{}, false, nil
	}
	sec := file.section(name)

	r := Rule{AppID: appID, Title: title}
	r.Description, _ = sec.get("Description")
	flag := func(key string) bool {
		v, _ := sec.get(key)
		return v == "true"
	}
	r.AlwaysOnTop = flag("above")
	policy, _ := sec.get("decorationpolicy")
	r.NoBorder = flag("noborder") || policy == decorationNone
	r.SkipTaskbar = flag("skiptaskbar")
	r.SkipSwitcher = flag("skipswitcher")
	r.SkipPager = flag("skippager")
	accept, _ := sec.get("acceptfocus")
	r.NoFocus = accept == "false"
	if v, ok := sec.get("opacityactive"); ok {
		r.Opacity, _ = strconv.Atoi(v)
	}
	return r, true, nil
}

// load reads the rules file. A file that is not there is an empty one: a
// desktop with no custom rules yet is the normal case, not an error.
func load(path string) (*iniFile, error) {
	f, err := os.Open(path) // nolint:gosec // the path is the user's own config
	if os.IsNotExist(err) {
		return &iniFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	parsed, err := parseINI(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return parsed, nil
}

// save writes the file back through a temporary file in the same directory and
// a rename, so an interrupted write cannot leave the user with half a rules
// file and no window rules at all.
func save(path string, file *iniFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".kwinrulesrc-*")
	if err != nil {
		return fmt.Errorf("creating a temporary file beside %s: %w", path, err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()

	if _, err := tmp.WriteString(file.render()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", name, err)
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return fmt.Errorf("setting the mode of %s: %w", name, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}

// ruleList is the file's list of rule groups and the format it is kept in.
type ruleList struct {
	general *iniSection
	// order is KWin 6.8's format: the list in `Order`, no `rules` or `count`.
	order bool
	names []string
}

/*
readList reads the rule list in whichever format the file is in.

A file with `Order` is a 6.8 file. A `rules` left beside it — by an earlier
version of this package — is folded in, so that store removes it before KWin
reads it as the only rules to keep. Groups the list does not name are rules
too: KWin 6.8 prepends them to its order.

A file with neither key but with rule groups is also a 6.8 file, since only 6.8
leaves rules unlisted, and a legacy list naming one of them would delete the
others. A file with `rules` alone, or with no rules at all, stays legacy: KWin
6.7 reads only that, and 6.8 migrates a list that names every group without
losing any.
*/
func readList(file *iniFile) ruleList {
	l := ruleList{general: file.ensure(generalSection)}
	order, hasOrder := l.general.get(orderKey)
	legacy, hasLegacy := l.general.get(legacyKey)

	add := func(name string) {
		if file.section(name) != nil && !slices.Contains(l.names, name) {
			l.names = append(l.names, name)
		}
	}
	if hasLegacy && !hasOrder {
		l.names = splitNames(legacy)
		return l
	}
	l.order = true
	for _, n := range splitNames(order) {
		add(n)
	}
	for _, n := range splitNames(legacy) {
		add(n)
	}
	for _, s := range file.sections {
		if !isMetaData(s.name) {
			add(s.name)
		}
	}
	if !hasOrder && len(l.names) == 0 {
		l.order = false // an empty file: the legacy keys, which every KWin reads
	}
	return l
}

// store writes the list back in the file's format.
func (l ruleList) store() {
	joined := strings.Join(l.names, ",")
	if l.order {
		l.general.set(orderKey, joined)
		l.general.unset(legacyKey)
		l.general.unset(countKey)
		return
	}
	l.general.set(legacyKey, joined)
	l.general.set(countKey, strconv.Itoa(len(l.names)))
}

// splitNames splits a list value, empty entries dropped.
func splitNames(raw string) []string {
	var out []string
	for _, n := range strings.Split(raw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// isMetaData is KWin's own test for a group that is not a rule.
func isMetaData(name string) bool {
	return name == generalSection || strings.HasPrefix(name, "$")
}

// sectionFor finds the listed rule whose wmclass is appID and whose title is
// title (no title key at all when title is empty), or "".
func sectionFor(file *iniFile, list ruleList, appID, title string) string {
	for _, name := range list.names {
		sec := file.section(name)
		if sec == nil {
			continue
		}
		class, ok := sec.get("wmclass")
		if !ok || class != appID {
			continue
		}
		have, _ := sec.get("title")
		if have == title {
			return name
		}
	}
	return ""
}

// appendRule adds a new numbered section and lists it. The number is one past
// the highest group in the file, listed or not, rather than the count: a file
// whose rules were removed out of order has gaps, and a number reused would
// write this rule over one the user, or another program, still has.
func appendRule(file *iniFile, list *ruleList) *iniSection {
	next := 1
	for _, s := range file.sections {
		if v, err := strconv.Atoi(s.name); err == nil && v >= next {
			next = v + 1
		}
	}
	name := strconv.Itoa(next)

	list.names = append(list.names, name)
	sec := &iniSection{name: name}
	file.sections = append(file.sections, sec)
	return sec
}

// write lays r into a section, clearing the keys it does not set so a rule
// edited down from a larger one does not keep the parts that were removed.
// order is the file's format (readList), which decides how NoBorder is said.
func write(sec *iniSection, r Rule, order bool) {
	sec.set("Description", descriptionOf(r))
	sec.set("wmclass", r.AppID)
	sec.set("wmclassmatch", "1") // exact
	if r.Title != "" {
		sec.set("title", r.Title)
		sec.set("titlematch", "1") // exact
	} else {
		sec.unset("title")
		sec.unset("titlematch")
	}

	setFlag(sec, "above", r.AlwaysOnTop, ruleForce)
	// KWin 6.8 replaced noborder with a decoration policy, and migrates one to
	// the other when it loads; 6.7 knows only noborder.
	setFlag(sec, "noborder", r.NoBorder && !order, ruleForce)
	if r.NoBorder && order {
		sec.set("decorationpolicy", decorationNone)
		sec.set("decorationpolicyrule", ruleForce)
	} else {
		sec.unset("decorationpolicy")
		sec.unset("decorationpolicyrule")
	}
	setFlag(sec, "skiptaskbar", r.SkipTaskbar, ruleForce)
	setFlag(sec, "skipswitcher", r.SkipSwitcher, ruleForce)
	setFlag(sec, "skippager", r.SkipPager, ruleForce)
	if r.NoFocus {
		sec.set("acceptfocus", "false")
		sec.set("acceptfocusrule", ruleForce)
	} else {
		sec.unset("acceptfocus")
		sec.unset("acceptfocusrule")
	}

	if r.Opacity > 0 {
		v := strconv.Itoa(r.Opacity)
		sec.set("opacityactive", v)
		sec.set("opacityactiverule", ruleForce)
		sec.set("opacityinactive", v)
		sec.set("opacityinactiverule", ruleForce)
	} else {
		for _, k := range []string{
			"opacityactive", "opacityactiverule",
			"opacityinactive", "opacityinactiverule",
		} {
			sec.unset(k)
		}
	}

	for _, k := range staleKeys {
		sec.unset(k)
	}
}

// setFlag writes a boolean property and its rule type, or removes both.
func setFlag(sec *iniSection, key string, on bool, ruleType string) {
	if on {
		sec.set(key, "true")
		sec.set(key+"rule", ruleType)
		return
	}
	sec.unset(key)
	sec.unset(key + "rule")
}

// descriptionOf is the rule's description, falling back to something the user
// can recognise in System Settings.
func descriptionOf(r Rule) string {
	if r.Description != "" {
		return r.Description
	}
	if r.Title != "" {
		return r.AppID + " " + r.Title + " (glance window)"
	}
	return r.AppID + " (glance window)"
}

// DBusCall names a method on the session bus.
type DBusCall struct {
	Destination string
	Path        string
	Interface   string
	Method      string
}

// ReconfigureCall is the session-bus method that makes KWin re-read its rules.
// A rule written by Install does nothing until this is called, or until the
// user logs out and back in.
//
// It is returned as data rather than made here: this module's packages take no
// dependency beyond Fyne, and a D-Bus client is not Fyne. A program that
// already has a session bus connection makes the call itself.
func ReconfigureCall() DBusCall {
	return DBusCall{
		Destination: "org.kde.KWin",
		Path:        "/KWin",
		Interface:   "org.kde.KWin",
		Method:      "reconfigure",
	}
}
