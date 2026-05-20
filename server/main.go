package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := NewMultiplexer()

	// Start multiplexer loop
	go mux.Run()

	// Start data producers
	startProducers(mux.EventChannel)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/events", handleEvents(mux))
	http.HandleFunc("/stats", handleStats(mux))

	// Serve frontend static files
	fs := http.FileServer(http.Dir("/app/frontend"))
	// Fallback to local ./frontend if not in Docker
	if _, err := os.Stat("/app/frontend"); os.IsNotExist(err) {
		fs = http.FileServer(http.Dir("../frontend"))
	}

	http.Handle("/", fs)

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
