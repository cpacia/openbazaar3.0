package api

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type connection struct {
	// The websocket connection
	ws *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// The hub
	h *hub
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Errorf("Websocket read error: %s", err.Error())
			break
		}

		// Just echo for now until we set up the API
		c.h.Broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Errorf("Websocket write error: %s", err.Error())
			break
		}
	}
	c.ws.Close()
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type hub struct {
	// Registered connections
	connections map[*connection]bool

	// Outbound messages to the connections
	Broadcast chan []byte

	// Register requests from the connections
	register chan *connection

	// Unregister requests from connections
	unregister chan *connection
}

func newHub() *hub {
	return &hub{
		Broadcast:   make(chan []byte),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
			log.Debug("Registered new websocket connection")
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
			log.Debug("Unregistered websocket connection")
		case m := <-h.Broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
				}
			}
		}
	}
}

type websocketHandler struct {
	hub *hub
}

func newWebsocketHandler(hub *hub) *websocketHandler {
	handler := websocketHandler{
		hub: hub,
	}
	return &handler
}

func (wsh websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Error upgrading websocket:", err)
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws, h: wsh.hub}
	c.h.register <- c
	defer func() { c.h.unregister <- c }()
	go c.writer()
	c.reader()
}
