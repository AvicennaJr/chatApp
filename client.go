package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// check heartbeat
var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

// for managing new clients

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager, // will help to broadcast to other clients
		egress:     make(chan Event),
	}
}

func (c *Client) readMessages() {

	defer func() {
		// cleanup connection
		c.manager.removeClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
	}

	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payLoad, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payLoad, &request); err != nil {
			log.Println("Error marshling event: ", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("error handling message: ", err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)
	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("failed to send message: ", err)
			}

			log.Println("message sent")

		case <-ticker.C:
			log.Println("ping")

			// send a ping to the clients

			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("Send ping error: ", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
