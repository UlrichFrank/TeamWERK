package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <db_path>\n", os.Args[0])
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	cmds := []string{
		`ALTER TABLE duty_types ADD COLUMN same_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (same_day_behavior IN ('normal', 'skip', 'reduced'))`,
		`ALTER TABLE duty_types ADD COLUMN same_day_variant_id INTEGER REFERENCES duty_types(id)`,
		`ALTER TABLE duty_types ADD COLUMN adjacent_day_behavior TEXT NOT NULL DEFAULT 'normal' CHECK (adjacent_day_behavior IN ('normal', 'skip', 'reduced'))`,
		`ALTER TABLE duty_types ADD COLUMN adjacent_day_variant_id INTEGER REFERENCES duty_types(id)`,
	}

	for _, sql := range cmds {
		_, err := db.Exec(sql)
		if err == nil {
			fmt.Printf("OK: %s...\n", sql[:60])
		} else if err.Error() == "UNIQUE constraint failed: sqlite_master.name" {
			fmt.Printf("SKIP (already exists): %s...\n", sql[:60])
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: %v for: %s\n", err, sql[:60])
		}
	}
}
