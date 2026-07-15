package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*websocket.Conn]bool)}
}

func (h *Hub) RegisterRoutes(r chi.Router) {
	r.Get("/ws/game/{gameID}", h.handleGameWS)
}

func (h *Hub) Broadcast(gameID string, event Event) {
	h.mu.RLock()
	conns := h.rooms[gameID]
	h.mu.RUnlock()
	if len(conns) == 0 {
		return
	}
	data, _ := json.Marshal(event)
	for conn := range conns {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *Hub) handleGameWS(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.addConn(gameID, conn)
	defer h.removeConn(gameID, conn)
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (h *Hub) addConn(gameID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[gameID] == nil {
		h.rooms[gameID] = make(map[*websocket.Conn]bool)
	}
	h.rooms[gameID][conn] = true
}

func (h *Hub) removeConn(gameID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[gameID], conn)
	if len(h.rooms[gameID]) == 0 {
		delete(h.rooms, gameID)
	}
}
