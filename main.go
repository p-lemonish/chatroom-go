package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/*
TODOs
- save something into db for practice
- make websocket usage safer
    - message size limits
*/

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
		"http://18.192.11.118",
		"http://18.192.11.118:80",
		"http://100.27.185.143:80"}
	r.Use(cors.New(config))
	r.GET("/users", getUsers)
	r.POST("/start", postUser)
	r.GET("/chat", func(ctx *gin.Context) {
		handleWebsocket(ctx)
	})
	r.Run()
}
