package videos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// renditionRe erlaubt nur Bezeichner wie "720p"/"360p" — verhindert
// Path-Traversal über den {rendition}-Pfadsegment.
var renditionRe = regexp.MustCompile(`^[0-9]{3,4}p$`)

// segmentRe erlaubt index.m3u8 sowie Segmentdateien seg_NNN.ts — verhindert
// Path-Traversal über den {segment}-Pfadsegment.
var segmentRe = regexp.MustCompile(`^(index\.m3u8|seg_[0-9]{1,6}\.ts)$`)

// streamUIDKey transportiert die aus dem Stream-Token gewonnene uid für Logging.
type streamUIDKey struct{}

// loadVideoForView lädt die für die Berechtigungsprüfung nötige Teilmenge eines
// Videos plus `duration_sec` für die dauerabhängige Stream-Token-TTL. Liefert
// (nil, nil) wenn das Video nicht existiert. TeamIDs wird aus video_teams
// befüllt, sodass CanViewVideo alle zugeordneten Teams prüfen kann.
func (h *Handler) loadVideoForView(id int) (*Video, error) {
	v := &Video{ID: id}
	var duration sql.NullInt64
	err := h.db.QueryRow(
		`SELECT team_id, duration_sec FROM videos WHERE id = ?`, id,
	).Scan(&v.TeamID, &duration)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if duration.Valid {
		v.DurationSec = duration.Int64
	}
	rows, err := h.db.Query(`SELECT team_id FROM video_teams WHERE video_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tid int
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		v.TeamIDs = append(v.TeamIDs, tid)
	}
	return v, rows.Err()
}

// Play liefert einen kurzlebigen Stream-Token + die Master-Playlist-URL aus.
// GET /api/videos/{id}/play (Authenticated-Tier).
func (h *Handler) Play(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	video, err := h.loadVideoForView(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ok, err := h.CanViewVideo(claims, video)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	token, err := h.Sign(id, claims.UserID, int(video.DurationSec))
	if err != nil {
		// Fehlt das Secret (Fehlkonfiguration), ist Streaming nicht verfügbar.
		http.Error(w, "streaming unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":      token,
		"master_url": "/api/videos/" + strconv.Itoa(id) + "/hls/master.m3u8",
	})
}

// StreamTokenMiddleware schützt die HLS-Routen: kein JWT, sondern Verifikation
// des ?st=-Tokens gegen das {id}-Pfadsegment. Bei Fehler 403. CORS-Preflight
// (OPTIONS) läuft ungeprüft durch — der Preflight ist per HTTP-Definition
// credential-frei, Auth erfolgt beim eigentlichen GET-Follow-up.
func (h *Handler) StreamTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		token := r.URL.Query().Get("st")
		if token == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		uid, err := h.Verify(token, id)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), streamUIDKey{}, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// renditionLinePrefix matcht eine Rendition-Playlist-Referenz im Master-Manifest
// (z.B. "720p/index.m3u8"). master.m3u8 referenziert Renditions relativ.
var renditionLinePrefix = regexp.MustCompile(`^[0-9]{3,4}p/index\.m3u8$`)

// ServeMaster liefert master.m3u8 aus und hängt an jede referenzierte
// Rendition-Playlist-URL den eingehenden ?st=-Token an, damit hls.js ihn
// weiterträgt. GET /api/videos/{id}/hls/master.m3u8 (Token-geschützt).
func (h *Handler) ServeMaster(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	path := MasterManifestPath(h.cfg.VideoStorageDir, id)
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	token := r.URL.Query().Get("st")
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if renditionLinePrefix.MatchString(trimmed) {
			lines[i] = trimmed + "?st=" + token
		}
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	setHLSCORSHeaders(w)
	w.Write([]byte(strings.Join(lines, "\n")))
}

// setHLSCORSHeaders erlaubt Chromecast-Default-Receivern das Cross-Origin-Laden
// der Manifeste/Segmente. Auth bleibt allein der `?st=`-Token — es werden keine
// Credentials/Cookies gesendet oder akzeptiert. Nur GET (HEAD implizit) ist
// erlaubt; PUT/POST/DELETE gibt es auf diesen Routen ohnehin nicht.
func setHLSCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
}

// HLSPreflight beantwortet CORS-Preflight-Requests auf den HLS-Routen mit 204
// und den gleichen `Access-Control-Allow-*`-Headern wie die GET-Handler. Manche
// Chromecast-Firmwares schicken vor dem eigentlichen Manifest-Fetch ein OPTIONS
// voraus; ohne diese Antwort verweigert der Default-Receiver das Playback.
func (h *Handler) HLSPreflight(w http.ResponseWriter, _ *http.Request) {
	setHLSCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// ServeRenditionFile liefert die index.m3u8 oder ein Segment einer Rendition aus.
// Range-Support + ETag via http.ServeContent. Beide Pfadsegmente werden strikt
// gegen Allowlists geprüft (Path-Traversal-Schutz).
// GET /api/videos/{id}/hls/{rendition}/{segment} (Token-geschützt).
func (h *Handler) ServeRenditionFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	rendition := chi.URLParam(r, "rendition")
	segment := chi.URLParam(r, "segment")
	if !renditionRe.MatchString(rendition) || !segmentRe.MatchString(segment) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	dir := RenditionDir(h.cfg.VideoStorageDir, id, rendition)
	full := filepath.Join(dir, segment)
	// Defense-in-depth: der aufgelöste Pfad muss unterhalb des Rendition-Dirs
	// bleiben (auch wenn die Regexe das bereits garantieren).
	if !strings.HasPrefix(filepath.Clean(full), filepath.Clean(dir)+string(os.PathSeparator)) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if strings.HasSuffix(segment, ".m3u8") {
		// Rendition-Playlist: Segment-Referenzen on-the-fly mit ?st= versehen,
		// damit hls.js die Token-geschützten .ts-Endpunkte erreicht. Die Datei
		// auf Disk listet seg_NNN.ts ohne Query — ohne diesen Rewrite scheitert
		// jeder Segment-GET an der StreamTokenMiddleware mit 403.
		data, err := os.ReadFile(full)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		token := r.URL.Query().Get("st")
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if segmentRe.MatchString(trimmed) && strings.HasSuffix(trimmed, ".ts") {
				lines[i] = trimmed + "?st=" + token
			}
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		setHLSCORSHeaders(w)
		w.Write([]byte(strings.Join(lines, "\n")))
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	setHLSCORSHeaders(w)
	// http.ServeContent übernimmt Range (206), If-Range und ETag-Handling.
	http.ServeContent(w, r, segment, info.ModTime(), f)
}
