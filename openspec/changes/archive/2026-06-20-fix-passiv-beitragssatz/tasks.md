# Tasks: Passiv-Beitragssatz ab Saisonstart gültig

## 1. Datenbank-Migration

- [x] 1.1 `internal/db/migrations/046_passiv_beitragssatz_saisonstart.up.sql` anlegen:
  - `INSERT OR IGNORE INTO beitrags_saetze (kategorie, betrag_eur, valid_from) VALUES ('passiv', 6000, '2026-07-01');`
  - Kommentar mit Bezug auf den Datums-Bug (Stichtag 01.07., alter Satz erst ab 2027-01-01).
  - Commit: `fix(db): Migration 046 — Passiv-Satz ab Saisonstart 2026/27 gültig`
- [x] 1.2 Korrespondierende `.down.sql`:
  - `DELETE FROM beitrags_saetze WHERE kategorie='passiv' AND valid_from='2026-07-01' AND betrag_eur=6000;`
  - Commit: Teil von 1.1.
- [x] 1.3 Up-Migration auf Temp-DB verifizieren: `SELECT * FROM beitrags_saetze WHERE kategorie='passiv'` zeigt zwei Sätze (2026-07-01 und 2027-01-01).

## 2. Test

- [x] 2.1 In `internal/beitragslauf/handler_test.go` einen Fall ergänzen, der eine Saison mit Start **2026-07-01** verwendet und ein Mitglied mit `status='passiv'` prüft:
  - Erwartung: `it.Included == true`, `it.Kategorie == "passiv"`, `it.BetragCent == 6000`, keine `kein_beitragssatz`-Exclusion.
  - (Bestehender Test bei `handler_test.go:165` nutzt Saison 2027/28 — der neue Fall sichert explizit die Saison 2026/27 ab, in der der Bug auftrat.)
  - Commit: `test(beitragslauf): passives Mitglied wird in Saison 2026/27 einbezogen`

## 3. Verifikation & Abschluss

- [x] 3.1 `/verify-change` ausführen (Build/Test/Lint, Migrationsnummer, `openspec validate`).
- [x] 3.2 `make test` grün (inkl. `internal/beitragslauf`).
- [x] 3.3 Proposal archivieren.
