# Wizard windows

A wizard is a window that takes a person through a fixed sequence of pages.
Each page asks for one thing or shows one thing. Next is available only when
the current page is complete. The last pages run a job and report how it
went. An installer has this shape, and so do a first-run setup and an import.

[design-system.md](design-system.md) holds the rules for an application
window, which is the shell. [glance.md](glance.md) holds the rules for a
glance window. This page holds the rules for the third shape.

A shell lets a person go to any section in any order. A wizard has an order,
a current page, and a point after which Back has no meaning. The two shapes
share `Status`, the palette, the fonts, the threading rules and the voice of
the copy.

![A fixed-size window titled "Example installer". On the left, the page list: Welcome marked with a play arrow as the current page, then Licence, Location, Options, Ready, Installing and Done, each with an empty circle. On the right, the bold page title "Welcome" over a rule, and two lines of Markdown, "This installs Example 1.0, which does nothing." with the name in bold. Along the bottom, under a rule, Back disabled, Next highlighted in blue, and Cancel.](img/installer-wizard.png)

This is `examples/installer-wizard` on its first page, in Breeze Dark.

Package `wizard` implements this. `examples/installer-wizard` goes through
every standard page, and its tests and the package's tests pin the rules
below. The first program on the package is
[fynstall](https://github.com/ushineko/fynstall), an installer framework.

## Contents

- [The window](#the-window)
- [Pages](#pages)
- [Moving between pages](#moving-between-pages)
- [The job](#the-job)
- [Leaving](#leaving)
- [Standard pages](#standard-pages)
- [Testing](#testing)
- [A window that asks once](#a-window-that-asks-once)

## The window

```
Border{
  left:   the page list, in the vocabulary of package steps
  center: Border{ top: the page heading, center: the one scroller }
  bottom: Separator, then Border{ left: the message line,
                                  right: Back, Next, Cancel }
}
```

- The window has a fixed size, 820 x 560 by default. A wizard is a short
  task, and a window that changes size between pages moves its own buttons.
  Some window managers ignore the fixed size, so the content is in a
  scroller and lays out at any size.
- The page list shows every page. The current page is marked running, the
  pages before it are done, and a page that does not apply is marked
  skipped. The list is not a navigation: a person cannot click a page to go
  to it.
- The page content is in one scroller, as a shell section is. A component on
  a page fills the space it receives, or it has a fixed height and its own
  scrollbar, as the log on the progress page does.

**The message line replaces the banner.** The leading part of the button row
is one line of text in a `Status` colour. It is always present, so a message
never moves the interface. A shell shows a result in a floating banner; a
wizard shows it here, because the person is looking at the buttons when the
result arrives.

**The buttons have fixed widths.** Next can read Next, Install or Finish.
Its width is that of the widest label it can show, so the row never changes
width. A button that the person cannot use is disabled, never hidden.

## Pages

A page is an interface with two methods: `Title` and `Build`. Build runs
once, the first time the page is shown, and the page updates what it built
in place.

**A page's constructor holds data, and only Build touches widgets.** A
program makes its pages before `Run` creates the app, and Fyne cannot
refresh a widget before there is an app. That is why `Form` takes the
initial values and sets them in Build. Optional behaviour is a separate interface, in the way a shell
section opts into `Arriver`:

| Interface | Method | Use |
|---|---|---|
| `Validator` | `Valid() bool` | Next is enabled only while it is true. The page calls `Wizard.Revalidate` when its state changes. |
| `Enterer` | `Enter(w)` | Runs each time the page becomes current. A summary reads the earlier choices here. |
| `Leaver` | `Leave(w, forward) error` | Runs before the wizard leaves the page. An error keeps the page and shows on the message line. |
| `Skipper` | `Skip() bool` | Back and Next pass over the page while it is true. |
| `Labeller` | `NextLabel() string` | The text of Next on this page, such as Install. |
| `PointOfNoReturn` | `PointOfNoReturn() bool` | After this page starts, Back is not available again. |
| `Changer` | `Changed() bool` | Cancel asks for confirmation while a page that was shown reports a change. |

## Moving between pages

- Next goes to the next page that applies. On the last page it is Finish,
  and it closes the window with the outcome `Finished`.
- Back goes to the previous page that applies. It is disabled on the first
  page, while a job runs, and on every page from the point of no return on.
- A page that the person returns to keeps what they entered. Build does not
  run again.

## The job

The progress page runs a `Job` and shows a step list beside a log. The job
receives a context and a `Reporter`. `Advance`, `Finish` and `Log` are safe
to call from the job's goroutine; they move to the UI thread with `fyne.Do`.

- The job starts the first time the progress page becomes current. That page
  is the point of no return.
- While the job runs, Back and Next are disabled. Cancel asks for
  confirmation, cancels the context, and waits for the job to return.
- When the job succeeds, the message line shows the page's message in green
  and Next goes on.
- When the job fails, the current step is marked failed, the message line
  shows the error, the log stays visible, and Cancel becomes Close.

**A bar for a long job.** `Progress(...).WithBar()` adds a determinate bar,
a status line and the current item under the step list and the log. The
job calls `Reporter.Progress(fraction, status, item)`, as often as once per
file: an installer of thousands of files reports each one. The page keeps
the latest values and draws them every 100 ms, and once more when the job
ends, so thousands of reports cost a few draws. The item line is shortened
in the middle, so a long path keeps its start and its file name. The region
is there from the start, so nothing moves when counting begins, and it is
opt-in, because a job with no counts would show an empty bar.

**The job's error decides the outcome.** A job that was stopped returns an
error that wraps `context.Canceled`. A job that finished its work as Cancel
arrived returns nil, and the wizard reports it as finished. An installer
that reported such an install as cancelled would tell the person that
nothing was installed when the program was in fact on the disk.

## Leaving

Cancel and the close button of the title bar do the same thing:

| When | Effect |
|---|---|
| Before the point of no return, no page changed | The window closes. The outcome is `Cancelled`. |
| Before the point of no return, a page changed | Confirmation first, then as above. |
| While the job runs | Confirmation, then the job is cancelled. The window closes when the job returns. |
| After the job failed | Cancel reads Close. The outcome is `Failed`, with the error. |
| After the job succeeded | Cancel is disabled. The title bar's close finishes. |

`Run` returns a `Result` with the outcome, the job's error, and the state of
each check on the finish page, such as "Launch now".

## Standard pages

Each standard page is built only from components that the module already
has.

| Page | Built from |
|---|---|
| `Welcome(title, body)` | Markdown, rendered block by block in the page's scroller. |
| `Licence(body)` | Markdown and an "I accept" check. Valid while the check is set. |
| `Directory(title, label, initial, check)` | An entry with Browse. Valid while `check` returns nil; the error shows below the entry. |
| `Form(title, form, values, valid)` | A `forms.Form`, with `values` set into it in Build. Valid while `valid` returns true, asked again after each change. |
| `Summary(title, next, facts)` | Rows of facts, read again each time the page is entered. |
| `Progress(title, steps, done, job)` | A `steps.List` beside a `logpane.Pane`. |
| `Finish(title, body, checks...)` | Markdown and checks that the `Result` reports. |

## Testing

`wizard.Headless` builds a wizard over a test app with no window. A job runs
inline, and Cancel's confirmation is accepted at once, so a test is
deterministic. A test drives the wizard with `Next`, `Back` and `Cancel`, and
finds controls in `Content` with `fynetest`.

## A window that asks once

Some tasks need one decision and not a sequence of pages. An uninstaller
asks "Uninstall Hello 0.1.0?", runs, and reports. A wizard with a page list
is too much for that, and a dialog needs a window to sit over, which an
uninstaller started from the launcher does not have.

`RunConfirm` shows a small window of its own in three stages. The layout
does not change between them:

1. **Asking.** The question in bold, Markdown detail under it, and Cancel
   beside the action button. The detail says what will happen and what is
   left alone.
2. **Running.** The action button is disabled and an indeterminate bar
   runs in a region of fixed height.
3. **Result.** The message line shows the `Done` text or the error. The
   action button becomes Close and Cancel is disabled.

The rules are the wizard's. The job's own error decides the outcome, so a
job that finished as Cancel arrived is Finished. Cancel or the title bar's
close while the job runs asks, cancels the job's context, and closes when
the job returns. `SkipQuestion` runs the job at once and shows only the
result, for a program's `--yes`.

![A small fixed window titled "Uninstall Example". At the top, in bold, "Uninstall Example 1.0?" over a rule. Under it, two lines of Markdown: "It removes ~/.local/share/example and puts back anything the install replaced." and "Your settings in ~/.config/example are left where they are." Along the bottom, under a rule, Cancel and a red Uninstall button.](img/confirm-window.png)

This is `examples/installer-wizard -confirm`, before the question is
answered.

### A second question

Sometimes the job finds something that needs its own decision. After an
uninstall, the program may have left files that the install did not create.
An uninstaller lists them and asks whether to remove them too, and only the
uninstall knows what they are. `Then` is that question:

```go
var left []string
o.Job = func(ctx context.Context) error { left, err = uninstall(ctx); return err }
o.Then = &wizard.ConfirmStep{
	Action: "Remove them too", Destructive: true, Decline: "Keep them",
	Ask: func() (string, string, bool) {
		return fmt.Sprintf("%d files are left.", len(left)), list(left), len(left) > 0
	},
	Job:  func(ctx context.Context) error { return remove(ctx, left) },
	Done: "Removed, with the files it made.",
}
```

After the job succeeds, `Ask` runs on the UI thread and returns the
question and its Markdown detail; `false` shows the plain result instead.
The question and detail replace the first ones, the message line keeps the
first job's `Done`, and the buttons read `Decline` and `Action`. The labels
are fields, not part of `Ask`, so the buttons are wide enough for them from
the start and nothing moves.

The first job's work is done by the time the step is asked. So `Decline`,
the title bar's close, and a step cancelled while it runs all end with
Finished. A step whose job fails ends with Failed. `SkipQuestion` skips only
the first question: the step is a separate decision.

`examples/installer-wizard -confirm -leftovers` shows it.
