package terminal

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type windowSize struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// HandleWebSocket upgrades an HTTP connection to a WebSocket, spawns a shell
// with a pseudo-terminal, and bridges I/O between the two. Resize messages
// (JSON with type:"resize") are forwarded as PTY window-size changes.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("terminal ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("pty start: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("\r\nFailed to start terminal: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		ptmx.Close()
	}()

	var once sync.Once
	done := make(chan struct{})

	// PTY stdout -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}()

	// WebSocket -> PTY stdin
	go func() {
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() { close(done) })
				return
			}

			if mt == websocket.TextMessage {
				var ws windowSize
				if json.Unmarshal(msg, &ws) == nil && ws.Type == "resize" {
					_ = pty.Setsize(ptmx, &pty.Winsize{Rows: ws.Rows, Cols: ws.Cols})
					continue
				}
			}

			if _, err := ptmx.Write(msg); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}()

	<-done
}
