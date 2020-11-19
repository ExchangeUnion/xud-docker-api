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
		s.SetContext("")
		logger.Infof("[SocketIO] New client connected: ID=%v, RemoteAddr=%v", s.ID(), s.RemoteAddr())
		return nil
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		logger.Errorf("[SocketIO] Client %v got an error: %v", s.ID(), e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		logger.Infof("[SocketIO] Client %v disconnected: %v", s.ID(), reason)
	})

	server.OnEvent("/", "test", func(s socketio.Conn, data string) {
		logger.Debugf("test: %v", data)
	})

	return server, nil
}
