// Package trainingdiary ist das Trainingstagebuch: von Spielern selbst
// erfasste Trainingseinheiten außerhalb des Mannschaftstrainings, mit
// optionalem Nachweis.
//
// Bewusst getrennt von `internal/trainings` — dort liegen terminierte
// Mannschaftseinheiten mit RSVP und Trainer-erfasster Anwesenheit. Hier gibt es
// keinen Termin, keine Zu-/Absage und keine Fremderfassung: Schreibrecht hat
// ausschließlich der Eigentümer, Trainer und Eltern lesen nur.
package trainingdiary

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// diaryEvent ist das einzige Live-Update-Ereignis dieses Packages. Bewusst
// payload-frei und global: die Nutzlast wäre privat, ein nackter String ist es
// nicht — jeder Client lädt nach Empfang nur das nach, was seine ACL hergibt.
const diaryEvent = "training-diary-changed"

// kinds ist der feste Artenkatalog. Deckungsgleich mit dem CHECK-Constraint in
// Migration 040 — eine achte Art ist deshalb eine Migration, kein Config-Eintrag.
var kinds = map[string]bool{
	"kraft": true, "ausdauer": true, "athletik": true, "technik": true,
	"beweglichkeit": true, "reha": true, "sonstiges": true,
}

const (
	maxCustomKindLen = 60
	maxNoteLen       = 500
	minDurationMin   = 1
	maxDurationMin   = 600
	minRPE           = 1
	maxRPE           = 10
)

// Handler bündelt die HTTP-Endpoints des Trainingstagebuchs.
type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
	dir string
}

// NewHandler bindet DB, Event-Hub und das Verzeichnis der Nachweis-Dateien.
// Das Verzeichnis wird wie bei media.NewHandler beim Start angelegt — schlägt
// das fehl, ist die Installation kaputt und der Prozess soll nicht starten.
func NewHandler(db *sql.DB, h *hub.EventHub, dir string) *Handler {
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(fmt.Sprintf("trainingdiary: cannot create storage dir %s: %v", dir, err))
	}
	return &Handler{db: db, hub: h, dir: dir}
}

// entry ist die API-Repräsentation eines Tagebuch-Eintrags.
type entry struct {
	ID          int     `json:"id"`
	MemberID    int     `json:"member_id"`
	SeasonID    *int    `json:"season_id"`
	TrainedOn   string  `json:"trained_on"`
	Kind        string  `json:"kind"`
	KindCustom  *string `json:"kind_custom"`
	DurationMin int     `json:"duration_min"`
	RPE         int     `json:"rpe"`
	Note        *string `json:"note"`
	// ProofStatus ist "none" (nie einer da), "present" (abrufbar) oder
	// "purged" (von der Retention gelöscht). Das Frontend unterscheidet daran
	// „kein Nachweis" von „Nachweis gelöscht" und vermeidet im zweiten Fall
	// einen Bildabruf, der zwangsläufig 410 liefern würde.
	ProofStatus   string  `json:"proof_status"`
	ProofMime     *string `json:"proof_mime"`
	ProofPurgedAt *string `json:"proof_purged_at"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// entryRequest ist der Request-Body für Anlegen und Ändern.
type entryRequest struct {
	TrainedOn   string  `json:"trained_on"`
	Kind        string  `json:"kind"`
	KindCustom  *string `json:"kind_custom"`
	DurationMin int     `json:"duration_min"`
	RPE         int     `json:"rpe"`
	Note        *string `json:"note"`
}

// validated ist das Ergebnis von validateEntry — normalisierte, prüfbare Werte.
type validated struct {
	trainedOn   string
	kind        string
	kindCustom  any
	durationMin int
	rpe         int
	note        any
}

// validateEntry prüft den Request gegen die fachlichen Regeln und normalisiert
// ihn. Die DB-CHECK-Constraints sind der Backstop; hier entstehen die
// sprechenden Fehlermeldungen.
func validateEntry(req entryRequest) (validated, error) {
	var v validated

	day := strings.TrimSpace(req.TrainedOn)
	if len(day) > 10 {
		// Toleriert einen ISO-Timestamp aus dem Client (vgl. Gotcha
		// „SQLite DATE-Felder"), speichert aber immer nur das Datum.
		day = day[:10]
	}
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return v, fmt.Errorf("trained_on muss ein Datum im Format JJJJ-MM-TT sein")
	}
	// Zukunft ausschließen. Vergleich auf Tagesebene in lokaler Zeit — ein
	// Spieler, der abends um 23:00 seine Einheit einträgt, soll den heutigen
	// Tag wählen dürfen.
	today := time.Now().Format("2006-01-02")
	if t.Format("2006-01-02") > today {
		return v, fmt.Errorf("trained_on darf nicht in der Zukunft liegen")
	}
	v.trainedOn = t.Format("2006-01-02")

	if !kinds[req.Kind] {
		return v, fmt.Errorf("kind ist unbekannt")
	}
	v.kind = req.Kind

	// kind_custom ist genau dann gesetzt, wenn kind='sonstiges'. Bei jeder
	// anderen Art wird ein mitgeschickter Wert verworfen statt abgelehnt —
	// der Client soll das Feld nicht leeren müssen, wenn der Nutzer die Art
	// im Formular wechselt.
	if req.Kind == "sonstiges" {
		custom := ""
		if req.KindCustom != nil {
			custom = strings.TrimSpace(*req.KindCustom)
		}
		if custom == "" {
			return v, fmt.Errorf("kind_custom ist bei kind='sonstiges' erforderlich")
		}
		if len([]rune(custom)) > maxCustomKindLen {
			return v, fmt.Errorf("kind_custom darf höchstens %d Zeichen haben", maxCustomKindLen)
		}
		v.kindCustom = custom
	} else {
		v.kindCustom = nil
	}

	if req.DurationMin < minDurationMin || req.DurationMin > maxDurationMin {
		return v, fmt.Errorf("duration_min muss zwischen %d und %d liegen", minDurationMin, maxDurationMin)
	}
	v.durationMin = req.DurationMin

	if req.RPE < minRPE || req.RPE > maxRPE {
		return v, fmt.Errorf("rpe muss zwischen %d und %d liegen", minRPE, maxRPE)
	}
	v.rpe = req.RPE

	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		if len([]rune(note)) > maxNoteLen {
			return v, fmt.Errorf("note darf höchstens %d Zeichen haben", maxNoteLen)
		}
		if note != "" {
			v.note = note
		}
	}

	return v, nil
}

// entryColumns ist die gemeinsame Spaltenliste aller SELECTs, damit scanEntry
// überall dieselbe Reihenfolge erwarten darf.
const entryColumns = `id, member_id, season_id, trained_on, kind, kind_custom,
	duration_min, rpe, note, proof_disk_name, proof_mime, proof_purged_at,
	created_at, updated_at`

// scanner deckt *sql.Row und *sql.Rows ab.
type scanner interface {
	Scan(dest ...any) error
}

// scanEntry liest eine Zeile in die API-Repräsentation und leitet dabei
// proof_status aus den beiden Nachweis-Spalten ab.
func scanEntry(s scanner) (entry, error) {
	var e entry
	var seasonID sql.NullInt64
	var kindCustom, note, proofDisk, proofMime, proofPurged sql.NullString

	err := s.Scan(&e.ID, &e.MemberID, &seasonID, &e.TrainedOn, &e.Kind, &kindCustom,
		&e.DurationMin, &e.RPE, &note, &proofDisk, &proofMime, &proofPurged,
		&e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return e, err
	}

	if seasonID.Valid {
		id := int(seasonID.Int64)
		e.SeasonID = &id
	}
	if kindCustom.Valid && kindCustom.String != "" {
		e.KindCustom = &kindCustom.String
	}
	if note.Valid && note.String != "" {
		e.Note = &note.String
	}

	switch {
	case proofDisk.Valid && proofDisk.String != "":
		e.ProofStatus = "present"
		if proofMime.Valid {
			e.ProofMime = &proofMime.String
		}
	case proofPurged.Valid && proofPurged.String != "":
		e.ProofStatus = "purged"
		e.ProofPurgedAt = &proofPurged.String
	default:
		e.ProofStatus = "none"
	}

	return e, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
