package main

import (
	"math"
	"sort"

	"github.com/ushineko/fynedesygn/glance"
)

/*
The drawers: what a card has no room to say, beside the cards and as tall as
them. A click anywhere on a card opens a drawer on that card; a click on
another card changes the subject; a click on the same card closes it. Those
three are this program's decisions, not the library's, which only says what
was clicked.

A Shift-click opens the card's drawer beside the ones already open, or closes
it when it is open, and with several open a plain click closes them all (spec
079). Where the room beside the window cannot be read the row is held to
glance.DrawersWithoutRoom; at the cap the outermost drawer takes the new card.

The thermal card has two subjects, and the choice between them is a switch in
the drawer (spec 069): one target on the card, and no going back to it to see
the other. The usage card's drawer is every account's quotas, more than the
cards are tall, so it scrolls; the lists do not, and are cut to the height.

The lists are generated, like every reading here: greatest first, and `f`
filters them.
*/
type drawer struct {
	dr     *glance.Drawer
	sw     *glance.Switch
	filter *glance.Filter
	rows   []*glance.Row

	// quotas are the usage card's drawer: accounts of meters with a rule
	// between them. quotaPieces is the same, as the drawer's content, and
	// showsQuotas whether that is the content now.
	quotas      []*glance.Meter
	quotaPieces []glance.Piece
	showsQuotas bool

	// subject is the card the drawer is open on, and thermal the thermal
	// card's subject it shows.
	subject *glance.Card
	thermal string

	// listed is the names drawn, top to bottom.
	listed []string
}

// drawerRows is how many rows the drawer holds; it shows as many as fit.
const drawerRows = 40

// processLabel is the width of a process's name. Pinned, because a name
// arrives with every reading and must not move the figure beside it.
const processLabel float32 = 150

var processNames = []string{
	"firefox", "code", "steam", "spotify", "dropbox", "kwin_wayland", "plasmashell",
	"pipewire", "syncthing", "tailscaled", "gopls", "docker", "obs", "discord",
	"terraria", "konsole", "baloo_file", "systemd", "Xwayland", "zsh",
	"firefox-bin", "chrome", "chrome_crashpad", "kded6", "kscreen", "ssh-agent",
}

// The thermal card's two subjects, and how each is chosen.
var thermalChoices = []glance.Choice{
	{ID: "cpu", Label: "CPU", Key: 'c'},
	{ID: "coolant", Label: "Coolant", Key: 'o'},
}

var thermalTitles = map[string]string{"cpu": "CPU by process", "coolant": "Coolant"}

// quotaAccounts and quotaLimits are the usage drawer: that many accounts of
// that many limits each, which is more than four cards are tall.
const (
	quotaAccounts = 4
	quotaLimits   = 4
)

// buildDrawer makes the first drawer's pieces and points the panel's clicks
// at the drawers. More are made as Shift-clicks ask for them.
func (m *monitor) buildDrawer() {
	m.thermalLast = "cpu"
	m.drawer = m.newDrawer(m.win.Panel().Drawer())
	m.win.Panel().OnTapped = m.tapped
}

// newDrawer gives a drawer of the panel's its own pieces.
func (m *monitor) newDrawer(dr *glance.Drawer) *drawer {
	d := &drawer{dr: dr, sw: glance.NewSwitch(), filter: glance.NewFilter(), thermal: "cpu"}
	d.filter.OnChanged = func() { m.renderOne(d) }
	d.sw.OnChanged = func(id string) {
		d.thermal, m.thermalLast = id, id
		dr.SetTitle(thermalTitles[id])
		m.renderOne(d)
	}
	for range drawerRows {
		d.rows = append(d.rows, glance.NewRowWidth("", glance.NoRate(), processLabel))
	}
	for account := range quotaAccounts {
		if account > 0 {
			d.quotaPieces = append(d.quotaPieces, glance.NewRule())
		}
		d.quotaPieces = append(d.quotaPieces, glance.NewRow("account "+string(rune('A'+account)), ""))
		for range quotaLimits {
			q := glance.NewMeter("", meterLabel)
			d.quotas = append(d.quotas, q)
			d.quotaPieces = append(d.quotaPieces, q)
		}
	}
	dr.SetContent(d.listPieces()...)
	dr.SetFilter(d.filter)
	dr.SetSwitch(d.sw)
	m.drawers = append(m.drawers, d)
	return d
}

// listPieces are a list's content: the switch, the filter and the rows.
func (d *drawer) listPieces() []glance.Piece {
	pieces := []glance.Piece{d.sw, d.filter}
	for _, r := range d.rows {
		pieces = append(pieces, r)
	}
	return pieces
}

// titleOf is the heading of a card's drawer.
func (m *monitor) titleOf(d *drawer, card *glance.Card) string {
	switch card {
	case m.bandwidth:
		return "Bandwidth by process"
	case m.battery:
		return "Peripherals"
	case m.usage:
		return "Usage by account"
	default:
		return thermalTitles[d.thermal]
	}
}

// open is the drawers that are out.
func (m *monitor) open() []*drawer {
	var out []*drawer
	for _, d := range m.drawers {
		if d.dr.Open() {
			out = append(out, d)
		}
	}
	return out
}

// tapped opens a drawer on the card clicked, changes its subject, or closes
// it; a Shift-click adds or closes one beside the others. What on the card
// was clicked makes no difference.
func (m *monitor) tapped(tap glance.Tap) {
	if tap.Shift {
		m.shiftTapped(tap.Card)
		return
	}
	open := m.open()
	switch {
	case len(open) > 1:
		for _, d := range open {
			d.dr.SetOpen(false)
		}
	case len(open) == 1 && open[0].subject == tap.Card:
		open[0].dr.SetOpen(false)
	case len(open) == 1:
		m.show(open[0], tap.Card)
	default:
		m.show(m.drawer, tap.Card)
	}
}

// shiftTapped closes the card's drawer when one is open, and otherwise opens
// one beside the rest: in a drawer of its own where there is room, in the
// outermost where there is not.
func (m *monitor) shiftTapped(card *glance.Card) {
	open := m.open()
	for _, d := range open {
		if d.subject == card {
			d.dr.SetOpen(false)
			return
		}
	}
	if len(open) == 0 {
		m.show(m.drawer, card)
		return
	}
	p := m.win.Panel()
	fits, known := p.FitsHere()
	if !known {
		fits = p.Opened() < glance.DrawersWithoutRoom
	}
	if !fits {
		for _, d := range open {
			if d.dr == p.Outermost() {
				m.show(d, card)
				return
			}
		}
	}
	for _, d := range m.drawers {
		if !d.dr.Open() {
			m.show(d, card)
			return
		}
	}
	m.show(m.newDrawer(p.AddDrawer()), card)
}

// show points a drawer at a card and opens it.
func (m *monitor) show(d *drawer, card *glance.Card) {
	dr := d.dr
	d.subject = card
	d.filter.Clear()

	// The content is replaced when its shape changes, a list for the quotas,
	// and not otherwise.
	if quotas := card == m.usage; quotas != d.showsQuotas {
		d.showsQuotas = quotas
		dr.SetScrolling(quotas)
		if quotas {
			dr.SetContent(d.quotaPieces...)
		} else {
			dr.SetContent(d.listPieces()...)
		}
	}
	// Only the thermal card has something to choose between, and it opens
	// on the subject chosen last.
	if card == m.thermal {
		d.thermal = m.thermalLast
		d.sw.SetChoices(thermalChoices)
		d.sw.Select(d.thermal)
	} else {
		d.sw.SetChoices(nil)
	}
	dr.SetTitle(m.titleOf(d, card))
	dr.SetOpen(true)
	m.renderOne(d)
}

// renderQuotas draws a usage drawer's meters, each a generated share.
func (m *monitor) renderQuotas(d *drawer) {
	windows := []string{"5h", "7d", "model", "month"}
	for i, q := range d.quotas {
		share := math.Abs(math.Sin(float64(m.ticks)/float64(40+3*i) + float64(i)))
		q.SetLabel(windows[i%quotaLimits])
		q.Set(share, glance.Percent(share*100), quotaStatus(share*100))
	}
}

// renderDrawer draws every open drawer. It runs on the UI thread.
func (m *monitor) renderDrawer() {
	for _, d := range m.drawers {
		m.renderOne(d)
	}
}

// renderOne draws a drawer's list: the names the filter passes, greatest
// first, in as many rows as fit.
func (m *monitor) renderOne(d *drawer) {
	dr := d.dr
	if !dr.Open() {
		return
	}
	if d.showsQuotas {
		m.renderQuotas(d)
		return
	}
	fit := min(dr.Fit(d.rows[0], d.sw, d.filter), len(d.rows))
	// The two thermal lists are the same names in another order.
	turn := 0
	if d.subject == m.thermal && d.thermal == "coolant" {
		turn = 5
	}

	type process struct {
		name string
		rate float64
	}
	var list []process
	for i, name := range processNames {
		if !d.filter.Matches(name) {
			continue
		}
		// Rates spread over many magnitudes, so the list reorders and the
		// fixed-width format is seen doing its work.
		wave := math.Abs(math.Sin(float64(m.ticks)/float64(7+i) + float64(i+turn)))
		list = append(list, process{name, wave * float64(int64(1)<<(10+(i+turn)%14))})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].rate > list[j].rate })

	d.listed = d.listed[:0]
	for i, r := range d.rows {
		shown := i < fit && i < len(list)
		if shown {
			d.listed = append(d.listed, list[i].name)
			r.SetLabel(list[i].name)
			r.Set(glance.Measured(glance.Rate(list[i].rate)))
		}
		r.SetShown(shown)
	}
}
