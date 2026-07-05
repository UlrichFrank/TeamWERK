package duties

import (
	"context"
	"database/sql"
	"errors"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// matchReportDutyTypeName ist der name-Match für den Spielbericht-Duty-Type
// (Seed in Migration 020). Namens-Match statt ID, damit Prod- und Test-DBs
// mit unterschiedlichen IDs beide erkannt werden.
const matchReportDutyTypeName = "Spielbericht"

// errMatchReportRoleRequired wird von Claim zurückgegeben, wenn ein
// Nicht-Presseteam-User einen Spielbericht-Slot ziehen will.
var errMatchReportRoleRequired = errors.New("role_required")

// assertSlotTakePermitted verweigert das Ziehen eines Spielbericht-Slots
// durch einen User ohne role IN (presseteam, admin).
//
// Nicht-Spielbericht-Slots laufen ungehindert durch — die Funktion tut nichts.
// Wird per name-Lookup auf duty_types entschieden (siehe
// matchReportDutyTypeName-Kommentar).
func (h *Handler) assertSlotTakePermitted(ctx context.Context, slotID string, claims *auth.Claims) error {
	var typeName string
	err := h.db.QueryRowContext(ctx,
		`SELECT dt.name
		 FROM duty_slots ds
		 JOIN duty_types dt ON dt.id = ds.duty_type_id
		 WHERE ds.id = ?`,
		slotID,
	).Scan(&typeName)
	if errors.Is(err, sql.ErrNoRows) {
		// Kein Slot mit der ID — nächster Handler-Schritt (UPDATE) liefert 409.
		return nil
	}
	if err != nil {
		// Bei Query-Fehler sicherheitshalber nicht blockieren; der Haupt-Handler
		// wird gleich eh einen Fehler produzieren.
		return nil
	}
	if typeName != matchReportDutyTypeName {
		return nil
	}
	if claims == nil {
		return errMatchReportRoleRequired
	}
	if claims.Role == auth.RolePressTeam || claims.Role == auth.RoleAdmin {
		return nil
	}
	return errMatchReportRoleRequired
}
