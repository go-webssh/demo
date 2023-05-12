package pkg

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
	"time"
)

func NewSshClient(addr, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
	}
	config.Auth = []ssh.AuthMethod{ssh.Password(password)}
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type wsMsg struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

type SshConn struct {
	StdinPipe   io.WriteCloser
	ComboOutput *wsBufferWriter
	Session     *ssh.Session
}

func flushComboOutput(w *wsBufferWriter, wsConn *websocket.Conn) error {
	if w.buffer.Len() != 0 {
		err := wsConn.WriteMessage(websocket.TextMessage, w.buffer.Bytes())
		if err != nil {
			return err
		}
		w.buffer.Reset()
	}
	return nil
}

func NewSshConn(cols, rows int, sshClient *ssh.Client) (*SshConn, error) {
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}

	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}

	comboWriter := new(wsBufferWriter)
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	return &SshConn{StdinPipe: stdinP, ComboOutput: comboWriter, Session: sshSession}, nil
}

func (s *SshConn) Close() {
	if s.Session != nil {
		s.Session.Close()
	}

}

// ReceiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (s *SshConn) ReceiveWsMsg(wsConn *websocket.Conn, logBuff *bytes.Buffer, exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if err != nil {
				logrus.WithError(err).Error("reading webSocket message failed")
				return
			}
			//unmashal bytes into struct
			msgObj := wsMsg{
				Type: "cmd",
				Cmd:  "",
				Rows: 50,
				Cols: 180,
			}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				logrus.WithError(err).WithField("wsData", string(wsData)).Error("unmarshal websocket message failed")
			}
			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := s.Session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						logrus.WithError(err).Error("ssh pty change windows size failed")
					}
				}
			case wsMsgCmd:
				decodeBytes := []byte(msgObj.Cmd)
				if err != nil {
					logrus.WithError(err).Error("websock cmd string base64 decoding failed")
				}
				if _, err := s.StdinPipe.Write(decodeBytes); err != nil {
					logrus.WithError(err).Error("ws cmd bytes write to ssh.stdin pipe failed")
				}
				if _, err := logBuff.Write(decodeBytes); err != nil {
					logrus.WithError(err).Error("write received cmd into log buffer failed")
				}
			}
		}
	}
}
func (s *SshConn) SendComboOutput(wsConn *websocket.Conn, exitCh chan bool) {
	defer setQuit(exitCh)

	tick := time.NewTicker(time.Millisecond * time.Duration(120))
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := flushComboOutput(s.ComboOutput, wsConn); err != nil {
				logrus.WithError(err).Error("ssh sending combo output to webSocket failed")
				return
			}
		case <-exitCh:
			return
		}
	}
}

func (s *SshConn) SessionWait(quitChan chan bool) {
	if err := s.Session.Wait(); err != nil {
		logrus.WithError(err).Error("ssh session wait failed")
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
