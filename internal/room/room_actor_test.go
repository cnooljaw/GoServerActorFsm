package room

import (
	"context"
	"testing"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/gamelogic"
)

func TestRoomActorGroupsPlayersByConfiguredAttackSize(t *testing.T) {
	roomActor := NewRoomActor(Config{
		RoomSize:  3,
		HoleCount: 9,
		Timing:    gamelogic.ShrewTiming{WaitMS: 1000, UpMS: 300, StandMS: 2000, DownMS: 300, DizzyMS: 500},
		NowMS:     func() int64 { return 10_000 },
	})

	replies := make([]JoinSessionReply, 0, 4)
	for sessionID := uint64(1); sessionID <= 4; sessionID++ {
		reply := make(chan JoinSessionReply, 1)
		roomActor.Handle(context.Background(), JoinSession{
			Session: SessionRef{SessionID: sessionID, Outbound: make(chan []byte, 8)},
			Reply:   reply,
		})
		got := <-reply
		if got.Err != nil {
			t.Fatalf("JoinSession(%d) Err = %v", sessionID, got.Err)
		}
		replies = append(replies, got)
	}

	if replies[0].AttackID != replies[1].AttackID || replies[1].AttackID != replies[2].AttackID {
		t.Fatalf("first 3 AttackID = %d,%d,%d, want same", replies[0].AttackID, replies[1].AttackID, replies[2].AttackID)
	}
	if replies[3].AttackID == replies[0].AttackID {
		t.Fatalf("4th AttackID = %d, want new attack", replies[3].AttackID)
	}
	for i, wantSeat := range []int{1, 2, 3, 1} {
		if replies[i].SeatIndex != wantSeat {
			t.Fatalf("reply[%d].SeatIndex = %d, want %d", i, replies[i].SeatIndex, wantSeat)
		}
	}
}

func TestRoomActorRoutesClientEnvelopeToAssignedAttack(t *testing.T) {
	roomRef := actor.Start(NewRoomActor(Config{
		RoomSize:  3,
		HoleCount: 9,
		Timing:    gamelogic.ShrewTiming{WaitMS: 1000, UpMS: 300, StandMS: 2000, DownMS: 300, DizzyMS: 500},
		NowMS:     func() int64 { return 10_000 },
	}))
	defer roomRef.Stop()

	reply := make(chan JoinSessionReply, 1)
	if err := roomRef.Tell(JoinSession{
		Session: SessionRef{SessionID: 1, Outbound: make(chan []byte, 8)},
		Reply:   reply,
	}); err != nil {
		t.Fatalf("Tell JoinSession error = %v", err)
	}
	joined := <-reply
	if joined.Err != nil {
		t.Fatalf("JoinSession Err = %v", joined.Err)
	}

	if joined.AttackID == 0 || joined.PlayerID == 0 || joined.AttackEpoch == 0 {
		t.Fatalf("JoinSessionReply = %+v, want non-zero ids", joined)
	}
}
