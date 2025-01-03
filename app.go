package main

import (
	"net/http"
	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
)

var lmt *limiter.Limiter

func initRateLimiter() {
	lmt = tollbooth.NewLimiter(1, nil)
	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "CF-Connecting-IP",
		IndexFromRight: 0,
	})
}

func isRateLimited(c *gin.Context) bool {
	httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
	return httpError != nil
}

func HelloHandler(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!\nYour IP address: " + c.GetHeader("CF-Connecting-IP"))
}

func main() {
	initRateLimiter()

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	rateLimitMiddleware := func(c *gin.Context) {
		if isRateLimited(c) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}

	r.Use(rateLimitMiddleware)

	r.GET("/", HelloHandler)

	r.Run(":5000")
}