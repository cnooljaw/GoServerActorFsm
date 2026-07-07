# AGENTS.md

本文件是 Agent 进入 `GoServerActorFsm` 后的极简入口，只放硬规则和索引。架构知识、教程和计划写到 `docs/`。

## 项目定位

- 为兄弟目录 `../LayaEcsDemo` 提供配套 Go 游戏服务器。
- 通信协议使用 Google Protocol Buffers；客户端当前 JSON 协议参考 `../LayaEcsDemo/src/network/ProtocolTypes.ts`。
- Actor 模式使用轻量自研实现，不使用 Hollywood。
- 项目同时服务代码实现和教学演示，文档必须简单、可读、可跟做。

## 强制边界

```text
WebSocket -> Protocol -> Actor -> FSM -> GameLogic -> Actor -> Protocol -> WebSocket
```

- `WebSocket` 只管连接、收发、心跳、断线清理。
- `Protocol` 只管 protobuf 编解码、cmd 路由、版本兼容。
- `Actor` 只管 mailbox、串行处理、生命周期、并发隔离。
- `FSM` 只管流程状态迁移。
- `GameLogic` 只管纯规则，必须可单测。

## 每次优先阅读

1. `AGENTS.md`
2. `docs/server-tutorial.md`：教学版架构说明。
3. `docs/build-plan.md`：项目建设计划。
4. `docs/skill-map.md`：当前可用技能和缺失技能。
5. `docs/client-sync-contract.md`：客户端同步协议契约。
6. 需要对齐客户端实现时再读 `../LayaEcsDemo/AGENTS.md`。

## 工作流

- 继续沿用 TDD：先写失败测试，再写最小实现，再重构。
- 文档改动不强行补测试；代码改动必须说明跑了哪些测试。
- 协议改动先改 `.proto`，生成代码不可手改。
- 查文件优先用 `rg` / `rg --files`。
- 不使用 `git reset --hard` 或 `git checkout -- <file>` 丢弃用户改动。

## 必要指令

- 生成 Go protobuf：`sh scripts/gen-proto.sh`
- 跑全部测试：`go test ./...`
- 启动服务器：`go run ./cmd/server`
- 默认 WebSocket 地址：`ws://127.0.0.1:9000/ws`
- 默认配置写在 Go 代码里：`internal/config/config.go`
