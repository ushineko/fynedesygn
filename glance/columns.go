package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Arrangement is how a panel lays its cards out.
type Arrangement int

const (
	// Stack is one card above another, full width. The shape of a narrow
	// desktop panel and the default.
	Stack Arrangement = iota
	// Grid reflows the cards into as many columns as the width fits. A width
	// that fits one column is Stack, so a panel that is never widened never
	// looks any different.
	Grid
)

// CardWidthInEms is MinCardWidth in text sizes: about twenty-two characters,
// which is a label and a reading with something between them.
const CardWidthInEms = 22

// MinCardWidth is the narrowest a column may be before the grid gives up and
// stacks.
//
// Below it a card's label and its value have no room between them and the
// reflow is worse than no reflow. It is a function of the text size for the
// reason CardPadH is: a width set for one face is wrong at another.
func MinCardWidth() float32 { return theme.TextSize() * CardWidthInEms }

/*
cardsLayout lays a panel's cards out, in one column or in several.

One layout for both arrangements rather than swapping containers, because the
arrangement changes while the panel is on screen -- it is a setting -- and a
container swapped under a window mid-run loses the objects' sizes and makes
the panel jump.

Column-major, so reading down a column follows the order the cards were added
in. That is the terminal panel's rule and there is no reason for the window to
disagree with it: the same setting produces the same reading order in both.
*/
type cardsLayout struct {
	panel *Panel

	// width is what the panel was last laid out at. MinSize has to answer in
	// columns and Fyne does not tell it the width, so it is remembered here
	// -- the same reason cellGridLayout keeps one. Before the first layout it
	// is zero, which reports one column and is the honest answer for a panel
	// nobody has sized yet.
	width float32
}

// gap is the space between cards, in the panel's own face.
func (l *cardsLayout) gap() float32 {
	if l.panel == nil {
		return CardGap()
	}
	return l.panel.gap()
}

// minCard is the narrowest column, in the panel's own face.
func (l *cardsLayout) minCard() float32 {
	if l.panel == nil {
		return MinCardWidth()
	}
	return l.panel.textSize() * CardWidthInEms
}

// columns is how many the width fits, and never more than there are cards.
//
// One column is Stack. A grid that drew a single narrow column would be a
// stack with worse spacing, and the caller does not have to special-case it.
func (l *cardsLayout) columns(width float32, cards int) int {
	if l.panel == nil || l.panel.arrangement != Grid || cards == 0 {
		return 1
	}
	gap, minCard := l.gap(), l.minCard()
	n := int((width + gap) / (minCard + gap))
	return max(1, min(n, cards))
}

// perColumn is how many cards each column holds, column-major.
func perColumn(cards, columns int) int {
	if columns < 1 {
		return cards
	}
	return (cards + columns - 1) / columns
}

/*
shape is how many columns the cards actually occupy at a width, and how many
go in each.

The count the width fits is not the count that gets drawn. Four cards in three
columns is two per column -- and two columns hold all four, so the third is
empty. Sizing for the count the width fits then left a third of the window
blank with the cards crammed into the rest, which is what a panel widened to
900 with four cards did: three columns of 292 where two of 444 were meant.

So the columns are counted back from the rows. It never grows the count and it
never leaves a column empty.
*/
func (l *cardsLayout) shape(width float32, cards int) (cols, rows int) {
	rows = perColumn(cards, l.columns(width, cards))
	if rows < 1 {
		return 1, cards
	}
	return (cards + rows - 1) / rows, rows
}

// Layout places the cards.
func (l *cardsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	shown := visible(objects)
	if len(shown) == 0 {
		return
	}
	l.width = size.Width
	cols, rows := l.shape(size.Width, len(shown))
	gap := l.gap()
	colWidth := (size.Width - gap*float32(cols-1)) / float32(cols)

	for i, o := range shown {
		col, row := i/rows, i%rows
		y := float32(0)
		for r := range row {
			y += shown[col*rows+r].MinSize().Height + gap
		}
		o.Move(fyne.NewPos(float32(col)*(colWidth+gap), y))
		o.Resize(fyne.NewSize(colWidth, o.MinSize().Height))
	}

	// Anything hidden is moved out of the way rather than left where it was:
	// a hidden card that kept a position still reserves it from Fyne's point
	// of view the moment it is shown again.
	for _, o := range objects {
		if !o.Visible() {
			o.Resize(fyne.NewSize(0, 0))
		}
	}
}

// MinSize is the tallest column's height and one column's minimum width.
//
// One column wide, because a panel that demanded room for a pair would set
// the width of a window that only ever holds one card. The height has to be
// the tallest column or the last card in it is cut off.
func (l *cardsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	shown := visible(objects)
	if len(shown) == 0 {
		return fyne.Size{}
	}

	_, rows := l.shape(l.width, len(shown))
	gap := l.gap()

	widest, tallest := float32(0), float32(0)
	for col := 0; col*rows < len(shown); col++ {
		height := float32(0)
		for row := 0; row < rows && col*rows+row < len(shown); row++ {
			o := shown[col*rows+row]
			if row > 0 {
				height += gap
			}
			height += o.MinSize().Height
			widest = max(widest, o.MinSize().Width)
		}
		tallest = max(tallest, height)
	}
	return fyne.NewSize(widest, tallest)
}
