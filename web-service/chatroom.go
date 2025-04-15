package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type user struct {
	Username string `json:"username"`
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}
		c.send <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("write error: %v", err)
			break
		}
	}
}

func handleWebsocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}
	go client.writePump()
	go client.readPump()
}

var users = []user{}

func getUsers(c *gin.Context) {
	c.JSON(http.StatusOK, users)
}

func postUser(c *gin.Context) {
	var newUser user
	if err := c.BindJSON(&newUser); err != nil {
		return
	}
	users = append(users, newUser)
	c.JSON(http.StatusCreated, gin.H{
		"message": "started chat with user " + newUser.Username,
	})
}

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	r.Use(cors.New(config))
	r.GET("/users", getUsers)
	r.POST("/start", postUser)
	r.GET("/main", func(c *gin.Context) {
		handleWebsocket(c)
	})
	fmt.Println("WS started at 8080")
	r.Run()
}
