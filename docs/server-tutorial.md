# GoServerActorFsm 教程

这份文档面向学习和跟做。目标不是一次写出大型商业服务器，而是用最少概念搭出一个能和 `LayaEcsDemo` 通信的 Go 游戏服务器。

## 1. 这个服务器做什么

客户端是 `../LayaEcsDemo`，玩法是打地鼠。现在地鼠产生、地鼠状态时间、命中校验都由服务端决定，客户端只按服务端时间线表现动画。

当前核心链路：

```text
LayaEcsDemo WebSocket
  -> JoinRoomRequest
  -> RoomActor 分配 AttackActor
  -> AttackActor 返回 GameSnapshot
  -> 客户端按服务端时间线播放地鼠
  -> KickRequest{attack_epoch, spawn_seq}
  -> AttackActor 校验时间线
  -> PlayerActor 更新个人状态
  -> KickResponse + ShrewStatePush
```

## 2. 为什么分成五层

```text
WebSocket -> Protocol -> Actor -> FSM -> GameLogic
```

每层只学一个问题。

- `WebSocket`：连接怎么进来，消息怎么读写。
- `Protocol`：字节怎么变成 typed request，response 怎么编码回字节。
- `Actor`：一个玩家的消息如何排队串行处理，避免到处加锁。
- `FSM`：玩家或房间当前处于什么流程状态，什么事件能让它进入下一个状态。
- `GameLogic`：给定输入和当前数据，怎么算出结果。

这样拆的好处是教学清楚，测试也清楚。`GameLogic` 不需要启动服务器就能测；`FSM` 不需要真实 socket 就能测；`Actor` 不需要真实玩法规则也能测消息顺序。

## 3. 轻量自研 Actor 模式

这里不使用 Hollywood。教学项目只需要一个最小 Actor：

```text
ActorRef
  -> mailbox chan Message
  -> goroutine 循环读取
  -> Handle(ctx, msg)
  -> Stop 关闭生命周期
```

核心规则：

- 每个 Actor 自己拥有状态。
- 外部不能直接改 Actor 内部字段，只能发消息。
- 同一个 Actor 的消息按 mailbox 顺序串行处理。
- Actor 之间通过 `ActorRef.Tell(msg)` 通信。
- 先实现 `Tell`，后续需要请求-响应时再加 `Ask`。

当前 Actor：

- WebSocket Session：不是业务 Actor，只做连接、读 loop、写 loop。
- `RoomActor`：大的接入入口，负责把连接分配给某个组。
- `AttackActor`：几个人一组的游戏 owner，默认 3 人一组。
- `PlayerActor`：保存单个玩家状态，处理个人奖励、怒气、锤子等。

分层关系：

```text
RoomActor
  -> AttackActor(group 1)
      -> PlayerActor(player 1)
      -> PlayerActor(player 2)
      -> PlayerActor(player 3)
  -> AttackActor(group 2)
      -> PlayerActor(...)
```

`AttackActor` 持有本组唯一的 `ShrewTimeline`。同一个组内的多个客户端看到同一份 `attack_epoch`、`timeline_rev` 和洞位时间线。

## 4. FSM 管流程，不管规则

FSM 只回答两个问题：

- 当前状态是什么？
- 收到事件后能不能迁移到下一个状态？

第一阶段玩家 FSM 可以很小：

```text
Disconnected
  -> Connected
  -> InGame
  -> Disconnected
```

击打流程可以是：

```text
InGame + KickReceived -> InGame
```

不要把金币、奖励、连击计算写进 FSM。FSM 只判断当前状态能不能处理 `KickReceived`。

## 5. GameLogic 管规则

`GameLogic` 应该像普通函数一样容易测试：

```text
输入：玩家状态、锤子类型、命中的地鼠、comboId、spawnSeq
输出：金币、怒气、体力、分数、combo、地鼠奖励明细
```

它不应该知道：

- WebSocket 是什么。
- protobuf 是什么。
- Actor mailbox 是什么。
- goroutine 是什么。

## 6. protobuf 协议

客户端当前 TypeScript JSON 协议在：

```text
../LayaEcsDemo/src/network/ProtocolTypes.ts
```

协议源是：

```text
api/proto/kick.proto
```

当前核心消息：

- `Envelope`
- `JoinRoomRequest`
- `JoinRoomResponse`
- `GameSnapshotRequest`
- `GameSnapshotResponse`
- `TimeSyncRequest`
- `TimeSyncResponse`
- `KickRequest`
- `KickResponse`
- `ShrewTimelinePush`
- `ShrewStatePush`
- `ErrorResponse`
- `Ping`
- `Pong`

迁移原则：

- `.proto` 是唯一协议源。
- Go 和 TypeScript 都从 `.proto` 生成。
- 生成文件不要手改。
- `Envelope.seq_id` 是请求和回包匹配字段。
- `Envelope.msg_id` 是协议路由字段。
- `KickRequest.attack_epoch` 防止旧局请求打到新局。
- `KickShrew.spawn_seq` 防止旧地鼠点击误打到新一轮地鼠。
- 服务端 push 使用 `seq_id = 0`。

## 7. TDD 怎么落地

代码顺序固定：

1. 先写 `internal/gamelogic` 测试，验证奖励计算。
2. 再写 `internal/fsm` 测试，验证非法状态不能处理击打。
3. 再写 `internal/actor` 测试，验证同一 Actor 串行处理消息。
4. 再写 `internal/protocol` 测试，验证 protobuf 编解码。
5. 再写 `internal/room` 测试，验证分组、快照、命中广播。
6. 最后写 `internal/ws` 集成测试或手动联调。

不要从 WebSocket 开始写。网络最难调，规则最容易测，先把最容易测的部分打牢。

## 8. 推荐目录

```text
api/proto/              .proto 协议源
cmd/server/             服务入口
internal/protocol/      protobuf 编解码和 cmd 路由
internal/ws/            WebSocket 连接和读写循环
internal/actor/         轻量自研 Actor runtime
internal/fsm/           自研 FSM
internal/gamelogic/     纯游戏规则
internal/game/          玩家 Actor 和玩家局内状态
internal/room/          RoomActor、AttackActor、分组和广播
internal/config/        游戏配置
docs/                   教程、计划、约束
```

## 9. 第一阶段完成标准

- 本地 Go server 能启动。
- 客户端能通过 WebSocket 发送 `JoinRoomRequest`。
- 服务端能返回带 snapshot 的 `JoinRoomResponse`。
- 客户端能按 snapshot 的 `spawn_seq` 发送一次击打。
- 服务端能返回 protobuf `KickResponse` 并广播 `ShrewStatePush`。
- `GameLogic`、`FSM`、`Actor`、`Room`、`WebSocket` 都有测试覆盖。
