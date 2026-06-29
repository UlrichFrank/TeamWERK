// Package attendance bündelt die aggregierte Anwesenheits-Statistik über
// Trainings und Spiele. Aufbau- und Berechtigungs-Details:
// openspec/changes/anwesenheits-statistik/design.md.
package attendance

// Category ist die Drei-Säulen-Klassifikation pro (Termin, Mitglied) plus
// ein Hilfswert für nicht erfassbare Datensätze (Datenloch).
type Category string

const (
	CategoryPresent  Category = "present"
	CategoryMissed   Category = "missed"
	CategoryExcused  Category = "excused"
	CategoryUnknown  Category = "unknown"
	CategoryCanceled Category = "cancelled"
)

// Classify ordnet einem Termin-/Mitgliedspaar genau eine Säule zu.
//
// Reihenfolge (siehe design.md D1):
//  1. attendance.present = 1                            → present
//  2. attendance.present = 0                            → missed
//  3. response.status = 'declined' AND absence_id ≠ ∅   → excused
//  4. sonst                                             → unknown (Datenloch)
//
// Liegt eine Attendance vor, überschreibt sie eine etwaige
// Auto-Decline-Response (D1: explizite Trainer-Erfassung gewinnt).
func Classify(present *bool, declined bool, hasAbsence bool) Category {
	if present != nil {
		if *present {
			return CategoryPresent
		}
		return CategoryMissed
	}
	if declined && hasAbsence {
		return CategoryExcused
	}
	return CategoryUnknown
}
