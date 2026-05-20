document.addEventListener('DOMContentLoaded', () => {
    const scorecardsContainer = document.getElementById('scorecards');
    const template = document.getElementById('scorecard-template');
    const statusIndicator = document.querySelector('[data-testid="connection-status"]');
    
    let eventSource = null;

    function connectSSE() {
        if (eventSource) {
            eventSource.close();
        }

        // Subscribe to all 3 games defined in the backend
        const games = ['game-01', 'game-02', 'game-03'].join(',');
        eventSource = new EventSource(`/events?games=${games}`);

        eventSource.onopen = () => {
            statusIndicator.dataset.status = 'connected';
            statusIndicator.textContent = 'Connected';
            statusIndicator.className = 'connection-status connected';
        };

        eventSource.onerror = (err) => {
            console.error("SSE Error", err);
            if (eventSource.readyState === EventSource.CONNECTING) {
                statusIndicator.dataset.status = 'reconnecting';
                statusIndicator.textContent = 'Reconnecting...';
                statusIndicator.className = 'connection-status reconnecting';
            } else {
                statusIndicator.dataset.status = 'disconnected';
                statusIndicator.textContent = 'Disconnected';
                statusIndicator.className = 'connection-status disconnected';
            }
        };

        eventSource.addEventListener('initial_state', (e) => {
            const data = JSON.parse(e.data);
            createOrUpdateScorecard(data);
        });

        eventSource.addEventListener('score_update', (e) => {
            const data = JSON.parse(e.data);
            createOrUpdateScorecard(data);
        });
    }

    function createOrUpdateScorecard(data) {
        const { game_id, home_team, away_team, home_score, away_score, game_clock } = data;
        
        let card = document.querySelector(`[data-testid="scorecard-${game_id}"]`);
        
        if (!card) {
            // Create new card from template
            const clone = template.content.cloneNode(true);
            card = clone.querySelector('.scorecard');
            card.dataset.testid = `scorecard-${game_id}`;
            
            card.querySelector('[data-home-team]').textContent = home_team;
            card.querySelector('[data-away-team]').textContent = away_team;
            
            const homeScoreEl = card.querySelector('[data-home-score]');
            homeScoreEl.dataset.testid = `score-home-${game_id}`;
            
            const awayScoreEl = card.querySelector('[data-away-score]');
            awayScoreEl.dataset.testid = `score-away-${game_id}`;
            
            scorecardsContainer.appendChild(card);
        }

        // Update values
        const homeScoreEl = card.querySelector(`[data-testid="score-home-${game_id}"]`);
        const awayScoreEl = card.querySelector(`[data-testid="score-away-${game_id}"]`);
        const clockEl = card.querySelector('[data-clock]');

        if (homeScoreEl.textContent !== String(home_score)) {
            homeScoreEl.textContent = home_score;
            triggerFlash(homeScoreEl);
        }

        if (awayScoreEl.textContent !== String(away_score)) {
            awayScoreEl.textContent = away_score;
            triggerFlash(awayScoreEl);
        }

        clockEl.textContent = game_clock;
    }

    function triggerFlash(element) {
        element.classList.remove('score-flash');
        void element.offsetWidth; // trigger reflow
        element.classList.add('score-flash');
    }

    function fetchStats() {
        fetch('/stats')
            .then(res => res.json())
            .then(data => {
                document.getElementById('clients-count').textContent = data.connected_clients;
                document.getElementById('eps-count').textContent = data.events_per_second.toFixed(1);
                document.getElementById('dropped-count').textContent = data.total_dropped_events;
            })
            .catch(err => console.error('Failed to fetch stats', err));
    }

    // Start
    connectSSE();
    setInterval(fetchStats, 2000);
});
