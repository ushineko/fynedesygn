# Validation report: spec 005 examples and screenshots

Date: 2026-09-18. Milestone: the reference programs and the first captures.
Spec: `specs/005-examples-and-screenshots.md`.

## Scope

`forms.Saver`, the `steps` package, four example programs under `examples/`,
`tools/screenshot.sh` and `tools/crop.py`, ten gallery captures under
`docs/img/`, README Screenshots and Examples sections, Makefile
`build-examples`, `screenshots` and `check-diagrams` targets, CI building the
examples and checking both diagram sets.

## Phase 3: tests

`go test -race ./...` on Linux, no display: all packages pass.

| Package | Coverage |
|---|---|
| `forms` | 100 % |
| `steps` | 89.4 % |
| `examples/master-detail` | 90.2 % |
| `examples/job-runner` | 85.3 % |
| `examples/document-viewer` | 80.0 % |
| `examples/settings` | 73.2 % |

The settings example's test was rewritten once: the saver's one-second timer
fired during the test and drew a banner on its own goroutine, which under the
test driver races with the test measuring text (quirk 11). The test now
flushes the pending save before continuing. Module total 9,213 lines of Go.

`go vet`, `gofmt`, `shellcheck tools/screenshot.sh`: clean.

## Phase 4: code quality

- golangci-lint v2.12.2: 0 issues after wrapping two cancellation errors.
- Each example is one `main.go` with an `options()` function the test calls
  through `shell.Headless`, so the program and its test share one description
  of the window.
- The harness is angou's script made generic (class and binary as flags);
  the per-app demo-data setup that angou had is left to each program.

## Phase 5: security review

- `govulncheck -mode binary` on the built job-runner (the example importing
  the most packages): no vulnerabilities.
- The screenshot harness runs the program under a throwaway HOME and XDG
  directories so a capture cannot show the developer's data, and starts no
  subprocess beyond kdotool, spectacle and python3.
- The settings example writes only under the user's configuration directory
  when run as a program; its test writes to `t.TempDir()`.
- No credentials, no network. Secrets grep: clean.

## Phase 5.5: release safety

Nothing consumes the module yet. Rollback is `git revert`. The captures are
plain PNG files.

## Spec reconciliation

Eight of nine acceptance criteria verified; AC9's CI part is verified by the
run this commit triggers.

## Gaps

- The captures show Breeze Dark only; macOS Dark and Windows Light were
  captured and checked by eye but not committed, since the README set is
  meant to be one scheme.
- `Decide` in the gallery's Dialogs section was not exercised on a display
  in this session.
