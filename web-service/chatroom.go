package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type user struct {
	Username string `json:"username"`
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
	r.Run()
}
