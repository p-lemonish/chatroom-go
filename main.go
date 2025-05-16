package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/*
TODOs
- make websocket usage safer
    - message size limits
*/

func main() {
	r := gin.Default()
	r.SetTrustedProxies([]string{
		"172.17.0.0/16", // Docker bridge subnet
	})

	config := cors.DefaultConfig()
	allowed := os.Getenv("ALLOWED_ORIGIN")
	if allowed == "" {
		log.Fatal("No allowed origin set")
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
