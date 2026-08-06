package h4aimport

import (
	"regexp"
	"strconv"
	"strings"
)

// reStaffelPrefix zerlegt das Präfix eines BWHV-Staffelcodes: ein Geschlechts-
// Buchstabe gefolgt von der Altersklasse, z.B. "mC-OL-3-BW", "gD-BOL-SRM".
// Der Rest des Codes (Liga, Bezirk, Nummer) ist für die Mannschaftszuordnung
// ohne Belang und wird bewusst nicht interpretiert.
var reStaffelPrefix = regexp.MustCompile(`^([mwg])([A-E])(?:[-.]|$)`)

// staffelGender bildet den H4A-Geschlechtsbuchstaben auf teams.gender ab.
// "w" (weiblich) heißt in der DB "f" — der einzige nicht-offensichtliche Fall.
var staffelGender = map[string]string{"m": "m", "w": "f", "g": "mixed"}

// ParseStaffel zerlegt einen Staffelcode in teams.gender und teams.age_class.
// Codes, die dem Muster nicht folgen (Turniere, Sonderrunden), liefern ok=false —
// solche Zeilen bleiben ohne Vorschlag und damit Handarbeit, statt geraten zu werden.
func ParseStaffel(staffel string) (gender, ageClass string, ok bool) {
	m := reStaffelPrefix.FindStringSubmatch(strings.TrimSpace(staffel))
	if m == nil {
		return "", "", false
	}
	g, known := staffelGender[m[1]]
	if !known {
		return "", "", false
	}
	return g, m[2] + "-Jugend", true
}

// reLeagueToken schneidet das Liga-Kürzel hinter dem Geschlecht/Altersklasse-Präfix
// heraus: "mC-OL-3-BW" → "OL", "mA-BOL-SRM" → "BOL".
var reLeagueToken = regexp.MustCompile(`^[mwg][A-E]-([A-Za-z]+)`)

// leagueRank ordnet die Spielklassen der Jugend nach Ebene — kleinere Zahl = höhere
// Liga. Die Rangfolge folgt der Ligenpyramide von Bundes-, Verbands- und Bezirksebene:
//
//	10 Jugend-Bundesliga (JBLH)  nur A-/B-Jugend, DHB
//	20 Regionalliga              höchste Landesebene, eingleisig
//	30 Oberliga                  zweithöchste Landesebene, regionale Staffeln
//	50 Bezirksoberliga           höchste Bezirksebene
//	60 Bezirksliga
//	70 Bezirksklasse
//	80 Kreisklasse               unterste Ebene
//
// Verbandsliga (VL) und Kreisliga (KL) sind bewusst NICHT belegt — die Ebenen sind
// für diesen Verein nicht relevant. Taucht ein solches Kürzel doch auf, greift der
// Fail-safe (unbekannt → kein Vorschlag) statt einer stillen Fehlzuordnung.
//
// Auch sonst bewusst unvollständig: ein unbekanntes Kürzel liefert ok=false, und der
// Aufrufer verzichtet dann auf einen Vorschlag, statt eine Rangfolge zu erfinden.
// Belegt aus echten Staffelcodes sind bisher nur OL und BOL — die übrigen Kürzel sind
// die üblichen Abkürzungen der jeweiligen Ebene und werden korrigiert, sobald sie
// real auftauchen.
var leagueRank = map[string]int{
	"JBLH": 10,
	"RL":   20,
	"OL":   30,
	"BOL":  50,
	"BZL":  60,
	"BZK":  70,
	"KK":   80,
}

// LeagueRank liefert die Spielklasse eines Staffelcodes als Rang (kleiner = höher).
// Grundlage für die Regel „Team Stuttgart 2 spielt immer in der niedrigeren Staffel".
func LeagueRank(staffel string) (int, bool) {
	m := reLeagueToken.FindStringSubmatch(strings.TrimSpace(staffel))
	if m == nil {
		return 0, false
	}
	r, ok := leagueRank[strings.ToUpper(m[1])]
	return r, ok
}

// reAliasNumber liest die Mannschaftsnummer aus dem H4A-Vereinsnamen:
// "Team Stuttgart 2" → 2. Ohne Suffix ist es die erste Mannschaft.
var reAliasNumber = regexp.MustCompile(`\s+(\d+)$`)

// TeamNumberFromAlias liefert die Mannschaftsnummer eines H4A-Vereinsnamens
// (ohne Nummer: 1). Sie unterscheidet "C-Jugend männlich" von "C-Jugend männlich 2".
func TeamNumberFromAlias(alias string) int {
	m := reAliasNumber.FindStringSubmatch(strings.TrimSpace(alias))
	if m == nil {
		return 1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// TeamNumberFromName liest dieselbe Nummer aus einem TeamWERK-Mannschaftsnamen
// ("B-Jugend männlich 2" → 2, "B-Jugend männlich" → 1). Gegenstück zu
// TeamNumberFromAlias, damit beide Seiten nach derselben Regel verglichen werden.
func TeamNumberFromName(name string) int { return TeamNumberFromAlias(name) }
