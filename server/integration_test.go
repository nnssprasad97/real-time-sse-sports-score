package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationSSEEndpoint(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	event := GameEvent{
		ID:        "100",
		GameID:    "game-01",
		HomeTeam:  "Lakers",
		AwayTeam:  "Warriors",
		HomeScore: 10,
		AwayScore: 5,
		GameClock: "10:00",
		EventType: "score_update",
		CreatedAt: time.Now(),
	}
	mux.EventChannel <- event
	time.Sleep(50 * time.Millisecond)
	
	server := httptest.NewServer(handleEvents(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/events?games=game-01")
	if err != nil {
		t.Fatalf("Failed to GET /events: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", ct)
	}

	reader := bufio.NewReader(resp.Body)
	
	idLine, _ := reader.ReadString('\n')
	eventLine, _ := reader.ReadString('\n')
	dataLine, _ := reader.ReadString('\n')
	
	if !strings.Contains(idLine, "id: 100") {
		t.Errorf("Expected id 100, got %s", idLine)
	}
	if !strings.Contains(eventLine, "event: initial_state") {
		t.Errorf("Expected event: initial_state, got %s", eventLine)
	}
	if !strings.Contains(dataLine, "Lakers") {
		t.Errorf("Expected data containing Lakers, got %s", dataLine)
	}
}

func TestIntegrationReplayAndFiltering(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	evt1 := GameEvent{ID: "1", GameID: "g1", EventType: "score_update", CreatedAt: time.Now()}
	evt2 := GameEvent{ID: "2", GameID: "g2", EventType: "score_update", CreatedAt: time.Now()}
	evt3 := GameEvent{ID: "3", GameID: "g1", EventType: "score_update", CreatedAt: time.Now()}
	
	mux.EventChannel <- evt1
	mux.EventChannel <- evt2
	mux.EventChannel <- evt3
	time.Sleep(50 * time.Millisecond)

	server := httptest.NewServer(handleEvents(mux))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/events?games=g1", nil)
	req.Header.Set("Last-Event-ID", "1")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	idLine, _ := reader.ReadString('\n')
	eventLine, _ := reader.ReadString('\n')
	dataLine, _ := reader.ReadString('\n')

	if !strings.Contains(idLine, "id: 3") {
		t.Errorf("Expected id 3 due to replay, got %s", idLine)
	}
	if !strings.Contains(eventLine, "event: score_update") {
		t.Errorf("Expected event: score_update, got %s", eventLine)
	}
	if !strings.Contains(dataLine, `"game_id":"g1"`) {
		t.Errorf("Expected data for g1, got %s", dataLine)
	}
}

func TestIntegrationMissingLastEventIDFallback(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	evt := GameEvent{ID: "10", GameID: "g1", EventType: "score_update", CreatedAt: time.Now()}
	mux.EventChannel <- evt
	time.Sleep(50 * time.Millisecond)

	server := httptest.NewServer(handleEvents(mux))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/events?games=g1", nil)
	req.Header.Set("Last-Event-ID", "unknown-id")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	idLine, _ := reader.ReadString('\n')
	eventLine, _ := reader.ReadString('\n')

	if !strings.Contains(idLine, "id: 10") {
		t.Errorf("Expected fallback to send id 10, got %s", idLine)
	}
	if !strings.Contains(eventLine, "event: initial_state") {
		t.Errorf("Expected event: initial_state fallback, got %s", eventLine)
	}
}

func TestIntegrationEmptySubscriptions(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	server := httptest.NewServer(handleEvents(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestIntegrationHeartbeat(t *testing.T) {
	mux := NewMultiplexer()
	mux.HeartbeatInterval = 50 * time.Millisecond // very short for testing
	go mux.Run()

	server := httptest.NewServer(handleEvents(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/events?games=g1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	
	done := make(chan bool)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.Contains(line, ": ping") {
				done <- true
				return
			}
		}
	}()

	select {
	case <-done:
		// success
	case <-time.After(250 * time.Millisecond):
		t.Errorf("Did not receive heartbeat within expected time")
	}
}

func TestIntegrationStatsEndpoint(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()
	
	// Inject some state
	mux.EventChannel <- GameEvent{ID: "1", GameID: "g1", EventType: "score_update", CreatedAt: time.Now()}
	time.Sleep(50 * time.Millisecond)

	// Add a client
	client := &Client{ID: "client1", Channel: make(chan GameEvent, 10), Subscriptions: map[string]bool{"g1": true}}
	mux.AddClient(client)

	// Drop an event manually to test metric
	mux.DroppedEvents.Add(5)

	server := httptest.NewServer(handleStats(mux))
	defer server.Close()

	resp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}

	if stats.ConnectedClients != 1 {
		t.Errorf("Expected 1 connected client, got %d", stats.ConnectedClients)
	}
	if stats.TotalDroppedEvents != 5 {
		t.Errorf("Expected 5 dropped events, got %d", stats.TotalDroppedEvents)
	}
	if len(stats.ActiveGames) != 1 || stats.ActiveGames[0].GameID != "g1" {
		t.Errorf("Expected active game g1")
	}
}
