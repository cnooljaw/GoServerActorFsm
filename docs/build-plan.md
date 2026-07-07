# GoServerActorFsm Build Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a minimal Go WebSocket game server for `../LayaEcsDemo` using protobuf, a lightweight self-built Actor model, a self-built FSM, service-authoritative shrew timelines, and TDD.

**Architecture:** WebSocket owns connections, Protocol owns protobuf encoding, `RoomActor` owns large access and grouping, `AttackActor` owns one group's shrew timeline, `PlayerActor` owns one player's state, FSM owns process transitions, and GameLogic owns pure game rules.

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
- `internal/room/`: RoomActor, AttackActor, group assignment, snapshot and broadcast.
- `docs/server-tutorial.md`: learning-oriented architecture guide.
- `docs/client-sync-contract.md`: client protocol and behavior contract.

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

- [x] Add WebSocket accept route.
- [x] Split WebSocket read loop and write loop.
- [x] Decode protobuf packets into actor messages.
- [x] Encode actor responses back to protobuf packets.
- [ ] Add heartbeat with `Ping` and `Pong`.
- [x] Notify RoomActor on disconnect.

Expected result: a real client connection can complete one kick round trip.

## Phase 7: Service-Authoritative Timeline

- [x] Add `JoinRoomReqID` and `JoinRoomRespID`.
- [x] Add `GameSnapshotReqID` and `GameSnapshotRespID`.
- [x] Add `TimeSyncReqID` and `TimeSyncRespID`.
- [x] Add `ShrewTimelinePushID` and `ShrewStatePushID` message definitions.
- [x] Add `attack_epoch` to `KickRequest`.
- [x] Add `spawn_seq` to `KickShrew`.
- [x] Add pure `ShrewTimeline` tests.
- [x] Add `RoomActor` grouping tests.
- [x] Add `AttackActor` snapshot, kick, and state push tests.
- [ ] Add periodic `ShrewTimelinePush` generation for future cycles.
- [ ] Add reconnect identity binding instead of treating every connection as a new player.

Expected result: clients can render server-provided timelines and server can reject stale hit requests.

## Phase 8: Client Alignment

- [x] Document client contract in `docs/client-sync-contract.md`.
- [ ] Update `LayaEcsDemo` network layer plan to consume GoServerActorFsm proto messages.
- [ ] Preserve request-response matching behavior through authoritative `Envelope.seq_id`.
- [ ] Route `seq_id = 0` push messages separately from pending request responses.
- [ ] Stop client-side random shrew spawning.
- [ ] Store `attack_epoch`, `timeline_rev`, and per-hole `spawn_seq` on the client.
- [ ] Add simple TimeSync offset calculation on the client.
- [ ] Run one manual round trip: click client, observe `KickResponse`.
- [ ] Run two or three clients and verify they share the same `attack_id` until room size is full.

Expected result: server and client agree on protocol shape and request matching.

## Phase 9: Teaching Polish

- [x] Add small diagrams for Actor, FSM, and request flow.
- [x] Add one tutorial section per implemented package.
- [ ] Keep examples short and runnable.
- [x] Update `AGENTS.md` only when the reading order or hard rules change.

Expected result: the project remains useful as both codebase and teaching material.

## Verification

- [x] Run `go test ./...`.
- [x] Run protobuf generation and confirm no dirty generated drift after rerun.
- [ ] Start server locally.
- [x] Complete one server-side WebSocket kick round trip in tests.
- [ ] Complete one real LayaEcsDemo client-server kick round trip.

## Scope Guard

Do not add database, login, persistence, ranking, deployment scripts, distributed actor behavior, or complex latency compensation until the first LayaEcsDemo client-server synchronized timeline round trip is working.
