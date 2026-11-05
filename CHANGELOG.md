# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added

- Deliver IR-01 MVP (identity authentication and article management)
- Add migrations, test fixtures and docs

### Fixed

- Reclaim non-retained test fixtures via `t.TempDir()` (`ISSUE-identity-auth-0002`)
- Default S4.14 recovery fixtures to the test temp dir; retain only on an explicit evidence root (ADR-0039)

### Documentation

- Close IR-01 tasks and record scope decisions
- Record IR-01 delivery state
- Correct IR-01 push status
- Correct IR-01 push blocker cause (corporate proxy upload limit, not credentials)
- Add project README
- Correct mis-pasted evidence status lines in task cards
- Add ADR-0039: test temp and evidence directory policy
- Record `ISSUE-identity-auth-0002` partial and root-cause fixes
- Verify `ISSUE-article-management-0001` evidence integrity
- Record self-review of the test-scaffold fix
- Mark dead `%TEMP%` evidence links in S9.2 and S1.5

### Tests

- Record integration suite run map (12 packages pass; 5 blocked by missing dedicated instances)
- Verify the offline bundle clones, builds and tests clean
- Record `-race` unavailability (no C toolchain) and static check results

