package gamestats

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// catalogEntry ist eine im Katalog gefundene Staffel.
type catalogEntry struct {
	SubOrgID int
	ClassID  string
	Name     string
}

// catalog ist der einmal je Lauf geladene Staffel-Katalog von Verband und
// Bezirken.
//
// Er existiert aus einem Effizienzgrund, der in der Praxis über Erfolg oder
// Misserfolg entscheidet: die Auflösung EINES Staffelcodes braucht einen
// Perioden-, einen Org- und bis zu neun Katalog-Abrufe. Ohne diesen Cache
// wiederholte ein Lauf über neun Mannschaften das neunmal — rund hundert
// Abrufe, die wegen der Höflichkeitspause (750 ms) über siebzig Sekunden
// brauchen, und die Bezirks-Kataloge neunmal identisch. Mit dem Cache sind es
// elf Abrufe für den ganzen Lauf.
type catalog struct {
	Period  string
	entries map[string]catalogEntry
}

// normalizeCode macht den Vergleich unabhängig von Groß-/Kleinschreibung und
// Leerraum — der Code kommt teils aus Freitext-Eingabe.
func normalizeCode(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// loadCatalog holt Spielzeit, Organisationsliste und die Kataloge aller
// Organisationen.
//
// Ein einzelner nicht erreichbarer Bezirk lässt den Lauf NICHT scheitern: seine
// Staffeln bleiben dann unauflösbar und werden einzeln gemeldet, statt die
// übrigen acht mitzureißen.
func loadCatalog(ctx context.Context, c *bwhv.Client, orgID int) (*catalog, error) {
	_, period, err := c.FetchPeriods(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if period == "" {
		return nil, fmt.Errorf("spielzeit des Verbands nicht ermittelbar")
	}
	orgs, err := c.FetchOrgs(ctx, orgID)
	if err != nil {
		return nil, err
	}

	cat := &catalog{Period: period, entries: map[string]catalogEntry{}}
	subOrgs := []int{0} // 0 = Verbandsebene (kein o-Parameter)
	for id := range orgs {
		n, convErr := strconv.Atoi(id)
		if convErr != nil || n == orgID || n <= 0 {
			continue
		}
		subOrgs = append(subOrgs, n)
	}
	for _, sub := range subOrgs {
		classes, err := c.FetchCatalog(ctx, orgID, sub, period)
		if err != nil {
			continue
		}
		for _, cl := range classes {
			key := normalizeCode(cl.Sname)
			// Verbandsebene zuerst: ein auf beiden Ebenen geführter Code
			// behält seine erste, höhere Fundstelle.
			if _, schon := cat.entries[key]; schon {
				continue
			}
			cat.entries[key] = catalogEntry{SubOrgID: sub, ClassID: cl.ID, Name: cl.Lname}
		}
	}
	if len(cat.entries) == 0 {
		return nil, fmt.Errorf("katalog des Verbands war leer")
	}
	return cat, nil
}

// lookup findet eine Staffel im Katalog.
func (cat *catalog) lookup(code string) (catalogEntry, bool) {
	e, ok := cat.entries[normalizeCode(code)]
	return e, ok
}
