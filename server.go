package socketigo

import (
	"net/http"

	"github.com/goccy/go-json"
	ws "github.com/gorilla/websocket"
)

/*
Events:
- preconnect: Gets called when the connection is established but the handshake is not yet completed.
- connected: Gets called when the connection is established and the handshake is completed.
- disconnected: Gets called when the connection is closed.
*/
type IgoServer struct {
	Clients             []*Client
	Rooms               []*Room
	upgrader            *ws.Upgrader
	preConnectHandler   func(conn *ws.Conn)
	connectedHandler    func(client *Client)
	disconnectedHandler func(client *Client)
	errHandler          func(err error)
}

type IgoServerOptions struct {
	ReadBufferSize  int
	WriteBufferSize int
}

type IgoServerHandle func(w http.ResponseWriter, r *http.Request)

func CreateIgoServer(options *IgoServerOptions) *IgoServer {
	if options == nil {
		options = &IgoServerOptions{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
	}

	return &IgoServer{
		Clients: make([]*Client, 0),
		Rooms:   make([]*Room, 0),
		upgrader: &ws.Upgrader{
			ReadBufferSize:  options.ReadBufferSize,
			WriteBufferSize: options.WriteBufferSize,
		},
		preConnectHandler:   nil,
		connectedHandler:    nil,
		disconnectedHandler: nil,
		errHandler:          nil,
	}
}

func (s *IgoServer) OnPreConnect(listener func(conn *ws.Conn)) {
	s.preConnectHandler = listener
}

func (s *IgoServer) OnConnected(listener func(client *Client)) {
	s.connectedHandler = listener
}

func (s *IgoServer) OnDisconnected(listener func(client *Client)) {
	s.disconnectedHandler = listener
}

func (s *IgoServer) Emit(eventName string, data interface{}) {
	for _, client := range s.Clients {
		client.Emit(eventName, data)
	}
}

func (s *IgoServer) EmitExcept(client *Client, eventName string, data interface{}) {
	for _, c := range s.Clients {
		if c != client {
			c.Emit(eventName, data)
		}
	}
}

func (s *IgoServer) CreateRoom(name string) *Room {
	room := &Room{
		Id:      name,
		clients: make([]*Client, 0),
	}
	s.Rooms = append(s.Rooms, room)
	return room
}

func (s *IgoServer) GetRoom(name string) *Room {
	for _, room := range s.Rooms {
		if room.Id == name {
			return room
		}
	}
	return nil
}

func (s *IgoServer) DeleteRoom(room *Room) {
	for i, r := range s.Rooms {
		if r == room {
			s.Rooms = append(s.Rooms[:i], s.Rooms[i+1:]...)
			return
		}
	}
}

func (s *IgoServer) Handle() IgoServerHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		s.upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}

		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			if s.errHandler != nil {
				s.errHandler(err)
			}
			return
		}

		if s.preConnectHandler != nil {
			s.preConnectHandler(conn)
		}

		client := createClient(s, conn)
		s.Clients = append(s.Clients, client)

		if s.connectedHandler != nil {
			s.connectedHandler(client)
		}

		go wsReader(client)
	}
}

func wsReader(client *Client) {
	for {
		_, data, err := client.socket.ReadMessage()
		if err != nil {
			for i, c := range client.server.Clients {
				if c == client {
					client.server.Clients = append(client.server.Clients[:i], client.server.Clients[i+1:]...)
					break
				}
			}

			client.socket.Close()

			if client.server.disconnectedHandler != nil {
				client.server.disconnectedHandler(client)
			}
			break
		}

		result := make(map[string]interface{})

		err = json.Unmarshal(data, &result)
		if err != nil {
			if client.server.errHandler != nil {
				client.server.errHandler(err)
			}
			continue
		}

		handleClientData(client, result)
	}
}
