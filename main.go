package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-webssh/demo/api"
	"github.com/go-webssh/demo/middlewares"
	"net/http"
)

func main() {
	r := gin.Default()
	r.Use(middlewares.Cors())
	r.LoadHTMLFiles("ui/index.html")
	r.Static("/static", "./ui/static")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "demo",
		})
	})
	r.GET("/ws", api.Websocket)
	_ = r.Run()
}
