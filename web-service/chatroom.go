package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type user struct {
	Username string `json:"username"`
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	user *user
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

var users = []user{}
var hub = newHub()

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				client.hub.broadcast <- []byte(client.user.Username + " has left the chat!")
				delete(h.clients, client)
				close(client.send)
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}
		switch {
		case msg.Type == "auth":
			if msg.Username != "" {
				pieces := strings.SplitN(msg.Username, "_", -1)
				for idx, piece := range pieces {
					if piece == "supersecretmessage" {
						msg.Username = strings.Join(pieces[:idx], "")
					}
				}
				c.user.Username = msg.Username
				c.hub.broadcast <- []byte(c.user.Username + " has joined the chat!")
			} else {
				c.user.Username = "anonymous"
				log.Println("Didn't find username in auth call. Setting user as anonymous")
			}
		case msg.Type == "message":
			message := append([]byte(c.user.Username+": "), []byte(msg.Text)...)
			c.hub.broadcast <- message
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)
		if err := w.Close(); err != nil {
			return
		}
	}
}

func handleWebsocket(ctx *gin.Context) {
	go hub.run()
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		user: &user{},
	}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func getUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, users)
}

func postUser(ctx *gin.Context) {
	var newUser user
	if err := ctx.BindJSON(&newUser); err != nil {
		return
	}
	users = append(users, newUser)
}

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	r.Use(cors.New(config))
	r.GET("/users", getUsers)
	r.POST("/start", postUser)
	r.GET("/main", func(ctx *gin.Context) {
		handleWebsocket(ctx)
	})
	r.Run()
}
