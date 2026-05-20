package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	ID            string
	Channel       chan GameEvent
	Subscriptions map[string]bool
}

type Multiplexer struct {
	Clients         map[string]*Client
	ClientsMutex    sync.RWMutex
	EventChannel    chan GameEvent
	History         map[string][]GameEvent // GameID -> History
	HistoryMutex    sync.RWMutex
	LatestState     map[string]*GameState // GameID -> State
	StateMutex      sync.RWMutex
	DroppedEvents   atomic.Uint64
	TotalEventsSeen atomic.Uint64
	StartTime       time.Time
}

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		Clients:      make(map[string]*Client),
		EventChannel: make(chan GameEvent, 1000),
		History:      make(map[string][]GameEvent),
		LatestState:  make(map[string]*GameState),
		StartTime:    time.Now(),
	}
}

func (m *Multiplexer) AddClient(client *Client) {
	m.ClientsMutex.Lock()
	defer m.ClientsMutex.Unlock()
	m.Clients[client.ID] = client
}

func (m *Multiplexer) RemoveClient(clientID string) {
	m.ClientsMutex.Lock()
	defer m.ClientsMutex.Unlock()
	if client, ok := m.Clients[clientID]; ok {
		close(client.Channel)
		delete(m.Clients, clientID)
	}
}

func (m *Multiplexer) Run() {
	for event := range m.EventChannel {
		m.TotalEventsSeen.Add(1)

		// Update history and state
		m.updateState(event)

		// Fan out to clients
		m.ClientsMutex.RLock()
		for _, client := range m.Clients {
			if client.Subscriptions[event.GameID] {
				select {
				case client.Channel <- event:
					// Success
				default:
					// Buffer is full, client is slow
					log.Printf("Dropped event %s for client %s", event.ID, client.ID)
					m.DroppedEvents.Add(1)
				}
			}
		}
		m.ClientsMutex.RUnlock()
	}
}

func (m *Multiplexer) updateState(event GameEvent) {
	m.StateMutex.Lock()
	m.LatestState[event.GameID] = &GameState{
		HomeTeam:  event.HomeTeam,
		AwayTeam:  event.AwayTeam,
		HomeScore: event.HomeScore,
		AwayScore: event.AwayScore,
		GameClock: event.GameClock,
		LastEvent: event,
		UpdatedAt: time.Now(),
	}
	m.StateMutex.Unlock()

	m.HistoryMutex.Lock()
	history := m.History[event.GameID]
	history = append(history, event)

	// Keep events only from the last 5 minutes
	cutoffTime := time.Now().Add(-5 * time.Minute)
	validIndex := 0
	for i, evt := range history {
		if evt.CreatedAt.After(cutoffTime) {
			validIndex = i
			break
		}
	}
	// Slice the history array to remove old events
	history = history[validIndex:]

	m.History[event.GameID] = history
	m.HistoryMutex.Unlock()
}

func (m *Multiplexer) GetHistoryAfter(gameID, lastEventID string) []GameEvent {
	m.HistoryMutex.RLock()
	defer m.HistoryMutex.RUnlock()

	var missedEvents []GameEvent
	history, ok := m.History[gameID]
	if !ok {
		return missedEvents
	}

	found := false
	for _, evt := range history {
		if found {
			missedEvents = append(missedEvents, evt)
		}
		if evt.ID == lastEventID {
			found = true
		}
	}
	return missedEvents
}

func (m *Multiplexer) GetLatestState(gameID string) (*GameState, bool) {
	m.StateMutex.RLock()
	defer m.StateMutex.RUnlock()
	state, ok := m.LatestState[gameID]
	return state, ok
}

func (m *Multiplexer) GetStats() Stats {
	m.ClientsMutex.RLock()
	clientsCount := len(m.Clients)
	m.ClientsMutex.RUnlock()

	uptime := time.Since(m.StartTime).Seconds()
	eps := float64(m.TotalEventsSeen.Load()) / uptime

	m.StateMutex.RLock()
	activeGames := make([]ActiveGameStats, 0, len(m.LatestState))
	for gameID, state := range m.LatestState {
		activeGames = append(activeGames, ActiveGameStats{
			GameID:     gameID,
			LastUpdate: state.UpdatedAt.Format(time.RFC3339),
		})
	}
	m.StateMutex.RUnlock()

	return Stats{
		ConnectedClients:   clientsCount,
		EventsPerSecond:    eps,
		TotalDroppedEvents: int(m.DroppedEvents.Load()),
		ActiveGames:        activeGames,
	}
}
