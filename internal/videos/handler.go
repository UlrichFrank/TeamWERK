// Package videos verwaltet selbst gehostete Spielvideos: Upload, Transcode,
// HLS-Streaming und Management. Dieses File enthält nur das Handler-Skelett;
// die einzelnen Routen/Worker liegen in den weiteren Files des Pakets.
package videos

import (
	"database/sql"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// Handler bündelt die Abhängigkeiten der Video-Routen.
type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
	cfg *appconfig.Config

	// remux ist die ffmpeg-Naht für GET /api/videos/{id}/download
	// (video-download-duty-upload); Produktion nutzt realFFmpegRemux, Tests
	// injizieren eine Fake analog zu Worker.transcode.
	remux remuxFunc
}

// NewHandler verdrahtet DB, Event-Hub und Config zu einem Video-Handler.
func NewHandler(db *sql.DB, h *hub.EventHub, cfg *appconfig.Config) *Handler {
	return &Handler{db: db, hub: h, cfg: cfg, remux: realFFmpegRemux}
}
