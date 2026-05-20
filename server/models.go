package main

import "time"

// GameEvent represents a single score update or state change.
type GameEvent struct {
	ID        string `json:"id"`
	GameID    string `json:"game_id"`
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeScore int    `json:"home_score"`
	AwayScore int    `json:"away_score"`
	GameClock string `json:"game_clock"`
	EventType string `json:"event_type"` // "score_update", "initial_state"
}

// ActiveGameStats represents the last update time of a game.
type ActiveGameStats struct {
	GameID     string    `json:"game_id"`
	LastUpdate string    `json:"last_update"` // ISO 8601 string
}

// Stats represents the metrics payload.
type Stats struct {
	ConnectedClients   int               `json:"connected_clients"`
	EventsPerSecond    float64           `json:"events_per_second"`
	TotalDroppedEvents int               `json:"total_dropped_events"`
	ActiveGames        []ActiveGameStats `json:"active_games"`
}

// GameState holds the absolute latest state of a game.
type GameState struct {
	HomeTeam  string
	AwayTeam  string
	HomeScore int
	AwayScore int
	GameClock string
	LastEvent GameEvent
	UpdatedAt time.Time
}
