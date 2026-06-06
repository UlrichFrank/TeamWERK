package hub

import "sync"

type EventHub struct {
	mu          sync.Mutex
	clients     map[chan string]struct{}
	userClients map[int]map[chan string]struct{}
}

func NewHub() *EventHub {
	return &EventHub{
		clients:     make(map[chan string]struct{}),
		userClients: make(map[int]map[chan string]struct{}),
	}
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

func (h *EventHub) SubscribeUser(userID int) chan string {
	ch := make(chan string, 4)
	h.mu.Lock()
	if h.userClients[userID] == nil {
		h.userClients[userID] = make(map[chan string]struct{})
	}
	h.userClients[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) UnsubscribeUser(userID int, ch chan string) {
	h.mu.Lock()
	if chans := h.userClients[userID]; chans != nil {
		delete(chans, ch)
		if len(chans) == 0 {
			delete(h.userClients, userID)
		}
	}
	h.mu.Unlock()
}

func (h *EventHub) BroadcastToUser(userID int, event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.userClients[userID] {
		select {
		case ch <- event:
		default:
		}
	}
}
