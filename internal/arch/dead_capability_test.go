package arch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deadCapabilities listet Capability-Strings, die aus dem Vokabular entfernt
// wurden und nirgends mehr vorkommen dürfen — weder als Konstante im Backend
// noch als hasCapability()-Abfrage im Frontend.
//
// Warum als Arch-Test und nicht als normaler Unit-Test: eine entfernte
// Capability scheitert nicht. Bleibt im Frontend ein
// hasCapability("broadcast_all") stehen, liefert es nach der Backend-Änderung
// dauerhaft false — der zugehörige Button ist für ALLE unsichtbar, ohne dass
// ein Request fehlschlägt oder ein Log etwas meldet. Genau dieselbe Familie
// stiller Fehler wie der Zielgruppen-Resolver, der gegen users.role auflöste
// und immer null Empfänger traf.
var deadCapabilities = map[string]string{
	// mitteilung-zielgruppen: admin, vorstand und sportliche_leitung dürfen
	// dieselben vier Zielgruppen — eine engere zweite Stufe trennt nichts mehr.
	"broadcast_all": "mitteilung-zielgruppen (ersetzt durch broadcast_messages)",
}

// scannedTrees sind die Wurzeln, in denen Capability-Strings vorkommen können,
// relativ zum Repo-Root.
var scannedTrees = []string{"internal", "web/src"}

func TestArchitecture_KeineToteCapability(t *testing.T) {
	root := repoRoot(t)

	for _, tree := range scannedTrees {
		treePath := filepath.Join(root, tree)
		if _, err := os.Stat(treePath); err != nil {
			t.Fatalf("Scan-Wurzel %s nicht gefunden: %v", tree, err)
		}

		err := filepath.WalkDir(treePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "node_modules" || d.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !scannableSource(path) {
				return nil
			}
			// Diese Datei selbst nennt die toten Strings naturgemäß.
			if filepath.Base(path) == "dead_capability_test.go" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			for cap, reason := range deadCapabilities {
				if strings.Contains(content, cap) {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s nennt die entfernte Capability %q (%s) — "+
						"eine Abfrage darauf liefert dauerhaft false und versteckt ihr Feature stumm",
						rel, cap, reason)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
}

// scannableSource meldet, ob die Datei Quellcode ist, in dem ein
// Capability-String stehen könnte.
func scannableSource(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".ts", ".tsx", ".js", ".jsx":
		return true
	}
	return false
}

// repoRoot liefert die Modulwurzel. Der Test liegt in internal/arch/, also zwei
// Ebenen darüber.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(filepath.Dir(wd))
}
