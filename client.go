package main

import (
	"log"

	"github.com/gorilla/websocket"
)

// for managing new clients

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager, // will help to broadcast to other clients
	}
}

func (c *Client) readMessages() {

	defer func() {
		// cleanup connection
		c.manager.removeClient(c)
	}()

	for {
		messageType, payLoad, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		log.Println(messageType)
		log.Println(string(payLoad))
	}
}
