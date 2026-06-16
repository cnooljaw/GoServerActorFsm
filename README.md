# GoServerActorFsm

`GoServerActorFsm` 是给兄弟目录 `../LayaEcsDemo` 配套的 Go 游戏服务器教学项目。

项目目标不是先做复杂框架，而是用最少代码把游戏服务器的核心边界讲清楚：

```text
WebSocket -> Protocol -> Actor -> FSM -> GameLogic -> Actor -> Protocol -> WebSocket
```

- `WebSocket` 管连接、收发、心跳、断线清理。
- `Protocol` 管 protobuf 编解码、消息号路由、协议兼容。
- `Actor` 管 mailbox、串行处理、生命周期、并发隔离。
- `FSM` 管玩家流程状态迁移。
- `GameLogic` 管纯规则，并保持可单测。

## 当前状态

当前已经完成最小可运行骨架：

- Go WebSocket 服务启动。
- protobuf `Envelope` 封包。
- `KickRequest -> KickResponse` 协议链路。
- 轻量自研 Actor。
- 玩家 FSM。
- 可单测的打地鼠规则。
- 简单日志模块，能看到连接、请求、回包和断线情况。

当前请求链路：

```text
Envelope(seq_id=N, msg_id=KickReqID, payload=KickRequest)
  -> WebSocket Session
  -> PlayerActor
  -> FSM
  -> GameLogic
  -> KickResponse
  -> Envelope(seq_id=N, msg_id=KickRespID, payload=KickResponse)
```

## 快速开始

运行测试：

```bash
go test ./...
```

启动服务器：

```bash
go run ./cmd/server
```

默认 WebSocket 地址：

```text
ws://127.0.0.1:9000/ws
```

默认配置写在 Go 代码里：

```text
internal/config/config.go
```

当前默认值：

```go
Root:   "./"
Thread: 2
Daemon: ""
Port:   9000
```

## 协议说明

协议文件：

```text
api/proto/kick.proto
```

生成 Go protobuf 代码：

```bash
sh scripts/gen-proto.sh
```

`Envelope` 是所有消息的传输层封包：

```proto
message Envelope {
  uint32 seq_id = 1;
  uint32 msg_id = 2;
  bytes payload = 3;
}
```

约定：

- `seq_id` 是请求和回包匹配的权威字段，客户端发什么，服务端就回什么。
- `msg_id` 是消息路由的权威字段。
- `payload` 只放业务 message 的 protobuf bytes。
- 业务 message 不再重复放 `seq_id`。

当前消息号定义在：

```text
internal/protocol/codec.go
```

当前已定义：

```go
PingReqID   = 1
PongRespID  = 2
KickReqID   = 2001
KickRespID  = 2002
ErrorRespID = 9001
```

## 目录结构

```text
api/proto/          protobuf 协议定义
cmd/server/         服务启动入口
docs/               教程、计划、技能地图
internal/actor/     轻量自研 Actor
internal/config/    Go 代码内默认配置
internal/fsm/       玩家状态机
internal/game/      玩家 Actor 和应用层命令
internal/gamelogic/ 纯游戏规则
internal/logx/      简单日志模块
internal/protocol/  协议编解码和消息号
internal/ws/        WebSocket 服务和会话
scripts/            工程脚本
```

## 开发规则

- 继续使用 TDD：先写失败测试，再写最小实现，再重构。
- 协议变更先改 `.proto`，再运行 `sh scripts/gen-proto.sh`。
- 生成代码不可手改。
- `GameLogic` 必须保持纯规则，不能依赖网络、Actor 或全局状态。
- WebSocket、Protocol、Actor、FSM、GameLogic 的职责不要混在一起。

## 文档入口

- `AGENTS.md`：Agent 极简入口和硬规则。
- `docs/server-tutorial.md`：教学版架构说明。
- `docs/build-plan.md`：项目建设计划。
- `docs/skill-map.md`：开发所需技能地图。
