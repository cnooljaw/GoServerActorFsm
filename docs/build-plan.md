# GoServerActorFsm Build Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a minimal Go WebSocket game server for `../LayaEcsDemo` using protobuf, a lightweight self-built Actor model, a self-built FSM, and TDD.

**Architecture:** WebSocket owns connections, Protocol owns protobuf encoding, Actor owns concurrent isolation, FSM owns process transitions, and GameLogic owns pure game rules. The first deliverable is one working `KickRequest -> KickResponse` loop.

**Tech Stack:** Go, protobuf, WebSocket, lightweight in-repo Actor runtime, in-repo FSM, Go `testing`.

---

## File Map

- `go.mod`: Go module definition.
- `api/proto/kick.proto`: shared first-stage protocol.
- `cmd/server/main.go`: server bootstrap.
- `internal/gamelogic/`: pure hit/reward rules.
- `internal/fsm/`: generic or small-purpose FSM implementation.
- `internal/actor/`: lightweight Actor runtime.
- `internal/protocol/`: protobuf envelope and codec.
- `internal/ws/`: WebSocket session handling.
- `internal/game/`: wires player/session use cases.
- `docs/server-tutorial.md`: learning-oriented architecture guide.

## Phase 1: Project Skeleton

- [x] Create Go module.
- [x] Add directories from the file map.
- [x] Add a minimal `cmd/server/main.go` that starts and exits cleanly.
- [x] Add `go test ./...` as the default verification command.

Expected result: project has a compilable empty server skeleton.

## Phase 2: Protocol First

- [x] Write `api/proto/kick.proto` from `../LayaEcsDemo/src/network/ProtocolTypes.ts`.
- [x] Add generation commands for Go.
- [ ] Add generation commands for TypeScript.
- [x] Add codec tests that encode and decode `KickRequest` and `KickResponse`.
- [x] Document that generated files are never edited manually.

Expected result: protobuf becomes the only protocol source.

## Phase 3: GameLogic TDD

- [x] Write a failing test for one successful hit returning reward and score.
- [x] Implement minimal `CalculateKickResult`.
- [x] Write a failing test for miss returning no shrew reward.
- [x] Write a failing test for combo id preservation.
- [x] Refactor names and data types only after tests are green.

Expected result: reward rules are pure and testable without server runtime.

## Phase 4: FSM TDD

- [x] Write a failing test for `Disconnected -> Connected`.
- [x] Write a failing test for `Connected -> InGame`.
- [x] Write a failing test that `Disconnected + KickReceived` is rejected.
- [x] Implement minimal FSM transition table.
- [x] Keep reward calculation outside FSM.

Expected result: process legality is separated from reward rules.

## Phase 5: Lightweight Actor TDD

- [x] Write a failing test proving one Actor processes messages in send order.
- [x] Implement `ActorRef`, `Message`, `Handler`, `Start`, `Tell`, and `Stop`.
- [x] Write a failing test proving `Stop` prevents later messages from being handled.
- [x] Add `PlayerActor` that receives kick messages and calls FSM + GameLogic.
- [x] Keep Actor runtime small; do not add supervision trees until needed.

Expected result: one player can process requests serially without locks in game state.

## Phase 6: WebSocket Integration

- [ ] Add WebSocket accept route.
- [ ] Create one `SessionActor` per connection.
- [ ] Decode protobuf packets into actor messages.
- [ ] Encode actor responses back to protobuf packets.
- [ ] Add heartbeat with `Ping` and `Pong`.
- [ ] Cleanly stop session actor on disconnect.

Expected result: a real client connection can complete one kick round trip.

## Phase 7: Client Alignment

- [ ] Update `LayaEcsDemo` network layer plan to replace JSON mock packets with protobuf packets.
- [ ] Preserve request-response matching behavior through authoritative `Envelope.seq_id`.
- [ ] Add a temporary compatibility note if JSON mock and protobuf server coexist during migration.
- [ ] Run one manual round trip: click client, observe `KickResponse`.

Expected result: server and client agree on protocol shape and request matching.

## Phase 8: Teaching Polish

- [ ] Add small diagrams for Actor, FSM, and request flow.
- [ ] Add one tutorial section per implemented package.
- [ ] Keep examples short and runnable.
- [ ] Update `AGENTS.md` only when the reading order or hard rules change.

Expected result: the project remains useful as both codebase and teaching material.

## Verification

- [x] Run `go test ./...`.
- [ ] Run protobuf generation and confirm no dirty generated drift after rerun.
- [ ] Start server locally.
- [ ] Complete one client-server kick round trip.

## Scope Guard

Do not add database, login, multi-room matchmaking, persistence, ranking, deployment scripts, or distributed actor behavior until the first protobuf kick round trip is working.
