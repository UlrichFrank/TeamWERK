package policy

import (
	"strings"
	"time"
)

// RSVP-Rückmeldefristen: bis so lange vor Beginn dürfen Spieler und Eltern
// zu- oder absagen; Trainer/Vorstand/Admin können danach weiter pflegen
// (auth.Claims.CanOverrideRSVPCutoff). Liegen hier statt in games/trainings,
// weil auch attendance (Rückmelde-Matrix) sie kennen muss und Domänen sich
// nicht gegenseitig importieren.
const (
	TrainingRSVPCutoff = 2 * time.Hour
	GameRSVPCutoff     = 18 * time.Hour
)

// RSVPReasonMissing meldet eine Ab- oder Vielleicht-Sage ohne Grund auf einem
// Termin mit rsvp_require_reason. Die Oberfläche fragt den Grund per Dialog ab;
// diese Prüfung schließt den Weg an ihr vorbei (direkter API-Aufruf). Wer die
// Frist übergehen darf (auth.Claims.CanOverrideRSVPCutoff), pflegt die Liste
// für andere und ist wie dort ausgenommen — das prüft der Aufrufer.
func RSVPReasonMissing(requireReason bool, status, reason string) bool {
	return requireReason && (status == "declined" || status == "maybe") && strings.TrimSpace(reason) == ""
}
