package socketigo

type Room struct {
	Id            string
	clients       []*Client
	joinedHandler func(client *Client)
	leftHandler   func(client *Client)
}

func (r *Room) OnClientJoined(listener func(client *Client)) {
	r.joinedHandler = listener
}

func (r *Room) OnClientLeft(listener func(client *Client)) {
	r.leftHandler = listener
}

func (r *Room) Emit(eventName string, data interface{}) {
	for _, client := range r.clients {
		client.Emit(eventName, data)
	}
}

func (r *Room) EmitExcept(client *Client, eventName string, data interface{}) {
	for _, c := range r.clients {
		if c != client {
			c.Emit(eventName, data)
		}
	}
}
