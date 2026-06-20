package beitragslauf

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Satz ist ein Beitragssatz mit Gültigkeitsbeginn (für die In-Memory-Auswahl).
type Satz struct {
	Kategorie  string
	BetragCent int
	ValidFrom  time.Time
}

// LoadSaetzeMap lädt alle Beitragssätze und gruppiert sie pro Kategorie,
// absteigend nach valid_from sortiert (neuester zuerst).
func LoadSaetzeMap(db *sql.DB) (map[string][]Satz, error) {
	rows, err := db.Query(`SELECT kategorie, betrag_eur, valid_from FROM beitrags_saetze`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Satz{}
	for rows.Next() {
		var s Satz
		var vf string
		if err := rows.Scan(&s.Kategorie, &s.BetragCent, &vf); err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02", vf[:min(10, len(vf))])
		if err != nil {
			return nil, fmt.Errorf("valid_from %q: %w", vf, err)
		}
		s.ValidFrom = t
		out[s.Kategorie] = append(out[s.Kategorie], s)
	}
	for k := range out {
		list := out[k]
		sort.Slice(list, func(i, j int) bool { return list[i].ValidFrom.After(list[j].ValidFrom) })
		out[k] = list
	}
	return out, nil
}

// MemberRow sind die für den Beitragslauf relevanten Felder eines Mitglieds.
type MemberRow struct {
	ID             int
	FirstName      string
	LastName       string
	Status         string
	Beitragsfrei   bool
	SepaMandat     bool
	IBAN           string
	SepaMandatPath string
	MemberNumber   string
	Street         string
	Zip            string
	City           string
	HomeClub       string // Freitext (Audit-Spur); für die Kategorie irrelevant
	HasHomeClub    bool   // home_club_id IS NOT NULL → bestimmt aktiv_mit/aktiv_ohne
	AccountHolder  string
	SepaMandatDate string
}

// LoadMembersForLauf lädt alle Mitglieder (ohne Status-Filter; die
// Kategorisierung passiert im Compute).
func LoadMembersForLauf(db *sql.DB) ([]MemberRow, error) {
	rows, err := db.Query(`
		SELECT id, first_name, last_name, status,
		       COALESCE(beitragsfrei,0), COALESCE(sepa_mandat,0),
		       COALESCE(iban,''), COALESCE(sepa_mandat_path,''), COALESCE(member_number,''),
		       COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''),
		       COALESCE(home_club,''), (home_club_id IS NOT NULL), COALESCE(account_holder,''), COALESCE(sepa_mandat_date,'')
		FROM members`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemberRow{}
	for rows.Next() {
		var m MemberRow
		var beitragsfrei, sepaMandat, hasHomeClub int
		if err := rows.Scan(&m.ID, &m.FirstName, &m.LastName, &m.Status,
			&beitragsfrei, &sepaMandat, &m.IBAN, &m.SepaMandatPath, &m.MemberNumber,
			&m.Street, &m.Zip, &m.City, &m.HomeClub, &hasHomeClub, &m.AccountHolder, &m.SepaMandatDate); err != nil {
			return nil, err
		}
		m.Beitragsfrei = beitragsfrei != 0
		m.SepaMandat = sepaMandat != 0
		m.HasHomeClub = hasHomeClub != 0
		if len(m.SepaMandatDate) > 10 {
			m.SepaMandatDate = m.SepaMandatDate[:10]
		}
		out = append(out, m)
	}
	return out, nil
}

// LookupBetragCent liefert den zum Stichtag (Saisonstart) gültigen Betrag:
// den Satz mit dem größten valid_from, das <= stichtag ist.
func LookupBetragCent(saetze map[string][]Satz, kategorie string, stichtag time.Time) (int, error) {
	list := saetze[kategorie]
	for _, s := range list { // bereits DESC sortiert
		if !s.ValidFrom.After(stichtag) {
			return s.BetragCent, nil
		}
	}
	return 0, fmt.Errorf("kein Beitragssatz für %q zum %s hinterlegt", kategorie, stichtag.Format("2006-01-02"))
}
