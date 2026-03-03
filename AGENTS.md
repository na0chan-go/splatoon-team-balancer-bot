# AGENTS.md

## Project Overview

This repository contains a Discord bot written in Go that automatically creates balanced teams for Splatoon 3 private matches.

Players join a room by declaring their X Power.
The bot selects 8 players (if more than 8 join) and splits them into two teams of 4 with the smallest possible difference in total X Power.

The algorithm guarantees an optimal solution using exhaustive search.

---

## Core Rules

### Team Balancing

- Maximum players: 10
- Players per match: 8
- Spectators: remaining players
- Team size: 4 vs 4

The matchmaking algorithm must:

- Minimize `abs(sum(teamA) - sum(teamB))`
- Search all possible combinations (10C8 × 8C4/2 worst case)

---

## Project Structure

cmd/bot/main.go
Entry point for the Discord bot.

internal/bot/
Discord command handling.

internal/app/usecase/
Command use cases and application workflow.

internal/adapter/
Discord/SQLite adapters.

internal/domain/
Core business logic such as matchmaking algorithms.

internal/domain/room/
Room state domain model.

internal/store/
State persistence (memory or database).

internal/util/
Formatting and helper functions.

---

## Development Rules

1. Prefer **small commits and minimal diffs**.
2. Business logic must live in `internal/domain`.
3. Discord-specific code must not contain matchmaking logic.
4. All core algorithms must have unit tests.
5. Avoid unnecessary dependencies.
6. Always create a feature branch for changes, then open a PR, perform a self-review, and merge after completion.

---

## Coding Guidelines

- Language: Go
- Use Go standard formatting (`go fmt`)
- Use idiomatic Go (simple structs, minimal abstraction)
- Prefer pure functions for algorithmic logic
- Avoid global state

---

## Testing Requirements

The matchmaking algorithm must include tests for:

- Exact 8-player balancing
- 10-player balancing with spectators
- Deterministic results when using the same random seed

Run tests using:

go test ./...

---

## Performance Expectations

Even in the worst case (10 players):

Total combinations ≈ 1575

This should complete within a few milliseconds.

Do not prematurely optimize the algorithm.

---

## AI Agent Instructions

When modifying the code:

1. Never change the algorithm goal (minimal team power difference).
2. Do not introduce complex abstractions.
3. Keep the matchmaking algorithm readable.
4. Always update tests when modifying logic.
5. Ensure the project still builds and tests pass.

---

## Future Improvements

Potential extensions:

- Spectator rotation
- Prevent consecutive same-team assignments
- Discord message embeds for better UI
- Persistent storage with SQLite
