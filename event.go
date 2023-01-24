package main

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	PayLoad json.RawMessage `json:"payload"`
}

type EventHandler func(even Event, c *Client) error

const (
	EventSendMessage = "send_message"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}
