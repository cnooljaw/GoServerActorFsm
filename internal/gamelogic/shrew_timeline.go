package gamelogic

import "math/rand"

const (
	ShrewActionWait  = 1
	ShrewActionUp    = 2
	ShrewActionStand = 3
	ShrewActionDown  = 4
	ShrewActionDizzy = 5
)

type ShrewTiming struct {
	WaitMS  int
	UpMS    int
	StandMS int
	DownMS  int
	DizzyMS int
}

func DefaultShrewTiming() ShrewTiming {
	return ShrewTiming{
		WaitMS:  1000,
		UpMS:    300,
		StandMS: 2000,
		DownMS:  300,
		DizzyMS: 500,
	}
}

type ShrewCycle struct {
	HoleIndex    int
	SpawnSeq     uint64
	ShrewType    int
	ProtectType  int
	HP           int
	WaitStartMS  int64
	UpStartMS    int64
	StandStartMS int64
	DownStartMS  int64
	EndMS        int64
}

type ShrewTimeline struct {
	holeCount int
	timing    ShrewTiming
	rng       *rand.Rand
	cycles    map[int]ShrewCycle
	rev       uint64
}

func NewShrewTimeline(holeCount int, timing ShrewTiming, startMS int64, rng *rand.Rand) *ShrewTimeline {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	timeline := &ShrewTimeline{
		holeCount: holeCount,
		timing:    timing,
		rng:       rng,
		cycles:    make(map[int]ShrewCycle, holeCount),
	}
	for hole := 1; hole <= holeCount; hole++ {
		timeline.cycles[hole] = timeline.newCycle(hole, 1, startMS)
	}
	timeline.rev = 1
	return timeline
}

func (t *ShrewTimeline) Revision() uint64 {
	return t.rev
}

func (t *ShrewTimeline) Timing() ShrewTiming {
	return t.timing
}

func (t *ShrewTimeline) ActiveCycles(nowMS int64) []ShrewCycle {
	cycles := make([]ShrewCycle, 0, len(t.cycles))
	for hole := 1; hole <= t.holeCount; hole++ {
		cycle, ok := t.cycles[hole]
		if !ok || cycle.EndMS <= nowMS {
			continue
		}
		cycles = append(cycles, cycle)
	}
	return cycles
}

// Advance moves every hole to the cycle that contains nowMS. The returned
// value is true when at least one hole received a newly generated cycle.
func (t *ShrewTimeline) Advance(nowMS int64) bool {
	advanced := false
	for hole := 1; hole <= t.holeCount; hole++ {
		cycle := t.cycles[hole]
		for cycle.EndMS <= nowMS {
			cycle = t.newCycle(hole, cycle.SpawnSeq+1, cycle.EndMS)
			advanced = true
		}
		t.cycles[hole] = cycle
	}
	if advanced {
		t.rev++
	}
	return advanced
}

func (t *ShrewTimeline) ValidateHit(holeIndex int, spawnSeq uint64, nowMS int64) (ShrewCycle, bool) {
	cycle, ok := t.cycles[holeIndex]
	if !ok {
		return ShrewCycle{}, false
	}
	if cycle.SpawnSeq != spawnSeq || cycle.HP <= 0 {
		return ShrewCycle{}, false
	}
	if nowMS < cycle.StandStartMS || nowMS >= cycle.DownStartMS {
		return ShrewCycle{}, false
	}
	return cycle, true
}

func (t *ShrewTimeline) ApplyHit(holeIndex int, spawnSeq uint64, nowMS int64) (ShrewCycle, bool) {
	cycle, ok := t.ValidateHit(holeIndex, spawnSeq, nowMS)
	if !ok {
		return ShrewCycle{}, false
	}
	// A hit ends the current appearance. The short terminal interval is
	// represented as Down in the durable cycle and as Dizzy by StatePush.
	cycle.HP = 0
	cycle.DownStartMS = nowMS
	cycle.EndMS = nowMS + int64(t.timing.DizzyMS)
	t.cycles[holeIndex] = cycle
	t.rev++
	return cycle, true
}

func (t *ShrewTimeline) newCycle(holeIndex int, spawnSeq uint64, waitStartMS int64) ShrewCycle {
	waitMS := int64(t.timing.WaitMS)
	upMS := int64(t.timing.UpMS)
	standMS := int64(t.timing.StandMS)
	downMS := int64(t.timing.DownMS)
	shrewType := 1 + t.rng.Intn(3)
	hp := 1
	if shrewType == 2 {
		hp = 2
	}

	upStartMS := waitStartMS + waitMS
	standStartMS := upStartMS + upMS
	downStartMS := standStartMS + standMS
	return ShrewCycle{
		HoleIndex:    holeIndex,
		SpawnSeq:     spawnSeq,
		ShrewType:    shrewType,
		ProtectType:  0,
		HP:           hp,
		WaitStartMS:  waitStartMS,
		UpStartMS:    upStartMS,
		StandStartMS: standStartMS,
		DownStartMS:  downStartMS,
		EndMS:        downStartMS + downMS,
	}
}
