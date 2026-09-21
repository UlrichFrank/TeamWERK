package bwhv

import (
	"strings"
	"testing"
)

// Harte Stufe: weicht der Kopf-Endstand von der Summe des Spielverlaufs ab,
// hat der Parser ein Ereignis verloren. Dann darf NICHTS gespeichert werden.
func TestKreuzprobe_EndstandAbweichungVerwirftBericht(t *testing.T) {
	r, err := ParseReport(loadFixture(t, "spielbericht_endstand_falsch.pdf"))
	if err == nil {
		t.Fatalf("erwartet: Fehlschlag bei falschem Endstand, bekam %d Ereignisse", len(r.Events))
	}
	if r != nil {
		t.Error("bei hartem Fehlschlag darf kein Report zurückkommen — sonst landen Teilergebnisse in der DB")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Fehlertyp = %T, erwartet *ParseError", err)
	}
	for _, want := range []string{"30:25", "29:25"} {
		if !strings.Contains(pe.Reason, want) {
			t.Errorf("Grund nennt %q nicht: %q", want, pe.Reason)
		}
	}
}

// Weiche Stufe: weicht nur die Mannschaftsliste vom Verlauf ab, bleibt der
// Bericht gültig und die Abweichung wird benannt.
func TestKreuzprobe_DetailabweichungWirdGespeichertUndGewarnt(t *testing.T) {
	r, err := ParseReport(loadFixture(t, "spielbericht_liste_abweichend.pdf"))
	if err != nil {
		t.Fatalf("erwartet: Bericht bleibt gültig, bekam Fehler %v", err)
	}
	if len(r.Warnings) == 0 {
		t.Fatal("erwartet: mindestens eine Warnung zur abweichenden Summenspalte")
	}
	joined := WarningsText(r.Warnings)
	for _, want := range []string{"#42", "Tore", "Mannschaftsliste", "Spielverlauf"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Warnung nennt %q nicht: %q", want, joined)
		}
	}
	if len(r.Events) == 0 || len(r.Home.Players) == 0 {
		t.Error("trotz Warnung müssen Ereignisse und Kader gespeichert werden")
	}
}

// Der unveränderte Bericht ist in sich stimmig — sonst wäre jede Warnung im
// Produktivbetrieb Rauschen und niemand sähe mehr hin.
func TestKreuzprobe_UnveraenderterBerichtIstWarnungsfrei(t *testing.T) {
	r := parseFixture(t, "spielbericht_905272.pdf")
	if len(r.Warnings) != 0 {
		t.Errorf("erwartet: keine Warnungen, bekam %d: %s", len(r.Warnings), WarningsText(r.Warnings))
	}
}

// checkScore direkt: ein Tor ohne zuordenbare Mannschaft ist ein harter
// Fehlschlag, weil der Endstand sonst nicht prüfbar wäre.
func TestCheckScore_TorOhneSeiteIstFehlschlag(t *testing.T) {
	r := &Report{
		Header: Header{HomeTeam: "A", GuestTeam: "B", HomeGoals: 1, GuestGoals: 0},
		Events: []Event{{Kind: EventGoal, Side: "", RawText: "Tor durch X (7, Unbekannt)"}},
	}
	err := checkScore(r)
	if err == nil {
		t.Fatal("erwartet: Fehlschlag bei Tor ohne Seite")
	}
	if !strings.Contains(err.Error(), "zuordenbare") {
		t.Errorf("Grund = %q, sollte die fehlende Zuordnung benennen", err)
	}
}

func TestCheckScore_SiebenmeterTorZaehltZumEndstand(t *testing.T) {
	r := &Report{
		Header: Header{HomeTeam: "A", GuestTeam: "B", HomeGoals: 2, GuestGoals: 0},
		Events: []Event{
			{Kind: EventGoal, Side: "home"},
			{Kind: EventSevenMGoal, Side: "home"},
			{Kind: EventSevenMMiss, Side: "home"},
			{Kind: EventTwoMin, Side: "guest"},
		},
	}
	if err := checkScore(r); err != nil {
		t.Fatalf("erwartet: 1 Feldtor + 1 Siebenmeter-Tor = 2:0, bekam %v", err)
	}
}

// checkRosters direkt: die Zählung aus dem Verlauf muss Siebenmeter-Tore
// sowohl bei den Toren als auch bei den Siebenmetern führen.
func TestCheckRosters_SiebenmeterZaehltDoppelt(t *testing.T) {
	r := &Report{
		Header: Header{HomeTeam: "A", GuestTeam: "B"},
		Home: Roster{TeamName: "A", Side: "home", Players: []RosterPlayer{
			{Number: 7, Name: "P", Goals: 1, SevenMAtt: 1, SevenMGoals: 1},
		}},
		Guest:  Roster{TeamName: "B", Side: "guest"},
		Events: []Event{{Kind: EventSevenMGoal, Side: "home", Number: 7}},
	}
	if w := checkRosters(r); len(w) != 0 {
		t.Errorf("erwartet: keine Warnung, bekam %q", WarningsText(w))
	}
}

func TestCheckRosters_AbweichungWirdBenannt(t *testing.T) {
	r := &Report{
		Header: Header{HomeTeam: "A", GuestTeam: "B"},
		Home: Roster{TeamName: "A", Side: "home", Players: []RosterPlayer{
			{Number: 7, Name: "P", Goals: 5, TwoMin: 2},
		}},
		Guest:  Roster{TeamName: "B", Side: "guest"},
		Events: []Event{{Kind: EventGoal, Side: "home", Number: 7}},
	}
	w := checkRosters(r)
	joined := WarningsText(w)
	if !strings.Contains(joined, "Tore: Mannschaftsliste 5, Spielverlauf 1") {
		t.Errorf("Tor-Abweichung fehlt: %q", joined)
	}
	if !strings.Contains(joined, "Zeitstrafen: Mannschaftsliste 2, Spielverlauf 0") {
		t.Errorf("Strafen-Abweichung fehlt: %q", joined)
	}
}
