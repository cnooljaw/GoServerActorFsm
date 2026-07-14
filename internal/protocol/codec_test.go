package protocol

import (
	"testing"

	kickpb "goserveractorfsm/internal/protocol/pb"
)

func TestEncodeDecodeKickRequest(t *testing.T) {
	input := &kickpb.KickRequest{
		HammerType: 1,
		KickShrew:  true,
		NumOfShrew: 1,
		ComboId:    11,
		Shrews: []*kickpb.KickShrew{
			{ShrewIndex: 4, ProtectType: 0},
		},
	}

	encoded, err := Encode(input)
	if err != nil {
		t.Fatalf("Encode(KickRequest) error = %v", err)
	}

	decoded, err := DecodeKickRequest(encoded)
	if err != nil {
		t.Fatalf("DecodeKickRequest error = %v", err)
	}
	if decoded.GetShrews()[0].GetShrewIndex() != 4 {
		t.Fatalf("ShrewIndex = %d, want 4", decoded.GetShrews()[0].GetShrewIndex())
	}
}

func TestEncodeDecodeKickResponse(t *testing.T) {
	input := &kickpb.KickResponse{
		Ret:        0,
		Money:      10,
		LevelScore: 10,
		ComboId:    11,
		ShrewResp: []*kickpb.ShrewReward{
			{ShrewIndex: 4, Reward: 10},
		},
	}

	encoded, err := Encode(input)
	if err != nil {
		t.Fatalf("Encode(KickResponse) error = %v", err)
	}

	decoded, err := DecodeKickResponse(encoded)
	if err != nil {
		t.Fatalf("DecodeKickResponse error = %v", err)
	}
	if decoded.GetMoney() != 10 {
		t.Fatalf("Money = %d, want 10", decoded.GetMoney())
	}
}

func TestMsgNameReturnsStableNames(t *testing.T) {
	if got := MsgName(KickReqID); got != "kick_req" {
		t.Fatalf("MsgName(KickReqID) = %q, want kick_req", got)
	}
	if got := MsgName(JoinRoomReqID); got != "join_room_req" {
		t.Fatalf("MsgName(JoinRoomReqID) = %q, want join_room_req", got)
	}
	if got := MsgName(ShrewTimelinePushID); got != "shrew_timeline_push" {
		t.Fatalf("MsgName(ShrewTimelinePushID) = %q, want shrew_timeline_push", got)
	}
	if got := MsgName(MapStatePushID); got != "map_state_push" {
		t.Fatalf("MsgName(MapStatePushID) = %q, want map_state_push", got)
	}
	if got := MsgName(MsgID(999999)); got != "unknown" {
		t.Fatalf("MsgName(unknown) = %q, want unknown", got)
	}
}

func TestKickShrewCarriesSpawnSeq(t *testing.T) {
	input := &kickpb.KickShrew{
		ShrewIndex:  4,
		ProtectType: 0,
		SpawnSeq:    99,
	}

	encoded, err := Encode(input)
	if err != nil {
		t.Fatalf("Encode(KickShrew) error = %v", err)
	}

	decoded := &kickpb.KickShrew{}
	if err := Decode(encoded, decoded); err != nil {
		t.Fatalf("Decode(KickShrew) error = %v", err)
	}
	if decoded.GetSpawnSeq() != 99 {
		t.Fatalf("SpawnSeq = %d, want 99", decoded.GetSpawnSeq())
	}
}
