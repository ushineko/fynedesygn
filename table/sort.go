package table

import "fyne.io/fyne/v2/widget"

// SortDir is the direction a sortable column is currently sorted in.
type SortDir int

const (
	// Unsorted is a column that is not the sort key; its title carries no arrow.
	Unsorted SortDir = iota
	// Ascending sorts low to high and appends an up arrow to the title.
	Ascending
	// Descending sorts high to low and appends a down arrow to the title.
	Descending
)

// SortTitle is the header text for a column: the title alone when the column
// is not the sort key, otherwise the title followed by two spaces and an
// arrow for the direction.
func SortTitle(title string, dir SortDir) string {
	switch dir {
	case Ascending:
		return title + "  ↑"
	case Descending:
		return title + "  ↓"
	case Unsorted:
	}
	return title
}

// SortHeader returns the header button the sortable tables use: low
// importance, leading-aligned, with the sort arrow appended to the title in
// the text. It is a button rather than a label because a header cell holds
// one object and a label cannot receive a tap.
//
// Contract for tapped: after re-sorting, clear the table's selection and
// disable the row-action buttons. The selection indexed into the old order,
// so it no longer names the row the user picked; carrying it over would leave
// a destructive action armed against a different row than the one
// highlighted. See docs/design-system.md, "Tables and lists".
func SortHeader(title string, dir SortDir, tapped func()) *widget.Button {
	b := widget.NewButton(SortTitle(title, dir), tapped)
	b.Importance = widget.LowImportance
	b.Alignment = widget.ButtonAlignLeading
	return b
}
