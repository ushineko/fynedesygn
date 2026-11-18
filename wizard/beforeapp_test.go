package wizard

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/forms"
)

/*
Every page and every form field can be made before there is an app (spec
084).

Page says a page is made before Run creates the app, and Form takes a form
built by its caller, so the fields are made then too. forms.Entry set its
placeholder through SetPlaceHolder, which measures the text, and with no app
there is nothing to measure it with: fynstall's installer stopped with a nil
pointer before its window (ushineko/fynstall#52). Every other test here has
the test app behind it, which is why none of them said so.
*/
func TestPagesAndFieldsAreMadeBeforeThereIsAnApp(t *testing.T) {
	was := fyne.CurrentApp()
	fyne.SetCurrentApp(nil)
	t.Cleanup(func() { fyne.SetCurrentApp(was) })

	require.NotPanics(t, func() {
		f := forms.New(
			forms.Entry("name", "Name", "a placeholder"),
			forms.Select("kind", "Kind", []string{"a", "b"}),
			forms.Check("on", "On"),
			forms.Numeric("port", "Port", 1, 65535),
			forms.Path(nil, "dir", "Directory", true),
		)
		_ = forms.NumericEntry(1, 10, nil)
		_ = Welcome("Welcome", "**Hello**")
		_ = Licence("The licence")
		_ = Form("Settings", f, map[string]string{"name": "x"}, nil)
		_ = Directory("Location", "Install into", "/tmp/x", nil)
		_ = Summary("Ready", "Install", func() []Fact { return nil })
		_ = Progress("Installing", []string{"one"}, "Done", func(context.Context, *Reporter) error { return nil })
		_ = Finish("Done", "It is done.")
	})
}
