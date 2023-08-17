package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 5,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		// allow requests from any origin
		// the internal service can decide what origins to allow
		return true
	},
}

type Server struct {
	connLock *sync.RWMutex
	connStop bool
	conns    map[string]*websocket.Conn
}

func NewServer() *Server {
	return &Server{
		connLock: new(sync.RWMutex),
		conns:    make(map[string]*websocket.Conn),
	}
}

func (s *Server) Upgrade(rw http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		return
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()

	// no more connections allowed
	if s.connStop {
		_ = c.Close()
		return
	}

	// save connection for shutdown
	s.conns[c.RemoteAddr().String()] = c

	log.Printf("[Websocket] Dialing: '%s'\n", req.URL.String())

	// dial for internal connection
	ic, _, err := websocket.DefaultDialer.DialContext(req.Context(), req.URL.String(), req.Header)
	if err != nil {
		s.Remove(c)
		return
	}
	done := make(chan struct{}, 1)

	// relay messages each way
	s.wsRelay(done, c, ic)
	s.wsRelay(done, ic, c)

	// wait for done signal and close both connections
	go func() {
		<-done
		_ = c.Close()
		_ = ic.Close()
	}()
}

func (s *Server) wsRelay(done chan struct{}, a, b *websocket.Conn) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		mt, message, err := a.ReadMessage()
		if err != nil {
			return
		}
		if b.WriteMessage(mt, message) != nil {
			return
		}
	}
}

func (s *Server) Remove(c *websocket.Conn) {
	s.connLock.Lock()
	delete(s.conns, c.RemoteAddr().String())
	s.connLock.Unlock()
	_ = c.Close()
}

func (s *Server) Shutdown() {
	s.connLock.Lock()
	defer s.connLock.Unlock()

	// flag shutdown and close all open connections
	s.connStop = true
	for _, i := range s.conns {
		_ = i.Close()
	}

	// clear connections, not required but do it anyway
	s.conns = make(map[string]*websocket.Conn)
}
