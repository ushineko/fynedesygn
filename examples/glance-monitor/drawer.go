package main

import (
	"math"
	"sort"

	"github.com/ushineko/fynedesygn/glance"
)

/*
The drawer: what a card has no room to say, beside the cards and as tall as
them. A click on a card, or on one of the thermal card's rows, opens it on
that subject; a click on something else changes the subject; a click on the
same thing closes it. Those three are this program's decisions, not the
library's, which only says what was clicked.

The list is generated, like every reading here. It is as long as the drawer's
height fits, greatest first, and `f` filters it.
*/
type drawer struct {
	filter *glance.Filter
	rows   []*glance.Row

	// subject is what the drawer was opened on: the card, and the row when
	// the click landed on one that has a subject of its own.
	subject glance.Tap

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

func (m *monitor) buildDrawer() {
	d := &drawer{filter: glance.NewFilter()}
	d.filter.OnChanged = m.renderDrawer
	pieces := []glance.Piece{d.filter}
	for range drawerRows {
		r := glance.NewRowWidth("", glance.NoRate(), processLabel)
		d.rows = append(d.rows, r)
		pieces = append(pieces, r)
	}
	m.drawer = d

	dr := m.win.Panel().Drawer()
	dr.SetContent(pieces...)
	dr.SetFilter(d.filter)
	m.win.Panel().OnTapped = m.tapped
}

// subjectOf is what a click asks about, and its heading. The thermal card's
// rows are subjects of their own; anywhere else on a card is the card.
func (m *monitor) subjectOf(tap glance.Tap) (glance.Tap, string) {
	switch {
	case tap.Card == m.bandwidth:
		return glance.Tap{Card: tap.Card}, "Bandwidth by process"
	case tap.Card == m.battery:
		return glance.Tap{Card: tap.Card}, "Peripherals"
	case tap.Card == m.usage:
		return glance.Tap{Card: tap.Card}, "Usage by process"
	case tap.Row == m.cpu:
		return glance.Tap{Card: tap.Card, Row: tap.Row}, "CPU by process"
	case tap.Row == m.coolant:
		return glance.Tap{Card: tap.Card, Row: tap.Row}, "Coolant"
	default:
		return glance.Tap{Card: tap.Card}, "AIO"
	}
}

// tapped opens the drawer on what was clicked, and closes it on a second
// click of the same thing.
func (m *monitor) tapped(tap glance.Tap) {
	subject, title := m.subjectOf(tap)
	dr := m.win.Panel().Drawer()
	if dr.Open() && m.drawer.subject == subject {
		dr.SetOpen(false)
		return
	}
	m.drawer.subject = subject
	dr.SetTitle(title)
	dr.SetOpen(true)
	m.renderDrawer()
}

// renderDrawer draws the list: the names the filter passes, greatest first,
// in as many rows as fit. It runs on the UI thread.
func (m *monitor) renderDrawer() {
	dr := m.win.Panel().Drawer()
	d := m.drawer
	if d == nil || !dr.Open() {
		return
	}
	fit := min(dr.Fit(d.rows[0], d.filter), len(d.rows))

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
		wave := math.Abs(math.Sin(float64(m.ticks)/float64(7+i) + float64(i)))
		list = append(list, process{name, wave * float64(int64(1)<<(10+i%14))})
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
