package events

import (
	"sync"
)

// Event is a named occurrence with an arbitrary structured payload.
type Event struct {
	Name    string
	Payload map[string]any
}

// Handler processes a single event. Handlers must not block on I/O; offload work to goroutines when needed.
type Handler func(Event)

// Manager is a thread-safe pub/sub event bus.
type Manager struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

// NewManager constructs an empty event manager.
func NewManager() *Manager {
	return &Manager{
		subscribers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for a specific event name.
func (m *Manager) Subscribe(name string, handler Handler) {
	if m == nil || handler == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[name] = append(m.subscribers[name], handler)
}

// Trigger dispatches an event to all subscribers registered for name.
func (m *Manager) Trigger(name string, payload map[string]any) {
	if m == nil {
		return
	}
	m.mu.RLock()
	handlers := append([]Handler(nil), m.subscribers[name]...)
	m.mu.RUnlock()

	evt := Event{Name: name, Payload: payload}
	for _, handler := range handlers {
		handler(evt)
	}
}
