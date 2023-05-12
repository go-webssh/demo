package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"time"
)

func JsonError(c *gin.Context, msg interface{}) {
	c.AbortWithStatusJSON(200, gin.H{"ok": false, "msg": msg})
}

func HandleError(c *gin.Context, err error) bool {
	if err != nil {
		//logrus.WithError(err).Error("gin context http handler error")
		JsonError(c, err.Error())
		return true
	}
	return false
}

func WsHandleError(ws *websocket.Conn, err error) bool {
	if err != nil {
		logrus.WithError(err).Error("handler ws ERROR:")
		dt := time.Now().Add(time.Second)
		if err := ws.WriteControl(websocket.CloseMessage, []byte(err.Error()), dt); err != nil {
			logrus.WithError(err).Error("websocket writes control message failed:")
		}
		return true
	}
	return false
}
