package matchreports

// ConsentMember ist ein Mitglied ohne Foto-Freigabe, für den Warnhinweis
// im Bericht-Formular.
type ConsentMember struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// consentMissing liefert die Team-Mitglieder ohne Foto-Freigabe für ein Spiel.
// Weg: game_id → game_teams.team_id → team_memberships.member_id → members.
//
// Consent ist die dedizierte DSGVO-Einwilligung members.foto_veroeffentlichung
// („Fotos dürfen auf öffentlichen Kanälen des Vereins veröffentlicht werden").
//
// Mitglieder mit foto_veroeffentlichung=0 landen in der Liste. Fehler werden
// geloggt, Rückgabe ist im Fehlerfall leer (der Warnhinweis fehlt dann, aber der
// Report bleibt anzeigbar).
func (h *Handler) consentMissing(gameID int) []ConsentMember {
	rows, err := h.db.Query(
		`SELECT DISTINCT m.first_name, m.last_name
		 FROM game_teams gt
		 JOIN team_memberships tm ON tm.team_id = gt.team_id
		 JOIN members m ON m.id = tm.member_id
		 WHERE gt.game_id = ?
		   AND COALESCE(m.foto_veroeffentlichung, 0) = 0
		 ORDER BY m.last_name, m.first_name`,
		gameID,
	)
	if err != nil {
		logErr("matchreports.consentMissing", err, "game", gameID)
		return nil
	}
	defer rows.Close()

	var out []ConsentMember
	for rows.Next() {
		var m ConsentMember
		if err := rows.Scan(&m.FirstName, &m.LastName); err != nil {
			logErr("matchreports.consentMissing scan", err, "game", gameID)
			return out
		}
		out = append(out, m)
	}
	return out
}
