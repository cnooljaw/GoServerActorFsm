package gamelogic

import "testing"

func TestMapTimelineStartsAtMeadowAndCatchesUpByServerTime(t *testing.T) {
	timeline := NewMapTimeline(16_000)
	timeline.Start(10_800)

	initial := timeline.Snapshot()
	if initial.CurrentMap != MapMeadow {
		t.Fatalf("CurrentMap = %d, want Meadow", initial.CurrentMap)
	}
	if initial.NextSwitchMS != 26_800 {
		t.Fatalf("NextSwitchMS = %d, want 26800", initial.NextSwitchMS)
	}
	if timeline.Advance(26_799) {
		t.Fatal("Advance before boundary = true, want false")
	}
	if !timeline.Advance(26_800) {
		t.Fatal("Advance at boundary = false, want true")
	}
	if got := timeline.Snapshot().CurrentMap; got != MapShip {
		t.Fatalf("CurrentMap = %d, want Ship", got)
	}
	if !timeline.Advance(42_801) {
		t.Fatal("Advance after second boundary = false, want true")
	}
	if got := timeline.Snapshot().CurrentMap; got != MapSpace {
		t.Fatalf("CurrentMap = %d, want Space", got)
	}
}

func TestMapTimelineFillingSnapshotRemainsMeadow(t *testing.T) {
	timeline := NewMapTimeline(16_000)
	state := timeline.Snapshot()
	if state.CurrentMap != MapMeadow || state.NextSwitchMS != 0 || state.Revision != 0 {
		t.Fatalf("filling snapshot = %+v, want meadow revision 0 without boundary", state)
	}
}
