## Context

Die Mein-Team-Seite (`web/src/pages/MeinTeamPage.tsx`) zeigt heute pro Team eine aufklappbare Karte mit Roster-Tabs (Team/Trainer/Eltern), gespeist aus `GET /api/teams/{id}/roster` und `GET /api/teams/my`. Team-Zugehörigkeit ist ausschließlich über den Kader der **aktiven Saison** definiert:

- Spieler → `kader_members`, Trainer → `kader_trainers`, Erweiterter Kader → `kader_extended_members`, Eltern → `family_links`.
- `kader.team_id` (nullable) bindet einen Saison-Kader an die durable `teams`-Identität; `seasons.is_active` markiert die aktive Saison.
- Der Master-Zugriffs-View `user_accessible_teams` (Spieler + Trainer + Erweitert + **Eltern**) steuert, wer ein Team überhaupt sieht — und speist die Roster-Response.

Vereinsfunktionen (`member_club_functions`, global) sind orthogonal zur Team-Zugehörigkeit. Team-Scoping passiert nicht im Router, sondern in Handlern/`internal/policy/rules.go` (SQL-Fragmente, z.B. Trainer-Scoping via `kader_trainers`-Join auf `members.user_id`).

Dieses Feature setzt zwei team-interne Mechanismen (Aufgaben, Strafen) obendrauf — beide kader-scoped.

## Goals / Non-Goals

**Goals:**
- Aufgaben pro Spieler im Team sichtbar machen (Trainer pflegt Catalog + Zuweisung), ohne Semantik.
- Strafen mit variablem Betrag durch einen pro Kader ernannten **Strafenwart** verwalten (vergeben/stornieren/zurücksetzen).
- Eine harte Sichtbarkeitsgrenze: Strafen nur für Spieler + Trainer des Teams (Stamm + Erweitert), **nie** für Eltern oder Außenstehende.
- Konsistenz mit dem bestehenden Kader/Saison-Modell und den Hard Rules (Broadcast, Tests, brand-Tokens, lucide).

**Non-Goals:**
- Keine Zahlungs-Historie/Audit-Trail (Storno + Reset löschen echt; nur offener Kassenstand).
- Kein neuer globaler `member_club_functions`-Wert, keine Änderung am Berechtigungsmodell.
- Keine Erinnerungen/Push/Workflows für Aufgaben oder Strafen.
- Kein SEPA-/Beitrags-Bezug — die Mannschaftskasse ist rein informell.

## Decisions

### D1 — Strafenwart als per-Kader-Appointment statt globaler Vereinsfunktion
Neue Table `kader_strafenwarte(kader_id, member_id)`, Sibling von `kader_trainers`. Ernennung durch den Trainer des Kaders.

**Warum:** Da alles kader/saison-scoped ist, wäre ein globaler `member_club_functions`-Wert semantisch falsch („Strafenwart überall/für immer" statt „dieses Team, diese Saison"). Der per-Kader-Ansatz (a) vermeidet CHECK-Constraint-Migration + neuen JWT-Claim, (b) macht das Write-Gate zu einem simplen DB-Lookup analog zum bestehenden Trainer-Scoping, (c) löst gratis, dass ein Strafenwart nur sein eigenes Team bestrafen kann.

**Alternative verworfen:** globaler Funktionswert + Team-Membership-Check. Invasiver (Migration, JWT, `HasFunction`) und drückt „per Team" nicht sauber aus.

### D2 — Strafen NICHT auf der Roster-Response; eigener Endpoint mit eigenem Read-Gate
`GET /api/teams/{id}/penalties` ist eine eigene Route. Read-Gate: Caller-Member ist Spieler (`kader_members`) **oder** Trainer (`kader_trainers`) **oder** Erweiterter Kader (`kader_extended_members`) des Kaders der aktiven Saison — Eltern (`family_links`) und Außenstehende bekommen **403**.

**Warum:** Die Roster-Response wird über `user_accessible_teams` auch an Eltern ausgeliefert. Ein Strafen-Feld auf der Roster-Response würde die Sichtbarkeitsgrenze durchbrechen. Aufgaben dagegen dürfen auf der Roster-Response mitreiten, weil ihre Sichtbarkeit exakt der Roster-Sichtbarkeit entspricht.

**Alternative verworfen:** ein gemeinsamer „team-detail"-Endpoint mit gemischter Sichtbarkeit — zu fehleranfällig; die Asymmetrie muss auf Endpoint-Ebene hart getrennt sein.

### D3 — Snapshot statt Referenz bei Zuweisungen
`member_responsibilities.label`, `team_penalties.reason` und `team_penalties.amount_cent` sind **Snapshots** (kopierte Werte), nicht FKs auf den Catalog. Der Catalog (`responsibility_types`, `penalty_types`) treibt nur das Dropdown + Default-Betrag.

**Warum:** Bei Geld darf ein späterer Catalog-Edit eine bereits vergebene Strafe nicht rückwirkend ändern. Snapshots entkoppeln „Vorschlag" von „Fakt". Gleiche Logik für Aufgaben-Labels (Freitext ist ohnehin nötig).

### D4 — Zwei echte Lösch-Operationen, kein Status
`team_penalties` hat **keine** Status-Spalte. **Storno** = `DELETE /api/teams/{id}/penalties/{pid}` (eine Row). **Zurücksetzen je Spieler** = `DELETE /api/teams/{id}/penalties?member={mid}` (alle Rows des Members im Kader). Beides hard delete.

**Warum:** Bewusst einfach (siehe Non-Goals). Kein Strikethrough/„storniert"-Zustand — der offene Kassenstand ist die einzige Wahrheit.

### D5 — Write-Gates
- Aufgaben-Catalog-CRUD, Aufgaben-Zuweisung, Strafen-Catalog-CRUD, Strafenwart-Ernennung: **Trainer des Kaders** (`kader_trainers`-Lookup; `admin` passt via bestehender Konvention immer).
- Strafe vergeben/stornieren/zurücksetzen: **Strafenwart des Kaders** (`kader_strafenwarte`-Lookup).

Beide Gates sind DB-Lookups im Handler (bzw. als Helper in `internal/policy/rules.go`), nicht Router-Middleware — Team-Scoping ist im Projekt grundsätzlich handler-seitig.

### D6 — SSE-Events
Mutationen broadcasten `responsibilities` bzw. `penalties` via `h.hub.Broadcast(...)`. `MeinTeamPage` abonniert via `useLiveUpdates` und lädt die betroffenen Rosters/Strafen neu. Erfüllt die Broadcast-Hard-Rule und `internal/arch/broadcast_test.go`.

### D7 — Team→Kader-Auflösung
Die Endpoints sind team-scoped (`/teams/{id}/...`, konsistent mit Roster). Intern wird `team_id` + aktive Saison → `kader_id` aufgelöst, exakt wie `GetRoster` es heute tut (inkl. dessen Umgang mit ggf. mehreren Kadern pro Team/Saison via `team_number`). Die neuen Tables referenzieren `kader_id`.

## Risks / Trade-offs

- **Sichtbarkeitsleck bei Strafen** → schwerste Fehlerklasse. Mitigation: eigener Endpoint (D2), explizite Negativ-Tests (Eltern → 403, Fremd-Team-Strafenwart → 403), Read-Gate als benannter, getesteter Helper.
- **Fremd-Team-Bestrafung durch globalen Strafenwart** → durch D1 (per-Kader-Appointment) strukturell ausgeschlossen; zusätzlich Test (c).
- **Kein Audit-Trail** → bewusst (Non-Goal). Risiko: Streit „hab ich doch bezahlt". Akzeptiert für v1; Follow-up-Change möglich.
- **Mehrere Kader pro Team/Saison** (`team_number`) → Auflösung an der bestehenden `GetRoster`-Logik ausrichten (D7), nicht neu erfinden; sonst Drift.
- **Catalog verwaist bei Kader-Löschung** → FKs mit `ON DELETE CASCADE` auf `kader(id)` (wie `kader_members`), damit keine Leichen bleiben.
- **RAM/VPS** → keine neuen Dependencies, reine SQLite-Tables + Handler. Kein Footprint-Risiko.

## Migration Plan

1. Neue Migration `0NN_mannschaft_aufgaben_strafen.up.sql` / `.down.sql` (nächste freie Nummer): fünf Tables, alle FK auf `kader(id) ON DELETE CASCADE` (bzw. `members(id)` für `member_id`/`created_by_member_id`). Beträge als `INTEGER` (Cent).
2. Backend-Handler + Routen + Policy-Helper + Tests.
3. Frontend (MeinTeamPage) + `useLiveUpdates`.
4. Rollback: `.down.sql` droppt die fünf Tables (rein additiv, kein Datenverlust an Bestand).

## Open Questions

- Genaue UI-Platzierung der Trainer-Verwaltung (inline auf MeinTeamPage vs. eigener Abschnitt/Modal) — wird in der Umsetzung entschieden, keine Spec-Auswirkung.
- Anzeige-Detail Kassenstand (nur Summe pro Spieler vs. zusätzlich Team-Gesamtsumme) — Darstellungsdetail, keine Spec-Auswirkung.
