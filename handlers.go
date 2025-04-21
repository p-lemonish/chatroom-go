package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var users = make(map[string]*user)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func getUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, users)
}

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
	hub := getHub(roomname)
	user, exists := users[msg.Username]
	if !exists {
		log.Println("user does not exist")
		return
	}
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
