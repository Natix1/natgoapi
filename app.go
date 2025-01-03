package main

import (
	"fmt"
	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var lmt *limiter.Limiter

/*
While rate limiting for my use case is definitely not necessary, it's included.
*/

func initRateLimiter() {
	lmt = tollbooth.NewLimiter(2, nil)
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
		}
	}

	if createFile {
		file, err := os.Create("visits.txt")
		if err != nil {
			fmt.Println("Error while failsafe was running and tried creating views file: ", err)
		}

		file.WriteString("0")
		defer file.Close()
	}

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fmt.Println("Current Go file directory:", dir)
	workingDirOld, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory (before CD): ", err)
		return 1
	}

	fmt.Println("Working directory: ", workingDirOld)

	err = os.Chdir(dir)
	if err != nil {
		fmt.Println("Error changing directory:", err)
		return 1
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return 1
	}

	fmt.Println("Working Directory after Chdir: ", workingDir)
	return 0
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
	strVisits, err := os.ReadFile("visits.txt")
	if err != nil {
		fmt.Println("Failed to read visits:", err)

		c.String(http.StatusInternalServerError, "")
	}

	frmStrVisits := strings.ReplaceAll(strings.TrimSpace(string(strVisits)), "\n", "")
	intVisits, err := strconv.Atoi(frmStrVisits)
	if err != nil {
		fmt.Println("Failed to convert visits count: ", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	c.String(http.StatusOK, fmt.Sprint(intVisits))
	file, err := os.Create("visits.txt")
	if err != nil {
		fmt.Println("Failed to open views for reading: ", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	defer file.Close()

	intVisits += 1
	_, err = file.WriteString(fmt.Sprint(intVisits))
	if err != nil {
		fmt.Println("Failed to write visits: ", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}
}

func main() {
	initRateLimiter()
	initFailSafe()

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
	r.GET("/headers", HeaderHandler)
	r.GET("/ip", GetIPHandler)
	r.GET("/visits/increment", visitsHandler)
	r.Run(":5000")
}
