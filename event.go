package main

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type    string          `json:"type"`
	PayLoad json.RawMessage `json:"payload"`
}

type EventHandler func(even Event, c *Client) error

const (
	EventSendMessage = "send_message"
	EventNewMessage  = "new_message"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type NewMessageEvent struct {
	SendMessageEvent
	Sent time.Time `json:"sent"`
}
