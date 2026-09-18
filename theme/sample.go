package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Sample is the live preview an Appearance section shows under its pickers:
// regular and bold text at the chosen size, a monospace line in the chosen
// console font, and the three status colours side by side, so a change is
// judged on the spot rather than on the next section. The monospace line is
// the caller's, since a log line that looks like the program's own is the
// useful preview.
func Sample(monoLine string) fyne.CanvasObject {
	good := widget.NewLabel("good")
	good.Importance = widget.SuccessImportance
	warn := widget.NewLabel("warning")
	warn.Importance = widget.WarningImportance
	bad := widget.NewLabel("failed")
	bad.Importance = widget.DangerImportance
	return container.NewVBox(
		widget.NewLabelWithStyle("Sample", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Regular text at the chosen size."),
		widget.NewLabelWithStyle("Bold text, as used for headings.", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle(monoLine, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
		container.NewHBox(good, warn, bad),
	)
}
