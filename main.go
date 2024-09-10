package main

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.POST("/api/v1/getPassword", getPassword)

	router.Run(":8080")
}

func getPassword(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encodedPassword := base64.StdEncoding.EncodeToString([]byte(data.Password))

	c.JSON(http.StatusOK, gin.H{"result": encodedPassword})
}
