package gamelogic

import (
	"math/rand"
	"testing"
)

func TestShrewTimelineCreatesServerAbsoluteCycles(t *testing.T) {
	timing := ShrewTiming{
		WaitMS:  1000,
		UpMS:    300,
		StandMS: 2000,
		DownMS:  300,
		DizzyMS: 500,
	}

	timeline := NewShrewTimeline(9, timing, 10_000, rand.New(rand.NewSource(1)))
	cycles := timeline.ActiveCycles(10_000)

	if len(cycles) != 9 {
		t.Fatalf("len(cycles) = %d, want 9", len(cycles))
	}

	cycle := cycles[0]
	if cycle.HoleIndex != 1 {
		t.Fatalf("HoleIndex = %d, want 1", cycle.HoleIndex)
	}
	if cycle.SpawnSeq != 1 {
		t.Fatalf("SpawnSeq = %d, want 1", cycle.SpawnSeq)
	}
	if cycle.WaitStartMS != 10_000 {
		t.Fatalf("WaitStartMS = %d, want 10000", cycle.WaitStartMS)
	}
	if cycle.UpStartMS != 11_000 {
		t.Fatalf("UpStartMS = %d, want 11000", cycle.UpStartMS)
	}
	if cycle.StandStartMS != 11_300 {
		t.Fatalf("StandStartMS = %d, want 11300", cycle.StandStartMS)
	}
	if cycle.DownStartMS != 13_300 {
		t.Fatalf("DownStartMS = %d, want 13300", cycle.DownStartMS)
	}
	if cycle.EndMS != 13_600 {
		t.Fatalf("EndMS = %d, want 13600", cycle.EndMS)
	}
}

func TestShrewTimelineValidatesCurrentSpawnSeqAndClickableWindow(t *testing.T) {
	timing := ShrewTiming{WaitMS: 1000, UpMS: 300, StandMS: 2000, DownMS: 300, DizzyMS: 500}
	timeline := NewShrewTimeline(9, timing, 10_000, rand.New(rand.NewSource(1)))
	cycle := timeline.ActiveCycles(10_000)[0]

	if _, ok := timeline.ValidateHit(cycle.HoleIndex, cycle.SpawnSeq, cycle.StandStartMS); !ok {
		t.Fatalf("ValidateHit at stand start = false, want true")
	}
	if _, ok := timeline.ValidateHit(cycle.HoleIndex, cycle.SpawnSeq+1, cycle.StandStartMS); ok {
		t.Fatalf("ValidateHit with old/new mismatched spawn_seq = true, want false")
	}
	if _, ok := timeline.ValidateHit(cycle.HoleIndex, cycle.SpawnSeq, cycle.DownStartMS); ok {
		t.Fatalf("ValidateHit at down start = true, want false")
	}
}
