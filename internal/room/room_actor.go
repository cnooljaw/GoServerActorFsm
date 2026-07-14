package room

import (
	"context"
	"errors"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/gamelogic"
)

var ErrSessionNotJoined = errors.New("room: session not joined")

type Config struct {
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

type RoomActor struct {
	cfg          Config
	nextPlayerID uint64
	nextAttackID uint64
	current      *attackEntry
	attacks      map[uint64]*attackEntry
	sessions     map[uint64]*sessionEntry
}

type attackEntry struct {
	id    uint64
	ref   *actor.ActorRef
	count int
}

type sessionEntry struct {
	attackID uint64
	attack   *attackEntry
}

func NewRoomActor(cfg Config) *RoomActor {
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
	return &RoomActor{
		cfg:      cfg,
		attacks:  make(map[uint64]*attackEntry),
		sessions: make(map[uint64]*sessionEntry),
	}
}

func (r *RoomActor) Handle(ctx context.Context, msg actor.Message) {
	switch command := msg.(type) {
	case JoinSession:
		r.handleJoin(command)
	case LeaveSession:
		r.handleLeave(command)
	case ClientEnvelope:
		r.handleEnvelope(command)
	}
}

func (r *RoomActor) handleJoin(command JoinSession) {
	attack := r.pickAttack()
	r.nextPlayerID++

	reply := make(chan joinAttackReply, 1)
	if err := attack.ref.Tell(joinAttack{
		Session:  command.Session,
		PlayerID: r.nextPlayerID,
		Reply:    reply,
	}); err != nil {
		command.Reply <- JoinSessionReply{Err: err}
		return
	}

	joined := <-reply
	if joined.Err != nil {
		command.Reply <- JoinSessionReply{Err: joined.Err}
		return
	}

	attack.count++
	r.sessions[command.Session.SessionID] = &sessionEntry{
		attackID: attack.id,
		attack:   attack,
	}
	command.Reply <- JoinSessionReply{
		PlayerID:    joined.PlayerID,
		AttackID:    joined.AttackID,
		AttackEpoch: joined.AttackEpoch,
		SeatIndex:   joined.SeatIndex,
		RoomSize:    joined.RoomSize,
	}
}

func (r *RoomActor) handleLeave(command LeaveSession) {
	session, ok := r.sessions[command.SessionID]
	if !ok {
		return
	}
	_ = session.attack.ref.Tell(leaveAttack{SessionID: command.SessionID})
	if session.attack.count > 0 {
		session.attack.count--
	}
	delete(r.sessions, command.SessionID)
}

func (r *RoomActor) handleEnvelope(command ClientEnvelope) {
	session, ok := r.sessions[command.SessionID]
	if !ok {
		return
	}
	_ = session.attack.ref.Tell(attackEnvelope{
		SessionID: command.SessionID,
		Envelope:  command.Envelope,
	})
}

func (r *RoomActor) pickAttack() *attackEntry {
	if r.current != nil && r.current.count < r.cfg.RoomSize {
		return r.current
	}

	r.nextAttackID++
	attack := &attackEntry{
		id: r.nextAttackID,
		ref: actor.Start(NewAttackActor(AttackConfig{
			AttackID:            r.nextAttackID,
			RoomSize:            r.cfg.RoomSize,
			HoleCount:           r.cfg.HoleCount,
			MinPlayersToStart:   r.cfg.MinPlayersToStart,
			InitialActiveShrews: r.cfg.InitialActiveShrews,
			MaxActiveShrews:     r.cfg.MaxActiveShrews,
			InterSpawnMS:        r.cfg.InterSpawnMS,
			MapCycleMS:          r.cfg.MapCycleMS,
			Timing:              r.cfg.Timing,
			NowMS:               r.cfg.NowMS,
		})),
	}
	r.attacks[attack.id] = attack
	r.current = attack
	return attack
}
