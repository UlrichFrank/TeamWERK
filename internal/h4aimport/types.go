// Package h4aimport liest den Spielplan des eigenen Vereins aus Handball4All
// (meinh4a.handball4all.de) über die xajax-Schnittstelle von edit.php ein.
//
// Der Import ist bewusst admin-getriggert (kein Cron): fremde H4A-Zugangsdaten
// werden nur transient im preview-Request verarbeitet, nie geloggt oder
// persistiert. Details siehe openspec/changes/h4a-import/design.md §2.
package h4aimport

// Period ist eine H4A-Spielperiode/Saison-Option aus <select id="ge_periods">.
type Period struct {
	ID   string
	Name string
}

// RawGame ist eine geparste Zeile der H4A-Spieltabelle — roh, ohne TeamWERK-Mapping
// (Staffel→Mannschaft und Hallennummer→Venue passieren erst im Diff-Schritt).
type RawGame struct {
	InternalID string // H4A game-DOM-ID, z.B. "9715601"
	GameNo     string // BWHV-Spielnummer "Nr.", Idempotenz-Anker, z.B. "211004"
	Staffel    string // "mA-BOL-SRM"
	HallNumber string // "3029"
	Date       string // normalisiert "2026-09-19"
	Time       string // "12:30" oder "" (leer möglich)
	Home       string // "HSC Schm/Oeff"
	Guest      string // "Team Stuttgart 2"
	IsHome     bool   // true wenn das EIGENE Team im Heim-Feld steht (via <b>-Markierung)
	Comment    string // "Än. Vl." o.ä. (Änderungssignal), darf leer sein
}
