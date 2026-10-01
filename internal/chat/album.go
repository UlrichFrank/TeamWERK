package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/httpx"
)

// maxAlbumSize begrenzt die Bilder je Nachricht bzw. Mitteilung
// (chat-mehrere-bilder). Der CHECK auf message_media.position (0–9) erzwingt
// dieselbe Grenze in der DB.
const maxAlbumSize = 10

// codeInvalidMedia ist der einzige Fehlercode jeder Album-Verletzung. Bewusst
// einheitlich: der Server verrät nicht, ob eine fremde ID existiert, fremd ist
// oder schon benutzt wurde.
const codeInvalidMedia = "invalid_media"

// errInvalidMedia meldet eine Album-Verletzung (→ HTTP 400 invalid_media).
var errInvalidMedia = errors.New(codeInvalidMedia)

// albumFields sind die Bild-Felder im Request von SendMessage und
// SendBroadcast: mediaIds (1–10, Reihenfolge = Album) oder die Kurzform mediaId.
type albumFields struct {
	MediaID  *int  `json:"mediaId"`
	MediaIDs []int `json:"mediaIds"`
}

type rowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// resolveAlbum normalisiert die Bild-Felder zu einer geordneten ID-Liste
// (leer = kein Bild) und prüft mit einer Abfrage, dass jede ID existiert, vom
// Absender selbst hochgeladen wurde und noch an keiner Nachricht und keiner
// Mitteilung hängt. Jede Verletzung liefert errInvalidMedia; andere Fehler
// sind DB-Fehler.
//
// Die Eigentümerprüfung gilt auch für die Kurzform: vorher genügte die
// Existenz, und eine fremde media-ID an der eigenen Nachricht verschaffte über
// media.canSee Lesezugriff auf das Bild.
func resolveAlbum(ctx context.Context, q rowQueryer, userID int, in albumFields) ([]int, error) {
	if in.MediaID != nil && in.MediaIDs != nil {
		return nil, errInvalidMedia
	}
	ids := in.MediaIDs
	if in.MediaID != nil {
		ids = []int{*in.MediaID}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > maxAlbumSize {
		return nil, errInvalidMedia
	}
	seen := make(map[int]bool, len(ids))
	args := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		if seen[id] {
			return nil, errInvalidMedia
		}
		seen[id] = true
		args = append(args, id)
	}
	args = append(args, userID)

	var n int
	err := q.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM media med
		WHERE med.id IN (%s) AND med.uploaded_by = ?
		  AND NOT EXISTS (SELECT 1 FROM message_media mm WHERE mm.media_id = med.id)
		  AND NOT EXISTS (SELECT 1 FROM broadcast_media bm WHERE bm.media_id = med.id)`,
		strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")), args...).Scan(&n)
	if err != nil {
		return nil, err
	}
	if n != len(ids) {
		return nil, errInvalidMedia
	}
	return ids, nil
}

// imagePreview ist die Push-Vorschau einer Nachricht bzw. Mitteilung ohne Text.
func imagePreview(n int) string {
	if n == 1 {
		return "Bild"
	}
	return fmt.Sprintf("%d Bilder", n)
}

// mediaItem ist ein Albumbild in den Lese-Antworten.
type mediaItem struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Width  *int   `json:"width,omitempty"`
	Height *int   `json:"height,omitempty"`
}

// loadAlbums lädt mit einer Abfrage die Albumbilder aller übergebenen
// Nachrichten bzw. Mitteilungen in Positionsreihenfolge. table/keyCol
// sind feste Bezeichner (message_media/message_id, broadcast_media/broadcast_id),
// nie Nutzereingaben. Ein eigenes Nachladen statt eines JOINs im Haupt-SELECT,
// weil der JOIN die Zeilen vervielfachte und LIMIT-Paginierung bräche.
func loadAlbums(ctx context.Context, db *sql.DB, table, keyCol string, ids []int) (map[int][]mediaItem, error) {
	out := map[int][]mediaItem{}
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT x.%[2]s, x.media_id, med.width, med.height
		FROM %[1]s x JOIN media med ON med.id = x.media_id
		WHERE x.%[2]s IN (%[3]s)
		ORDER BY x.%[2]s, x.position`,
		table, keyCol, strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ownerID, mediaID int
		var width, height sql.NullInt64
		if err := rows.Scan(&ownerID, &mediaID, &width, &height); err != nil {
			return nil, err
		}
		item := mediaItem{ID: mediaID, URL: mediaURL(mediaID)}
		if width.Valid && height.Valid {
			w, h := int(width.Int64), int(height.Int64)
			item.Width, item.Height = &w, &h
		}
		out[ownerID] = append(out[ownerID], item)
	}
	return out, rows.Err()
}

func loadMessageMedia(ctx context.Context, db *sql.DB, ids []int) (map[int][]mediaItem, error) {
	return loadAlbums(ctx, db, "message_media", "message_id", ids)
}

func loadBroadcastMedia(ctx context.Context, db *sql.DB, ids []int) (map[int][]mediaItem, error) {
	return loadAlbums(ctx, db, "broadcast_media", "broadcast_id", ids)
}

// insertAlbum schreibt die Zuordnungszeilen eines Albums in der Transaktion
// des tragenden Inserts. Ein Unique-Verstoß (zwei Requests mit derselben ID im
// Rennen) lässt den Aufrufer die ganze Transaktion zurückrollen.
func insertAlbum(ctx context.Context, tx *sql.Tx, table, keyCol string, ownerID int64, ids []int) error {
	for pos, id := range ids {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			`INSERT INTO %s (%s, media_id, position) VALUES (?, ?, ?)`, table, keyCol),
			ownerID, id, pos); err != nil {
			return err
		}
	}
	return nil
}

// firstMedia ist der Wert für die Kompatibilitätsspalte media_id: das Bild auf
// Position 0 bzw. NULL ohne Bild (Invariante aus design.md §1).
func firstMedia(ids []int) sql.NullInt64 {
	if len(ids) == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(ids[0]), Valid: true}
}

// writeAlbumInsertError beantwortet einen Fehler beim Schreiben der
// Zuordnungszeilen: der Unique-Verstoß ist ein verlorenes Rennen um dieselbe
// media-ID (400 invalid_media), alles andere ein Serverfehler.
func writeAlbumInsertError(w http.ResponseWriter, r *http.Request, err error) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		httpx.WriteError(w, r, http.StatusBadRequest, codeInvalidMedia, err)
		return
	}
	httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
}
