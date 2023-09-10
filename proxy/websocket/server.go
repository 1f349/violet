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
	req.URL.Scheme = "ws"
	log.Printf("[Websocket] Upgrading request to '%s' from '%s'\n", req.URL.String(), req.Header.Get("Origin"))

	c, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		return
	}

	defer c.Close()
	s.connLock.Lock()

	// no more connections allowed
	if s.connStop {
		s.connLock.Unlock()
		return
	}

	// save connection for shutdown
	s.conns[c.RemoteAddr().String()] = c
	s.connLock.Unlock()

	log.Printf("[Websocket] Dialing: '%s'\n", req.URL.String())

	// dial for internal connection
	ic, _, err := websocket.DefaultDialer.DialContext(req.Context(), req.URL.String(), nil)
	if err != nil {
		log.Printf("[Websocket] Failed to dial '%s': %s\n", req.URL.String(), err)
		s.Remove(c)
		return
	}
	defer ic.Close()

	d1 := make(chan struct{}, 1)
	d2 := make(chan struct{}, 1)

	// relay messages each way
	go s.wsRelay(d1, c, ic)
	go s.wsRelay(d2, ic, c)

	// wait for done signal and close both connections
	log.Println("[Websocket] Completed websocket hijacking")

	// waiting until d1 or d2 close then automatically defer close both connections
	select {
	case <-d1:
	case <-d2:
	}
}

func (s *Server) wsRelay(done chan struct{}, a, b *websocket.Conn) {
	defer func() {
		close(done)
	}()
	for {
		mt, message, err := a.ReadMessage()
		if err != nil {
			log.Println("Websocket read message error: ", err)
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
