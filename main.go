package main

import (
	"net/http"
	"os"

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
	allowed := os.Getenv("ALLOWED_ORIGIN")
	if allowed == "" {
		allowed = "http://localhost:5173"
	}
	config.AllowOrigins = []string{allowed}
	r.Use(cors.New(config))
	r.GET("/users", getUsers)
	r.POST("/start", postUser)
	r.GET("/chat", func(ctx *gin.Context) {
		handleWebsocket(ctx)
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.Run()
}
