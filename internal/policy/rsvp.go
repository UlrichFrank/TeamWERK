package policy

import "time"

// RSVP-Rückmeldefristen: bis so lange vor Beginn dürfen Spieler und Eltern
// zu- oder absagen; Trainer/Vorstand/Admin können danach weiter pflegen
// (auth.Claims.CanOverrideRSVPCutoff). Liegen hier statt in games/trainings,
// weil auch attendance (Rückmelde-Matrix) sie kennen muss und Domänen sich
// nicht gegenseitig importieren.
const (
	TrainingRSVPCutoff = 2 * time.Hour
	GameRSVPCutoff     = 18 * time.Hour
)
