package room

import (
	"context"
	"math/rand"
	"time"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/game"
	"goserveractorfsm/internal/gamelogic"
	"goserveractorfsm/internal/protocol"
	kickpb "goserveractorfsm/internal/protocol/pb"

	"google.golang.org/protobuf/proto"
)

type AttackConfig struct {
	AttackID            uint64
	RoomSize            int
	HoleCount           int
	MinPlayersToStart   int
	InitialActiveShrews int
	MaxActiveShrews     int
	InterSpawnMS        int
	MapCycleMS          int
	Timing              gamelogic.ShrewTiming
	NowMS               func() int64
}

type AttackPhase int32

const (
	AttackPhaseFilling AttackPhase = 1
	AttackPhaseRunning AttackPhase = 2
)

type AttackActor struct {
	cfg         AttackConfig
	epoch       uint64
	phase       AttackPhase
	startAtMS   int64
	timeline    *gamelogic.ShrewTimeline
	mapTimeline *gamelogic.MapTimeline
	players     map[uint64]*attackPlayer
	sessionByID map[uint64]uint64
}

type attackPlayer struct {
	session SessionRef
	player  *actor.ActorRef
	seat    int
}

func NewAttackActor(cfg AttackConfig) *AttackActor {
	if cfg.RoomSize <= 0 {
		cfg.RoomSize = 3
	}
	if cfg.HoleCount <= 0 {
		cfg.HoleCount = 9
	}
	if cfg.MinPlayersToStart <= 0 {
		cfg.MinPlayersToStart = cfg.RoomSize
	}
	if cfg.InitialActiveShrews <= 0 {
		cfg.InitialActiveShrews = 1
	}
	if cfg.MaxActiveShrews <= 0 {
		cfg.MaxActiveShrews = 1
	}
	if cfg.InterSpawnMS <= 0 {
		cfg.InterSpawnMS = 800
	}
	if cfg.MapCycleMS <= 0 {
		cfg.MapCycleMS = 16_000
	}
	if cfg.Timing == (gamelogic.ShrewTiming{}) {
		cfg.Timing = gamelogic.DefaultShrewTiming()
	}
	if cfg.NowMS == nil {
		cfg.NowMS = unixMilli
	}
	return &AttackActor{
		cfg:         cfg,
		epoch:       1,
		phase:       AttackPhaseFilling,
		timeline:    gamelogic.NewShrewTimeline(cfg.HoleCount, cfg.Timing, cfg.MaxActiveShrews, int64(cfg.InterSpawnMS), rand.New(rand.NewSource(int64(cfg.AttackID)))),
		mapTimeline: gamelogic.NewMapTimeline(int64(cfg.MapCycleMS)),
		players:     make(map[uint64]*attackPlayer),
		sessionByID: make(map[uint64]uint64),
	}
}

func (a *AttackActor) Handle(ctx context.Context, msg actor.Message) {
	switch command := msg.(type) {
	case joinAttack:
		a.handleJoin(command)
	case leaveAttack:
		a.handleLeave(command)
	case attackEnvelope:
		a.handleEnvelope(command)
	}
}

func (a *AttackActor) handleJoin(command joinAttack) {
	seat := len(a.players) + 1
	player := actor.Start(game.NewPlayerActorWithID(command.PlayerID))
	a.players[command.PlayerID] = &attackPlayer{
		session: command.Session,
		player:  player,
		seat:    seat,
	}
	a.sessionByID[command.Session.SessionID] = command.PlayerID
	if a.phase == AttackPhaseFilling && len(a.players) >= a.cfg.MinPlayersToStart {
		a.startAtMS = a.cfg.NowMS() + int64(a.cfg.InterSpawnMS)
		a.phase = AttackPhaseRunning
		a.timeline.Start(a.startAtMS, a.cfg.InitialActiveShrews)
		a.mapTimeline.Start(a.startAtMS)
		a.broadcastTimeline(a.cfg.NowMS())
		a.broadcastMapState(a.cfg.NowMS())
	}
	command.Reply <- joinAttackReply{
		PlayerID:    command.PlayerID,
		AttackID:    a.cfg.AttackID,
		AttackEpoch: a.epoch,
		SeatIndex:   seat,
		RoomSize:    a.cfg.RoomSize,
	}
}

func (a *AttackActor) handleLeave(command leaveAttack) {
	playerID, ok := a.sessionByID[command.SessionID]
	if !ok {
		return
	}
	if player := a.players[playerID]; player != nil {
		player.player.Stop()
	}
	delete(a.players, playerID)
	delete(a.sessionByID, command.SessionID)
}

func (a *AttackActor) handleEnvelope(command attackEnvelope) {
	playerID, ok := a.sessionByID[command.SessionID]
	if !ok {
		return
	}
	player := a.players[playerID]
	if player == nil {
		return
	}

	a.advanceGameState(a.cfg.NowMS())

	switch protocol.MsgID(command.Envelope.GetMsgId()) {
	case protocol.JoinRoomReqID:
		a.send(player.session, protocol.JoinRoomRespID, command.Envelope.GetSeqId(), a.joinRoomResponse(playerID, player.seat))
	case protocol.GameSnapshotReqID:
		a.send(player.session, protocol.GameSnapshotRespID, command.Envelope.GetSeqId(), &kickpb.GameSnapshotResponse{
			Snapshot: a.snapshot(),
		})
	case protocol.TimeSyncReqID:
		req := &kickpb.TimeSyncRequest{}
		if err := protocol.Decode(command.Envelope.GetPayload(), req); err != nil {
			a.sendError(player.session, command.Envelope.GetSeqId(), err.Error())
			return
		}
		a.send(player.session, protocol.TimeSyncRespID, command.Envelope.GetSeqId(), &kickpb.TimeSyncResponse{
			ClientSendMs: req.GetClientSendMs(),
			ServerTimeMs: a.cfg.NowMS(),
		})
	case protocol.KickReqID:
		a.handleKick(player, command.Envelope)
	default:
		a.sendError(player.session, command.Envelope.GetSeqId(), "unknown command")
	}
}

func (a *AttackActor) handleKick(player *attackPlayer, envelope *kickpb.Envelope) {
	request := &kickpb.KickRequest{}
	if err := protocol.Decode(envelope.GetPayload(), request); err != nil {
		a.sendError(player.session, envelope.GetSeqId(), err.Error())
		return
	}
	if request.GetAttackEpoch() != a.epoch {
		a.sendError(player.session, envelope.GetSeqId(), "attack epoch mismatch")
		return
	}

	nowMS := a.cfg.NowMS()
	a.advanceGameState(nowMS)
	hitCycles := make([]gamelogic.ShrewCycle, 0, len(request.GetShrews()))
	for _, shrew := range request.GetShrews() {
		cycle, ok := a.timeline.ApplyHit(int(shrew.GetShrewIndex()), shrew.GetSpawnSeq(), nowMS)
		if !ok {
			a.sendError(player.session, envelope.GetSeqId(), "invalid shrew hit")
			return
		}
		hitCycles = append(hitCycles, cycle)
	}

	reply := make(chan game.KickReply, 1)
	if err := player.player.Tell(game.KickCommand{
		Input: toKickInput(request),
		Reply: reply,
	}); err != nil {
		a.sendError(player.session, envelope.GetSeqId(), err.Error())
		return
	}

	result := <-reply
	if result.Err != nil {
		a.sendError(player.session, envelope.GetSeqId(), result.Err.Error())
		return
	}

	a.send(player.session, protocol.KickRespID, envelope.GetSeqId(), toKickResponse(result.Result))
	// Timeline precedes the terminal state so a client that receives both can
	// apply the durable cycle first and retain the Dizzy override.
	a.broadcastTimeline(nowMS)
	for _, cycle := range hitCycles {
		a.broadcast(protocol.ShrewStatePushID, &kickpb.ShrewStatePush{
			ServerTimeMs: nowMS,
			AttackId:     a.cfg.AttackID,
			AttackEpoch:  a.epoch,
			TimelineRev:  a.timeline.Revision(),
			HoleIndex:    int32(cycle.HoleIndex),
			SpawnSeq:     cycle.SpawnSeq,
			ActionState:  int32(gamelogic.ShrewActionDizzy),
			PhaseStartMs: nowMS,
			PhaseEndMs:   nowMS + int64(a.timeline.Timing().DizzyMS),
			Hp:           int32(cycle.HP),
			Clickable:    false,
		})
	}
}

func (a *AttackActor) advanceGameState(nowMS int64) {
	if a.phase != AttackPhaseRunning {
		return
	}
	if a.timeline.Advance(nowMS) {
		a.broadcastTimeline(nowMS)
	}
	if a.mapTimeline.Advance(nowMS) {
		a.broadcastMapState(nowMS)
	}
}

func (a *AttackActor) broadcastTimeline(nowMS int64) {
	cycles := a.timeline.ActiveCycles(nowMS)
	pbCycles := make([]*kickpb.ShrewCycle, 0, len(cycles))
	for _, cycle := range cycles {
		pbCycles = append(pbCycles, toProtoCycle(cycle))
	}
	a.broadcast(protocol.ShrewTimelinePushID, &kickpb.ShrewTimelinePush{
		ServerTimeMs: nowMS,
		AttackId:     a.cfg.AttackID,
		AttackEpoch:  a.epoch,
		TimelineRev:  a.timeline.Revision(),
		Cycles:       pbCycles,
		RoomPhase:    int32(a.phase),
		PlayerCount:  int32(len(a.players)),
		RoomSize:     int32(a.cfg.RoomSize),
		StartAtMs:    a.startAtMS,
	})
}

func (a *AttackActor) broadcastMapState(nowMS int64) {
	a.broadcast(protocol.MapStatePushID, &kickpb.MapStatePush{
		ServerTimeMs: nowMS,
		AttackId:     a.cfg.AttackID,
		AttackEpoch:  a.epoch,
		Timeline:     toProtoMapTimeline(a.mapTimeline.Snapshot()),
	})
}

func (a *AttackActor) joinRoomResponse(playerID uint64, seat int) *kickpb.JoinRoomResponse {
	return &kickpb.JoinRoomResponse{
		PlayerId:    playerID,
		AttackId:    a.cfg.AttackID,
		AttackEpoch: a.epoch,
		SeatIndex:   int32(seat),
		RoomSize:    int32(a.cfg.RoomSize),
		Snapshot:    a.snapshot(),
	}
}

func (a *AttackActor) snapshot() *kickpb.GameSnapshot {
	nowMS := a.cfg.NowMS()
	a.advanceGameState(nowMS)
	cycles := a.timeline.ActiveCycles(nowMS)
	pbCycles := make([]*kickpb.ShrewCycle, 0, len(cycles))
	for _, cycle := range cycles {
		pbCycles = append(pbCycles, toProtoCycle(cycle))
	}
	timing := a.timeline.Timing()
	return &kickpb.GameSnapshot{
		ServerTimeMs: nowMS,
		AttackId:     a.cfg.AttackID,
		AttackEpoch:  a.epoch,
		TimelineRev:  a.timeline.Revision(),
		DefaultTiming: &kickpb.ShrewTiming{
			WaitMs:  int32(timing.WaitMS),
			UpMs:    int32(timing.UpMS),
			StandMs: int32(timing.StandMS),
			DownMs:  int32(timing.DownMS),
			DizzyMs: int32(timing.DizzyMS),
		},
		ActiveCycles: pbCycles,
		RoomPhase:    int32(a.phase),
		PlayerCount:  int32(len(a.players)),
		RoomSize:     int32(a.cfg.RoomSize),
		StartAtMs:    a.startAtMS,
		MapTimeline:  toProtoMapTimeline(a.mapTimeline.Snapshot()),
	}
}

func (a *AttackActor) send(session SessionRef, msgID protocol.MsgID, seqID uint32, payload proto.Message) {
	data, err := protocol.EncodeEnvelope(msgID, seqID, payload)
	if err != nil {
		return
	}
	session.Outbound <- data
}

func (a *AttackActor) sendError(session SessionRef, seqID uint32, message string) {
	a.send(session, protocol.ErrorRespID, seqID, &kickpb.ErrorResponse{
		Code:    -1,
		Message: message,
	})
}

func (a *AttackActor) broadcast(msgID protocol.MsgID, payload proto.Message) {
	data, err := protocol.EncodeEnvelope(msgID, 0, payload)
	if err != nil {
		return
	}
	for _, player := range a.players {
		player.session.Outbound <- data
	}
}

func toProtoCycle(cycle gamelogic.ShrewCycle) *kickpb.ShrewCycle {
	return &kickpb.ShrewCycle{
		HoleIndex:    int32(cycle.HoleIndex),
		SpawnSeq:     cycle.SpawnSeq,
		ShrewType:    int32(cycle.ShrewType),
		ProtectType:  int32(cycle.ProtectType),
		Hp:           int32(cycle.HP),
		WaitStartMs:  cycle.WaitStartMS,
		UpStartMs:    cycle.UpStartMS,
		StandStartMs: cycle.StandStartMS,
		DownStartMs:  cycle.DownStartMS,
		EndMs:        cycle.EndMS,
	}
}

func toProtoMapTimeline(state gamelogic.MapState) *kickpb.MapTimeline {
	return &kickpb.MapTimeline{
		CurrentMap:   state.CurrentMap,
		MapRevision:  state.Revision,
		MapStartedMs: state.MapStartedMS,
		NextSwitchMs: state.NextSwitchMS,
		NextMap:      state.NextMap,
		CycleMs:      state.CycleMS,
	}
}

func toKickInput(request *kickpb.KickRequest) gamelogic.KickInput {
	shrews := make([]gamelogic.KickShrew, 0, len(request.GetShrews()))
	for _, shrew := range request.GetShrews() {
		shrews = append(shrews, gamelogic.KickShrew{
			Index:       int(shrew.GetShrewIndex()),
			ProtectType: int(shrew.GetProtectType()),
			SpawnSeq:    shrew.GetSpawnSeq(),
		})
	}

	return gamelogic.KickInput{
		HammerType: int(request.GetHammerType()),
		ComboID:    int(request.GetComboId()),
		Shrews:     shrews,
	}
}

func toKickResponse(result gamelogic.KickResult) *kickpb.KickResponse {
	rewards := make([]*kickpb.ShrewReward, 0, len(result.ShrewRewards))
	for _, reward := range result.ShrewRewards {
		rewards = append(rewards, &kickpb.ShrewReward{
			ShrewIndex: int32(reward.ShrewIndex),
			Reward:     int32(reward.Reward),
		})
	}

	return &kickpb.KickResponse{
		Ret:        0,
		Money:      int32(result.Money),
		Angry:      int32(result.Angry),
		Power:      int32(result.Power),
		LevelScore: int32(result.LevelScore),
		HammerId:   int32(result.HammerID),
		NumOfShrew: int32(result.NumOfShrew),
		ShrewResp:  rewards,
		Combo:      int32(result.Combo),
		ComboId:    int32(result.ComboID),
	}
}

func unixMilli() int64 {
	return time.Now().UnixMilli()
}
