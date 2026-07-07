package room

import kickpb "goserveractorfsm/internal/protocol/pb"

type SessionRef struct {
	SessionID uint64
	Outbound  chan<- []byte
}

type JoinSession struct {
	Session SessionRef
	Reply   chan<- JoinSessionReply
}

type JoinSessionReply struct {
	PlayerID    uint64
	AttackID    uint64
	AttackEpoch uint64
	SeatIndex   int
	RoomSize    int
	Err         error
}

type LeaveSession struct {
	SessionID uint64
}

type ClientEnvelope struct {
	SessionID uint64
	Envelope  *kickpb.Envelope
}

type joinAttack struct {
	Session  SessionRef
	PlayerID uint64
	Reply    chan<- joinAttackReply
}

type joinAttackReply struct {
	PlayerID    uint64
	AttackID    uint64
	AttackEpoch uint64
	SeatIndex   int
	RoomSize    int
	Err         error
}

type leaveAttack struct {
	SessionID uint64
}

type attackEnvelope struct {
	SessionID uint64
	Envelope  *kickpb.Envelope
}
