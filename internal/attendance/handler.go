package attendance

import (
	"database/sql"

	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// Handler bündelt die HTTP-Endpoints für die Anwesenheits-Statistik.
type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

// NewHandler bindet DB und Event-Hub. Der Hub wird hier vorgehalten, damit
// spätere Mutationen (z.B. Pflege-Bestätigungen) Broadcasts auslösen können.
func NewHandler(db *sql.DB, h *hub.EventHub) *Handler {
	return &Handler{db: db, hub: h}
}
