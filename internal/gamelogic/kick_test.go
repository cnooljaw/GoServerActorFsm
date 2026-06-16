package gamelogic

import "testing"

func TestCalculateKickResultSuccessfulHitReturnsRewardAndScore(t *testing.T) {
	result := CalculateKickResult(KickInput{
		HammerType: 1,
		ComboID:    7,
		Shrews: []KickShrew{
			{Index: 3, ProtectType: 0},
		},
	})

	if result.Money != 10 {
		t.Fatalf("Money = %d, want 10", result.Money)
	}
	if result.LevelScore != 10 {
		t.Fatalf("LevelScore = %d, want 10", result.LevelScore)
	}
	if result.NumOfShrew != 1 {
		t.Fatalf("NumOfShrew = %d, want 1", result.NumOfShrew)
	}
	if len(result.ShrewRewards) != 1 {
		t.Fatalf("len(ShrewRewards) = %d, want 1", len(result.ShrewRewards))
	}
	if result.ShrewRewards[0].ShrewIndex != 3 || result.ShrewRewards[0].Reward != 10 {
		t.Fatalf("ShrewRewards[0] = %+v, want shrew 3 reward 10", result.ShrewRewards[0])
	}
}

func TestCalculateKickResultMissReturnsNoShrewReward(t *testing.T) {
	result := CalculateKickResult(KickInput{
		HammerType: 1,
		ComboID:    8,
		Shrews:     nil,
	})

	if result.Money != 0 {
		t.Fatalf("Money = %d, want 0", result.Money)
	}
	if result.LevelScore != 0 {
		t.Fatalf("LevelScore = %d, want 0", result.LevelScore)
	}
	if result.NumOfShrew != 0 {
		t.Fatalf("NumOfShrew = %d, want 0", result.NumOfShrew)
	}
	if len(result.ShrewRewards) != 0 {
		t.Fatalf("len(ShrewRewards) = %d, want 0", len(result.ShrewRewards))
	}
}

func TestCalculateKickResultPreservesComboID(t *testing.T) {
	result := CalculateKickResult(KickInput{
		HammerType: 1,
		ComboID:    99,
		Shrews: []KickShrew{
			{Index: 1, ProtectType: 0},
		},
	})

	if result.ComboID != 99 {
		t.Fatalf("ComboID = %d, want 99", result.ComboID)
	}
}
