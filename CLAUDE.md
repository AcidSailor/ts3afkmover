# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-purpose daemon that polls a TeamSpeak 3 server's **WebQuery HTTP API**
and moves idle clients into a configured "AFK" channel, notifying them via a
global message. Runs headless (no CLI subcommands despite the `cli`/`cobra`
copier answers — `main.go` is a plain ticker loop). Deployed as a container.

## Commands

The taskfile (`taskfile.yml`) is the entry point; it comes from the `go-scaffolds`
template (see the parent-workspace CLAUDE.md for fleet-wide tooling).

```bash
task run                    # go run .
task test                   # go test -race ./...
task lint                   # golangci-lint fmt + run --fix (mutates files)
task ci                     # read-only fmt --diff + run (what CI enforces)
task check                  # lint + test
task build                  # goreleaser snapshot build (all platforms)
task update                 # pull latest go-scaffolds v1 template tooling
```

Run a single test: `go test -race -run TestName ./internal/usecases/`.
There are currently **no `*_test.go` files** in the repo.

Releases are tag-triggered (`task release-tag` cuts the next `svu` tag → GoReleaser
CI). This repo's default branch is **`master`** (most fleet repos use `main`).

## Architecture

Three-layer flow, wired by hand in `main.go` (no DI framework):

- **`configs/config.go`** — all config comes from `TS3_*` env vars via
  `caarlos0/env` (see README table). `NewConfig` trims a trailing `/` off the URL.
  Time units matter: `IdleTime`/`IdleCheckInterval` are **minutes**,
  `RequestTimeout` is **seconds**.
- **`internal/controllers/ts3.go`** — `Ts3Client`, the HTTP layer. Three
  operations, each a self-contained POST to the WebQuery API: `ClientListWithTimes`
  (`/{vserver}/clientlist` with `-times`), `MoveClient` (`/{vserver}/clientmove`),
  `SendGM` (`/gm` global message). Auth is the `X-Api-Key` header. Every method
  **swallows errors by logging and returning a zero value / `false`** rather than
  propagating — the daemon must never crash on a transient TS3 hiccup.
- **`internal/usecases/idle.go`** — `Ts3IdleUsecase.MoveIdleClients` is the core
  loop body. WebQuery returns all fields as **strings**, so it `strconv.Atoi`s
  `client_type`, `clid`, `cid`, `client_idle_time`. Skip rules: query clients
  (`client_type != 0`) and clients already in the idle channel are left alone.
  A client is idle when `client_idle_time` (ms) exceeds `IdleTime` (min). Each run
  logs a `stats` struct (moved/notIdle/skipped/errored).

`main.go` drives it: a `time.Ticker` at `IdleCheckInterval` calls
`MoveIdleClients` per tick, in one goroutine, until SIGINT/SIGTERM.

## Gotchas

- **Errors don't propagate.** By design the controller returns `false`/zero
  instead of an `error`. When adding logic, follow that pattern (log + return a
  sentinel) rather than introducing `error` returns that `main.go` can't handle.
- **Ticker only, no immediate run.** The first sweep happens after one
  `IdleCheckInterval`, not at startup.
- **Copier answers lie about shape.** `.copier-answers.yml` says
  `project_type: cli` / `cli_framework: cobra`, but there's no cobra and no
  subcommands. `.air.toml` also points at a nonexistent `./cmd/main.go` (the real
  entry is `./main.go` at repo root). Don't trust those templated files.
- **`main.go`, `go.mod`, `README.md` are seed-once** in go-scaffolds — `task
  update` won't touch them, only tooling files propagate.

## Local integration testing

`test/docker-compose.test.yaml` spins up a real `teamspeak` server (+ MariaDB +
ts3-manager UI on :8080). WebQuery HTTP is on **:10080**. Point `TS3_URL` at
`http://127.0.0.1:10080` and grab an API key from the ts3 server admin to exercise
the daemon end-to-end. `test/query_ip_allowlist.txt` is mounted to allow query
access from private ranges.
