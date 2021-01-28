package main

import (
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	polling2 "github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"net/http"
)

func NewSioServer(network string) (*socketio.Server, error) {
	pt := polling2.Default
	wt := websocket.Default
	wt.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	server, err := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			pt,
			wt,
		},
	})
	if err != nil {
		logger.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		logger.Debugf("[SocketIO/%s] CONNECT: RemoteAddr=%v", s.ID(), s.RemoteAddr())
		t := s.RemoteHeader().Get("X-Type")
		if t != "" {
			logger.Debugf("[SocketIO/%s] Type=%s", s.ID(), t)
			logger.Debugf("[SocketIO/%s] User-Agent=%s", s.ID(), s.RemoteHeader().Get("User-Agent"))
			s.Join("launchers")
			s.Emit("welcome! registered launcher id")
		}
		return nil
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		if s != nil {
			//removeConsoles(s.ID())
			//logger.Debugf("[SocketIO/%s] ERROR: %s", s.ID(), e)
			logger.Debugf("[SocketIO/%v] ERROR: %s", s.ID(), e)
		} else {
			logger.Debugf("[SocketIO] ERROR: %s", e)
		}
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		removeConsoles(s.ID())
		logger.Infof("[SocketIO/%s] DISCONNECTED: %s", s.ID(), reason)
	})

	server.OnEvent("/", "test", func(s socketio.Conn, data string) {
		logger.Debugf("[SocketIO/%s] TEST: %s", s.ID(), data)
	})

	return server, nil
}
