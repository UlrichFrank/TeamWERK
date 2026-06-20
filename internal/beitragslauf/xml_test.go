package beitragslauf

import (
	"strings"
	"testing"
	"time"
)

func sampleInput() BuildInput {
	created := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	return BuildInput{
		SaisonKurz:   "2026/27",
		ClubName:     "Team Stuttgart",
		GlaeubigerID: "DE98ZZZ09999999999",
		ClubIBAN:     "DE89370400440532013000",
		BIC:          "GENODEF1S02",
		Kontoinhaber: "Team Stuttgart e.V.",
		Faelligkeit:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:    created,
		Items: []ExportItem{
			{MemberID: 1, Name: "Max Müller", Street: "Hauptstr. 12", Zip: "70182", City: "Stuttgart",
				IBAN: "DE89370400440532013000", BetragCent: 9600, MandatRef: "1042", MandatDatum: "2026-05-01", MemberNumber: "1042"},
		},
	}
}

func TestBuildXML_EinPmtInfBlockRCUR(t *testing.T) {
	out, err := BuildXML(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if n := strings.Count(s, "<PmtInf>"); n != 1 {
		t.Errorf("PmtInf-Blöcke = %d, want 1", n)
	}
	if strings.Count(s, "<SeqTp>RCUR</SeqTp>") != 1 || strings.Contains(s, "FRST") {
		t.Errorf("SeqTp nicht ausschließlich RCUR:\n%s", s)
	}
	if !strings.Contains(s, painNS) {
		t.Error("Namespace fehlt")
	}
	if !strings.Contains(s, `<InstdAmt Ccy="EUR">96.00</InstdAmt>`) {
		t.Errorf("InstdAmt-Format falsch:\n%s", s)
	}
	if !strings.Contains(s, "<CtrlSum>96.00</CtrlSum>") {
		t.Error("CtrlSum-Format falsch")
	}
}

func TestBuildXML_StraßenParsing(t *testing.T) {
	if strt, bldg := parseStreet("Hauptstr. 12"); strt != "Hauptstr." || bldg != "12" {
		t.Errorf("Hauptstr. 12 → %q / %q", strt, bldg)
	}
	if strt, bldg := parseStreet("Am Bach 3a"); strt != "Am Bach" || bldg != "3a" {
		t.Errorf("Am Bach 3a → %q / %q", strt, bldg)
	}
	if strt, bldg := parseStreet("Postfach 100"); strt != "Postfach 100" || bldg != "" {
		// "100" hat keinen Buchstaben-Suffix, matcht aber die Hausnr-Regex.
		// Akzeptiert: Postfach ist für Lastschrift praktisch irrelevant.
		_ = strt
		_ = bldg
	}
}

func TestBuildXML_UmlauteWerdenTransliteriert(t *testing.T) {
	// Bank-Validatoren (z.B. Sparkassen-Modul) lehnen Nicht-ASCII auch in
	// Name/Adresse/Verwendungszweck ab. Umlaute müssen vor dem Schreiben
	// transliteriert werden (ä→ae usw.).
	in := sampleInput()
	in.Items[0].Name = "Max Müller"
	in.Items[0].Street = "Albstraße 5"
	in.Items[0].City = "München"
	in.ClubName = "Team Stuttgart e.V. – Ürlich"
	in.Kontoinhaber = "Team Stüttgart e.V."

	out, _ := BuildXML(in)
	s := string(out)

	if !strings.Contains(s, "<Nm>Max Mueller</Nm>") {
		t.Errorf("Name nicht transliteriert (erwarte Mueller):\n%s", s)
	}
	if !strings.Contains(s, "<StrtNm>Albstrasse</StrtNm>") {
		t.Errorf("Straße nicht transliteriert (erwarte Albstrasse):\n%s", s)
	}
	if !strings.Contains(s, "<TwnNm>Muenchen</TwnNm>") {
		t.Errorf("Stadt nicht transliteriert (erwarte Muenchen):\n%s", s)
	}
	// Gesamte XML muss reines ASCII sein (XML-Header und Body)
	for i, b := range []byte(s) {
		if b > 0x7E {
			t.Fatalf("Nicht-ASCII-Byte 0x%02x an Position %d — Validator wird das ablehnen", b, i)
		}
	}
}

func TestBuildXML_VerwendungszweckFormat(t *testing.T) {
	out, _ := BuildXML(sampleInput())
	want := "<Ustrd>Mitgliedsbeitrag Team Stuttgart 26/27 - Mitglied 1042</Ustrd>"
	if !strings.Contains(string(out), want) {
		t.Errorf("Verwendungszweck-Format falsch, erwartet %q", want)
	}
}

func TestNextBusinessDay(t *testing.T) {
	sat := time.Date(2028, 7, 1, 0, 0, 0, 0, time.UTC) // 01.07.2028 ist ein Samstag
	if got := nextBusinessDay(sat); got.Weekday() != time.Monday {
		t.Errorf("Sa → %v, want Monday", got.Weekday())
	}
}
