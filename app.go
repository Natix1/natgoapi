package main

import (
	"fmt"
	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var lmt *limiter.Limiter

// For cors
const DOMAIN = "https://natixone.xyz"

/*
While rate limiting for my use case is definitely not necessary, it's included.
*/

func initRateLimiter() {
	lmt = tollbooth.NewLimiter(5, nil)
	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "X-Forwarded-For",
		IndexFromRight: 0,
	})
}

func isRateLimited(c *gin.Context) bool {
	httpError := tollbooth.LimitByRequest(lmt, c.Writer, c.Request)
	return httpError != nil
}

func initFailSafe() int {
	createFile := false
	_, err := os.Stat("visits.txt")
	if err != nil {
		if os.IsNotExist(err) {
			createFile = true
		} else {
			fmt.Println("Unexpected error while checking for existence of views: ", err)
			return 1
		}
	}

	if createFile {
		file, err := os.Create("visits.txt")
		if err != nil {
			fmt.Println("Error while failsafe was running and tried creating views file: ", err)
			return 1
		}

		file.WriteString("0")
		defer file.Close()
	}
	return 0
}

func getVisits() (int, error) {
	strVisits, err := os.ReadFile("visits.txt")
	if err != nil {
		return 0, err
	}

	frmStrVisits := strings.ReplaceAll(strings.TrimSpace(string(strVisits)), "\n", "")
	intVisits, err := strconv.Atoi(frmStrVisits)
	if err != nil {
		return 0, err
	}
	return intVisits, nil
}

func writeVisits(visits int) error {
	file, err := os.Create("visits.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	visits += 1
	_, err = file.WriteString(fmt.Sprint(visits))
	if err != nil {
		return err
	}

	return nil
}

/* Handlers */

/*  /  */
func HelloHandler(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!\nThis is a small api for natixone.xyz.\nServer time: "+time.Now().Format("15:04"))
}

/*  /headers  */
func HeaderHandler(c *gin.Context) {
	var headersStr strings.Builder

	headers := c.Request.Header
	for key, values := range headers {
		for _, value := range values {
			headersStr.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	c.String(http.StatusOK, headersStr.String())
}

/*  /ip  */
func GetIPHandler(c *gin.Context) {
	fullIP := c.GetHeader("X-Forwarded-For")
	ipParts := strings.Split(fullIP, ",")
	for i, ip := range ipParts {
		ipParts[i] = strings.TrimSpace(ip)
	}
	clientIP := ipParts[1]
	c.String(http.StatusOK, clientIP)
}

/*
	/visits/increment

Increments the counter and returns the current views before the increment
*/

func visitsHandler(c *gin.Context) {
	intVisits, err := getVisits()
	if err != nil {
		fmt.Println("Failed to open visits: ", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("%d", intVisits))
	go writeVisits(intVisits)
}

func main() {
	initRateLimiter()
	codefail := initFailSafe()
	if codefail != 0 {
		fmt.Println("Exception inside initFailSafe, error code 1...")
		return
	}
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
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{DOMAIN},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
	}))

	r.GET("/", HelloHandler)
	r.GET("/headers", HeaderHandler)
	r.GET("/ip", GetIPHandler)
	r.GET("/visits/increment", visitsHandler)
	r.Run(":5000")
}
