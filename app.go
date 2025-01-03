package main

import (
	"net/http"
	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
)

func HelloHandler(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!")
}

func main() {
	// Create a new tollbooth limiter
	lmt := tollbooth.NewLimiter(1, nil)

	// Set up IP lookup for rate limiting using the "CF-Connecting-IP" header
	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "CF-Connecting-IP",
		IndexFromRight: 0,
	})

	r := gin.Default()

	rateLimitMiddleware := func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
		if httpError != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}

	r.Use(rateLimitMiddleware)

	r.GET("/", HelloHandler)

	r.Run(":8080")
}
