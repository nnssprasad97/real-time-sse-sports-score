package main

import (
	"testing"
	"time"
)

func TestMultiplexerAddRemoveClient(t *testing.T) {
	mux := NewMultiplexer()
	client := &Client{
		ID:            "test-client",
		Channel:       make(chan GameEvent, 10),
		Subscriptions: map[string]bool{"game-01": true},
	}

	mux.AddClient(client)
	if len(mux.Clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(mux.Clients))
	}

	mux.RemoveClient("test-client")
	if len(mux.Clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(mux.Clients))
	}
}

func TestMultiplexerBackpressure(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	client := &Client{
		ID:            "slow-client",
		Channel:       make(chan GameEvent, 1), // Buffer size 1
		Subscriptions: map[string]bool{"game-01": true},
	}
	mux.AddClient(client)

	event := GameEvent{
		ID:        "1",
		GameID:    "game-01",
		EventType: "score_update",
	}

	// Send 3 events, buffer is 1, so 2 should drop
	mux.EventChannel <- event
	mux.EventChannel <- event
	mux.EventChannel <- event

	time.Sleep(100 * time.Millisecond) // Give run loop time to process

	if mux.DroppedEvents.Load() == 0 {
		t.Errorf("Expected dropped events > 0, got %d", mux.DroppedEvents.Load())
	}
}
