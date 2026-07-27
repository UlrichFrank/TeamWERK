package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// runE2ESeed legt eine deterministische Test-Datenbank für die Playwright-E2E-Suite an.
// Verwendung: teamwerk e2e-seed --db=./e2e.db
//
// Die Datei wird bei jedem Lauf frisch erzeugt (idempotent: gleicher Aufruf → gleicher
// Zustand). Enthält 1 Admin + 3 Standard-Nutzer, eine kleine Text-Chat-Konversation,
// eine zukünftige Dienstbörsen-Zeile, einen minimalen Verein + Mitglied (Tresor-Smoke)
// sowie Chat-Konversationen mit Bildern / Unread-Zustand für die Scroll-Tests.
// Bild-Dateien landen im MEDIA_DIR (Default ./storage/media, überschreibbar per Env).
func runE2ESeed() {
	fs := flag.NewFlagSet("e2e-seed", flag.ExitOnError)
	dbPath := fs.String("db", "", "Pfad zur SQLite-Datenbank (Pflicht)")
	_ = fs.Parse(os.Args[2:])
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "Verwendung: teamwerk e2e-seed --db=<pfad>")
		os.Exit(1)
	}

	// Frische DB erzwingen (deterministisch). WAL-Seitendateien mit entfernen.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(*dbPath + suffix)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		fatal("e2e-seed: open db failed", "error", err)
	}
	defer database.Close()

	r := seedE2E(database)

	fmt.Printf("e2e-seed: DB %s angelegt (admin=%d, users=%v)\n", *dbPath, r.adminID, r.userIDs)
	fmt.Printf("e2e-seed: duty_slot=%d, member=%d, media_dir=%s\n", r.dutySlotID, r.memberID, r.mediaDir)
}

// e2eSeedResult bündelt die IDs/Pfade, die runE2ESeed für die Zusammenfassung
// ausgibt und Tests zum Assertieren nutzen.
type e2eSeedResult struct {
	adminID    int
	userIDs    []int
	dutySlotID int
	memberID   int
	mediaDir   string
}

// seedE2E migriert die (bereits geöffnete) DB und trägt den deterministischen
// E2E-Datensatz ein. Bewusst von der CLI-Hülle (runE2ESeed) getrennt, damit die
// Seed-Logik in Go-Tests (Idempotenz, Login) direkt ausführbar ist.
func seedE2E(database *sql.DB) e2eSeedResult {
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		fatal("e2e-seed: migrate failed", "error", err)
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte("E2ETestPassword!"), bcrypt.DefaultCost)
	if err != nil {
		fatal("e2e-seed: bcrypt failed", "error", err)
	}
	adminID := mustInsertUser(database, "e2e@test.local", string(adminHash), "admin", "E2E", "Admin")

	// Drei Standard-Nutzer (gleiches Passwort für Einfachheit).
	stdHash, _ := bcrypt.GenerateFromPassword([]byte("E2ETestPassword!"), bcrypt.DefaultCost)
	userIDs := []int{
		mustInsertUser(database, "user1@test.local", string(stdHash), "standard", "Uwe", "Eins"),
		mustInsertUser(database, "user2@test.local", string(stdHash), "standard", "Uta", "Zwei"),
		mustInsertUser(database, "user3@test.local", string(stdHash), "standard", "Udo", "Drei"),
	}

	// Eine Gruppen-Konversation Admin + User1 mit ein paar Textnachrichten.
	seedTextConversation(database, "E2E Chat", []int{adminID, userIDs[0]}, adminID, 8)

	// Golden-Path-Daten für Dienstbörse, Tresor/Bank und Chat-Bilder/-Scroll.
	dutySlotID := seedDienstboerse(database)
	memberID := seedClubAndMember(database)
	mediaDir := seedChatMedia(database, adminID, userIDs)

	// Sehr lange, bildlastige Threads für die Scroll-/Divider-/Chip-Regressionstests.
	seedChatLongThreads(database, mediaDir, adminID, userIDs)

	return e2eSeedResult{adminID: adminID, userIDs: userIDs, dutySlotID: dutySlotID, memberID: memberID, mediaDir: mediaDir}
}

func mustInsertUser(database *sql.DB, email, hash, role, first, last string) int {
	res, err := database.Exec(
		`INSERT INTO users (email, password, role, first_name, last_name) VALUES (?, ?, ?, ?, ?)`,
		email, hash, role, first, last)
	if err != nil {
		fatal("e2e-seed: insert user failed", "email", email, "error", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// seedTextConversation legt eine Gruppen-Konversation mit Mitgliedern + n Textnachrichten an.
func seedTextConversation(database *sql.DB, title string, memberIDs []int, senderID, n int) int {
	res, err := database.Exec(
		`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`, title, senderID)
	if err != nil {
		fatal("e2e-seed: insert conversation failed", "error", err)
	}
	convID, _ := res.LastInsertId()
	for _, m := range memberIDs {
		if _, err := database.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, m); err != nil {
			fatal("e2e-seed: insert conversation_member failed", "error", err)
		}
	}
	for i := 0; i < n; i++ {
		if _, err := database.Exec(
			`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
			convID, senderID, fmt.Sprintf("E2E Nachricht %d", i+1)); err != nil {
			fatal("e2e-seed: insert message failed", "error", err)
		}
	}
	return int(convID)
}

// seedDienstboerse legt eine aktive (zukünftig endende) Saison, ein Team, einen Duty-Type
// und einen offenen, in der Zukunft liegenden Dienst-Slot an. Gibt die duty_slot-ID zurück.
func seedDienstboerse(database *sql.DB) int {
	seasonRes, err := database.Exec(
		`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 1)`,
		"2025/26", "2025-09-01", "2027-06-30")
	if err != nil {
		fatal("e2e-seed: insert season failed", "error", err)
	}
	seasonID, _ := seasonRes.LastInsertId()

	teamRes, err := database.Exec(
		`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`,
		"E2E Team", "Erwachsene", "mixed")
	if err != nil {
		fatal("e2e-seed: insert team failed", "error", err)
	}
	teamID, _ := teamRes.LastInsertId()

	dtRes, err := database.Exec(
		`INSERT INTO duty_types (name, hours_value) VALUES (?, ?)`, "Kasse", 1.0)
	if err != nil {
		fatal("e2e-seed: insert duty_type failed", "error", err)
	}
	dutyTypeID, _ := dtRes.LastInsertId()

	slotRes, err := database.Exec(
		`INSERT INTO duty_slots
		    (event_name, event_date, event_time, duty_type_id, slots_total, slots_filled, team_id, season_id, game_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		"E2E Dienst", "2026-08-15", "10:00", dutyTypeID, 2, 0, teamID, seasonID)
	if err != nil {
		fatal("e2e-seed: insert duty_slot failed", "error", err)
	}
	slotID, _ := slotRes.LastInsertId()
	return int(slotID)
}

// seedClubAndMember legt (falls nötig) eine minimale clubs-Zeile an — der Tresor-Setup
// (config/vault.go) aktualisiert die erste clubs-Zeile und schlägt ohne Zeile still fehl —
// und ein Mitglied für den Bank-/Tresor-Smoke-Test. Gibt die member-ID zurück.
func seedClubAndMember(database *sql.DB) int {
	var clubCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM clubs`).Scan(&clubCount); err != nil {
		fatal("e2e-seed: count clubs failed", "error", err)
	}
	if clubCount == 0 {
		if _, err := database.Exec(`INSERT INTO clubs (name) VALUES (?)`, "E2E Verein"); err != nil {
			fatal("e2e-seed: insert club failed", "error", err)
		}
	}

	memRes, err := database.Exec(
		`INSERT INTO members (first_name, last_name, status, join_date) VALUES (?, ?, ?, ?)`,
		"E2E", "Kontakt", "aktiv", "2025-09-01")
	if err != nil {
		fatal("e2e-seed: insert member failed", "error", err)
	}
	memberID, _ := memRes.LastInsertId()
	return int(memberID)
}

// seedChatMedia legt drei Chat-Konversationen für die Scroll-/Unread-/Deep-Link-Tests an
// und schreibt die zugehörigen PNG-Dateien ins MEDIA_DIR. Gibt das verwendete MEDIA_DIR
// zurück. userIDs: [user1=2, user2=3, user3=4].
func seedChatMedia(database *sql.DB, adminID int, userIDs []int) string {
	user1, user2 := userIDs[0], userIDs[1]

	mediaDir := getEnvOrDefault("MEDIA_DIR", "./storage/media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		fatal("e2e-seed: mkdir media_dir failed", "dir", mediaDir, "error", err)
	}

	// 1) Gruppe mit Bildern: Text + 4 Bild-Nachrichten, alle vom Admin (→ für Admin gelesen).
	imgConvID := seedTextConversation(database, "E2E Chat mit Bildern", []int{adminID, user1}, adminID, 4)
	imgIndex := 1
	for i := 0; i < 4; i++ {
		mediaID := seedImage(database, mediaDir, adminID, imgIndex, true)
		imgIndex++
		if _, err := database.Exec(
			`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`,
			imgConvID, adminID, mediaID); err != nil {
			fatal("e2e-seed: insert image message failed", "error", err)
		}
	}

	// 2) Gruppe mit Unread: 28 Nachrichten von user1 (NICHT Admin). message_reads für
	//    alle außer den letzten 3 als gelesen markieren → unread(admin) = 3.
	unreadConvID := seedGroupConversation(database, "E2E Chat unread", []int{adminID, user1}, adminID)
	var msgIDs []int
	for i := 0; i < 28; i++ {
		res, err := database.Exec(
			`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
			unreadConvID, user1, fmt.Sprintf("Unread Nachricht %d", i+1))
		if err != nil {
			fatal("e2e-seed: insert unread message failed", "error", err)
		}
		id, _ := res.LastInsertId()
		msgIDs = append(msgIDs, int(id))
	}
	// Alle außer den letzten 3 als vom Admin gelesen markieren.
	for _, mid := range msgIDs[:len(msgIDs)-3] {
		if _, err := database.Exec(
			`INSERT INTO message_reads (message_id, user_id) VALUES (?, ?)`, mid, adminID); err != nil {
			fatal("e2e-seed: insert message_read failed", "error", err)
		}
	}

	// 3) Direkt-Konversation Admin + user2 mit ~20 Textnachrichten (Deep-Link-Ziel).
	directConvID := seedDirectConversation(database, []int{adminID, user2}, adminID)
	for i := 0; i < 20; i++ {
		if _, err := database.Exec(
			`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
			directConvID, adminID, fmt.Sprintf("Direct Nachricht %d", i+1)); err != nil {
			fatal("e2e-seed: insert direct message failed", "error", err)
		}
	}

	return mediaDir
}

// seedGroupConversation legt eine leere Gruppen-Konversation mit Mitgliedern an.
func seedGroupConversation(database *sql.DB, title string, memberIDs []int, createdBy int) int {
	res, err := database.Exec(
		`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`, title, createdBy)
	if err != nil {
		fatal("e2e-seed: insert group conversation failed", "error", err)
	}
	convID, _ := res.LastInsertId()
	for _, m := range memberIDs {
		if _, err := database.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, m); err != nil {
			fatal("e2e-seed: insert conversation_member failed", "error", err)
		}
	}
	return int(convID)
}

// seedDirectConversation legt eine Direkt-Konversation (type='direct', name NULL) an.
func seedDirectConversation(database *sql.DB, memberIDs []int, createdBy int) int {
	res, err := database.Exec(
		`INSERT INTO conversations (type, name, created_by) VALUES ('direct', NULL, ?)`, createdBy)
	if err != nil {
		fatal("e2e-seed: insert direct conversation failed", "error", err)
	}
	convID, _ := res.LastInsertId()
	for _, m := range memberIDs {
		if _, err := database.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, m); err != nil {
			fatal("e2e-seed: insert conversation_member failed", "error", err)
		}
	}
	return int(convID)
}

// seedImage erzeugt ein deterministisches 300x450-PNG (Hochformat, wie ein Foto),
// schreibt es als e2e-img-<index>.png ins mediaDir und legt die zugehörige media-Zeile an.
// Gibt die media-ID zurück.
//
// withDims steuert, ob die media-Zeile width/height trägt: `true` = normaler Pfad
// (Server kennt die Dimensionen, Frontend rendert ohne Layout-Shift). `false` legt eine
// Zeile mit width=NULL/height=NULL an (wie ein Bestandsbild vor Migration 030) — das
// exerziert den AuthImage-Fallback-Pfad im Frontend (Placeholder→Bild-Shift).
func seedImage(database *sql.DB, mediaDir string, uploadedBy, index int, withDims bool) int {
	const w, h = 300, 450
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Feste, pro Index leicht variierende Farbe (deterministisch, aber unterscheidbar).
	fill := color.RGBA{R: uint8(40 + index*30), G: uint8(90 + index*20), B: 200, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{fill}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		fatal("e2e-seed: png encode failed", "error", err)
	}

	diskName := fmt.Sprintf("e2e-img-%d.png", index)
	if err := os.WriteFile(filepath.Join(mediaDir, diskName), buf.Bytes(), 0o644); err != nil {
		fatal("e2e-seed: write png failed", "file", diskName, "error", err)
	}

	var (
		res sql.Result
		err error
	)
	if withDims {
		res, err = database.Exec(
			`INSERT INTO media (disk_name, mime_type, size, uploaded_by, width, height) VALUES (?, ?, ?, ?, ?, ?)`,
			diskName, "image/png", buf.Len(), uploadedBy, w, h)
	} else {
		// NULL-Dims: exerziert den AuthImage-Fallback (kein Server-Dim → Layout-Shift).
		res, err = database.Exec(
			`INSERT INTO media (disk_name, mime_type, size, uploaded_by) VALUES (?, ?, ?, ?)`,
			diskName, "image/png", buf.Len(), uploadedBy)
	}
	if err != nil {
		fatal("e2e-seed: insert media failed", "file", diskName, "error", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// longThreadBaseTime ist die feste Basiszeit für gespreizte sent_at-Werte in
// seedLongThread — deterministisch (KEIN time.Now()), damit die Seeds reproduzierbar
// sind. Der Schritt (17 min) sorgt dafür, dass 150–250 Nachrichten mehrere Tage
// überspannen und im Frontend DaySeparator-Trenner erscheinen.
var longThreadBaseTime = time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)

const longThreadStep = 17 * time.Minute

// seedLongThread legt eine Gruppen-Konversation mit n Nachrichten an und setzt sent_at
// EXPLIZIT (deterministisch, aufsteigend gespreizt), damit Scroll-/Divider-/DaySeparator-
// Verhalten im Frontend testbar ist.
//
//   - senderFor(i)  → Absender der i-ten Nachricht (0-basiert). Nachrichten von einem
//     anderen Nutzer als dem Leser (Admin) zählen als unread.
//   - imageAt(i)    → (isImage, withDims): true legt statt Text ein Bild an (media_id
//     gesetzt, body=”), withDims steuert width/height der media-Zeile.
//   - imgCounter    → wird pro Bild hochgezählt, garantiert eindeutige disk_names über
//     alle Konversationen hinweg.
//
// Gibt convID + alle message-IDs (aufsteigend nach sent_at) zurück.
func seedLongThread(
	database *sql.DB, mediaDir, title string, memberIDs []int, createdBy, n int,
	senderFor func(i int) int, imageAt func(i int) (bool, bool), imgCounter *int,
) (int, []int) {
	convID := seedGroupConversation(database, title, memberIDs, createdBy)

	msgIDs := make([]int, 0, n)
	for i := 0; i < n; i++ {
		sentAt := longThreadBaseTime.Add(time.Duration(i) * longThreadStep).Format(time.RFC3339)
		sender := senderFor(i)

		isImage, withDims := imageAt(i)
		var (
			res sql.Result
			err error
		)
		if isImage {
			*imgCounter++
			mediaID := seedImage(database, mediaDir, sender, *imgCounter, withDims)
			res, err = database.Exec(
				`INSERT INTO messages (conversation_id, sender_id, body, media_id, sent_at) VALUES (?, ?, '', ?, ?)`,
				convID, sender, mediaID, sentAt)
		} else {
			// Body bewusst OHNE den Konversationstitel — sonst enthält die
			// Last-Message-Vorschau in der Sidebar den Titel und getByText(name)
			// im E2E-Test kollidiert (strict-mode: Name + Vorschau).
			res, err = database.Exec(
				`INSERT INTO messages (conversation_id, sender_id, body, sent_at) VALUES (?, ?, ?, ?)`,
				convID, sender, fmt.Sprintf("Nachricht %d", i+1), sentAt)
		}
		if err != nil {
			fatal("e2e-seed: insert long-thread message failed", "title", title, "error", err)
		}
		id, _ := res.LastInsertId()
		msgIDs = append(msgIDs, int(id))
	}
	return convID, msgIDs
}

// markReadByAdmin markiert die übergebenen message-IDs als vom Admin gelesen
// (INSERT INTO message_reads). Nachrichten, die der Admin selbst gesendet hat, sind
// für die Unread-Berechnung ohnehin ausgenommen — doppeltes Markieren schadet nicht.
func markReadByAdmin(database *sql.DB, adminID int, msgIDs []int) {
	for _, mid := range msgIDs {
		if _, err := database.Exec(
			`INSERT INTO message_reads (message_id, user_id) VALUES (?, ?)`, mid, adminID); err != nil {
			fatal("e2e-seed: insert long-thread message_read failed", "error", err)
		}
	}
}

// imageAtSet baut aus einer Map (message-Index → withDims) die imageAt-Funktion für
// seedLongThread. Ist der Index in der Map, wird ein Bild gesetzt (mit dem hinterlegten
// withDims-Flag); sonst Text.
func imageAtSet(withDimsByIdx map[int]bool) func(i int) (bool, bool) {
	return func(i int) (bool, bool) {
		wd, ok := withDimsByIdx[i]
		return ok, wd
	}
}

// seedChatLongThreads legt die drei langen, bildlastigen Chat-Threads für die
// Scroll-/Divider-/Chip-Regressionstests an. Leser/Login ist der Admin (e2e@test.local).
//
//	"E2E Chat lang gelesen"        150 Nachrichten, ~20 Bilder, komplett vom Admin
//	                               → unreadCount(admin) = 0  (zuverlässiges „ans Ende").
//	"E2E Chat lang unread"         150 Nachrichten von user1, Bilder über UND unter dem
//	                               Divider; Admin liest alle außer den letzten 40
//	                               → unreadCount(admin) = 40 (Divider auf 100er-Seite).
//	"E2E Chat unread älter als Seite" 250 Nachrichten von user1, Admin liest alle außer
//	                               den letzten 180 → unreadCount(admin) = 180 (>100,
//	                               erste Ungelesene älter als die geladene Seite → Chip).
func seedChatLongThreads(database *sql.DB, mediaDir string, adminID int, userIDs []int) {
	user1 := userIDs[0]
	// Eindeutige Bild-Indizes ab 1000, damit disk_names nicht mit seedChatMedia (1–4) kollidieren.
	imgCounter := 1000

	// 1) "E2E Chat lang gelesen": 150 Nachrichten, alle vom Admin → unread=0 (Sender==Leser).
	//    ~20 Bilder über den ganzen Verlauf verteilt, dims abwechselnd mit/ohne.
	langGelesenImgs := map[int]bool{}
	for i := 5; i < 150; i += 7 { // 5,12,...,145 → 21 Bilder
		langGelesenImgs[i] = (i/7)%2 == 0 // abwechselnd with/without dims
	}
	seedLongThread(database, mediaDir, "E2E Chat lang gelesen", []int{adminID, user1}, adminID, 150,
		func(int) int { return adminID }, imageAtSet(langGelesenImgs), &imgCounter)

	// 2) "E2E Chat lang unread": 150 Nachrichten von user1. Admin liest alle außer den
	//    letzten 40 → unread=40 (Divider innerhalb der neuesten 100er-Seite).
	//    Bilder über dem Divider (Indizes < 110, gelesen) und darunter (>= 110, ungelesen).
	langUnreadImgs := map[int]bool{
		9: true, 24: false, 44: true, 69: false, 94: true, // über dem Divider (gelesen)
		118: false, 133: true, 147: false, // unter dem Divider (ungelesen)
	}
	_, unreadMsgIDs := seedLongThread(database, mediaDir, "E2E Chat lang unread", []int{adminID, user1}, adminID, 150,
		func(int) int { return user1 }, imageAtSet(langUnreadImgs), &imgCounter)
	markReadByAdmin(database, adminID, unreadMsgIDs[:len(unreadMsgIDs)-40]) // erste 110 gelesen

	// 3) "E2E Chat unread älter als Seite": 250 Nachrichten von user1. Admin liest alle
	//    außer den letzten 180 → unread=180 (>100 → erste Ungelesene älter als die
	//    geladene 100er-Seite → Chip-Fall). Ein paar Bilder verteilt.
	aelterImgs := map[int]bool{
		15: true, 60: false, 120: true, 175: false, 230: true,
	}
	_, aelterMsgIDs := seedLongThread(database, mediaDir, "E2E Chat viele ungelesen", []int{adminID, user1}, adminID, 250,
		func(int) int { return user1 }, imageAtSet(aelterImgs), &imgCounter)
	markReadByAdmin(database, adminID, aelterMsgIDs[:len(aelterMsgIDs)-180]) // erste 70 gelesen
}
