package main

import (
	"encoding/json"
	"fmt"

	cmap "github.com/orcaman/concurrent-map"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	lastClientId int
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Init messages from the clients.
	init chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	coords cmap.ConcurrentMap

	coordinateUpdate chan []byte
}

type Coords struct {
	X *int
	Y *int
}

type Msg struct {
	init      bool
	X         *int    `json:"x"`
	Y         *int    `json:"y"`
	Method    *string `json:"method"`
	SessionId int     `json:"sessionId"`
}

func newHub() *Hub {
	return &Hub{
		coords:           cmap.New(),
		coordinateUpdate: make(chan []byte, 10),
		broadcast:        make(chan []byte),
		init:             make(chan []byte),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		clients:          make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client, ok := <-h.unregister:
			if !ok {
				continue
			}
			if _, ok := h.clients[client]; ok {
				h.coords.Remove(fmt.Sprintf("%d", client.id))
				delete(h.clients, client)
				close(client.send)
			}
		case message, ok := <-h.broadcast:
			if !ok {
				continue
			}
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					fmt.Println("close")
					close(client.send)
					delete(h.clients, client)
				}
			}
		case message, ok := <-h.coordinateUpdate:
			if !ok {
				continue
			}
			var msg Msg
			err := json.Unmarshal(message, &msg)
			if err != nil {
				continue
			}

			h.coords.Set(fmt.Sprintf("%d", msg.SessionId), Coords{
				X: msg.X,
				Y: msg.Y,
			})
		}
	}
}
