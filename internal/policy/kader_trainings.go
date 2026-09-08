package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
)

// KaderTrainingCount zählt Trainingstermine und -serien eines Kaders.
//
// Beide Löschpfade brauchen dieselbe Zahl: DELETE /api/kader/{id}
// (Mannschaftsvariante) und DELETE /api/practice-groups/{id} (Übungsgruppe).
// Sie liegt hier statt in einem der beiden Domänen-Pakete, weil die zwei sich
// gegenseitig nicht importieren dürfen — und weil die Zusage („Trainingshistorie
// überlebt jedes Löschen") für beide Varianten dieselbe ist.
func KaderTrainingCount(ctx context.Context, db *sql.DB, kaderID int) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM training_sessions WHERE kader_id=?)
		     + (SELECT COUNT(*) FROM training_series   WHERE kader_id=?)`,
		kaderID, kaderID).Scan(&n)
	return n, err
}

// WriteKaderTrainingConflict schreibt die gemeinsame 409-Antwort der
// Trainings-Guard. Dieselbe Gestalt wie die bestehende Mitglieder-Guard in
// kader.DeleteKader: ein sprechender `error` plus die blockierende Zahl.
func WriteKaderTrainingConflict(w http.ResponseWriter, count int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"error":          "Kader hat noch Trainingstermine",
		"training_count": count,
	})
}
