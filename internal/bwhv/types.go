// Package bwhv liest Spielpläne, Tabellen und Spielberichte des Baden-
// Württembergischen Handball-Verbands über die öffentliche JSON-Schnittstelle
// von Handball4All (spo.handball4all.de/service/if_g_json.php) sowie die dort
// verlinkten Spielbericht-PDFs.
//
// Anders als internal/h4aimport nimmt dieses Paket KEINE fremden Zugangsdaten
// entgegen: sowohl die JSON-Schnittstelle als auch die Berichte sind ohne
// Anmeldung abrufbar. Es gibt hier kein Geheimnis zu schützen.
//
// Foundation-Paket: kein Datenbankzugriff, keine Domänen-Importe. Die
// Persistenz liegt in internal/gamestats.
package bwhv

// Class ist eine Staffel aus dem Katalog (cmd=po). Sname ist der Staffelcode,
// wie ihn auch der Verein verwendet ("mB-RL-BW"), und damit der stabile
// Schlüssel — die ID wechselt mit der Saison.
type Class struct {
	ID    string // gClassID, saisonabhängig
	Sname string // gClassSname, z.B. "mB-RL-BW"
	Lname string // gClassLname, z.B. "männliche B-Jugend Regionalliga"
}

// Game ist eine Begegnung aus dem Spielplan (cmd=ps).
//
// SGID ist leer, solange kein Spielbericht freigegeben ist — die Schnittstelle
// liefert dort die Zahl 0. Genau dieses Feld ist das Bereitschaftssignal für
// den Berichtsabruf; ein zeitgesteuerter Versuch ist damit überflüssig.
type Game struct {
	GameNo       string // gNo, BWHV-Spielnummer = games.external_id
	SGID         string // leer = kein Bericht
	Date         string // normalisiert "2006-01-02"
	Time         string // "16:00", darf leer sein
	HomeTeam     string
	GuestTeam    string
	HomeGoals    *int
	GuestGoals   *int
	HomeGoalsHT  *int
	GuestGoalsHT *int
	HallNumber   string // gGymnasiumNo = venues.hall_number
	HallName     string
	HallTown     string
}

// Played meldet, ob für die Begegnung ein Ergebnis vorliegt.
func (g Game) Played() bool { return g.HomeGoals != nil && g.GuestGoals != nil }

// TableRow ist eine Zeile des Tabellenstands (cmd=ps, content.score).
//
// Position kommt aus der Reihenfolge der Antwort, NICHT aus tabScore: dieses
// Feld trägt nur in der ersten Zeile eine Zahl und ist in allen folgenden ein
// leerer String (Anzeigekonvention des Dienstes für gleiche Ränge).
type TableRow struct {
	Position     int
	TeamName     string
	Games        int
	Won          int
	Drawn        int
	Lost         int
	GoalsFor     int
	GoalsAgainst int
	PointsPlus   int
	PointsMinus  int
}

// Schedule ist die Antwort eines Spielplan-Abrufs: alle Begegnungen einer
// Staffel über die ganze Saison plus der Tabellenstand.
type Schedule struct {
	Games     []Game
	Table     []TableRow
	ReportURL string // head.repURL, Präfix für sboPublicReports.php?sGID=
}

// Period ist eine Spielzeit-Option (menu.period).
type Period struct {
	ID   string
	Name string
}
