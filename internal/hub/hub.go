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

// Broadcast delivers event to every connected client — both legacy global
// subscribers (h.clients) and all per-user subscribers (h.userClients). Since
// /api/events now subscribes per user (SubscribeUser), global topics
// (venues/settings/beitragssatz-changed/stammvereine) must reach the per-user
// streams too; iterating both maps keeps those vereinsweiten Topics global.
func (h *EventHub) Broadcast(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
	for _, chans := range h.userClients {
		for ch := range chans {
			select {
			case ch <- event:
			default:
			}
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

// SubscribedUserCount returns the number of distinct users with at least one
// active per-user subscription. Used by tests to await stream registration.
func (h *EventHub) SubscribedUserCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.userClients)
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

// BroadcastToUsers delivers event to the per-user streams of the given user IDs
// (adressierter Domänen-Event-Versand). It is non-blocking per channel — a full
// buffer drops the event for that channel exactly like Broadcast/BroadcastToUser.
// Duplicate IDs are deduplicated so a user appearing twice in the audience
// receives the event only once per stream. A user without an active
// /api/events subscription is silently skipped.
func (h *EventHub) BroadcastToUsers(userIDs []int, event string) {
	if len(userIDs) == 0 {
		return
	}
	seen := make(map[int]struct{}, len(userIDs))
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, uid := range userIDs {
		if _, dup := seen[uid]; dup {
			continue
		}
		seen[uid] = struct{}{}
		for ch := range h.userClients[uid] {
			select {
			case ch <- event:
			default:
			}
		}
	}
}
