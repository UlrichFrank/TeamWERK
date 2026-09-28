package trainings_test

// Begründungspflicht (rsvp_require_reason) serverseitig: bisher erzwang nur der
// Dialog der Oberfläche den Grund, ein direkter API-Aufruf kam ohne durch.

import (
	"net/http"
	"testing"
)

// Innerhalb der Frist, damit ausschließlich die Begründungsprüfung greift.
const reasonNow = "2026-06-13 18:00"

func countResponses(t *testing.T, f rsvpPushFixture) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id = ? AND member_id = ?`, f.sessionID, f.mPlayer).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestRespond_BegruendungFehlt_400(t *testing.T) {
	for _, tc := range []struct{ name, status, reason string }{
		{"Absage ohne Grund", "declined", ""},
		{"Vielleicht ohne Grund", "maybe", ""},
		{"Grund nur Leerzeichen", "declined", "   "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTeamRSVPPushFixture(t)
			if st := f.respond(t, reasonNow, f.playerTok, map[string]any{"status": tc.status, "reason": tc.reason}); st != http.StatusBadRequest {
				t.Fatalf("status %d, want 400", st)
			}
			if n := countResponses(t, f); n != 0 {
				t.Errorf("%d Antwortzeilen gespeichert, want 0", n)
			}
		})
	}
}

func TestRespond_BegruendungVorhanden_204(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	if st := f.respond(t, reasonNow, f.playerTok, map[string]any{"status": "declined", "reason": "krank"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	if n := countResponses(t, f); n != 1 {
		t.Errorf("%d Antwortzeilen, want 1", n)
	}
}

func TestRespond_ZusageBrauchtKeinenGrund_204(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	if st := f.respond(t, reasonNow, f.playerTok, map[string]any{"status": "confirmed"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
}

func TestRespond_OhnePflicht_AbsageOhneGrund_204(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	if _, err := f.db.Exec(`UPDATE training_sessions SET rsvp_require_reason = 0 WHERE id = ?`, f.sessionID); err != nil {
		t.Fatalf("update: %v", err)
	}
	if st := f.respond(t, reasonNow, f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
}

// Wer die Frist übergehen darf, pflegt die Liste für andere und ist wie dort
// von der Begründungspflicht ausgenommen.
func TestRespond_TrainerFuerSpielerOhneGrund_204(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	if st := f.respond(t, reasonNow, f.trainerTok, map[string]any{"status": "declined", "member_id": f.mPlayer}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
}
