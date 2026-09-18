/*
Package forms holds the form shapes the consuming programs share: a form
whose widgets are held apart from the section so Save and Revert are testable
headlessly and so Revert can revert, the slider-and-entry pair where only one
control commits, and the validated numeric entry.

Ported from nmsbonker's settingsForm, editField and paramRow and
clockwork-orange's numericEntry (same author, MIT).

One rule runs through the package: programmatic writes go through a re-entry
guard, because Fyne's setters (Check.SetChecked, Select.SetSelected,
Entry.SetText) fire the change handlers, and a section build must never
schedule a save.
*/
package forms

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/dialogs"
	"github.com/ushineko/fynedesygn/widgets"
)

// NumericWidth is the width of a Numeric field's entry.
const NumericWidth float32 = 120

// Field is one labelled control that reads and writes a string value. The
// string shape is deliberate: the programs compare a map of strings against a
// settings file, and a Revert that reverts is a Set of that map.
type Field struct {
	// Key names the value in Set and Values; Label is shown beside it.
	Key, Label string
	// Widget is what the form lays out.
	Widget fyne.CanvasObject
	// Get reads the control; Set writes it without firing OnChange.
	Get func() string
	Set func(string)

	// notify is wired by Form.New; controls call it on a user change.
	notify func(value string)
	// setting is the re-entry guard shared with the form.
	setting *bool
}

func (f *Field) changed(v string) {
	if f.setting != nil && *f.setting {
		return
	}
	if f.notify != nil {
		f.notify(v)
	}
}

// Entry is a single-line text field.
func Entry(key, label, placeholder string) *Field {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	f := &Field{Key: key, Label: label, Widget: e, Get: func() string { return e.Text }}
	f.Set = e.SetText
	e.OnChanged = f.changed
	return f
}

// Select is a drop-down over fixed options.
func Select(key, label string, options []string) *Field {
	s := widget.NewSelect(options, nil)
	f := &Field{Key: key, Label: label, Widget: s, Get: func() string { return s.Selected }}
	f.Set = s.SetSelected
	s.OnChanged = f.changed
	return f
}

// Check is a boolean whose value is "true" or "false".
func Check(key, label string) *Field {
	c := widget.NewCheck("", nil)
	f := &Field{Key: key, Label: label, Widget: c, Get: func() string { return strconv.FormatBool(c.Checked) }}
	f.Set = func(v string) { c.SetChecked(strings.EqualFold(strings.TrimSpace(v), "true")) }
	c.OnChanged = func(b bool) { f.changed(strconv.FormatBool(b)) }
	return f
}

// Path is an entry with a Browse button; dir picks a folder chooser.
func Path(win fyne.Window, key, label string, dir bool) *Field {
	e := widget.NewEntry()
	f := &Field{Key: key, Label: label, Widget: dialogs.WithBrowse(win, e, dir), Get: func() string { return e.Text }}
	f.Set = e.SetText
	e.OnChanged = f.changed
	return f
}

// Numeric is a whole-number entry validated to lo..hi, NumericWidth wide.
// OnChange fires only for valid values.
func Numeric(key, label string, lo, hi int) *Field {
	e := widget.NewEntry()
	e.Validator = IntRange(lo, hi)
	f := &Field{Key: key, Label: label, Widget: widgets.FixedWidth(e, NumericWidth), Get: func() string { return e.Text }}
	f.Set = e.SetText
	e.OnChanged = func(s string) {
		if e.Validator(strings.TrimSpace(s)) == nil {
			f.changed(strings.TrimSpace(s))
		}
	}
	return f
}

// Custom wraps a control of the caller's: get and set read and write it, and
// the caller arranges for notify to be called on a user change (through the
// returned field's Notify).
func Custom(key, label string, w fyne.CanvasObject, get func() string, set func(string)) *Field {
	return &Field{Key: key, Label: label, Widget: w, Get: get, Set: set}
}

// Notify reports a user change on a Custom field. It is ignored while the
// form is setting values.
func (f *Field) Notify(value string) { f.changed(value) }

// Form is a set of fields with one guard and one change hook.
type Form struct {
	fields  []*Field
	setting bool
	// OnChange is called with the key and new value after a user change.
	// Programs write into their document here and schedule one coalesced
	// save; the guard keeps a Set from arriving here.
	OnChange func(key, value string)
}

// New wires the fields to one form.
func New(fields ...*Field) *Form {
	f := &Form{fields: fields}
	for _, fld := range fields {
		fld.setting = &f.setting
		key := fld.Key
		fld.notify = func(v string) {
			if f.OnChange != nil {
				f.OnChange(key, v)
			}
		}
	}
	return f
}

// Set writes values into the fields without firing OnChange. Keys the form
// does not have are ignored; fields not in the map are left alone.
func (f *Form) Set(values map[string]string) {
	f.setting = true
	defer func() { f.setting = false }()
	for _, fld := range f.fields {
		if v, ok := values[fld.Key]; ok {
			fld.Set(v)
		}
	}
}

// Values reads every field.
func (f *Form) Values() map[string]string {
	out := make(map[string]string, len(f.fields))
	for _, fld := range f.fields {
		out[fld.Key] = fld.Get()
	}
	return out
}

// Field finds a field by key, or nil.
func (f *Form) Field(key string) *Field {
	for _, fld := range f.fields {
		if fld.Key == key {
			return fld
		}
	}
	return nil
}

// Fields are the form's fields in order.
func (f *Form) Fields() []*Field { return f.fields }

// Widget lays the fields out as a widget.Form.
func (f *Form) Widget() fyne.CanvasObject {
	items := make([]*widget.FormItem, 0, len(f.fields))
	for _, fld := range f.fields {
		items = append(items, widget.NewFormItem(fld.Label, fld.Widget))
	}
	return widget.NewForm(items...)
}

// IntRange validates a whole number from lo to hi inclusive, with messages a
// reader can act on.
func IntRange(lo, hi int) fyne.StringValidator {
	return func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("enter a whole number")
		}
		if n < lo || n > hi {
			return fmt.Errorf("enter a number from %d to %d", lo, hi)
		}
		return nil
	}
}

// NumericEntry is a validated whole-number entry that calls onChange only
// with values in range.
func NumericEntry(lo, hi int, onChange func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.Validator = IntRange(lo, hi)
	e.OnChanged = func(s string) {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n >= lo && n <= hi {
			onChange(n)
		}
	}
	return e
}

// Group is a titled block of rows with a dim blurb: a handful of fields with
// one line saying what they are.
func Group(title, blurb string, rows ...fyne.CanvasObject) fyne.CanvasObject {
	head := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	items := []fyne.CanvasObject{head}
	if blurb != "" {
		items = append(items, widgets.Dim(blurb))
	}
	return container.NewVBox(append(items, rows...)...)
}
