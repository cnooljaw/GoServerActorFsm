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
		AttackID:            7,
		RoomSize:            3,
		HoleCount:           9,
		MinPlayersToStart:   3,
		InitialActiveShrews: 1,
		MaxActiveShrews:     1,
		InterSpawnMS:        800,
		MapCycleMS:          16_000,
		Timing:              gamelogic.ShrewTiming{WaitMS: 1000, UpMS: 300, StandMS: 2000, DownMS: 300, DizzyMS: 500},
		NowMS:               func() int64 { return nowMS },
	})
	out1 := make(chan []byte, 12)
	out2 := make(chan []byte, 12)
	out3 := make(chan []byte, 12)
	joinAttackPlayer(t, attack, 101, 1, out1)
	joinAttackPlayer(t, attack, 102, 2, out2)
	joinAttackPlayer(t, attack, 103, 3, out3)
	drainOutbound(t, out1, 2)
	drainOutbound(t, out2, 2)
	drainOutbound(t, out3, 2)

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
	if snapshotResp.GetSnapshot().GetRoomPhase() != int32(AttackPhaseRunning) {
		t.Fatalf("RoomPhase = %d, want running", snapshotResp.GetSnapshot().GetRoomPhase())
	}
	if snapshotResp.GetSnapshot().GetMapTimeline().GetCurrentMap() != gamelogic.MapMeadow {
		t.Fatalf("CurrentMap = %d, want meadow", snapshotResp.GetSnapshot().GetMapTimeline().GetCurrentMap())
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

	timelineEnvelope := readEnvelope(t, out2)
	if timelineEnvelope.GetSeqId() != 0 {
		t.Fatalf("timeline seq_id = %d, want 0", timelineEnvelope.GetSeqId())
	}
	if timelineEnvelope.GetMsgId() != protocol.ShrewTimelinePushID.Uint32() {
		t.Fatalf("timeline msg_id = %d, want %d", timelineEnvelope.GetMsgId(), protocol.ShrewTimelinePushID)
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

func TestAttackActorKeepsFillingRoomEmptyOnSnapshot(t *testing.T) {
	attack := NewAttackActor(AttackConfig{
		AttackID:            8,
		RoomSize:            3,
		HoleCount:           9,
		MinPlayersToStart:   3,
		InitialActiveShrews: 1,
		MaxActiveShrews:     1,
		InterSpawnMS:        800,
		MapCycleMS:          16_000,
		NowMS:               func() int64 { return 10_000 },
	})
	outbound := make(chan []byte, 4)
	joinAttackPlayer(t, attack, 101, 1, outbound)

	attack.Handle(context.Background(), attackEnvelope{
		SessionID: 1,
		Envelope:  envelope(t, 11, protocol.GameSnapshotReqID, &kickpb.GameSnapshotRequest{}),
	})

	response := &kickpb.GameSnapshotResponse{}
	if err := proto.Unmarshal(readEnvelope(t, outbound).GetPayload(), response); err != nil {
		t.Fatalf("Unmarshal GameSnapshotResponse error = %v", err)
	}
	snapshot := response.GetSnapshot()
	if snapshot.GetRoomPhase() != int32(AttackPhaseFilling) {
		t.Fatalf("RoomPhase = %d, want filling", snapshot.GetRoomPhase())
	}
	if len(snapshot.GetActiveCycles()) != 0 {
		t.Fatalf("len(active_cycles) = %d, want 0", len(snapshot.GetActiveCycles()))
	}
	if snapshot.GetMapTimeline().GetCurrentMap() != gamelogic.MapMeadow || snapshot.GetMapTimeline().GetNextSwitchMs() != 0 {
		t.Fatalf("filling map timeline = %+v, want meadow without switch", snapshot.GetMapTimeline())
	}
}

func TestAttackActorBroadcastsMapStateToEveryPlayerInItsRoom(t *testing.T) {
	nowMS := int64(10_000)
	attack := NewAttackActor(AttackConfig{
		AttackID:            9,
		RoomSize:            3,
		HoleCount:           9,
		MinPlayersToStart:   3,
		InitialActiveShrews: 1,
		MaxActiveShrews:     1,
		InterSpawnMS:        800,
		MapCycleMS:          16_000,
		NowMS:               func() int64 { return nowMS },
	})
	outbounds := []chan []byte{make(chan []byte, 8), make(chan []byte, 8), make(chan []byte, 8)}
	for index, outbound := range outbounds {
		joinAttackPlayer(t, attack, uint64(index+1), uint64(index+1), outbound)
	}
	for _, outbound := range outbounds {
		drainOutbound(t, outbound, 2)
	}

	nowMS = 26_800
	attack.Handle(context.Background(), attackEnvelope{
		SessionID: 1,
		Envelope:  envelope(t, 1, protocol.TimeSyncReqID, &kickpb.TimeSyncRequest{}),
	})

	for _, outbound := range outbounds {
		push := readMapStatePush(t, outbound, 3)
		if push.GetTimeline().GetCurrentMap() != gamelogic.MapShip {
			t.Fatalf("CurrentMap = %d, want ship", push.GetTimeline().GetCurrentMap())
		}
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

func drainOutbound(t *testing.T, outbound <-chan []byte, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		readEnvelope(t, outbound)
	}
}

func readMapStatePush(t *testing.T, outbound <-chan []byte, maxMessages int) *kickpb.MapStatePush {
	t.Helper()
	for i := 0; i < maxMessages; i++ {
		envelope := readEnvelope(t, outbound)
		if envelope.GetMsgId() != protocol.MapStatePushID.Uint32() {
			continue
		}
		push := &kickpb.MapStatePush{}
		if err := proto.Unmarshal(envelope.GetPayload(), push); err != nil {
			t.Fatalf("Unmarshal MapStatePush error = %v", err)
		}
		return push
	}
	t.Fatal("timed out waiting for MapStatePush")
	return nil
}
