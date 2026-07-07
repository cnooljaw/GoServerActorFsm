package room

import (
	"context"
	"testing"
	"time"

	"goserveractorfsm/internal/gamelogic"
	"goserveractorfsm/internal/protocol"
	kickpb "goserveractorfsm/internal/protocol/pb"

	"google.golang.org/protobuf/proto"
)

func TestAttackActorReturnsSnapshotAndBroadcastsKickState(t *testing.T) {
	nowMS := int64(10_000)
	attack := NewAttackActor(AttackConfig{
		AttackID:  7,
		RoomSize:  3,
		HoleCount: 9,
		Timing:    gamelogic.ShrewTiming{WaitMS: 1000, UpMS: 300, StandMS: 2000, DownMS: 300, DizzyMS: 500},
		NowMS:     func() int64 { return nowMS },
	})
	out1 := make(chan []byte, 8)
	out2 := make(chan []byte, 8)
	joinAttackPlayer(t, attack, 101, 1, out1)
	joinAttackPlayer(t, attack, 102, 2, out2)

	attack.Handle(context.Background(), attackEnvelope{
		SessionID: 1,
		Envelope:  envelope(t, 11, protocol.GameSnapshotReqID, &kickpb.GameSnapshotRequest{}),
	})

	snapshotEnvelope := readEnvelope(t, out1)
	if snapshotEnvelope.GetSeqId() != 11 {
		t.Fatalf("snapshot seq_id = %d, want 11", snapshotEnvelope.GetSeqId())
	}
	if snapshotEnvelope.GetMsgId() != protocol.GameSnapshotRespID.Uint32() {
		t.Fatalf("snapshot msg_id = %d, want %d", snapshotEnvelope.GetMsgId(), protocol.GameSnapshotRespID)
	}
	snapshotResp := &kickpb.GameSnapshotResponse{}
	if err := proto.Unmarshal(snapshotEnvelope.GetPayload(), snapshotResp); err != nil {
		t.Fatalf("Unmarshal GameSnapshotResponse error = %v", err)
	}
	cycle := snapshotResp.GetSnapshot().GetActiveCycles()[0]
	nowMS = cycle.GetStandStartMs() + 1

	attack.Handle(context.Background(), attackEnvelope{
		SessionID: 1,
		Envelope: envelope(t, 55, protocol.KickReqID, &kickpb.KickRequest{
			AttackEpoch: snapshotResp.GetSnapshot().GetAttackEpoch(),
			HammerType:  1,
			KickShrew:   true,
			NumOfShrew:  1,
			Shrews: []*kickpb.KickShrew{
				{ShrewIndex: cycle.GetHoleIndex(), ProtectType: cycle.GetProtectType(), SpawnSeq: cycle.GetSpawnSeq()},
			},
		}),
	})

	kickEnvelope := readEnvelope(t, out1)
	if kickEnvelope.GetSeqId() != 55 {
		t.Fatalf("kick seq_id = %d, want 55", kickEnvelope.GetSeqId())
	}
	if kickEnvelope.GetMsgId() != protocol.KickRespID.Uint32() {
		t.Fatalf("kick msg_id = %d, want %d", kickEnvelope.GetMsgId(), protocol.KickRespID)
	}

	pushEnvelope := readEnvelope(t, out2)
	if pushEnvelope.GetSeqId() != 0 {
		t.Fatalf("push seq_id = %d, want 0", pushEnvelope.GetSeqId())
	}
	if pushEnvelope.GetMsgId() != protocol.ShrewStatePushID.Uint32() {
		t.Fatalf("push msg_id = %d, want %d", pushEnvelope.GetMsgId(), protocol.ShrewStatePushID)
	}
	push := &kickpb.ShrewStatePush{}
	if err := proto.Unmarshal(pushEnvelope.GetPayload(), push); err != nil {
		t.Fatalf("Unmarshal ShrewStatePush error = %v", err)
	}
	if push.GetHoleIndex() != cycle.GetHoleIndex() || push.GetSpawnSeq() != cycle.GetSpawnSeq() {
		t.Fatalf("push target = hole %d seq %d, want hole %d seq %d", push.GetHoleIndex(), push.GetSpawnSeq(), cycle.GetHoleIndex(), cycle.GetSpawnSeq())
	}
	if push.GetActionState() != int32(gamelogic.ShrewActionDizzy) {
		t.Fatalf("push action_state = %d, want %d", push.GetActionState(), gamelogic.ShrewActionDizzy)
	}
}

func joinAttackPlayer(t *testing.T, attack *AttackActor, playerID uint64, sessionID uint64, outbound chan<- []byte) {
	t.Helper()
	reply := make(chan joinAttackReply, 1)
	attack.Handle(context.Background(), joinAttack{
		Session:  SessionRef{SessionID: sessionID, Outbound: outbound},
		PlayerID: playerID,
		Reply:    reply,
	})
	if got := <-reply; got.Err != nil {
		t.Fatalf("joinAttack Err = %v", got.Err)
	}
}

func envelope(t *testing.T, seqID uint32, msgID protocol.MsgID, payload proto.Message) *kickpb.Envelope {
	t.Helper()
	payloadData, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload error = %v", err)
	}
	return &kickpb.Envelope{SeqId: seqID, MsgId: msgID.Uint32(), Payload: payloadData}
}

func readEnvelope(t *testing.T, outbound <-chan []byte) *kickpb.Envelope {
	t.Helper()
	var data []byte
	select {
	case data = <-outbound:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for outbound envelope")
	}
	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		t.Fatalf("Unmarshal Envelope error = %v", err)
	}
	return envelope
}
