package kader

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/h4aimport"
)

// validateStaffel prüft einen Staffelcode gegen den Kader.
//
// Rückgabe 0 heißt "in Ordnung". Sonst HTTP-Status und Meldung:
//   - 404 Kader existiert nicht
//   - 409 Übungsgruppe — sie tritt in keiner Staffel an; die Einschränkung
//     steht hier statt als CHECK, weil SQLite dafür einen Tabellen-Rebuild
//     bräuchte (Migration 068)
//   - 400 Code nicht interpretierbar oder unpassend zu Geschlecht/Altersklasse
//
// Ein leerer Code entfernt die Zuordnung und ist immer zulässig — außer an
// einer Übungsgruppe, die nie eine hatte.
func (h *Handler) validateStaffel(ctx context.Context, kaderID, code string) (int, string) {
	var kind string
	var gender, ageClass sql.NullString
	err := h.db.QueryRowContext(ctx,
		`SELECT kind, gender, age_class FROM kader WHERE id = ?`, kaderID).
		Scan(&kind, &gender, &ageClass)
	switch {
	case err == sql.ErrNoRows:
		return http.StatusNotFound, "Kader nicht gefunden"
	case err != nil:
		return http.StatusInternalServerError, "interner Fehler"
	}
	if kind == "practice" {
		return http.StatusConflict, "Übungsgruppen treten in keiner Staffel an"
	}

	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return 0, ""
	}

	// ParseStaffel zerlegt den Code in Geschlecht und Altersklasse. Ein Code,
	// dessen Aufbau nicht erkennbar ist, wird abgelehnt statt ungeprüft
	// übernommen — sonst liefe der Abruf später still ins Leere.
	g, ac, ok := h4aimport.ParseStaffel(trimmed)
	if !ok {
		return http.StatusBadRequest, "Staffelcode nicht interpretierbar (erwartet z.B. mB-RL-BW)"
	}
	if gender.Valid && gender.String != "" && gender.String != g {
		return http.StatusBadRequest, "Staffelcode passt nicht zum Geschlecht des Kaders"
	}
	if ageClass.Valid && ageClass.String != "" && ageClass.String != ac {
		return http.StatusBadRequest, "Staffelcode passt nicht zur Altersklasse des Kaders"
	}
	return 0, ""
}

// nullIfEmpty macht aus einem leeren Code ein SQL-NULL, damit "keine Staffel"
// nicht als leerer String gespeichert wird und die Abfragen zwei Fälle kennen.
func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}
