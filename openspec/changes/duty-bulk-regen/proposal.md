## Why

Dienst-Slots entstehen heute ausschließlich als Nebenwirkung einer **einzelnen**
Spieländerung: `POST/PUT/DELETE /api/games/{id}` löst `runAutoRegen` über das Fenster
Datum ± 1 Tag aus. Das deckt den laufenden Betrieb ab, aber nicht die drei Fälle, in denen
der Vorstand tatsächlich am Dienstplan arbeitet:

1. **Nach dem H4A-Massenimport** trägt ein Teil der Spiele kein oder das falsche Template.
   Es gibt keinen Weg, „allen Heimspielen der Restsaison Template X" zuzuweisen, außer
   jedes Spiel einzeln zu öffnen und zu speichern (~70 Klicks pro Saisonhälfte).
2. **Nach einer Template-Änderung** (Dienstart ergänzt, Zeitversatz korrigiert) driften die
   bestehenden Slots stumm vom Template weg. Der Regen läuft nur bei Spieländerungen, nicht
   bei Template-Änderungen — die alten Slots bleiben stehen, und niemand sieht die Differenz.
3. **Aufräumen**: Termine, die versehentlich Dienste bekommen haben (oder keine bekommen
   haben), lassen sich nur einzeln korrigieren.

Die Engine dafür existiert bereits: `runAutoRegen(tx, dates, seasonID)` nimmt eine beliebige
Datumsliste, und der H4A-Apply fährt sie schon heute über ~146 Spiele in einer Transaktion
mit einem Broadcast. Es fehlt der Treiber davor und die Kontrolle darüber.

Der Grund, warum es diesen Treiber noch nicht gibt, ist nicht technisch, sondern der
**Kollateralschaden**: `DELETE FROM duty_slots` reißt via `ON DELETE CASCADE` alle
`duty_assignments` mit. Bei einem Spiel sind das zwei Leute; über eine Restsaison ist es der
gesamte künftige Dienstplan. Dieser Change macht den Massenlauf deshalb erst durch zwei
Zusicherungen benutzbar: eine **exakte Vorschau vor dem Schreiben** und die **Rettung der
Zuweisungen**, deren Slot unverändert wiederkommt.

## What Changes

- **Neuer Massenlauf** „Dienste aktualisieren" im Kalender, erreichbar über das bestehende
  Dropdown am `+ Event`-Split-Button (dieselbe Stelle wie der H4A-Import, identisch auf
  Mobil und Desktop).
- **Zeitraum wählbar**, Default = `[morgen, MAX(games.date)]` der aktiven Saison. Die
  Vergangenheit ist gesperrt (Begründung: `duty_accounts.ist`, siehe `design.md` §6).
- **Vier Zustände pro Termin** — pauschal je Terminart (`heim` / `auswärts` / `generisch`)
  vorbelegt, pro Zeile überschreibbar:
  | Zustand | `games.template_id` | `is_custom=0`-Slots | `is_custom=1`-Slots |
  |---|---|---|---|
  | `template` | wird gesetzt | neu aus Template | bleiben |
  | `none` („keine Dienste anlegen") | `NULL` | gelöscht | bleiben |
  | `purge` („alle Dienste löschen") | `NULL` | gelöscht | **gelöscht** |
  | ausgenommen (Checkbox) | unverändert | unangetastet | unangetastet |
- **Live-Vorschau**: jede Änderung im Modal löst einen serverseitigen Dry-Run aus
  (vollständige Apply-Transaktion mit `ROLLBACK` statt `COMMIT`, debounced). Zeigt pro Zeile
  und in Summe: angelegt / gelöscht / handgemacht behalten / **Zuweisungen erhalten** /
  **Zuweisungen verloren** / Konflikte. Kein Nachbau der Regen-Logik im Browser.
- **Zuweisungen werden gerettet**: kommt ein Slot mit identischem
  `(duty_type_id, event_time, team_id)` wieder, werden seine `duty_assignments` samt
  `status`/`cash_amount`/`fulfilled_at` wiederhergestellt und `slots_filled` korrigiert.
  Gilt für **alle** Regen-Pfade, nicht nur den Massenlauf.
- **Benachrichtigung abschaltbar** (`notify: false`), Default an. Betrifft nur die
  `dispatchRegenNotifications`-Pushes an Betroffene, nie die `regen_summary` an den Auslöser.
- **Neue Capability** `bulk_regen_duties` (Vorstand/Admin) — bewusst enger als
  `manage_duties`, das auch Trainer haben.
- **Keine Migration.** Der Change ist rein verhaltensbezogen; das Schema bleibt unberührt.

## Capabilities

### Added Capabilities

- **`duty-bulk-regen`** — Massen-Regeneration der Dienst-Slots über einen wählbaren
  Zeitraum, mit Template-Zuweisung je Terminart und je Termin, Ausnahmeliste, den
  Lösch-Zuständen `none`/`purge`, serverseitiger Live-Vorschau (Dry-Run mit Rollback) und
  abschaltbarer Benachrichtigung.
- **`duty-assignment-preservation`** — Eine Dienst-Zuweisung überlebt jede Regeneration, bei
  der ihr Slot mit identischer Dienstart, Uhrzeit und Mannschaft wieder entsteht. Betrifft
  auch den bestehenden Einzelspiel-Regen.

## Test-Anforderungen

**Routen** (beide Vorstand-Tier, Admin-Bypass):

| Route | Fall | Erwartung |
|---|---|---|
| `POST /api/duty-slots/bulk-regen/preview` | Vorstand, gültiger Zeitraum | 200 + `{rows, totals, range}` |
| | ohne `from`/`to` | 200, Server liefert Default-Range zurück |
| | Standard-Nutzer ohne `vorstand` | 403 |
| | keine aktive Saison | 400 |
| | `from` ≤ heute | 400 `range_in_past` |
| | unbekannte `template_id` | 400 `invalid_template` |
| `POST /api/duty-slots/bulk-regen/apply` | Vorstand, gültiger Plan | 200 + Ergebnis, Slots geschrieben |
| | Standard-Nutzer | 403 |
| | keine aktive Saison | 400 |
| | `from` ≤ heute | 400 `range_in_past` |

**Garantierte Invarianten** (jede bekommt einen eigenen Test):

1. **Preview schreibt nicht.** Vollständiger DB-Snapshot vor und nach einem Preview-Request
   ist byte-identisch — auch bei `purge` über den gesamten Zeitraum. (Mechanisch, im Stil
   von `TestPreviewH4A_CredentialsWerdenNichtPersistiertOderGeloggt`.)
2. **Preview sagt die Wahrheit.** Derselbe Request an `preview` und danach an `apply`
   liefert identische `totals`; die tatsächlichen DB-Zählungen nach `apply` stimmen mit den
   Preview-Zahlen überein.
3. **Ausnahme schützt, entfernt aber keinen Kontext.** Bei zwei Spielen am selben Tag, von
   denen eines ausgenommen ist: die Slot-IDs des ausgenommenen Spiels sind unverändert, und
   das einbezogene Spiel erhält dieselbe `same_day_behavior`-Reduktion wie ohne Ausnahme.
4. **Zuweisung überlebt identische Regeneration.** Slot mit Zuweisung → Regen mit
   unverändertem Template → Zuweisung existiert weiter (ggf. an neuer Slot-ID),
   `slots_filled` stimmt, **keine** Benachrichtigung.
5. **Zuweisung geht verloren, wenn der Slot verschwindet.** Regen mit `none` → Zuweisung
   weg, genau **eine** Benachrichtigung an die betroffene Person.
6. **Überzählige Zuweisungen sind deterministisch.** Schrumpft `slots_total` von 3 auf 2,
   überleben die zwei ältesten Zuweisungen (nach `duty_assignments.id`), die dritte erzeugt
   eine „entfernt"-Benachrichtigung.
7. **`purge` löscht handgemacht, `none` nicht.** Gleicher Ausgangszustand, beide Läufe:
   `is_custom=1`-Slot ist nach `purge` weg und nach `none` unverändert vorhanden.
8. **`notify: false` sendet nichts.** Lauf mit garantiert Betroffenen → Notification-Spy
   zählt 0, obwohl `regen_summary.notified_users` befüllt ist.
9. **Ein Broadcast pro Lauf.** Apply über 40 Termine an 25 Tagen → genau ein
   `Broadcast("duties")` und ein `Broadcast("games")`, kein Sturm.
10. **Vergangenheit ist unerreichbar.** `from` in der Vergangenheit → 400, und kein Slot mit
    `event_date <= heute` wird in irgendeinem Lauf angefasst.
11. **Dienstoptimierung für aufeinanderfolgende Heimspiele bleibt zustands- und
    range-unabhängig.** Zwei Heimspiel-Tage in Folge im selben Lauf, der zweite Tag bekommt
    `none` → der erste behält seine `adjacent_day_behavior`-Reduktion (das Nachbarspiel
    existiert weiterhin, nur seine Dienst-Konfiguration ändert sich). Ebenso: ein Heimspiel
    am Vortag von `from` (außerhalb des Zeitraums) löst dieselbe Reduktion am ersten Termin
    im Zeitraum aus wie beim Einzelspiel-Regen.

**Frontend (Vitest):** Pauschalwahl setzt alle Zeilen; Zeilen-Override sticht die
Pauschalwahl; Ausnahme-Checkbox nimmt die Zeile aus den Summen; `purge`-Zeilen sind visuell
als destruktiv markiert; Vorschau-Anfragen sind gedebounced und brechen die laufende ab.

## Impact

- `internal/games/regen.go` — Kern: Ausnahme-Menge bis `loadDayGames` durchreichen,
  `RegenSummary` um eine Aufschlüsselung pro Spiel erweitern, Zuweisungs-Restore nach den
  Inserts. Betrifft geteilten Code, an dem auch der Einzelspiel-Regen und der H4A-Import
  hängen → die bestehende Charakterisierungssuite ist der Schutzzaun.
- `internal/games/bulkregen_handler.go` (neu) — die beiden Routen; liegt bei `games`, weil
  `runAutoRegen` unexportiert ist (gleiche Begründung wie `h4aimport_handler.go`).
- `internal/app/router.go` — zwei Routen im Vorstand-Tier.
- `internal/policy/rules.go` — `CapBulkRegenDuties`, vergeben an `IsVorstandLike`.
- `internal/arch/broadcast_test.go` — Allowlist-Eintrag für `Games.PreviewBulkRegen`
  („Dry-Run mit Rollback, kein DB-Write; ApplyBulkRegen broadcastet").
- `web/src/components/DutyBulkRegenModal.tsx` (neu) + `web/src/pages/KalenderPage.tsx`
  (Menüeintrag, Sichtbarkeits-Gate von `canImportGames` auf
  `canImportGames || canBulkRegenDuties` erweitern).
- `docs/agent/06-gotchas.md` — Gotcha zur Zuweisungs-Rettung und zur Kontext-vs-Ausnahme-
  Unterscheidung; `docs/agent/04-api-db.md` — neue Routen im Vorstand-Tier.
- **Kein** Schema-Änderung, **keine** Migration.
