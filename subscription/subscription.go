package subscription

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	recvTopic string
	recvAt    time.Time
	data      []byte
}

type Data struct {
	Name   string
	Topic  string
	Qos    byte
	Format string
}

type Model struct {
	data       Data
	messages   chan []Message
	messagesMu *sync.Mutex
	messageIdx int
}

func NewModel(data Data) Model {
	s := Model{
		data:       data,
		messages:   make(chan []Message, 1),
		messagesMu: &sync.Mutex{},
	}
	s.messages <- make([]Message, 0)
	return s
}

func (m Model) Title() string       { return m.data.Name }
func (m Model) Description() string { return m.data.Topic }
func (m Model) FilterValue() string { return m.data.Topic }

func (m Model) OnPubHandler(client mqtt.Client, msg mqtt.Message) {
	var data []byte
	data = msg.Payload()
	switch m.data.Format {
	case "json":
		tmp := bytes.NewBuffer([]byte{})
		json.Indent(tmp, data, "", "  ")
		data = tmp.Bytes()
	}

	m.messagesMu.Lock()
	defer m.messagesMu.Unlock()
	newMessage := Message{
		recvTopic: msg.Topic(),
		recvAt:    time.Now(),
		data:      data,
	}

	messages := <-m.messages
	messages = append([]Message{newMessage}, messages...)

	m.messages <- messages
}

func (m Model) Messages() []Message {
	m.messagesMu.Lock()
	messages := <-m.messages
	m.messages <- messages
	m.messagesMu.Unlock()

	return messages
}

func (m Model) Data() Data { return m.data }

func (m Message) RecvTopic() string { return m.recvTopic }
func (m Message) RecvAt() time.Time { return m.recvAt }
func (m Message) Data() []byte      { return m.data }
