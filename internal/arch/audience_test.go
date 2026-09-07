// Empfänger-Invariante als Test (Harness-Engineering, Säule 2 — mechanisch
// erzwungene Konventionen). Change `terminmeldungen-erweiterter-kader`:
// notify.TeamAudience ist die EINZIGE Auflösung der Frage „wer bekommt eine
// Meldung zu einem Mannschaftstermin?".
//
// Der Anlass dieses Gates ist kein Denkfehler, sondern Kopiererei: games,
// trainings und scheduler trugen je eine eigene Fassung derselben Funktion,
// und über die Jahre liefen sie auseinander — der erweiterte Kader bekam
// Trainings-, aber keine Spielmeldungen, seine Eltern gar keine. Ohne
// mechanische Absicherung ist dieser Zustand in zwei Jahren wieder da.
//
// Heuristik: eine Funktion, die eine Nutzer-ID-Liste (`[]int`) liefert und in
// ihrem Rumpf `family_links` mit einer Kader-Tabelle verbindet, baut die
// Empfängermenge selbst nach. Bewusst über den Rückgabetyp gefiltert:
// Sichtbarkeits- und Berechtigungsprüfungen fragen dieselben Tabellen ab,
// liefern aber bool, Zeilen oder Team-IDs — sie sind nicht gemeint.
//
// Ausnahmen ausschließlich über die begründete Allowlist unten. Ein
// Allowlist-Eintrag ohne realen Fundort lässt den Test fehlschlagen
// (Anti-Verrottung, wie TestBroadcastAllowlist_NoOrphans).
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// kaderTables sind die Quellen einer Kader-Zugehörigkeit. `player_memberships`
// ist die View über `kader_members` — beide zählen.
var kaderTables = []string{"player_memberships", "kader_members", "kader_extended_members"}

// audienceAllowlist: "<Paket>.<Funktion>" -> Begründung, warum diese Funktion
// eine eigene Empfängerauflösung haben DARF. Jeder Eintrag MUSS auf eine real
// gefundene Funktion zeigen (sonst schlägt TestAudienceAllowlist_OhneWaisen fehl).
var audienceAllowlist = map[string]string{
	// Die Dienstbörse fragt „wer ist dienstpflichtig?", nicht „wen betrifft der
	// Termin?". Die Dienstpflicht folgt der Kader-Zugehörigkeit; wer im
	// erweiterten Kader aushilft, schuldet dem Verein keine Dienststunden und
	// soll auch nicht zur Übernahme aufgefordert werden. Siehe
	// openspec/specs/terminmeldung-empfaenger, Requirement „Abgrenzung zu
	// Dienstmeldungen und Sichtbarkeit".
	"duties.eligibleDutyRecipients": "Dienstpflicht folgt dem Stammkader, nicht der Terminbetroffenheit",

	// Mitfahrgelegenheiten adressieren bewusst NUR die Eltern (plus Trainer) und
	// schließen den Steller der Suche aus — gefragt ist „wer kann fahren?", nicht
	// „wen betrifft der Termin?". Der erweiterte Kader ist dort bereits über
	// seine Eltern enthalten. Zusätzlich saisonparametrisiert statt auf die
	// aktive Saison fixiert.
	"carpooling.kaderRecipients": "andere Frage (wer kann fahren?): nur Eltern + Trainer, ohne den Steller",

	// Die Video-Ready-Meldung adressiert Hochladenden, aktive Spieler, deren
	// Eltern und Trainer — gebunden an die Saison DES VIDEOS und gefiltert auf
	// members.status='aktiv'. Dass der erweiterte Kader dort fehlt, ist eine
	// eigene, offene Frage (Video-Sichtbarkeit), kein Rest dieses Changes.
	"videos.pushRecipients": "eigene Menge (Hochladender + aktive Spieler + Trainer), Saison des Videos",
}

type audienceOccurrence struct {
	pkg  string
	fn   string
	file string
	line int
}

func (o audienceOccurrence) key() string { return o.pkg + "." + o.fn }

// returnsUserIDList erkennt Funktionen mit Rückgabetyp []int bzw. ([]int, error).
func returnsUserIDList(fd *ast.FuncDecl) bool {
	if fd.Type.Results == nil || len(fd.Type.Results.List) == 0 || len(fd.Type.Results.List) > 2 {
		return false
	}
	arr, ok := fd.Type.Results.List[0].Type.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return false
	}
	ident, ok := arr.Elt.(*ast.Ident)
	return ok && ident.Name == "int"
}

// collectAudienceOccurrences sammelt alle Funktionen im Produktionscode unter
// internal/ (außer internal/notify selbst), die eine Nutzer-ID-Liste liefern und
// dabei family_links mit einer Kader-Tabelle verbinden.
func collectAudienceOccurrences(t *testing.T) []audienceOccurrence {
	t.Helper()
	root := internalRoot(t)
	var occs []audienceOccurrence

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg := strings.SplitN(rel, "/", 2)[0]
		if pkg == "notify" {
			return nil // der eine erlaubte Fundort
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return err
		}

		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil || !returnsUserIDList(fd) {
				continue
			}
			body := string(src[fd.Body.Pos()-1 : fd.Body.End()-1])
			if !strings.Contains(body, "family_links") {
				continue
			}
			hasKader := false
			for _, tbl := range kaderTables {
				if strings.Contains(body, tbl) {
					hasKader = true
					break
				}
			}
			if !hasKader {
				continue
			}
			occs = append(occs, audienceOccurrence{
				pkg:  pkg,
				fn:   fd.Name.Name,
				file: rel,
				line: fset.Position(fd.Pos()).Line,
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking internal/: %v", err)
	}
	return occs
}

func TestArchitecture_KeineEigeneEmpfaengerAufloesung(t *testing.T) {
	for _, o := range collectAudienceOccurrences(t) {
		if _, ok := audienceAllowlist[o.key()]; ok {
			continue
		}
		t.Errorf("%s:%d: internal/%s.%s löst eine Empfängermenge selbst auf (family_links + Kader-Tabelle) — "+
			"notify.TeamAudience verwenden oder mit Begründung in audienceAllowlist aufnehmen",
			o.file, o.line, o.pkg, o.fn)
	}
}

// TestAudienceAllowlist_OhneWaisen verhindert Verrottung: ein Eintrag, dessen
// Funktion es nicht mehr gibt, würde sonst stillschweigend eine spätere,
// gleichnamige Funktion durchwinken.
func TestAudienceAllowlist_OhneWaisen(t *testing.T) {
	found := map[string]bool{}
	for _, o := range collectAudienceOccurrences(t) {
		found[o.key()] = true
	}
	for key := range audienceAllowlist {
		if !found[key] {
			t.Errorf("audienceAllowlist-Eintrag %q zeigt auf keine gefundene Funktion (veraltet? Tippfehler?)", key)
		}
	}
}
