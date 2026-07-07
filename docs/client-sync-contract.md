# LayaEcsDemo 客户端同步契约

这份文档只写客户端接入需要遵守的协议和行为。客户端代码在 `../LayaEcsDemo` 修改，服务端代码以本仓库 `.proto` 为准。

## 1. 总原则

- 地鼠产生由服务端决定，客户端不再随机地鼠类型、等待时间和状态间隔。
- 客户端只负责表现：按服务端给的绝对时间播放 Wait、Up、Stand、Down、Dizzy。
- 请求和回包继续用 `Envelope.seq_id` 匹配。
- 服务端主动推送使用 `Envelope.seq_id = 0`。
- 客户端必须丢弃旧的 `attack_epoch` 或旧的 `timeline_rev`。

## 2. 进入游戏

连接 WebSocket 后，客户端发送：

```text
msg_id = JoinRoomReqID(1001)
payload = JoinRoomRequest
```

服务端返回：

```text
msg_id = JoinRoomRespID(1002)
payload = JoinRoomResponse
```

客户端需要保存：

- `player_id`
- `attack_id`
- `attack_epoch`
- `seat_index`
- `room_size`
- `snapshot.server_time_ms`
- `snapshot.timeline_rev`
- `snapshot.active_cycles`

`JoinRoomResponse.snapshot` 是第一份服务端权威地鼠时间线。

## 3. 时间同步

客户端发送：

```text
msg_id = TimeSyncReqID(1005)
payload = TimeSyncRequest{client_send_ms}
```

服务端返回：

```text
msg_id = TimeSyncRespID(1006)
payload = TimeSyncResponse{client_send_ms, server_time_ms}
```

第一版客户端使用简单 offset：

```text
server_offset_ms = server_time_ms - Date.now()
server_now_ms = Date.now() + server_offset_ms
```

先不做 RTT/2 延迟补偿。后续要优化时再用 `client_send_ms` 计算往返延迟。

## 4. 渲染地鼠时间线

客户端从 `GameSnapshot.active_cycles` 读取每个洞位当前周期：

```text
hole_index
spawn_seq
shrew_type
protect_type
hp
wait_start_ms
up_start_ms
stand_start_ms
down_start_ms
end_ms
```

客户端用 `server_now_ms` 判断当前阶段：

```text
server_now < up_start_ms       -> Wait
up_start_ms ~ stand_start_ms   -> Up
stand_start_ms ~ down_start_ms -> Stand，可点击
down_start_ms ~ end_ms         -> Down
server_now >= end_ms           -> 等待新 snapshot/timeline
```

第一版服务端已经支持 snapshot 和命中后的 `ShrewStatePush`。`ShrewTimelinePush` 协议已经定义，后续服务端加定时推进时，客户端按同一套字段更新本地时间线。

## 5. 击打请求

客户端点击地鼠时，必须发送当前洞位对应的 `spawn_seq`：

```proto
message KickRequest {
  uint64 attack_epoch = 1;
  int32 hammer_type = 2;
  bool kick_shrew = 3;
  int32 num_of_shrew = 4;
  repeated KickShrew shrews = 5;
  int32 combo_id = 6;
}

message KickShrew {
  int32 shrew_index = 1;
  int32 protect_type = 2;
  uint64 spawn_seq = 3;
}
```

服务端会校验：

- `attack_epoch` 是否等于当前组。
- `shrew_index` 是否存在。
- `spawn_seq` 是否等于该洞位当前周期。
- 当前服务端时间是否处于 Stand 可点击窗口。

校验失败返回 `ErrorResponse`。校验成功返回 `KickResponse`。

## 6. 服务端状态推送

命中成功后，服务端会向同组客户端广播：

```text
msg_id = ShrewStatePushID(3002)
seq_id = 0
payload = ShrewStatePush
```

客户端收到后，用它修正本地表现：

- `attack_epoch` 不一致：丢弃。
- `timeline_rev` 小于本地：丢弃。
- `hole_index + spawn_seq` 定位地鼠。
- `action_state = Dizzy` 时播放 Dizzy。
- `hp` 覆盖本地 hp。
- `clickable` 覆盖本地可点击状态。

## 7. 客户端需要修改的位置

建议客户端按这个顺序改：

1. `src/network/ProtocolTypes.ts`：补新 MsgID、新 message 类型、`spawnSeq`。
2. `src/network/KickProtoCodec.ts`：补 JoinRoom、GameSnapshot、TimeSync、ShrewStatePush 编解码。
3. `src/network/KickSocket.ts`：区分 request/response 和 `seq_id=0` push。
4. `src/game/features/shrew/ShrewComponents.ts`：给地鼠保存 `spawnSeq`、服务端阶段时间。
5. `src/game/features/shrew/ShrewStateSystem.ts`：停止本地随机，改为按 `server_now_ms` 推导状态。
6. `src/game/session/KickRequestMapper.ts`：点击时带上当前洞位的 `spawnSeq` 和 `attackEpoch`。

