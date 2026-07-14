package gamelogic

const (
	MapMeadow int32 = 2
	MapShip   int32 = 3
	MapSpace  int32 = 4
)

type MapState struct {
	CurrentMap   int32
	Revision     uint64
	MapStartedMS int64
	NextSwitchMS int64
	NextMap      int32
	CycleMS      int64
}

// MapTimeline is a room-level, server-authoritative schedule. Clients only
// render its absolute boundaries; they never choose a map themselves.
type MapTimeline struct {
	currentMap   int32
	revision     uint64
	started      bool
	mapStartedMS int64
	nextSwitchMS int64
	nextMap      int32
	cycleMS      int64
}

func NewMapTimeline(cycleMS int64) *MapTimeline {
	if cycleMS <= 0 {
		cycleMS = 16_000
	}
	return &MapTimeline{
		currentMap: MapMeadow,
		cycleMS:    cycleMS,
	}
}

func (t *MapTimeline) Start(startAtMS int64) {
	if t.started {
		return
	}
	t.started = true
	t.currentMap = MapMeadow
	t.mapStartedMS = startAtMS
	t.nextMap = MapShip
	t.nextSwitchMS = startAtMS + t.cycleMS
	t.revision++
}

func (t *MapTimeline) Advance(nowMS int64) bool {
	if !t.started || nowMS < t.nextSwitchMS {
		return false
	}
	changed := false
	for nowMS >= t.nextSwitchMS {
		t.currentMap = t.nextMap
		t.mapStartedMS = t.nextSwitchMS
		t.nextMap = nextMap(t.currentMap)
		t.nextSwitchMS += t.cycleMS
		t.revision++
		changed = true
	}
	return changed
}

func (t *MapTimeline) Snapshot() MapState {
	return MapState{
		CurrentMap:   t.currentMap,
		Revision:     t.revision,
		MapStartedMS: t.mapStartedMS,
		NextSwitchMS: t.nextSwitchMS,
		NextMap:      t.nextMap,
		CycleMS:      t.cycleMS,
	}
}

func nextMap(current int32) int32 {
	switch current {
	case MapMeadow:
		return MapShip
	case MapShip:
		return MapSpace
	default:
		return MapMeadow
	}
}
