package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)


type DashboardHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

var dashHub = &DashboardHub{clients: make(map[*websocket.Conn]bool)}

func DashboardWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	dashHub.mu.Lock()
	dashHub.clients[conn] = true
	dashHub.mu.Unlock()

	go func() {
		defer func() {
			dashHub.mu.Lock()
			delete(dashHub.clients, conn)
			dashHub.mu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func BroadcastMetrics(data interface{}) {
	dashHub.mu.RLock()
	defer dashHub.mu.RUnlock()
	msg, _ := json.Marshal(data)
	for conn := range dashHub.clients {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func RegisterDashboardRoute(r chi.Router) {
	r.Get("/ws/dashboard", DashboardWS)
}
