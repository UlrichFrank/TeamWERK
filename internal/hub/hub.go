package hub

import "sync"

type EventHub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

func NewHub() *EventHub {
	return &EventHub{clients: make(map[chan string]struct{})}
}

func (h *EventHub) Subscribe() chan string {
	ch := make(chan string, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

func (h *EventHub) Broadcast(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
}
