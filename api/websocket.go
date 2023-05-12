package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-webssh/demo/pkg"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Websocket(c *gin.Context) {
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if pkg.HandleError(c, err) {
		return
	}
	defer wsConn.Close()
	cols, err := strconv.Atoi(c.DefaultQuery("cols", "180"))
	if pkg.WsHandleError(wsConn, err) {
		return
	}
	rows, err := strconv.Atoi(c.DefaultQuery("rows", "38"))
	if pkg.WsHandleError(wsConn, err) {
		return
	}
	address := c.DefaultQuery("address", "127.0.0.1:222")
	user := c.DefaultQuery("user", "root")
	password := c.DefaultQuery("password", "123456")
	logrus.Infof("cols:%d rows:%d address:%s user:%s password:%s", cols, rows, address, user, password)
	ter := pkg.NewTerminal(wsConn, pkg.Options{
		Addr:     address,
		User:     user,
		Password: password,
		Cols:     cols,
		Rows:     rows,
	})
	ter.Run()
}
