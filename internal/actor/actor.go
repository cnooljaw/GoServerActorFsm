package actor

import (
	"context"
	"errors"
	"sync"
)

var ErrActorStopped = errors.New("actor: stopped")

type Message any

type Handler interface {
	Handle(context.Context, Message)
}

type HandlerFunc func(context.Context, Message)

func (fn HandlerFunc) Handle(ctx context.Context, msg Message) {
	fn(ctx, msg)
}

type ActorRef struct {
	mailbox chan Message
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.RWMutex
	stopped bool
}

func Start(handler Handler) *ActorRef {
	ctx, cancel := context.WithCancel(context.Background())
	ref := &ActorRef{
		mailbox: make(chan Message, 32),
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	go func() {
		defer close(ref.done)
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ref.mailbox:
				handler.Handle(ctx, msg)
			}
		}
	}()

	return ref
}

func (r *ActorRef) Tell(msg Message) error {
	r.mu.RLock()
	stopped := r.stopped
	r.mu.RUnlock()
	if stopped {
		return ErrActorStopped
	}

	r.mailbox <- msg
	return nil
}

func (r *ActorRef) Stop() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	r.cancel()
	r.mu.Unlock()

	<-r.done
}
