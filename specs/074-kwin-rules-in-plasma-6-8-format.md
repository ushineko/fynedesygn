# Spec 074: KWin rules in Plasma 6.8's format

**Issue**: [#219](https://github.com/ushineko/fynedesygn/issues/219)

## Status: COMPLETE

## Executive Summary

`glance/kwin` reads and writes the rule list in the format the file is in:
`Order` on KWin 6.8, `rules`/`count` before it. It no longer writes `rules=1`
into a 6.8 file, which KWin read as the only rule to keep, deleting the rest.
A new rule is numbered past every group in the file, and `NoBorder` is
`decorationpolicy=none` in the 6.8 format. Reviewers should look at
`readList` in `glance/kwin/rule.go` first, then `loadLikeKWin68` in
`plasma68_test.go`, which is the copy of KWin's load the tests depend on.

## Context

KWin 6.8 lists window rules in `[General] Order=`. `rules=` and `count=` are
kept only to clean old files (`src/rulebooksettingsbase.kcfg`: "Deprecated.
Kept to clean the existing configs"). `RuleBookSettings::usrRead`
(`src/rulebooksettings.cpp`, v6.8.0) does this on every load:

1. A non-empty `rules=` is an old file to migrate: every rule group it does
   not list is deleted, and `Order` becomes `rules`.
2. Names in `Order` with no group are dropped; groups not in `Order` are
   prepended to it.
3. A rule with `noborderrule` set gets `decorationpolicy` (`none` for
   `noborder=true`) and `decorationpolicyrule` instead.
4. The file is saved when anything changed, without `rules=` or `count=`.

`glance/kwin` reads and writes only `rules=` and `count=`. On a file KWin 6.8
has saved, `sectionFor` finds nothing, `appendRule` sees an empty list and
names the new rule `1`, `ensure("1")` reuses whatever `[1]` is already there,
and `rules=1` is written next to `Order=`. On reconfigure step 1 deletes every
other group. Seen on Plasma 6.8.0: a file of 16 rules came down to one, and
ototo and hayami overwrote each other's rule in `[1]` in turn. In both
programs, Lookup reports the rule as absent after KWin rewrites the file,
so a settings checkbox unticks itself.

## Requirements

- R1 The rule list is read from `Order` when the file has it, and from
  `rules` otherwise.
- R2 A file in the 6.8 format is written back in the 6.8 format: the list goes
  to `Order`, and `rules=` and `count=` are never written into it. A 6.8 file
  that already carries a `rules=` (left by an earlier version of this package)
  has that list folded into `Order` and the two legacy keys removed, so KWin
  no longer reads it as a list of the only rules to keep.
- R3 A file is in the 6.8 format when it has `Order`, or when it has neither
  list key and holds rule groups (only 6.8 leaves rules unlisted). A file
  with `rules` and no `Order`, or with no rules at all, keeps the legacy keys:
  KWin 6.7 reads only those, and 6.8 migrates a complete list without loss.
- R4 A new rule's group name is a number that no group in the file uses,
  listed or not. Install never writes into a group whose `wmclass` and `title`
  are not the rule's own.
- R5 In the 6.8 format, `NoBorder` is written as `decorationpolicy=none`,
  `decorationpolicyrule=2`; in the legacy format as `noborder`. Either form
  reads back as `NoBorder`, and turning it off removes both.
- R6 Remove updates the same list key the file uses (R2, R3).

## Acceptance Criteria

- [x] R1, R4: Lookup finds a rule in a 6.8 file, and Install updates it in
      place instead of adding a second (`TestInstallingIntoAPlasma68FileKeepsEveryRuleAfterKWinLoadsIt`, `TestTwoProgramsInstallingInTurnBothKeepTheirRule` (one rule per program after four installs)).
- [x] R2, R4: installing into a 6.8 file with several rules, then running
      KWin 6.8's load over the result, leaves every rule in place
      (falsified against the current `appendRule`) (`TestInstallingIntoAPlasma68FileKeepsEveryRuleAfterKWinLoadsIt`; fails against the previous `rule.go`).
- [x] R2: a 6.8 file carrying a stray `rules=1` comes back from Install with
      no `rules=`/`count=` and every group still listed in `Order` (`TestAStrayLegacyListInAPlasma68FileIsFoldedIntoOrder`).
- [x] R4: two programs installing alternately into a 6.8 file, with KWin's
      load between each step, both keep their rule (`TestTwoProgramsInstallingInTurnBothKeepTheirRule`, `TestUnlistedGroupsAreRulesTooAndAreNotWrittenOver`).
- [x] R3: a legacy file stays legacy (the existing byte-identical round trip
      still passes); an empty file gets `rules=`/`count=` (`TestRewritingAFileWithoutChangingItIsByteIdentical`, `TestAnEmptyFileGetsTheListEveryKWinReads`, `TestRuleGroupsWithNoListAreAPlasma68File`).
- [x] R5: `decorationpolicy=none` reads as `NoBorder`; a 6.8 install writes
      `decorationpolicy`, not `noborder` (`TestAPlasma68InstallSaysNoBorderAsADecorationPolicy`, `TestALegacyRuleIsStillFoundAfterKWin68MigratesIt`).
- [x] R6: Remove from a 6.8 file updates `Order`, writes no `rules=`, and
      leaves the other rules (`TestRemovingFromAPlasma68FileUpdatesOrder`).
- [x] `go test ./...` and the linter pass (`make lint`: 0 issues (golangci-lint v2.12.2); `govulncheck`: none).

## Risks & Assumptions

- KWin's load is reproduced in a test helper from the v6.8.0 source, not run.
  If KWin changes `usrRead` again, the helper is what goes stale. The real
  desk check is in the E2E plan.
- Plasma 6.7 is not on the desk. On 6.7 a file has only `rules=`, so this
  package writes what it wrote before (R3).
- Rules already deleted are not recovered by this change; they come back from
  the user's own copy of the file.
- Rollback: revert the commit; consumers pin a version.

## E2E Test Plan

- Environment: Plasma 6.8 on Wayland, a `kwinrulesrc` KWin 6.8 has saved
  (`Order=` present) holding several rules, ototo and hayami built against
  this change.
- Steps: install ototo's indicator rule and hayami's rule from their settings,
  in turn, twice each; after each step run `qdbus6 org.kde.KWin /KWin
  reconfigure` and wait for KWin to save the file.
- Expected: the file keeps every rule it had plus the two programs' rules;
  neither program's window gets a titlebar back; each settings checkbox stays
  ticked. Covers AC 2, 4, 5 on a real KWin.

### Result (2026-10-21)

- Desk: Plasma 6.8.0 on Wayland, CachyOS. `kwinrulesrc` was restored from a
  copy holding 16 rules, which KWin 6.8 then saved with `Order=` and no
  `rules=`. ototo 0.1.5 and hayami 0.11.3 were installed, both built against
  0.1.107.
- Each program's rule checkbox was used from its settings. The operator
  reported both checkboxes staying ticked, and both windows staying frameless.
- The file afterwards: `Order=` names all 16 groups, there are 16
  `Description=` lines, no `rules=` or `count=`, and ototo's and hayami's rules
  each carry `decorationpolicy=none`. Covers AC 2, 4 and 5 on a real KWin.
- Not done as planned: the two programs were not alternated twice each with a
  reconfigure timed between, and the checking was by eye and by reading the
  file, not scripted.
