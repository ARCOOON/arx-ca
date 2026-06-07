package notifications

import (
	"github.com/google/uuid"
)

const sseClientBuffer = 16

type sseClient struct {
	id string
	ch chan []byte
}

// RegisterSSEClient adds a live Server-Sent Events subscriber and returns its event channel.
// Call the returned unregister function when the HTTP connection closes.
func (d *Dispatcher) RegisterSSEClient() (id string, events <-chan []byte, unregister func()) {
	if d == nil {
		ch := make(chan []byte)
		close(ch)
		return "", ch, func() {}
	}

	client := &sseClient{
		id: uuid.NewString(),
		ch: make(chan []byte, sseClientBuffer),
	}

	d.sseMu.Lock()
	if d.sseClients == nil {
		d.sseClients = make(map[string]*sseClient)
	}
	d.sseClients[client.id] = client
	d.sseMu.Unlock()

	return client.id, client.ch, func() {
		d.sseMu.Lock()
		delete(d.sseClients, client.id)
		d.sseMu.Unlock()
		close(client.ch)
	}
}

func (d *Dispatcher) broadcastSSE(payload []byte) {
	if d == nil || len(payload) == 0 {
		return
	}

	d.sseMu.RLock()
	clients := make([]*sseClient, 0, len(d.sseClients))
	for _, client := range d.sseClients {
		clients = append(clients, client)
	}
	d.sseMu.RUnlock()

	for _, client := range clients {
		select {
		case client.ch <- payload:
		default:
		}
	}
}
