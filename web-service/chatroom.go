package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

/*
TODOs
- code into modules
- save something into db for practice
- remove data of room when last user leaves it
- can implement authorization with jwts for example
    - currently as "supersecretmessagefromgo"
- make websocket usage safer
    - message size limits
*/

type user struct {
	Username string
	Auth     string
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	user *user
}

type SafeCounter struct {
	mu  sync.Mutex
	val int
}

func (counter *SafeCounter) Inc() {
	counter.mu.Lock()
	counter.val++
	counter.mu.Unlock()
}

func (counter *SafeCounter) Val() int {
	counter.mu.Lock()
	defer counter.mu.Unlock()
	return counter.val
}

type Hub struct {
	roomname   string
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub(roomname string) *Hub {
	return &Hub{
		roomname:   roomname,
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

var hubs = make(map[string]*Hub)
var users = make(map[string]*user)

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
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
	Roomname string `json:"roomname"`
	Text     string `json:"text"`
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
			log.Printf("read error: %v", err)
			break
		}
		message := append([]byte(c.user.Username+": "), []byte(msg.Text)...)
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
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
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	var msg Message
	if err := conn.ReadJSON(&msg); err != nil {
		log.Printf("readjson error: %v", err)
		return
	}
	roomname := msg.Roomname
	hub, exists := hubs[roomname]
	if !exists {
		hub = newHub(roomname)
		hubs[roomname] = hub
	}
	user, exists := users[msg.Username]
	if !exists {
		log.Println("user does not exist")
		return
	}
	go hub.run()
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		user: user,
	}
	client.hub.register <- client
	client.hub.broadcast <- []byte(client.user.Username + " has joined the chat!")
	go client.writePump()
	go client.readPump()
}

func getUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, users)
}

var anoncounter = SafeCounter{val: 1}

func postUser(ctx *gin.Context) {
	var newUser user
	newUser.Auth = "supersecretmessagefromgo"
	if err := ctx.BindJSON(&newUser); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, err)
		return
	}
	if newUser.Username == "" {
		newUser.Username = fmt.Sprintf("anonymous%d", anoncounter.Val())
		anoncounter.Inc()
	}
	_, exists := users[newUser.Username]
	if exists {
		ctx.IndentedJSON(http.StatusBadRequest, newUser.Username)
		return
	}
	users[newUser.Username] = &newUser
	ctx.IndentedJSON(http.StatusOK, newUser)
}

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	r.Use(cors.New(config))
	r.GET("/users", getUsers)
	r.POST("/start", postUser)
	r.GET("/chat", func(ctx *gin.Context) {
		handleWebsocket(ctx)
	})
	r.Run()
}
