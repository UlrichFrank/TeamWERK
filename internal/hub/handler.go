package hub

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Handler struct {
	hub       *EventHub
	buildHash string
	// userIDFromCtx extracts the authenticated user's ID from the request
	// context. It is injected by the composition root (main.go) rather than
	// imported directly, because hub is a FOUNDATION package and auth already
	// imports hub — importing auth back would create a cycle and violate the
	// architecture test. Returns 0 when no authenticated user is present.
	userIDFromCtx func(context.Context) int
}

// NewHandler wires the SSE handler. userIDFromCtx maps the request context to
// the authenticated user's ID (typically auth.UserIDFromCtx); it enables the
// per-user subscription that makes domain events adressierbar.
func NewHandler(h *EventHub, buildHash string, userIDFromCtx func(context.Context) int) *Handler {
	return &Handler{hub: h, buildHash: buildHash, userIDFromCtx: userIDFromCtx}
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Per-user subscription: the CookieMiddleware has authenticated the request
	// and placed the user's ID in the context. Subscribing per user (like the
	// chat stream) makes domain events adressierbar via BroadcastToUsers, while
	// globally-scoped topics still reach everyone via Broadcast → all userClients.
	userID := h.userIDFromCtx(r.Context())
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ch := h.hub.SubscribeUser(userID)
	defer h.hub.UnsubscribeUser(userID, ch)

	fmt.Fprintf(w, "data: __version:%s\n\n", h.buildHash)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
