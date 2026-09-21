package bwhv

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("Fixture %s nicht lesbar: %v", name, err)
	}
	return b
}

func parseFixture(t *testing.T, name string) *Report {
	t.Helper()
	r, err := ParseReport(loadFixture(t, name))
	if err != nil {
		t.Fatalf("ParseReport(%s): %v", name, err)
	}
	return r
}

func TestParseReport_Kopfdaten(t *testing.T) {
	h := parseFixture(t, "spielbericht_905272.pdf").Header
	for _, c := range []struct{ name, got, want string }{
		{"Staffel", h.Staffel, "mB-RL-BW"},
		{"Spielnummer", h.GameNo, "905272"},
		{"Datum", h.Date, "2026-09-20"},
		{"Anwurf", h.Time, "16:00"},
		{"Hallennummer", h.HallNumber, "21005"},
		{"Spielort", h.VenueName, "Erich-Bamberger Stadthalle"},
		{"Ort", h.VenueTown, "Östringen"},
		{"Heim", h.HomeTeam, "Rhein-Neckar Löwen 2"},
		{"Gast", h.GuestTeam, "Team Stuttgart"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, erwartet %q", c.name, c.got, c.want)
		}
	}
	if h.HomeGoals != 29 || h.GuestGoals != 25 {
		t.Errorf("Endstand = %d:%d, erwartet 29:25", h.HomeGoals, h.GuestGoals)
	}
	if h.HomeGoalsHT != 14 || h.GuestGoalsHT != 13 {
		t.Errorf("Halbzeit = %d:%d, erwartet 14:13", h.HomeGoalsHT, h.GuestGoalsHT)
	}
}

// Die Spielnummer ist derselbe Anker wie games.external_id (Migration 042),
// die Hallennummer derselbe wie venues.hall_number.
func TestParseReport_AnkerFelderEntsprechenDemSpielplan(t *testing.T) {
	h := parseFixture(t, "spielbericht_905272.pdf").Header
	if h.GameNo == "" || h.HallNumber == "" {
		t.Fatalf("Spielnummer %q / Hallennummer %q — beide sind Fremdschlüssel und dürfen nicht leer sein",
			h.GameNo, h.HallNumber)
	}
}

func TestParseReport_Mannschaftslisten(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	if len(r.Home.Players) != 13 {
		t.Errorf("Heim-Kader = %d Spieler, erwartet 13", len(r.Home.Players))
	}
	if len(r.Guest.Players) != 13 {
		t.Errorf("Gast-Kader = %d Spieler, erwartet 13", len(r.Guest.Players))
	}
	if r.Home.Side != "home" || r.Guest.Side != "guest" {
		t.Errorf("Seiten falsch gesetzt: %q / %q", r.Home.Side, r.Guest.Side)
	}
}

// Die Spaltenzuordnung ist der heikelste Teil des Parsers: Werte liegen mal
// links, mal rechts ihres Kopfankers. Diese Zeile prüft Tore, Siebenmeter und
// Zeitstrafen an einem Datensatz, dessen Werte in drei verschiedenen Spalten
// stehen.
func TestParseReport_SpaltenzuordnungEinerSpielerzeile(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	byNumber := map[int]RosterPlayer{}
	for _, p := range r.Home.Players {
		byNumber[p.Number] = p
	}
	// #9: zwei Zeitstrafen (30:12 und 49:06), zwei Tore, keine Siebenmeter.
	p9, ok := byNumber[9]
	if !ok {
		t.Fatalf("Heim #9 fehlt, vorhanden: %v", numbersOf(r.Home))
	}
	if p9.Goals != 2 {
		t.Errorf("#9 Tore = %d, erwartet 2", p9.Goals)
	}
	if p9.TwoMin != 2 {
		t.Errorf("#9 Zeitstrafen = %d, erwartet 2 (zwei Hinausstellungsspalten belegt)", p9.TwoMin)
	}
	if p9.SevenMAtt != 0 {
		t.Errorf("#9 Siebenmeter-Versuche = %d, erwartet 0", p9.SevenMAtt)
	}
	// #20: drei Tore und ein vergebener Siebenmeter ("1/0").
	p20 := byNumber[20]
	if p20.Goals != 3 || p20.SevenMAtt != 1 || p20.SevenMGoals != 0 {
		t.Errorf("#20 = %d Tore, 7m %d/%d; erwartet 3 Tore, 7m 1/0",
			p20.Goals, p20.SevenMAtt, p20.SevenMGoals)
	}
}

func TestParseReport_Spielverlauf(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	if len(r.Events) != 66 {
		t.Errorf("Ereignisse = %d, erwartet 66", len(r.Events))
	}
	kinds := map[EventKind]int{}
	for _, e := range r.Events {
		kinds[e.Kind]++
	}
	for _, c := range []struct {
		kind EventKind
		want int
	}{
		{EventGoal, 52}, {EventSevenMGoal, 2}, {EventSevenMMiss, 2},
		{EventTwoMin, 3}, {EventWarning, 2}, {EventTimeout, 5},
	} {
		if kinds[c.kind] != c.want {
			t.Errorf("Ereignisart %s = %d, erwartet %d", c.kind, kinds[c.kind], c.want)
		}
	}
	if kinds[EventOther] != 0 {
		t.Errorf("%d Zeilen unerkannt — jede Form dieses Berichts ist bekannt", kinds[EventOther])
	}
}

// Die Spielzeit wird in Sekunden normalisiert, damit sie vergleichbar ist;
// die Uhrzeit bleibt als Text erhalten.
func TestParseReport_SpielzeitInSekunden(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	first := r.Events[0]
	if first.GameSecond != 99 {
		t.Errorf("erste Spielzeit = %d s, erwartet 99 (01:39)", first.GameSecond)
	}
	if first.ClockTime != "16:01:57" {
		t.Errorf("erste Uhrzeit = %q, erwartet 16:01:57", first.ClockTime)
	}
	var last int
	for _, e := range r.Events {
		if e.GameSecond < last {
			t.Errorf("Spielzeit läuft rückwärts bei Ereignis %d: %d nach %d", e.Seq, e.GameSecond, last)
			break
		}
		last = e.GameSecond
	}
}

// Jedes Tor braucht eine Seite, sonst ist der Endstand nicht prüfbar.
func TestParseReport_JedesTorHatEineSeite(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	for _, e := range r.Events {
		if e.Kind != EventGoal && e.Kind != EventSevenMGoal {
			continue
		}
		if e.Side != "home" && e.Side != "guest" {
			t.Errorf("Tor ohne Seite: %q (Mannschaft %q)", e.RawText, e.TeamName)
		}
		if e.PlayerName == "" || e.Number == 0 {
			t.Errorf("Tor ohne Schütze: %q", e.RawText)
		}
	}
}

// Der Spielverlauf nutzt Kurzformen ("R-N Löwen 2"), der Kopf die Langform.
// Die Zuordnung darf daran nicht scheitern.
func TestParseReport_KurzformDerMannschaftWirdZugeordnet(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	seen := map[string]string{}
	for _, e := range r.Events {
		if e.TeamName != "" && e.Side != "" {
			seen[e.TeamName] = e.Side
		}
	}
	if len(seen) < 2 {
		t.Fatalf("weniger als zwei Mannschaften im Verlauf zugeordnet: %v", seen)
	}
	sides := map[string]bool{}
	for _, s := range seen {
		sides[s] = true
	}
	if !sides["home"] || !sides["guest"] {
		t.Errorf("nicht beide Seiten vertreten: %v", seen)
	}
}

// Trikotnummern sind innerhalb EINES Spiels nicht eindeutig: in diesem Bericht
// tragen beide Mannschaften eine 16 und eine 46. Erst (Mannschaft, Nummer) ist
// eindeutig — deshalb ist team_name Teil der Spieler-Identität.
func TestParseReport_TrikotnummernKollidierenZwischenMannschaften(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	home := map[int]bool{}
	for _, p := range r.Home.Players {
		home[p.Number] = true
	}
	var kollisionen []int
	for _, p := range r.Guest.Players {
		if home[p.Number] {
			kollisionen = append(kollisionen, p.Number)
		}
	}
	if len(kollisionen) == 0 {
		t.Fatal("erwartet: mindestens eine Nummer in beiden Mannschaften (Fixture trägt 16 und 46)")
	}
}

func TestParseReport_PlatzhalterErzeugtKeinenSpieler(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	for _, ros := range []Roster{r.Home, r.Guest} {
		for _, p := range ros.Players {
			if isPlaceholderName(p.Name) {
				t.Errorf("Platzhalter als Spieler erfasst: %q in %q", p.Name, ros.TeamName)
			}
		}
	}
	if ref := r.Header.Referees; ref != "" {
		t.Errorf("Schiedsrichter = %q, erwartet leer (Bericht trägt N.N.-Platzhalter)", ref)
	}
}

func TestParseReport_KeinPDFIstEinParseError(t *testing.T) {
	_, err := ParseReport([]byte("kein PDF"))
	if err == nil {
		t.Fatal("erwartet: Fehler bei Nicht-PDF-Eingabe")
	}
	if _, ok := err.(*ParseError); !ok {
		t.Errorf("Fehlertyp = %T, erwartet *ParseError", err)
	}
}

func numbersOf(r Roster) []int {
	out := make([]int, 0, len(r.Players))
	for _, p := range r.Players {
		out = append(out, p.Number)
	}
	return out
}

// refereeDoc baut eine Textebene mit der Schiedsrichter-Zeile aus den
// übergebenen Textläufen. Ein Lauf je Name entspricht zwei Spalten im
// Dokument, ein einziger Lauf dem Einspalten-Fall.
func refereeDoc(runs ...string) *document {
	line := textLine{Page: 1, Y: 100, Groups: []textGroup{{X: 40, End: 70, S: "Name"}}}
	x := 100.0
	for _, r := range runs {
		line.Groups = append(line.Groups, textGroup{X: x, End: x + 80, S: r})
		x += 120
	}
	return &document{Lines: []textLine{
		{Page: 1, Y: 120, Groups: []textGroup{{X: 40, End: 120, S: "Schiedsrichter"}}},
		line,
	}}
}

// Zwei Textläufe heißt: die Trennung steht im Dokument. Sie wird übernommen
// und nicht als unsicher vermerkt.
func TestParseReferees_ZweiSpaltenZweiNamen(t *testing.T) {
	names, uncertain := parseReferees(refereeDoc("Max Mustermann", "Peter Müller"))
	if len(names) != 2 || names[0] != "Max Mustermann" || names[1] != "Peter Müller" {
		t.Fatalf("Namen = %#v, erwartet [Max Mustermann Peter Müller]", names)
	}
	if uncertain {
		t.Error("uncertain = true, erwartet false — die Trennung stammt aus dem Dokument")
	}
}

// Ein einziger Lauf: die Namensregel greift und der Schnitt ist geraten.
func TestParseReferees_EineSpalteIstUnsicher(t *testing.T) {
	names, uncertain := parseReferees(refereeDoc("Max Mustermann Peter Müller"))
	if len(names) != 2 || names[0] != "Max Mustermann" || names[1] != "Peter Müller" {
		t.Fatalf("Namen = %#v, erwartet zwei getrennte Namen", names)
	}
	if !uncertain {
		t.Error("uncertain = false, erwartet true — der Schnitt ist geraten")
	}
}

func TestParseReferees_PlatzhalterErzeugtNichts(t *testing.T) {
	names, uncertain := parseReferees(refereeDoc("N.N.", "N.N."))
	if len(names) != 0 {
		t.Fatalf("Namen = %#v, erwartet leer", names)
	}
	if uncertain {
		t.Error("uncertain = true, erwartet false — es wurde nichts geraten")
	}
}

// Eine Zeile, die die Namensregel nicht deuten kann, liefert nichts — lieber
// keine Namen als erfundene.
func TestParseReferees_UnbrauchbareZeileLiefertNichts(t *testing.T) {
	names, uncertain := parseReferees(refereeDoc("a b c d e f g"))
	if len(names) != 0 || uncertain {
		t.Fatalf("Namen = %#v / uncertain = %v, erwartet leer und false", names, uncertain)
	}
}

// Die Rohzeile bleibt erhalten: sie ist der Beleg für eine spätere,
// verbesserte Trennung ohne erneuten Fremdabruf.
func TestParseReferees_RohzeileBleibtErhalten(t *testing.T) {
	if got := refereeLineText(refereeDoc("Max Mustermann", "Peter Müller")); got != "Max Mustermann Peter Müller" {
		t.Errorf("Rohzeile = %q, erwartet %q", got, "Max Mustermann Peter Müller")
	}
}

// Der Bericht darf an einer unbrauchbaren Schiedsrichter-Zeile nicht
// scheitern: die Namen sind für die Auswertung des Spiels entbehrlich. Das
// Fixture trägt N.N.-Platzhalter und ist damit genau dieser Fall.
func TestParseReport_UnbrauchbareSchiedsrichterzeileKipptNicht(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	if len(r.Header.RefereeNames) != 0 {
		t.Errorf("RefereeNames = %#v, erwartet leer (Platzhalter)", r.Header.RefereeNames)
	}
	if r.Header.RefereesUncertain {
		t.Error("RefereesUncertain = true, erwartet false")
	}
	if len(r.Home.Players) == 0 || len(r.Events) == 0 {
		t.Fatal("erwartet: der Bericht ist trotz fehlender Schiedsrichter vollständig ausgewertet")
	}
}
