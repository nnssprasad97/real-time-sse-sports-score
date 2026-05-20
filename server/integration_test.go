package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationSSEEndpoint(t *testing.T) {
	mux := NewMultiplexer()
	go mux.Run()

	// Inject a known state
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
	time.Sleep(50 * time.Millisecond) // let run loop process
	
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

	// Request with Last-Event-ID = 1, filtered to g1 only. Should skip initial_state and just send evt3.
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
