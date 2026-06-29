package main

import (
	"net/http"
	"time"

	"github.com/Frosmin/cs2-skins-arbitrage/csFloat"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.Use(corsMiddleware("http://localhost:4200"))

	service := csfloat.NewService(&http.Client{Timeout: 15 * time.Second})
	handler := csfloat.NewHandler(service)

	router.GET("/api/listings", handler.GetListings)
	router.OPTIONS("/api/listings", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}

func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
