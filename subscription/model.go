package subscription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	recvTopic string
	recvAt    time.Time
	data      []byte
}

type Model struct {
	Topic      string
	Qos        byte
	Format     string
	messages   chan []Message
	messagesMu *sync.Mutex
	messageIdx int
}

func NewModel(topic string, qos byte, format string) Model {
	s := Model{
		Topic:      topic,
		Qos:        qos,
		Format:     format,
		messages:   make(chan []Message, 1),
		messagesMu: &sync.Mutex{},
	}
	s.messages <- make([]Message, 0)
	return s
}

func (m Model) Title() string       { return m.Topic }
func (m Model) Description() string { return "" }
func (m Model) FilterValue() string { return m.Topic }

func (m Model) OnPubHandler(client mqtt.Client, msg mqtt.Message) {
	str := fmt.Sprintf("subscription: Pub received on %s; %s\n", msg.Topic(), string(msg.Payload()))
	os.WriteFile("mqttui.log", []byte(str), 0666)

	var data []byte
	data = msg.Payload()
	switch m.Format {
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

func (m Message) RecvTopic() string { return m.recvTopic }
func (m Message) RecvAt() time.Time { return m.recvAt }
func (m Message) Data() []byte      { return m.data }
