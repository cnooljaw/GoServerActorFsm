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
	if got := MsgName(MsgID(999999)); got != "unknown" {
		t.Fatalf("MsgName(unknown) = %q, want unknown", got)
	}
}
