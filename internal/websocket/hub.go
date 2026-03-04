package websocket

import (
	"encoding/json"
	"net/http"
	"reflect"
)

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMessage
}

type broadcastMessage struct {
	patientID string
	event     interface{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case msg := <-h.broadcast:
			event := h.wrapEvent(msg.patientID, msg.event)
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			for client := range h.clients {
				if client.patientID == msg.patientID {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) BroadcastToPatient(patientID string, event interface{}) {
	h.broadcast <- broadcastMessage{patientID: patientID, event: event}
}

func (h *Hub) wrapEvent(patientID string, event interface{}) Event {
	eventType := extractEventType(event)
	return Event{
		Type:      eventType,
		PatientID: patientID,
		Data:      event,
	}
}

func extractEventType(event interface{}) string {
	v := reflect.ValueOf(event)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		if f := v.FieldByName("Type"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	patientID := r.URL.Query().Get("patient")
	if patientID == "" {
		http.Error(w, "patient query param required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		patientID: patientID,
		send:      make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}
