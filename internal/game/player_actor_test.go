package game

import (
	"testing"
	"time"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/gamelogic"
)

func TestPlayerActorHandlesKickWithGameLogicResult(t *testing.T) {
	ref := actor.Start(NewPlayerActor())
	defer ref.Stop()

	reply := make(chan KickReply, 1)
	err := ref.Tell(KickCommand{
		Input: gamelogic.KickInput{
			HammerType: 1,
			ComboID:    12,
			Shrews: []gamelogic.KickShrew{
				{Index: 2, ProtectType: 0},
			},
		},
		Reply: reply,
	})
	if err != nil {
		t.Fatalf("Tell(KickCommand) error = %v", err)
	}

	select {
	case got := <-reply:
		if got.Err != nil {
			t.Fatalf("KickReply.Err = %v, want nil", got.Err)
		}
		if got.Result.Money != 10 {
			t.Fatalf("Result.Money = %d, want 10", got.Result.Money)
		}
		if got.Result.ComboID != 12 {
			t.Fatalf("Result.ComboID = %d, want 12", got.Result.ComboID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for KickReply")
	}
}
