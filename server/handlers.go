package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func handleEvents(mux *Multiplexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*") // For local dev

		// Parse games from query string
		gamesQuery := r.URL.Query().Get("games")
		subscriptions := make(map[string]bool)
		for _, g := range strings.Split(gamesQuery, ",") {
			if g != "" {
				subscriptions[g] = true
			}
		}

		clientID := fmt.Sprintf("%d", time.Now().UnixNano())
		client := &Client{
			ID:            clientID,
			Channel:       make(chan GameEvent, 50), // Buffered channel for backpressure
			Subscriptions: subscriptions,
		}

		mux.AddClient(client)
		defer mux.RemoveClient(clientID)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		lastEventID := r.Header.Get("Last-Event-ID")

		if lastEventID != "" {
			// 2. Replay missed events if Last-Event-ID is provided
			for gameID := range subscriptions {
				missedEvents, found := mux.GetHistoryAfter(gameID, lastEventID)
				if found {
					for _, evt := range missedEvents {
						sendSSE(w, evt)
						flusher.Flush()
					}
				} else {
					// Fallback: Last-Event-ID not found (too old), send initial_state
					if state, ok := mux.GetLatestState(gameID); ok {
						initialEvent := state.LastEvent
						initialEvent.EventType = "initial_state"
						sendSSE(w, initialEvent)
						flusher.Flush()
					}
				}
			}
		} else {
			// 1. Send initial state for fresh connection
			for gameID := range subscriptions {
				if state, ok := mux.GetLatestState(gameID); ok {
					initialEvent := state.LastEvent
					initialEvent.EventType = "initial_state"
					sendSSE(w, initialEvent)
					flusher.Flush()
				}
			}
		}

		// 3. Start sending live events and heartbeats
		heartbeatTicker := time.NewTicker(15 * time.Second)
		defer heartbeatTicker.Stop()

		for {
			select {
			case event, ok := <-client.Channel:
				if !ok {
					return
				}
				// Reset ticker on activity
				heartbeatTicker.Reset(15 * time.Second)
				sendSSE(w, event)
				flusher.Flush()

			case <-heartbeatTicker.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()

			case <-r.Context().Done():
				return
			}
		}
	}
}

func handleStats(mux *Multiplexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		stats := mux.GetStats()
		json.NewEncoder(w).Encode(stats)
	}
}

func sendSSE(w http.ResponseWriter, event GameEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event %s: %v", event.ID, err)
		return
	}
	fmt.Fprintf(w, "id: %s\n", event.ID)
	fmt.Fprintf(w, "event: %s\n", event.EventType)
	fmt.Fprintf(w, "data: %s\n\n", data)
}
