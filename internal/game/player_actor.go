package game

import (
	"context"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/fsm"
	"goserveractorfsm/internal/gamelogic"
)

type KickCommand struct {
	Input gamelogic.KickInput
	Reply chan<- KickReply
}

type KickReply struct {
	Result gamelogic.KickResult
	Err    error
}

type PlayerActor struct {
	playerID uint64
	flow     *fsm.PlayerFSM
}

func NewPlayerActor() *PlayerActor {
	return NewPlayerActorWithID(0)
}

func NewPlayerActorWithID(playerID uint64) *PlayerActor {
	flow := fsm.NewPlayerFSM()
	_ = flow.Apply(fsm.EventConnect)
	_ = flow.Apply(fsm.EventEnterGame)

	return &PlayerActor{playerID: playerID, flow: flow}
}

func (p *PlayerActor) Handle(ctx context.Context, msg actor.Message) {
	switch command := msg.(type) {
	case KickCommand:
		p.handleKick(command)
	}
}

func (p *PlayerActor) handleKick(command KickCommand) {
	if err := p.flow.Apply(fsm.EventKickReceived); err != nil {
		command.Reply <- KickReply{Err: err}
		return
	}

	command.Reply <- KickReply{
		Result: gamelogic.CalculateKickResult(command.Input),
	}
}
