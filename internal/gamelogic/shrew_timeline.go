package gamelogic

import (
	"math/rand"
	"sort"
)

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

// ShrewTimeline owns the room's currently active shrew cycles. It never
// creates a cycle until Start, so a filling room has an empty snapshot.
type ShrewTimeline struct {
	holeCount     int
	timing        ShrewTiming
	rng           *rand.Rand
	maxActive     int
	interSpawnMS  int64
	started       bool
	startAtMS     int64
	nextSpawnAtMS int64
	nextSpawnSeq  uint64
	cycles        map[int]ShrewCycle
	rev           uint64
}

func NewShrewTimeline(holeCount int, timing ShrewTiming, maxActive int, interSpawnMS int64, rng *rand.Rand) *ShrewTimeline {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	if maxActive <= 0 {
		maxActive = 1
	}
	if maxActive > holeCount {
		maxActive = holeCount
	}
	if interSpawnMS < 0 {
		interSpawnMS = 0
	}
	return &ShrewTimeline{
		holeCount:    holeCount,
		timing:       timing,
		rng:          rng,
		maxActive:    maxActive,
		interSpawnMS: interSpawnMS,
		cycles:       make(map[int]ShrewCycle, maxActive),
	}
}

func (t *ShrewTimeline) Start(startAtMS int64, initialActive int) {
	if t.started {
		return
	}
	t.started = true
	t.startAtMS = startAtMS
	if initialActive < 0 {
		initialActive = 0
	}
	if initialActive > t.maxActive {
		initialActive = t.maxActive
	}
	for len(t.cycles) < initialActive {
		t.spawn(startAtMS)
	}
	if len(t.cycles) == 0 {
		t.nextSpawnAtMS = startAtMS
	} else {
		t.nextSpawnAtMS = t.latestEndMS() + t.interSpawnMS
	}
	t.rev++
}

func (t *ShrewTimeline) Revision() uint64 {
	return t.rev
}

func (t *ShrewTimeline) Timing() ShrewTiming {
	return t.timing
}

func (t *ShrewTimeline) ActiveCycles(nowMS int64) []ShrewCycle {
	cycles := make([]ShrewCycle, 0, len(t.cycles))
	for _, cycle := range t.cycles {
		if cycle.EndMS > nowMS {
			cycles = append(cycles, cycle)
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].HoleIndex < cycles[j].HoleIndex })
	return cycles
}

// Advance removes elapsed cycles and deterministically creates replacements
// until the current server time is represented by the active collection.
func (t *ShrewTimeline) Advance(nowMS int64) bool {
	if !t.started || nowMS < t.startAtMS {
		return false
	}

	advanced := false
	for {
		for hole, cycle := range t.cycles {
			if cycle.EndMS <= nowMS {
				delete(t.cycles, hole)
				advanced = true
			}
		}
		if len(t.cycles) >= t.maxActive || len(t.cycles) >= t.holeCount || nowMS < t.nextSpawnAtMS {
			break
		}
		t.spawn(t.nextSpawnAtMS)
		t.nextSpawnAtMS = t.latestEndMS() + t.interSpawnMS
		advanced = true
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
	nextAt := cycle.EndMS + t.interSpawnMS
	if t.nextSpawnAtMS == 0 || nextAt < t.nextSpawnAtMS {
		t.nextSpawnAtMS = nextAt
	}
	t.rev++
	return cycle, true
}

func (t *ShrewTimeline) spawn(waitStartMS int64) {
	hole := t.pickAvailableHole()
	if hole == 0 {
		return
	}
	t.nextSpawnSeq++
	t.cycles[hole] = t.newCycle(hole, t.nextSpawnSeq, waitStartMS)
}

func (t *ShrewTimeline) pickAvailableHole() int {
	available := make([]int, 0, t.holeCount-len(t.cycles))
	for hole := 1; hole <= t.holeCount; hole++ {
		if _, occupied := t.cycles[hole]; !occupied {
			available = append(available, hole)
		}
	}
	if len(available) == 0 {
		return 0
	}
	return available[t.rng.Intn(len(available))]
}

func (t *ShrewTimeline) latestEndMS() int64 {
	var latest int64
	for _, cycle := range t.cycles {
		if cycle.EndMS > latest {
			latest = cycle.EndMS
		}
	}
	return latest
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
