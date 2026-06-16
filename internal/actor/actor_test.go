package actor

import (
	"context"
	"testing"
	"time"
)

func TestActorProcessesMessagesInSendOrder(t *testing.T) {
	handled := make(chan int, 3)
	ref := Start(HandlerFunc(func(ctx context.Context, msg Message) {
		value, ok := msg.(int)
		if !ok {
			t.Fatalf("message = %T, want int", msg)
		}
		handled <- value
	}))
	defer ref.Stop()

	for i := 1; i <= 3; i++ {
		if err := ref.Tell(i); err != nil {
			t.Fatalf("Tell(%d) error = %v", i, err)
		}
	}

	for want := 1; want <= 3; want++ {
		select {
		case got := <-handled:
			if got != want {
				t.Fatalf("handled message = %d, want %d", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for message %d", want)
		}
	}
}

func TestActorRejectsMessagesAfterStop(t *testing.T) {
	ref := Start(HandlerFunc(func(ctx context.Context, msg Message) {}))
	ref.Stop()

	if err := ref.Tell("late"); err == nil {
		t.Fatal("Tell after Stop error = nil, want error")
	}
}
