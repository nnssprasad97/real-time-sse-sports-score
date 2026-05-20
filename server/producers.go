package main

import (
	"fmt"
	"math/rand"
	"time"
)

var activeGamesList = []struct {
	id   string
	home string
	away string
}{
	{"game-01", "Lakers", "Warriors"},
	{"game-02", "Bulls", "Celtics"},
	{"game-03", "Heat", "Knicks"},
}

func startProducers(eventChannel chan<- GameEvent) {
	for _, game := range activeGamesList {
		go simulateGame(game.id, game.home, game.away, eventChannel)
	}
}

func simulateGame(gameID, homeTeam, awayTeam string, eventChannel chan<- GameEvent) {
	homeScore := 0
	awayScore := 0
	seq := 0

	// Random initial delay so events don't all fire exactly at once
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

	for {
		if rand.Intn(2) == 0 {
			homeScore += rand.Intn(3) + 1
		} else {
			awayScore += rand.Intn(3) + 1
		}

		mins := 12 - (seq / 5)
		secs := 59 - ((seq * 12) % 60)
		if mins < 0 {
			mins = 0
			secs = 0
		}
		clock := fmt.Sprintf("%02d:%02d", mins, secs)

		timestamp := time.Now().UnixMilli()
		id := fmt.Sprintf("%d-%d", timestamp, seq)

		event := GameEvent{
			ID:        id,
			GameID:    gameID,
			HomeTeam:  homeTeam,
			AwayTeam:  awayTeam,
			HomeScore: homeScore,
			AwayScore: awayScore,
			GameClock: clock,
			EventType: "score_update",
		}

		eventChannel <- event

		seq++
		// Update every 1.5 to 3.5 seconds
		sleepMs := 1500 + rand.Intn(2000)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}
