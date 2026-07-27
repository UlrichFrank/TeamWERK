package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// runDevSeed legt eine wegwerfbare Entwickler-Datenbank an, mit der man die Chat-
// Scroll-/Divider-/Bild-Szenarien manuell im Browser durchklicken kann. Es nutzt exakt
// dieselbe Seed-Logik wie die Playwright-Suite (seedE2E): Admin + 3 Standard-Nutzer +
// alle Chat-Konversationen inklusive der langen, bildlastigen Threads.
//
// Verwendung: teamwerk dev-seed [--db=./storage/dev-seed.db]
//
// Die DB wird bei jedem Lauf frisch aufgesetzt (wipe+reseed, deterministisch). Bild-
// Dateien landen im MEDIA_DIR (Default ./storage/media, per Env überschreibbar) — genau
// dort muss auch der Server sie später suchen.
func runDevSeed() {
	fs := flag.NewFlagSet("dev-seed", flag.ExitOnError)
	dbPath := fs.String("db", "./storage/dev-seed.db", "Pfad zur SQLite-Dev-Datenbank")
	_ = fs.Parse(os.Args[2:])

	// Prod-Schutz: niemals die produktive DB anfassen.
	if strings.Contains(*dbPath, "/var/lib/teamwerk") {
		fmt.Fprintf(os.Stderr,
			"dev-seed: verweigert — --db=%q zeigt auf das Prod-Verzeichnis (/var/lib/teamwerk).\n", *dbPath)
		os.Exit(1)
	}
	// Prod-Schutz auch für das Bild-Verzeichnis: dev-seed schreibt Test-PNGs ins
	// MEDIA_DIR — niemals ins Prod-Media-Verzeichnis.
	if md := os.Getenv("MEDIA_DIR"); strings.Contains(md, "/var/lib/teamwerk") {
		fmt.Fprintf(os.Stderr,
			"dev-seed: verweigert — MEDIA_DIR=%q zeigt auf das Prod-Verzeichnis (/var/lib/teamwerk).\n", md)
		os.Exit(1)
	}

	if dir := filepath.Dir(*dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal("dev-seed: mkdir db dir failed", "dir", dir, "error", err)
		}
	}

	// Frische DB erzwingen (deterministisch). WAL-Seitendateien mit entfernen.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(*dbPath + suffix)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		fatal("dev-seed: open db failed", "error", err)
	}
	defer database.Close()

	r := seedE2E(database)

	fmt.Println("========================================================================")
	fmt.Println("  dev-seed: Entwickler-DB neu aufgesetzt")
	fmt.Println("========================================================================")
	fmt.Printf("  DB-Pfad     : %s\n", *dbPath)
	fmt.Printf("  MEDIA_DIR   : %s\n", r.mediaDir)
	fmt.Println("  Login       : e2e@test.local / E2ETestPassword!  (Rolle: admin)")
	fmt.Println()
	fmt.Println("  Chat-Konversationen (Namen matchen die Playwright-Tests):")
	fmt.Println("    - E2E Chat                         (8 Textnachrichten)")
	fmt.Println("    - E2E Chat mit Bildern             (Text + 4 Bilder)")
	fmt.Println("    - E2E Chat unread                  (28 Nachrichten, unread=3)")
	fmt.Println("    - E2E Chat lang gelesen            (150 Nachrichten, ~21 Bilder, unread=0)")
	fmt.Println("    - E2E Chat lang unread             (150 Nachrichten, unread=40)")
	fmt.Println("    - E2E Chat viele ungelesen         (250 Nachrichten, unread=180, Chip-Fall)")
	fmt.Println()
	fmt.Println("  Starten:")
	fmt.Printf("    1) Terminal A:  DB_PATH=%s MEDIA_DIR=%s go run ./cmd/teamwerk\n", *dbPath, r.mediaDir)
	fmt.Println("    2) Terminal B:  pnpm -C web dev")
	fmt.Println("    3) Browser:     http://localhost:5173/chat")
	fmt.Println("========================================================================")
}
