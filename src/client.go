package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	id int
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		method := "leave"
		msg := Msg{Method: &method, SessionId: c.id}
		m, err := json.Marshal(msg)
		if err == nil {
			c.hub.broadcast <- m
		}
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	method := "move"
	initMsg := Msg{Method: &method, SessionId: c.id}
	m, err := json.Marshal(initMsg)
	if err != nil {
		panic("Err init connection")
	}
	c.hub.broadcast <- m
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var broadcastMsg Msg
		err = json.Unmarshal(message, &broadcastMsg)
		if err != nil {
			return
		}
		broadcastMsg.SessionId = c.id

		message, err = json.Marshal(broadcastMsg)
		if err != nil {
			return
		}

		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			var msg Msg
			if err = json.Unmarshal(message, &msg); err == nil {
				if !msg.init && msg.X != nil && msg.Y != nil {
					c.hub.coordinateUpdate <- message
				}
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	hub.lastClientId = hub.lastClientId + 1
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), id: hub.lastClientId}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	go initMessages(hub)

}

func initMessages(hub *Hub) {
	method := "move"
	for session, cord := range hub.coords.Items() {
		sId, err := strconv.Atoi(session)
		if err != nil {
			continue
		}
		X := cord.(Coords).X
		Y := cord.(Coords).Y

		if X == nil || Y == nil {
			continue
		}

		if send, err := json.Marshal(Msg{Method: &method, SessionId: sId, X: X, Y: Y, init: true}); err == nil {
			hub.broadcast <- send
		}
	}
}
