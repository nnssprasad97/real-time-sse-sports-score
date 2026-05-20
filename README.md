# Real-Time SSE Sports Score Feed Multiplexer

A high-performance, real-time sports score feed aggregator using Server-Sent Events (SSE). This service efficiently distributes live game updates to thousands of concurrent web clients using a fan-out multiplexing pattern.

## Architecture

The backend is built in **Go**, chosen for its powerful concurrency primitives (channels, goroutines) which are ideal for managing multiplexed, non-blocking streams.

1. **Data Producers:** Independent goroutines simulate live events and publish them to a central, shared channel.
2. **SSE Multiplexer:** The core engine that consumes the central channel and fans out events to connected clients. It handles backpressure by safely dropping events for slow clients without blocking others.
3. **Frontend Client:** A vanilla HTML/JS/CSS web interface that connects via the native browser `EventSource` API, rendering dynamic, real-time score updates with a modern aesthetic.

## Endpoints

### `GET /events`
Establishes a persistent SSE connection.
- **Query Params:** `?games=game-01,game-02` (Filters the stream to only send updates for specified games).
- **Headers Handled:** `Last-Event-ID` for resuming a stream after a disconnect.

### `GET /stats`
Returns real-time metrics about the service (JSON).
- `connected_clients`: Total active SSE connections.
- `events_per_second`: Overall throughput.
- `total_dropped_events`: Counter of events dropped due to client backpressure.
- `active_games`: List of games currently producing updates.

## Core Mechanisms

- **Backpressure Handling:** When fanning out events, the multiplexer uses a non-blocking channel send (`select { case clientChannel <- event: default: drop() }`). If a client's buffer is full, the event is immediately dropped, preventing that slow client from bottlenecking the entire system.
- **Event History & Replay:** The server strictly retains the last 5 minutes of events in memory. When a client reconnects with a `Last-Event-ID` header, the server skips the initial state dispatch and immediately streams any missed events from this history buffer before resuming the live feed. This ensures perfect ordering and prevents duplicate data.
- **Heartbeats:** To keep idle connections alive through firewalls/proxies, an SSE comment (`: ping`) is emitted every 15 seconds.

## Run Instructions

The entire stack is containerized using Docker and Docker Compose. To start the application:

```bash
docker-compose up -d --build
```

Wait approximately 10 seconds for the health checks to pass. The application will be available at:
- Dashboard: [http://localhost:8080](http://localhost:8080)
- Stats API: [http://localhost:8080/stats](http://localhost:8080/stats)

## Testing

Comprehensive unit and integration tests are included in the `server` directory. The tests verify SSE endpoint headers, subscription filtering, initial state dispatch, and Last-Event-ID replay logic.

To run the tests locally:
```bash
cd server
go test -v ./...
```
