# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-purpose daemon that polls a TeamSpeak 3 server's **WebQuery HTTP API**
and moves idle clients into a configured "AFK" channel, notifying them via a
global message. Runs headless (no CLI subcommands despite the `cli`/`cobra`
copier answers — `main.go` is a plain ticker loop). Deployed as a container.

## Commands

Commands live in `taskfile.yml` (`task --list`; from the `go-scaffolds` template —
see the parent-workspace CLAUDE.md for fleet-wide tooling). Two non-obvious ones:
`task lint` **mutates files** (golangci-lint `fmt` + `run --fix`), while `task ci`
is the read-only gate CI enforces. Releases are tag-triggered (`task release-tag`
cuts the next `svu` tag → GoReleaser CI). This repo's default branch is **`main`**
(renamed from `master` during the go-scaffolds fleet onboarding).

## Architecture

Three-layer flow, wired by hand in `main.go` (no DI framework). Packages are
domain-named so types drop the `Ts3` prefix (`ts3.Client`, `idle.Mover`).
Logging uses the **global** `slog` default (set once in `main.go` via
`slog.SetDefault`); nothing carries a `*slog.Logger`.

- **`config/config.go`** — package `config`; `config.Config` holds all `TS3_*`
  env vars via `caarlos0/env` (see README table). `config.New` trims a trailing
  `/` off the URL and validates the interval/timeouts. Time units matter:
  `IdleTime`/`IdleCheckInterval` are **minutes**, `RequestTimeout` is **seconds**;
  the `IdleThreshold()`/`TickInterval()`/`RequestTimeoutDuration()` helpers apply
  the unit in one place.
- **`internal/ts3/ts3.go`** — package `ts3`; `ts3.Client`, the HTTP layer, built
  on the fleet's **`restkit`** transport core. Three operations POST to the
  WebQuery API (`ClientListWithTimes`, `MoveClient`, `SendGM`); auth is the
  `X-Api-Key` header. Every method **swallows errors by logging and returning a
  zero value / `false`** rather than propagating — the daemon must never crash on
  a transient TS3 hiccup. A 2xx with a non-zero TS3 `status.code` is still a
  failure (checked via `ResponseStatus.OK()`).
- **`internal/idle/idle.go`** — package `idle`; `idle.Mover.MoveIdleClients` is
  the core loop body. It depends on the unexported `ts3Client` interface (not
  `*ts3.Client` directly) so the sweep is unit-testable against a fake. WebQuery
  returns all fields as **strings**, so it `strconv.Atoi`s `client_type`, `clid`,
  `cid`, `client_idle_time`. Skip rules: query clients (`client_type != 0`) and
  clients already in the idle channel are left alone. A client is idle when
  `client_idle_time` (ms) exceeds `IdleTime` (min). A failed client-list fetch
  skips the whole sweep (not treated as "nobody online"). Each run logs a `stats`
  struct (moved/notIdle/skipped/errored).

`main.go` drives it: a `time.Ticker` at `IdleCheckInterval` calls
`MoveIdleClients` per tick in the main goroutine, selecting against a
`signal.NotifyContext` that cancels on SIGINT/SIGTERM (which also aborts any
in-flight HTTP call).

## Gotchas

- **Errors don't propagate.** By design `ts3.Client` returns `false`/zero
  instead of an `error` (translating restkit's errors into a logged sentinel).
  When adding logic, follow that pattern (log + return a sentinel) rather than
  introducing `error` returns that `main.go` can't handle.
- **Ticker only, no immediate run.** The first sweep happens after one
  `IdleCheckInterval`, not at startup.
- **Copier answers lie about shape.** `.copier-answers.yml` says
  `project_type: cli` / `cli_framework: cobra`, but there's no cobra and no
  subcommands. Don't trust those templated files. (`.air.toml` used to point at a
  nonexistent `./cmd/main.go`; it's now fixed to build the root `.` package.)
- **`main.go`, `go.mod`, `README.md` are seed-once** in go-scaffolds — `task
  update` won't touch them, only tooling files propagate.

## Local integration testing

Running the daemon end-to-end against a real TS3 server: see the
`ts3-integration-test` skill (`.claude/skills/ts3-integration-test/SKILL.md`).
