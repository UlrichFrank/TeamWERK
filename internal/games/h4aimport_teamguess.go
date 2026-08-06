package games

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/h4aimport"
)

// staffelKey ist der Verbundschlüssel der Mannschaftszuordnung: dieselbe Staffel
// kann beide Vereins-Entitäten enthalten, derselbe Verein spielt in mehreren Staffeln.
// Identisch zum Schlüssel von h4a_staffel_team_map.
type staffelKey struct {
	staffel string
	alias   string
}

// teamCandidate ist eine TeamWERK-Mannschaft als Zuordnungskandidat.
type teamCandidate struct {
	id      int
	name    string
	besetzt int // Kader-Mitglieder in der aktiven Saison
	spiele  int // bereits zugeordnete Spiele (Tiebreak bei Namensdubletten)
}

// suggestStaffelTeams schlägt für jede (Staffel, Verein)-Kombination des Abrufs eine
// Mannschaft vor. Der Vorschlag entsteht plan-weit und nicht zeilenweise, weil die
// fachliche Regel den Vergleich der Staffeln einer Altersklasse untereinander braucht:
//
//	„Team Stuttgart 2" ist immer die Mannschaft in der NIEDRIGEREN Staffel.
//
// Der H4A-Vereinsname allein trägt diese Information nicht — erst die Spielklasse der
// beteiligten Staffeln ordnet die Mannschaften. Deshalb: Staffeln je Altersklasse
// sammeln, nach Liga sortieren (höchste zuerst) und positionsweise auf die nach
// Mannschaftsnummer sortierten TeamWERK-Mannschaften abbilden.
//
// Fail-safe an jeder Stelle: passt die Zahl der Staffeln nicht zur Zahl der aktiven
// Mannschaften, ist ein Liga-Kürzel unbekannt oder bleibt eine Auswahl mehrdeutig,
// gibt es KEINEN Vorschlag. Ein falscher Vorschlag wäre teurer als Handarbeit — er
// wird beim apply als Mapping gelernt und wirkt auf alle Folgeimporte.
// Rückgabe ist zusätzlich ein Grund je abgelehnter Kombination. Ohne ihn steht im
// Modal nur „Mannschaft nicht zugeordnet" — richtig, aber nicht handlungsleitend:
// der Importeur kann nicht erkennen, ob eine Mannschaft fehlt, ein Kader unbesetzt
// ist oder ein Liga-Kürzel unbekannt.
func (h *Handler) suggestStaffelTeams(ctx context.Context, raw []h4aimport.RawGame) (map[staffelKey]teamCandidate, map[staffelKey]string) {
	// Schritt 1: (Staffel, Verein) je Altersklasse/Geschlecht sammeln.
	type groupKey struct{ gender, ageClass string }
	groups := map[groupKey]map[staffelKey]bool{}
	reasons := map[staffelKey]string{}
	for _, g := range raw {
		alias, _, _ := ownAlias(g)
		gender, ageClass, ok := h4aimport.ParseStaffel(g.Staffel)
		if !ok {
			reasons[staffelKey{g.Staffel, alias}] = "Staffelcode nicht interpretierbar"
			continue
		}
		gk := groupKey{gender, ageClass}
		if groups[gk] == nil {
			groups[gk] = map[staffelKey]bool{}
		}
		groups[gk][staffelKey{g.Staffel, alias}] = true
	}

	out := map[staffelKey]teamCandidate{}
	for gk, keySet := range groups {
		keys := make([]staffelKey, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		assignGroup(h.teamCandidates(ctx, gk.ageClass, gk.gender), keys, out, reasons)
	}
	return out, reasons
}

// rejectAll vermerkt denselben Ablehnungsgrund für alle Staffeln einer Gruppe.
func rejectAll(keys []staffelKey, reasons map[staffelKey]string, grund string) {
	for _, k := range keys {
		reasons[k] = grund
	}
}

// assignGroup bildet die Staffeln EINER Altersklasse auf deren Mannschaften ab.
//
// Vorgehen: Staffeln nach Spielklasse sortieren (höchste Liga zuerst); die Position
// in dieser Rangfolge IST die Mannschaftsnummer — höchste Staffel = Mannschaft 1,
// nächste = Mannschaft 2. Das ist die direkte Umsetzung von „Team Stuttgart 2 spielt
// immer in der niedrigeren Staffel". Zugeordnet wird dann die Mannschaft, deren Name
// diese Nummer trägt („C-Jugend männlich" = 1, „C-Jugend männlich 2" = 2).
//
// Eine besetzte Kaderzuordnung ist ausdrücklich KEINE Voraussetzung — auch neu
// angelegte Mannschaften ohne Spieler spielen bereits. Kaderbesetzung und
// Spielhistorie dienen nur noch als Tiebreak zwischen namensgleichen Dubletten
// (im Bestand heißen z.B. zwei Mannschaften „A-Jugend männlich 2").
func assignGroup(cands []teamCandidate, keys []staffelKey, out map[staffelKey]teamCandidate, reasons map[staffelKey]string) {
	if len(cands) == 0 {
		rejectAll(keys, reasons, "keine Mannschaft dieser Altersklasse/Geschlecht angelegt")
		return
	}

	// Staffeln nach Spielklasse ordnen. Ist auch nur ein Kürzel unbekannt, wird nicht
	// geraten — die Rangfolge wäre dann willkürlich.
	ranks := make(map[staffelKey]int, len(keys))
	for _, k := range keys {
		r, ok := h4aimport.LeagueRank(k.staffel)
		if !ok {
			rejectAll(keys, reasons, "Spielklasse aus Staffel "+k.staffel+" nicht ableitbar (Liga-Kürzel unbekannt)")
			return
		}
		ranks[k] = r
	}
	sort.Slice(keys, func(i, j int) bool {
		if ranks[keys[i]] != ranks[keys[j]] {
			return ranks[keys[i]] < ranks[keys[j]]
		}
		return keys[i].staffel < keys[j].staffel
	})
	for i := 1; i < len(keys); i++ {
		if ranks[keys[i]] == ranks[keys[i-1]] {
			rejectAll(keys, reasons, fmt.Sprintf(
				"Staffeln %s und %s liegen in derselben Spielklasse — keine Rangfolge ableitbar",
				keys[i-1].staffel, keys[i].staffel))
			return
		}
	}

	vergeben := map[int]bool{}
	for i, k := range keys {
		wantNo := i + 1
		passend := candidatesWithNumber(cands, wantNo, vergeben)

		// Einzige Staffel und einzige Mannschaft der Altersklasse: dann ist die
		// Zuordnung auch dann eindeutig, wenn die Namensnummer nicht passt (H4A
		// meldet „Team Stuttgart 2", TeamWERK nennt sie „A-Jugend männlich").
		if len(passend) == 0 && len(keys) == 1 && len(cands) == 1 {
			out[k] = cands[0]
			return
		}
		if len(passend) == 0 {
			reasons[k] = fmt.Sprintf(
				"Staffel %s ist die %d. Spielklasse, aber es gibt keine %d. Mannschaft dieser Altersklasse (vorhanden: %s)",
				k.staffel, wantNo, wantNo, strings.Join(namesOf(cands), ", "))
			continue
		}
		// Mehrere gleichnamige Kandidaten: Kader und Spielhistorie entscheiden.
		// Sind auch die gleich, ist nichts zu unterscheiden — dann lieber Handarbeit.
		if len(passend) > 1 && passend[0].besetzt == passend[1].besetzt && passend[0].spiele == passend[1].spiele {
			reasons[k] = fmt.Sprintf(
				"mehrere gleichrangige Mannschaften kommen infrage (%s) — nicht unterscheidbar",
				strings.Join(namesOf(passend), ", "))
			continue
		}
		out[k] = passend[0]
		vergeben[passend[0].id] = true
	}
}

// candidatesWithNumber liefert die noch freien Mannschaften mit der gesuchten
// Nummer im Namen, beste zuerst: besetzter Kader vor leerem, danach Spielhistorie,
// zuletzt die niedrigere ID.
func candidatesWithNumber(cands []teamCandidate, wantNo int, vergeben map[int]bool) []teamCandidate {
	var out []teamCandidate
	for _, c := range cands {
		if !vergeben[c.id] && h4aimport.TeamNumberFromName(c.name) == wantNo {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].besetzt != out[j].besetzt {
			return out[i].besetzt > out[j].besetzt
		}
		if out[i].spiele != out[j].spiele {
			return out[i].spiele > out[j].spiele
		}
		return out[i].id < out[j].id
	})
	return out
}

func namesOf(cands []teamCandidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.name
	}
	return out
}

// teamCandidates lädt die aktiven Mannschaften einer Altersklasse. Kaderbesetzung und
// Spielhistorie sind keine Filter, sondern Tiebreak-Merkmale zwischen Namensdubletten.
func (h *Handler) teamCandidates(ctx context.Context, ageClass, gender string) []teamCandidate {
	rows, err := h.db.QueryContext(ctx, `
		SELECT t.id, t.name,
		       (SELECT COUNT(*) FROM kader k
		          JOIN kader_members km ON km.kader_id = k.id
		         WHERE k.team_id = t.id
		           AND k.season_id = (SELECT id FROM seasons WHERE is_active = 1)) AS besetzt,
		       (SELECT COUNT(*) FROM game_teams gt WHERE gt.team_id = t.id) AS spiele
		  FROM teams t
		 WHERE t.age_class = ? AND t.gender = ? AND t.is_active = 1
		 ORDER BY t.id`, ageClass, gender)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []teamCandidate
	for rows.Next() {
		var c teamCandidate
		if err := rows.Scan(&c.id, &c.name, &c.besetzt, &c.spiele); err != nil {
			return nil
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}
