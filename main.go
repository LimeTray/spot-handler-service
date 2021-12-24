package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

var (
	ADDRESS string = "0.0.0.0"
	PORT    string = "8080"
)

func routes(r *gin.Engine) {
	// register routes
	r.GET("/health", healthCtrl)
	r.POST("/api/v1/notice", spotNoticeCtrl)
	r.POST("/api/v1/lifecycle-event", handleLifecycleEventctrl)
}
func server() *gin.Engine {
	// create server
	// r := gin.Default()
	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/health"),
		gin.Recovery(),
	)
	return router
}

func main() {
	fmt.Println("Welcome to spot handler (>_<)")
	ec2auth()
	registerLogger()
	r := server()
	routes(r)
	r.Run(fmt.Sprintf("%s:%s", ADDRESS, PORT))
}
