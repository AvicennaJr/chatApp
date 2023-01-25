package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// manage anything related to websocket

var (
	websocketUpgrader = websocket.Upgrader{ // upgrade htt to websocket
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type Manager struct {
	clients ClientList
	sync.RWMutex
	otps     RetentionMap
	handlers map[string]EventHandler
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}

	m.setupEventHandlers()
	return m
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {

	otp := r.URL.Query().Get("otp")
	if otp == "" || !m.otps.verifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Println("new connection")

	// upgrade regular http connection to websockert
	conn, err := websocketUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)
	m.addClient(client)

	// start client processes

	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "mojo" && req.Password == "123" {
		type response struct {
			OTP string `json:"otp"`
		}

		otp := m.otps.NewOTP()

		resp := response{
			OTP: otp.Key,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
}
func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}

}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("No such event exists.")
	}
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
	m.handlers[EventChangeRoom] = ChangeRoom
}

func ChangeRoom(event Event, c *Client) error {
	var changeRoomEvent ChangeChatRoomEvent

	if err := json.Unmarshal(event.PayLoad, &changeRoomEvent); err != nil {
		return fmt.Errorf("bad payload: %v", err)
	}

	c.chatroom = changeRoomEvent.Name

	return nil
}
func SendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent

	if err := json.Unmarshal(event.PayLoad, &chatevent); err != nil {
		return fmt.Errorf("bad payload: %v\n", err)
	}

	var broadcastMessage NewMessageEvent

	broadcastMessage.Sent = time.Now()
	broadcastMessage.Message = chatevent.Message
	broadcastMessage.From = chatevent.From

	data, err := json.Marshal(broadcastMessage)

	if err != nil {
		return fmt.Errorf("Failed to marshal: %v\n", err)
	}

	outgoingEvent := Event{
		PayLoad: data,
		Type:    EventNewMessage,
	}

	for client := range c.manager.clients {

		if client.chatroom == c.chatroom {
			client.egress <- outgoingEvent
		}
	}
	return nil
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "https://localhost:8080":
		return true
	default:
		return false
	}
}
