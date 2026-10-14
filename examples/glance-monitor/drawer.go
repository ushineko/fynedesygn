package main

import (
	"math"
	"sort"

	"github.com/ushineko/fynedesygn/glance"
)

/*
The drawer: what a card has no room to say, beside the cards and as tall as
them. A click anywhere on a card opens it on that card; a click on another
card changes the subject; a click on the same card closes it. Those three are
this program's decisions, not the library's, which only says what was
clicked.

The thermal card has two subjects, and the choice between them is a switch in
the drawer (spec 069): one target on the card, and no going back to it to see
the other. The usage card's drawer is every account's quotas, more than the
cards are tall, so it scrolls; the lists do not, and are cut to the height.

The lists are generated, like every reading here: greatest first, and `f`
filters them.
*/
type drawer struct {
	sw     *glance.Switch
	filter *glance.Filter
	rows   []*glance.Row

	// quotas are the usage card's drawer: accounts of meters with a rule
	// between them. quotaPieces is the same, as the drawer's content, and
	// showsQuotas whether that is the content now.
	quotas      []*glance.Meter
	quotaPieces []glance.Piece
	showsQuotas bool

	// subject is the card the drawer was opened on, and thermal the choice
	// last made on the thermal card's.
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

func (m *monitor) buildDrawer() {
	d := &drawer{sw: glance.NewSwitch(), filter: glance.NewFilter(), thermal: "cpu"}
	d.filter.OnChanged = m.renderDrawer
	d.sw.OnChanged = func(id string) {
		d.thermal = id
		m.win.Panel().Drawer().SetTitle(thermalTitles[id])
		m.renderDrawer()
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
	m.drawer = d

	dr := m.win.Panel().Drawer()
	dr.SetContent(d.listPieces()...)
	dr.SetFilter(d.filter)
	dr.SetSwitch(d.sw)
	m.win.Panel().OnTapped = m.tapped
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
func (m *monitor) titleOf(card *glance.Card) string {
	switch card {
	case m.bandwidth:
		return "Bandwidth by process"
	case m.battery:
		return "Peripherals"
	case m.usage:
		return "Usage by account"
	default:
		return thermalTitles[m.drawer.thermal]
	}
}

// tapped opens the drawer on the card clicked, and closes it on a second
// click of the same card. What on the card was clicked makes no difference.
func (m *monitor) tapped(tap glance.Tap) {
	d := m.drawer
	dr := m.win.Panel().Drawer()
	if dr.Open() && d.subject == tap.Card {
		dr.SetOpen(false)
		return
	}
	d.subject = tap.Card
	d.filter.Clear()

	// The content is replaced when its shape changes, a list for the quotas,
	// and not otherwise.
	if quotas := tap.Card == m.usage; quotas != d.showsQuotas {
		d.showsQuotas = quotas
		dr.SetScrolling(quotas)
		if quotas {
			dr.SetContent(d.quotaPieces...)
		} else {
			dr.SetContent(d.listPieces()...)
		}
	}
	// Only the thermal card has something to choose between.
	if tap.Card == m.thermal {
		d.sw.SetChoices(thermalChoices)
		d.sw.Select(d.thermal)
	} else {
		d.sw.SetChoices(nil)
	}
	dr.SetTitle(m.titleOf(tap.Card))
	dr.SetOpen(true)
	m.renderDrawer()
}

// renderQuotas draws the usage drawer's meters, each a generated share.
func (m *monitor) renderQuotas() {
	windows := []string{"5h", "7d", "model", "month"}
	for i, q := range m.drawer.quotas {
		share := math.Abs(math.Sin(float64(m.ticks)/float64(40+3*i) + float64(i)))
		q.SetLabel(windows[i%quotaLimits])
		q.Set(share, glance.Percent(share*100), quotaStatus(share*100))
	}
}

// renderDrawer draws the list: the names the filter passes, greatest first,
// in as many rows as fit. It runs on the UI thread.
func (m *monitor) renderDrawer() {
	dr := m.win.Panel().Drawer()
	d := m.drawer
	if d == nil || !dr.Open() {
		return
	}
	if d.showsQuotas {
		m.renderQuotas()
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
