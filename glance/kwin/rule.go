package kwin

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Description is what System Settings shows in its rule list. A rule the
	// user cannot identify is a rule they cannot remove.
	Description string

	// AlwaysOnTop keeps the window above others, forced. It duplicates
	// desktop.Window.RequestAlwaysOnTop and is worth setting anyway: the rule
	// survives a compositor restart, and the toolkit's request is one the
	// window manager is allowed to decline.
	AlwaysOnTop bool

	// NoBorder removes the titlebar, forced. It duplicates what a splash
	// window already is, for the same reason.
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
}

// The rule-type vocabulary KWin uses for the "…rule" companion of each
// property. Only the two this package sets are named.
const (
	ruleForce          = "2"
	ruleApplyInitially = "4"
)

// generalSection holds the list of rules and their count.
const generalSection = "General"

// staleKeys are keys an earlier version of a rule may have written that this
// one must clear.
//
// Position is deliberately not a rule. KWin's `position` rule selects a screen
// and snaps to its origin on Wayland; it does not honour intra-screen
// coordinates. A rule that half-works is worse than none, so a rule that once
// set one has it removed rather than left behind.
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
// AppID if one is already there and appending a new one otherwise. Every other
// rule in the file is left as it was found.
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

	sec := file.section(sectionFor(file, r.AppID))
	if sec == nil {
		sec = appendRule(file)
	}
	write(sec, r)

	return save(path, file)
}

// Remove deletes the rule matching appID, and says whether there was one. The
// file is left untouched when there was not.
func Remove(appID string) (bool, error) {
	path, err := Path()
	if err != nil {
		return false, err
	}
	file, err := load(path)
	if err != nil {
		return false, err
	}

	name := sectionFor(file, appID)
	if name == "" {
		return false, nil
	}
	file.remove(name)

	general := file.ensure(generalSection)
	names := ruleNames(general)
	kept := names[:0]
	for _, n := range names {
		if n != name {
			kept = append(kept, n)
		}
	}
	general.set("rules", strings.Join(kept, ","))
	general.set("count", strconv.Itoa(len(kept)))

	return true, save(path, file)
}

// Lookup returns the rule installed for appID, and whether there is one. It is
// what a program's settings section asks before offering to install.
func Lookup(appID string) (Rule, bool, error) {
	path, err := Path()
	if err != nil {
		return Rule{}, false, err
	}
	file, err := load(path)
	if err != nil {
		return Rule{}, false, err
	}
	name := sectionFor(file, appID)
	if name == "" {
		return Rule{}, false, nil
	}
	sec := file.section(name)

	r := Rule{AppID: appID}
	r.Description, _ = sec.get("Description")
	above, _ := sec.get("above")
	r.AlwaysOnTop = above == "true"
	noborder, _ := sec.get("noborder")
	r.NoBorder = noborder == "true"
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

// ruleNames is the General section's rule list, empty entries dropped.
func ruleNames(general *iniSection) []string {
	raw, _ := general.get("rules")
	var out []string
	for _, n := range strings.Split(raw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// sectionFor finds the listed rule whose wmclass is appID, or "".
func sectionFor(file *iniFile, appID string) string {
	general := file.section(generalSection)
	if general == nil {
		return ""
	}
	for _, name := range ruleNames(general) {
		if sec := file.section(name); sec != nil {
			if class, ok := sec.get("wmclass"); ok && class == appID {
				return name
			}
		}
	}
	return ""
}

// appendRule adds a new numbered section and lists it. The number is one past
// the highest already in use rather than the count, because a file whose rules
// were removed out of order has gaps, and reusing a number would collide with
// a rule the user still has.
func appendRule(file *iniFile) *iniSection {
	general := file.ensure(generalSection)
	names := ruleNames(general)

	next := 1
	for _, n := range names {
		if v, err := strconv.Atoi(n); err == nil && v >= next {
			next = v + 1
		}
	}
	name := strconv.Itoa(next)

	names = append(names, name)
	general.set("rules", strings.Join(names, ","))
	general.set("count", strconv.Itoa(len(names)))

	return file.ensure(name)
}

// write lays r into a section, clearing the keys it does not set so a rule
// edited down from a larger one does not keep the parts that were removed.
func write(sec *iniSection, r Rule) {
	sec.set("Description", descriptionOf(r))
	sec.set("wmclass", r.AppID)
	sec.set("wmclassmatch", "1") // exact

	setFlag(sec, "above", r.AlwaysOnTop, ruleForce)
	setFlag(sec, "noborder", r.NoBorder, ruleForce)

	if r.Opacity > 0 {
		v := strconv.Itoa(r.Opacity)
		sec.set("opacityactive", v)
		sec.set("opacityactiverule", ruleApplyInitially)
		sec.set("opacityinactive", v)
		sec.set("opacityinactiverule", ruleApplyInitially)
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
