# Skill Map

本项目需要的技能分为“当前已可用”和“建议补齐”两类。

## 当前已可用

- `superpowers:test-driven-development`：代码开发必须按 RED -> GREEN -> REFACTOR。
- `superpowers:writing-plans`：把多步骤实现拆成可执行计划。
- `game-client-architect`：对齐 `LayaEcsDemo` 的客户端-服务端契约。
- `layaair-developer`：理解客户端 Laya 网络层、生命周期和协议接入点。

## 建议补齐

- Go 后端开发技能：Go module、目录约定、测试、错误处理、context、并发。
- 游戏服务器架构技能：连接、会话、玩家、房间、心跳、断线、重连、压测边界。
- 轻量 Actor 模式技能：mailbox、ActorRef、Tell、Stop、串行处理、生命周期。
- FSM 设计技能：状态、事件、迁移表、非法事件、流程和规则分离。
- Protocol Buffers 跨端协议技能：`.proto` 设计、版本兼容、Go/TypeScript 生成、字段演进。

## 当前决策

- 不使用 Hollywood。原因是教学项目优先可读、可解释、可测试，开源 Actor 框架会提前引入监督树、调度和运行时概念。
- 先实现最小 Actor runtime。等项目出现真实复杂度后，再评估是否需要引入成熟框架。
