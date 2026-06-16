package fsm

import "fmt"

type State string

const (
	StateDisconnected State = "disconnected"
	StateConnected    State = "connected"
	StateInGame       State = "in_game"
)

type Event string

const (
	EventConnect      Event = "connect"
	EventEnterGame    Event = "enter_game"
	EventKickReceived Event = "kick_received"
	EventDisconnect   Event = "disconnect"
)

type PlayerFSM struct {
	state State
}

func NewPlayerFSM() *PlayerFSM {
	return &PlayerFSM{state: StateDisconnected}
}

func (m *PlayerFSM) State() State {
	return m.state
}

func (m *PlayerFSM) Apply(event Event) error {
	next, ok := playerTransitions[transitionKey{state: m.state, event: event}]
	if !ok {
		return fmt.Errorf("fsm: cannot apply event %q in state %q", event, m.state)
	}
	m.state = next
	return nil
}

type transitionKey struct {
	state State
	event Event
}

var playerTransitions = map[transitionKey]State{
	{state: StateDisconnected, event: EventConnect}:    StateConnected,
	{state: StateConnected, event: EventEnterGame}:     StateInGame,
	{state: StateInGame, event: EventKickReceived}:     StateInGame,
	{state: StateConnected, event: EventDisconnect}:    StateDisconnected,
	{state: StateInGame, event: EventDisconnect}:       StateDisconnected,
	{state: StateDisconnected, event: EventDisconnect}: StateDisconnected,
}
