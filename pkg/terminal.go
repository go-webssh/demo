package pkg

import (
	"bytes"
	"context"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"os"
)

type Terminal struct {
	opts     Options
	ip       string
	ws       *websocket.Conn
	stdin    *os.File
	stdinr   *os.File
	conn     *ssh.Client
	session  *SshConn
	inited   int32
	cancelFn context.CancelFunc
}

func NewTerminal(ws *websocket.Conn, opts Options) *Terminal {
	return &Terminal{opts: opts, ws: ws}
}

func (t *Terminal) Run() {
	var err error
	t.conn, err = NewSshClient(t.opts.Addr, t.opts.User, t.opts.Password)
	if WsHandleError(t.ws, err) {
		return
	}
	defer func() {
		t.conn.Close()
	}()
	//startTime := time.Now()
	t.session, err = NewSshConn(t.opts.Cols, t.opts.Rows, t.conn)

	if WsHandleError(t.ws, err) {
		return
	}
	defer func() {
		t.session.Close()
	}()

	quitChan := make(chan bool, 3)

	var logBuff = new(bytes.Buffer)

	go t.session.ReceiveWsMsg(t.ws, logBuff, quitChan)
	go t.session.SendComboOutput(t.ws, quitChan)
	go t.session.SessionWait(quitChan)

	<-quitChan
	logrus.Info("websocket finished")
}
