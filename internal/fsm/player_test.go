package fsm

import "testing"

func TestPlayerFSMConnectsFromDisconnected(t *testing.T) {
	machine := NewPlayerFSM()

	if err := machine.Apply(EventConnect); err != nil {
		t.Fatalf("Apply(EventConnect) error = %v", err)
	}
	if machine.State() != StateConnected {
		t.Fatalf("State() = %q, want %q", machine.State(), StateConnected)
	}
}

func TestPlayerFSMEntersGameFromConnected(t *testing.T) {
	machine := NewPlayerFSM()

	if err := machine.Apply(EventConnect); err != nil {
		t.Fatalf("Apply(EventConnect) error = %v", err)
	}
	if err := machine.Apply(EventEnterGame); err != nil {
		t.Fatalf("Apply(EventEnterGame) error = %v", err)
	}
	if machine.State() != StateInGame {
		t.Fatalf("State() = %q, want %q", machine.State(), StateInGame)
	}
}

func TestPlayerFSMRejectsKickWhenDisconnected(t *testing.T) {
	machine := NewPlayerFSM()

	if err := machine.Apply(EventKickReceived); err == nil {
		t.Fatal("Apply(EventKickReceived) error = nil, want error")
	}
	if machine.State() != StateDisconnected {
		t.Fatalf("State() = %q, want %q", machine.State(), StateDisconnected)
	}
}
