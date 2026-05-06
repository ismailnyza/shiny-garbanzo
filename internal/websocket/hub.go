package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/ismael/qr-restaurant/internal/shared/metrics"
)

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Hub struct {
	// Registered clients mapped by sessionToken.
	// Since multiple customers might scan the same table QR, a session can have multiple active clients.
	sessions map[string]map[*Client]bool
	mu       sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.sessions[client.sessionToken]; !ok {
		h.sessions[client.sessionToken] = make(map[*Client]bool)
	}
	h.sessions[client.sessionToken][client] = true
	metrics.Default.IncWSActive()
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.sessions[client.sessionToken]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
			metrics.Default.DecWSActive()
			if len(clients) == 0 {
				delete(h.sessions, client.sessionToken)
			}
		}
	}
}

func (h *Hub) BroadcastToSession(sessionToken string, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.sessions[sessionToken]
	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Buffer full, drop message and assume client is dead or stuck
			log.Printf("WebSocket: buffer full for client in session %s, dropping message", sessionToken)
		}
	}
}
