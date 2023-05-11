package socketigo

import (
	uuid "github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

type EventListener func(client *Client, data map[string]interface{}) interface{}

type Client struct {
	Id     uuid.UUID
	Events map[string]EventListener
	Server *IgoServer
	socket *ws.Conn
}

func createClient(server *IgoServer, socket *ws.Conn) *Client {
	return &Client{
		Server: server,
		socket: socket,
		Id:     uuid.New(),
		Events: make(map[string]EventListener),
	}
}

func handleClientData(client *Client, data map[string]interface{}) {
	eventName := data["event"].(string)
	eventData := data["data"].(map[string]interface{})
	ackId := ""

	if data["ackId"] != nil {
		ackId = data["ackId"].(string)
	}

	if listener, ok := client.Events[eventName]; ok {
		result := listener(client, eventData)

		if ackId != "" {
			response := map[string]interface{}{
				"result": result,
			}

			client.Emit(eventName+"@ack:"+ackId, response)
		}
	}
}

func (c *Client) Close() error {
	return c.socket.Close()
}

func (c *Client) Emit(eventName string, data interface{}) error {
	return c.socket.WriteJSON(map[string]interface{}{
		"event": eventName,
		"data":  data,
	})
}

func (c *Client) On(eventName string, listener EventListener) {
	c.Events[eventName] = listener
}

func (c *Client) Once(eventName string, listener EventListener) {
	c.Events[eventName] = func(client *Client, data map[string]interface{}) interface{} {
		delete(client.Events, eventName)
		return listener(client, data)
	}
}

func (c *Client) Off(eventName string) {
	delete(c.Events, eventName)
}

func (c *Client) Join(room *Room) {
	room.clients = append(room.clients, c)

	if room.joinedHandler != nil {
		room.joinedHandler(c)
	}
}

func (c *Client) Leave(room *Room) {
	for i, client := range room.clients {
		if client == c {
			room.clients = append(room.clients[:i], room.clients[i+1:]...)
			break
		}
	}

	if room.leftHandler != nil {
		room.leftHandler(c)
	}
}
