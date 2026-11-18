# Spec 084: form fields made before the app

**Issue**: [#248](https://github.com/ushineko/fynedesygn/issues/248)

## Status: COMPLETE

## Executive Summary

`forms.Entry` sets its placeholder through the field rather than
`SetPlaceHolder`, so it can be made before there is an app. A test makes
every `forms` field and every `wizard` page with no current app. A reviewer
should look first at the test, `wizard/beforeapp_test.go`, and at the list
of constructors it covers.

## Context

`wizard.Page` says pages are made before `Run` creates the app, so a page's
constructor holds data and only `Build` touches widgets. `wizard.Form`
takes a `*forms.Form` that its caller has already built, so the form's
fields are made before the app too.

`forms.Entry` called `SetPlaceHolder`, which measures the text through the
current app's driver. With no app, it stops with a nil pointer in
`fyne.MeasureText`. fynstall's installer stopped this way before its window
whenever its config had a parameter on the Settings page
(ushineko/fynstall#52). Every wizard test has the test app behind it, so no
test caught it.

## Requirements

- R1 `forms.Entry` can be made with no current app.
- R2 A test makes every `forms` field constructor and every `wizard` page
  constructor with no current app, and fails if one panics.
- R3 A changelog line.

## Acceptance Criteria

- [x] `forms.Entry` sets `PlaceHolder` directly; its placeholder is drawn
  when the entry is shown, and the existing form tests pass.
- [x] `TestPagesAndFieldsAreMadeBeforeThereIsAnApp` covers `Entry`,
  `Select`, `Check`, `Numeric`, `NumericEntry`, `Path`, `New`, `Welcome`,
  `Licence`, `Form`, `Directory`, `Summary`, `Progress` and `Finish`.
- [x] The test fails with `SetPlaceHolder` put back.
- [x] A changelog entry under Unreleased.

## Risks & Assumptions

- Setting the field skips the refresh that `SetPlaceHolder` does. Before
  the entry is shown there is nothing to refresh: the entry draws its
  placeholder when it is first rendered.
- The test sets the current app to nil for its duration and puts it back,
  so it does not affect the tests around it. The wizard tests do not run in
  parallel.
- `Finish` takes checks made by its caller. A caller that ticks one with
  `SetChecked` before the app exists has the same fault in its own code,
  outside this library (fynstall fixed its own, ushineko/fynstall#51).
- **Rollback**: a one-line change and a test; revert.

## E2E Test Plan

Through fynstall (ushineko/fynstall#52): `examples/greet`, whose config has
parameters on the Settings page, built against this release.

1. `WAYLAND_DISPLAY=wayland-none DISPLAY= ./greet-0.1.0-linux-amd64-installer --gui`
   stops with a message that there is no display, not a panic (R1).
2. On a desktop, the wizard opens and the Settings page shows its fields,
   with their placeholders (R1).

### Prod visibility

Not applicable: a library.

## Verification

- `make test` and `make lint` pass. `make vuln` (govulncheck): the code is
  affected by 0 vulnerabilities.
- Falsified: with `SetPlaceHolder` put back, the new test fails with the
  panic at `forms/forms.go:63`.
- The placeholder is drawn: an entry from `forms.Entry` shown in a test
  window has a visible `canvas.Text` with its placeholder. This was a
  one-off check, not a committed test; the gallery was not opened.
- The E2E steps run in fynstall once this is released.
