package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	user *user
}

type Hub struct {
	roomname   string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

var hubs = struct {
	sync.RWMutex
	data map[string]*Hub
}{data: make(map[string]*Hub)}

func newHub(roomname string) *Hub {
	return &Hub{
		roomname:   roomname,
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func getHub(roomname string) *Hub {
	hubs.Lock()
	defer hubs.Unlock()
	hub, exists := hubs.data[roomname]
	if !exists {
		hub = newHub(roomname)
		hubs.data[roomname] = hub
		go hub.run()
	}
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			close(client.send)
			delete(h.clients, client)
			if len(h.clients) == 0 {
				delete(hubs.data, h.roomname)
				return
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.broadcast <- []byte(c.user.Username + " has left the chat!")
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("readPump error: %v", err)
			return
		}
		message := append([]byte(c.user.Username+": "), []byte(msg.Text)...)
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("writePump NextWriter error: %v", err)
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				log.Printf("writePump close error: %v", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
