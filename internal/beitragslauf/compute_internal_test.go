package beitragslauf

import (
	"testing"
	"time"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// Saisonfenster 2027-09-01 .. 2028-06-30, Stichtag 2027-07-01.
func testSeason(inaugural bool) SeasonInfo {
	return SeasonInfo{
		Label:     "2027/28",
		Start:     mustDate("2027-09-01"),
		End:       mustDate("2028-06-30"),
		Stichtag:  mustDate("2027-07-01"),
		Inaugural: inaugural,
	}
}

func TestInWindow_Grenzen(t *testing.T) {
	start, end := mustDate("2027-09-01"), mustDate("2028-06-30")
	cases := []struct {
		date string
		want bool
	}{
		{"2027-09-01", true},  // Startgrenze inklusive
		{"2028-06-30", true},  // Endgrenze inklusive
		{"2027-08-31", false}, // ein Tag vor Start
		{"2028-07-01", false}, // ein Tag nach Ende
		{"2028-01-15", true},  // mittendrin
		{"", false},           // leer
		{"kaputt", false},     // ungültig
	}
	for _, c := range cases {
		if got := inWindow(c.date, start, end); got != c.want {
			t.Errorf("inWindow(%q) = %v, want %v", c.date, got, c.want)
		}
	}
}

func TestHalfFee_Prioritaet(t *testing.T) {
	cases := []struct {
		name       string
		m          MemberRow
		inaugural  bool
		wantHalf   bool
		wantReason string
	}{
		{"erstjahr schlägt alles", MemberRow{Status: "aktiv"}, true, true, "erstjahr"},
		{"eintritt im fenster", MemberRow{Status: "aktiv", JoinDate: "2027-10-01"}, false, true, "eintritt"},
		{"austritt im fenster", MemberRow{Status: "ausgetreten", ExitDate: "2027-10-01"}, false, true, "austritt"},
		{"eintritt vor austritt (prio)", MemberRow{Status: "ausgetreten", JoinDate: "2027-10-01", ExitDate: "2028-01-01"}, false, true, "eintritt"},
		{"ganzjährig voll", MemberRow{Status: "aktiv", JoinDate: "2020-01-01"}, false, false, ""},
		{"austritt außerhalb", MemberRow{Status: "ausgetreten", ExitDate: "2025-01-01"}, false, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			half, reason := halfFee(c.m, testSeason(c.inaugural))
			if half != c.wantHalf || reason != c.wantReason {
				t.Errorf("halfFee = (%v,%q), want (%v,%q)", half, reason, c.wantHalf, c.wantReason)
			}
		})
	}
}

// Ein- UND Austritt im selben Jahr halbiert nur einmal (nicht geviertelt).
func TestComputeItem_KeinStacking(t *testing.T) {
	saetze := map[string][]Satz{
		"aktiv_ohne": {{Kategorie: "aktiv_ohne", BetragCent: 22600, ValidFrom: mustDate("2026-07-01")}},
	}
	m := MemberRow{
		ID: 1, FirstName: "A", LastName: "B", Status: "ausgetreten",
		JoinDate: "2027-10-01", ExitDate: "2028-02-01",
		MemberNumber: "1", Street: "S", Zip: "1", City: "C",
		SepaMandat: true, SepaMandatPath: "/x.pdf", HasBank: true, BankCiphertext: "CT",
	}
	it := computeItem(m, saetze, testSeason(false))
	if !it.Half || it.BetragCent != 11300 {
		t.Fatalf("kein Stacking erwartet: half=%v betrag=%d (want 11300)", it.Half, it.BetragCent)
	}
}

// Exakte (ganzzahlige) Halbierung: ungerader Cent-Betrag wird abgeschnitten.
func TestComputeItem_ExakteHalbierung(t *testing.T) {
	saetze := map[string][]Satz{
		"aktiv_ohne": {{Kategorie: "aktiv_ohne", BetragCent: 22601, ValidFrom: mustDate("2026-07-01")}},
	}
	m := MemberRow{
		ID: 1, FirstName: "A", LastName: "B", Status: "aktiv", JoinDate: "2027-10-01",
		MemberNumber: "1", Street: "S", Zip: "1", City: "C",
		SepaMandat: true, SepaMandatPath: "/x.pdf", HasBank: true, BankCiphertext: "CT",
	}
	it := computeItem(m, saetze, testSeason(false))
	if it.BetragCent != 11300 { // 22601 / 2 = 11300 (Abschnitt)
		t.Fatalf("exakte Halbierung: betrag=%d, want 11300", it.BetragCent)
	}
}
