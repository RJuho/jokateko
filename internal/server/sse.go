package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents a typed message broadcast to connected Web UI clients.
type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// SSEHub manages active client connections and broadcasts live entity mutation events.
type SSEHub struct {
	mu          sync.RWMutex
	clients     map[chan SSEEvent]struct{}
	broadcastCh chan SSEEvent
	stopCh      chan struct{}
	closeOnce   sync.Once
}

// NewSSEHub initializes a new SSEHub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:     make(map[chan SSEEvent]struct{}),
		broadcastCh: make(chan SSEEvent, 64),
		stopCh:      make(chan struct{}),
	}
}

// Start begins the event distribution and periodic heartbeat ping loop.
func (h *SSEHub) Start() {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.Broadcast("ping", nil)
			case ev := <-h.broadcastCh:
				h.mu.RLock()
				for ch := range h.clients {
					select {
					case ch <- ev:
					default:
						// Drop event for lagging clients to avoid head-of-line blocking
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// Stop shuts down the hub and disconnects all clients.
func (h *SSEHub) Stop() {
	h.closeOnce.Do(func() {
		close(h.stopCh)
		h.mu.Lock()
		defer h.mu.Unlock()
		for ch := range h.clients {
			close(ch)
		}
		h.clients = make(map[chan SSEEvent]struct{})
	})
}

// Broadcast dispatches an event to all connected clients.
func (h *SSEHub) Broadcast(eventType string, payload any) {
	select {
	case h.broadcastCh <- SSEEvent{Type: eventType, Payload: payload}:
	default:
	}
}

// ServeHTTP handles incoming SSE client subscriptions at GET /api/events.
func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	clientCh := make(chan SSEEvent, 16)

	h.mu.Lock()
	h.clients[clientCh] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, clientCh)
		h.mu.Unlock()
	}()

	// Initial connect handshake
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-clientCh:
			if !ok {
				return
			}
			var data []byte
			if ev.Payload != nil {
				var err error
				data, err = json.Marshal(ev.Payload)
				if err != nil {
					continue
				}
			} else {
				data = []byte("{}")
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
			flusher.Flush()
		}
	}
}
