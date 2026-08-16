// Package eventlog hält fest, was einem Nutzer mitgeteilt wurde.
//
// Geschrieben wird ausschließlich aus notify.Send, und zwar aus der
// UNGEFILTERTEN Empfängerliste — bevor Push- und Email-Präferenzen ausgewertet
// werden. Push ist an sechs unabhängigen Stellen verlustbehaftet (Präferenz,
// fehlender VAPID-Key, fehlende Subscription, HTTP 410, Transportfehler und
// vor allem TTL=3600: ein Gerät, das einen Abend offline ist, verliert die
// Meldung). Der Log ist deshalb der vollständige Kanal, Push der Beschleuniger.
//
// Gelesen wird im Dashboard, aufgeräumt im Scheduler.
package eventlog

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// Event ist eine Log-Zeile aus Sicht des Empfängers.
type Event struct {
	ID        int    `json:"id"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
}

// defaultLimit ist die Deckelung des Dashboard-Reads. Der Log ist eine
// Übersicht der letzten Tage, kein Archiv.
const defaultLimit = 30

// retentionDays ist die Aufbewahrung NACH der ersten Ansicht.
const retentionDays = 3

// unseenCapDays ist eine Betriebs-Sicherung, keine fachliche Regel: ungesehene
// Zeilen bleiben grundsätzlich liegen ("wer im Urlaub war, verliert nichts"),
// aber ein Account, der nie wieder eingeloggt wird (ausgetretenes Mitglied,
// verwaister Kinder-Account), sammelte sonst unbegrenzt Zeilen. 90 Tage liegen
// weit oberhalb jeder plausiblen Abwesenheit — wer so lange weg war, hat nicht
// den Log verloren, sondern den Bezug zur Saison.
const unseenCapDays = 90

// Record schreibt je Empfänger eine Zeile.
//
// Ein Vereins-Fan-out sind bis zu ~180 Empfänger, deshalb ein einziges INSERT
// mit Multi-Row-VALUES statt N Einzel-Statements. Kein Chunking: fünf
// Bind-Parameter je Zeile gegen SQLITE_MAX_VARIABLE_NUMBER (32766 seit SQLite
// 3.32, von modernc.org/sqlite geerbt) reichen für ~6500 Empfänger — nachgemessen
// bis 2000. Ein Verein, der das reißt, hat andere Probleme.
//
// Fehler werden geloggt, nicht zurückgegeben: der Log ist nachrangig gegenüber
// dem Versand. Eine kaputte Log-Zeile darf keine Benachrichtigung verhindern.
func Record(db *sql.DB, userIDs []int, category, title, body, url string) {
	if len(userIDs) == 0 {
		return
	}

	rows := make([]string, len(userIDs))
	args := make([]any, 0, len(userIDs)*5)
	for i, uid := range userIDs {
		rows[i] = "(?,?,?,?,?)"
		args = append(args, uid, category, title, body, url)
	}

	query := `INSERT INTO user_events (user_id, category, title, body, url) VALUES ` +
		strings.Join(rows, ",")

	if _, err := db.Exec(query, args...); err != nil {
		slog.Error("eventlog record failed",
			"category", category, "recipients", len(userIDs), "error", err)
	}
}

// Queryer ist die schmale Teilmenge, die ListForUser braucht — erlaubt den
// Aufruf innerhalb einer Transaktion (*sql.Tx) wie außerhalb (*sql.DB), damit
// Read und Stempel (MarkSeen) im selben tx laufen können (design.md
// Decision 4).
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ListForUser liefert die jüngsten Zeilen eines Nutzers, neueste zuerst.
//
// Sekundär nach id, weil SQLites CURRENT_TIMESTAMP nur Sekundenauflösung hat:
// ein Fan-out und eine unmittelbar folgende zweite Meldung teilen sich sonst
// den Zeitstempel und die Reihenfolge wäre nicht deterministisch.
//
// Es wird NICHT gegen den aktuellen Kader oder die aktuelle Vereinsfunktion
// nachgefiltert. Der Log protokolliert, was gesendet wurde — nicht, was heute
// gelten würde. Eine Nachfilterung löge in beide Richtungen: sie verstecke
// Meldungen vor Nutzern, die sie nachweislich erhalten haben, und zeigte neu
// Hinzugekommenen rückwirkend Meldungen, die nie an sie gingen.
func ListForUser(ctx context.Context, db Queryer, userID, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	rows, err := db.QueryContext(ctx,
		`SELECT id, category, title, body, url, created_at
		   FROM user_events
		  WHERE user_id = ?
		  ORDER BY created_at DESC, id DESC
		  LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Category, &e.Title, &e.Body, &e.URL, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// DefaultLimit ist die Deckelung, mit der das Dashboard liest. Exportiert,
// damit Aufrufer und Tests dieselbe Zahl benutzen.
func DefaultLimit() int { return defaultLimit }

// Execer ist die schmale Teilmenge, die MarkSeen braucht — erlaubt den Aufruf
// innerhalb einer Transaktion (*sql.Tx) wie außerhalb (*sql.DB).
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// MarkSeen startet die Retention-Uhr für die übergebenen Zeilen.
//
// Nimmt bewusst IDs und KEINE user_id: der Dashboard-Read ist auf 30 Zeilen
// gedeckelt. Ein `WHERE user_id = ?` würde auch die Zeilen 31+ stempeln, die
// der Nutzer nie zu sehen bekam — sie verfielen ungesehen. Gestempelt wird
// genau, was ausgeliefert wurde.
//
// `AND seen_at IS NULL` sorgt dafür, dass wiederholtes Laden die Uhr nicht
// nach hinten schiebt.
func MarkSeen(ctx context.Context, ex Execer, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	args := make([]any, len(ids))
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		placeholders[i] = "?"
	}

	_, err := ex.ExecContext(ctx,
		`UPDATE user_events SET seen_at = CURRENT_TIMESTAMP
		  WHERE id IN (`+strings.Join(placeholders, ",")+`)
		    AND seen_at IS NULL`, args...)
	return err
}

// Purge löscht abgelaufene Zeilen und liefert die Anzahl.
//
// Zwei Zweige in einer Anweisung: gesehen + 3 Tage (die fachliche Regel) und
// ungesehen + 90 Tage (die Betriebs-Sicherung, siehe unseenCapDays).
//
// Ohne Idempotenzschutz — ein DELETE ist von Natur aus wiederholbar.
func Purge(db *sql.DB) (int64, error) {
	res, err := db.Exec(fmt.Sprintf(
		`DELETE FROM user_events
		  WHERE (seen_at IS NOT NULL AND seen_at    < datetime('now','-%d days'))
		     OR (seen_at IS     NULL AND created_at < datetime('now','-%d days'))`,
		retentionDays, unseenCapDays))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
