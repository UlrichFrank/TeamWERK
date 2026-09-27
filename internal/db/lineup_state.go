package db

import "strings"

// Codes des dreiwertigen Aufstellungsstatus eines Spielers für ein Spiel
// (Change aufstellung-status-termine). Die API liefert sie unverändert aus,
// das Frontend (web/src/lib/lineup.ts) und der iCal-Feed übersetzen sie in
// „aufgestellt" / „nicht aufgestellt" / „Aufstellung offen".
const (
	LineupIn   = "in"
	LineupOut  = "out"
	LineupOpen = "open"
)

// LineupStateSQL liefert einen SQL-Ausdruck, der für ein Spiel und ein
// Mitglied den Aufstellungsstatus ergibt — `'in'`, `'out'`, `'open'` oder
// NULL, wenn der Termin keine Aufstellung kennt (alles außer heim/auswärts).
//
// Die Regel „keine Zeile für das Spiel ist offen, nie nicht aufgestellt" lebt
// allein hier: an mehreren Stellen ausgewertet würde aus einer ungepflegten
// Aufstellung früher oder später eine Absage, die niemand ausgesprochen hat.
// Ob die Zeile überhaupt ein Spieler ist (und kein Trainer), entscheidet der
// Aufrufer — nur er kennt die Kader-Beziehung.
//
// Die drei Argumente sind SQL-Ausdrücke, gedacht für Spaltenreferenzen
// (`g.event_type`, `g.id`, `m.id`); gameIDExpr kommt zweimal vor.
func LineupStateSQL(eventTypeExpr, gameIDExpr, memberIDExpr string) string {
	r := strings.NewReplacer("{E}", eventTypeExpr, "{G}", gameIDExpr, "{M}", memberIDExpr)
	return r.Replace(`(CASE
		WHEN {E} NOT IN ('heim', 'auswärts') THEN NULL
		WHEN EXISTS (SELECT 1 FROM game_lineup gl_s WHERE gl_s.game_id = {G} AND gl_s.member_id = {M}) THEN 'in'
		WHEN EXISTS (SELECT 1 FROM game_lineup gl_s WHERE gl_s.game_id = {G}) THEN 'out'
		ELSE 'open' END)`)
}

// ResolveLineupState ist dieselbe Regel für Aufrufer, die die Aufstellung
// bereits geladen haben (Rückmelde-Matrix): lineupExists = mindestens eine
// Zeile für das Spiel, inLineup = eine Zeile für dieses Mitglied. Liefert ""
// für Termine ohne Aufstellung. TestLineupStateSQL prüft beide Formen auf
// denselben Fakten.
func ResolveLineupState(eventType string, lineupExists, inLineup bool) string {
	if eventType != "heim" && eventType != "auswärts" {
		return ""
	}
	return LineupFromFacts(lineupExists, inLineup)
}

// LineupFromFacts ist die Regel ohne Typ-Gate. Die Detailseite zeigt die
// Aufstellung auch bei generischen Terminen („Sonstiges"), Liste, Tabelle und
// Kalender-Abo dagegen nur bei Heim-/Auswärtsspielen.
func LineupFromFacts(lineupExists, inLineup bool) string {
	switch {
	case inLineup:
		return LineupIn
	case lineupExists:
		return LineupOut
	default:
		return LineupOpen
	}
}
