## Why

Auch nach `scoped-live-updates` (weniger Empfänger) und `list-endpoint-pagination` (erste Seite kleiner) überträgt jeder Live-Update-getriggerte Reload schwerer Listen die **komplette** (Seite der) Liste, obwohl sich meist nur ein Datensatz geändert hat: bei 1 von 100 geänderten Spielen wandern 100 Datensätze zum Client.

Dieser Change führt **pull-basierte inkrementelle Synchronisation** für die schweren Listen-Endpoints ein: der Client hält einen Cursor und fragt „was hat sich seit X geändert?". Kombiniert mit `scoped-live-updates` ergibt das das eigentliche Ziel: `O(betroffene Clients × geänderte Zeilen)` statt `O(alle Clients × ganze Liste)`.

**Warum pull-basiert (und nicht Push-Delta):** In `scoped-live-updates` wurden Push-Delta-Events bewusst verworfen, weil der SSE-Channel bei Bursts verwirft (Buffer 1) — ein verlorenes Delta hieße dauerhaft veralteter Client. Beim **Pull**-Modell besitzt der Client den Cursor: ein verworfenes Event ist harmlos, der nächste `?since=`-Aufruf holt die Lücke nach. Kein Resync-Protokoll nötig; das SSE-Event dient nur noch als „jetzt nachfragen"-Trigger.

**Herkunft:** Dieser Change wurde aus `incremental-sync` **herausgelöst**. Die dortige Phase 0 (inkrementeller Chat über id-Cursor, `?after=`/`?before=`) ist bereits umgesetzt und verbleibt unter `incremental-sync`. Dieses Proposal bündelt die noch offene Substanz (Tiers 1–3): `updated_at`-Nachrüstung, Tombstones und den `?since=`-Cursor auf den schweren Listen.

## What Changes

- **Cursor-Parameter auf schweren Listen:** `GET /api/games`, `/api/duty-slots`, `/api/training-sessions`, `/api/kader` akzeptieren `?since=<cursor>` und liefern nur Datensätze mit `updated_at > cursor` (plus Lösch-Marker, siehe unten) sowie einen neuen Cursor. Ohne `?since=` unverändertes Verhalten (voller, paginierter Abruf, koexistiert mit `limit`/`offset`). Der Client kombiniert den Bestand lokal.
- **`updated_at` nachrüsten (nur wo nötig):** Migrationen ergänzen `updated_at` (App-seitiges Setzen bei INSERT/UPDATE) auf `games`, `duty_slots`, `training_sessions` — dort existiert heute nur `created_at` (Backfill-Default = `created_at`). Auf `kader`, `mitfahrgelegenheiten`, `mitfahrt_paarungen` ist `updated_at` **bereits vorhanden** → dort ohne Spalten-Migration nutzbar. `videos`/`members` bewusst optional (niederfrequent bzw. via `lazy-rendering`/`scoped-live-updates` abgedeckt) — nur bei gemessenem Bedarf. `game_attendances` (keine Zeitspalte, hochfrequent) bleibt Tier 3 / später.
- **Lösch-Marker (Tombstones):** Damit `?since=` auch Löschungen meldet (eine gelöschte Zeile taucht in „changed since" sonst nicht auf), liefert die Response neben `items` eine `deleted_ids`-Liste. Umsetzung über ein schlankes, append-only `sync_tombstones(entity, entity_id, deleted_at)`-Log. Tombstones werden nach einer Aufbewahrungsfrist beschnitten; Clients mit einem Cursor älter als die Frist erhalten `full: true` und bauen ihren Bestand neu auf.

## Geltungsbereich & Domänen-Priorität (schema-verifiziert)

| Tier | Domäne(n) | `updated_at` | Delete-Pfade (Tombstone) | Kosten |
|---|---|---|---|---|
| **1 — gratis** | `kader`, `mitfahrgelegenheiten`, `mitfahrt_paarungen` | **schon vorhanden** | je 1 (`kader`, `mitfahrgelegenheiten`) | **niedrig** — nur `?since=` + Tombstone |
| **2 — ADD updated_at** | `games`, `duty_slots`, `training_sessions` | nur `created_at` → ADD + Backfill | `training_sessions` **6**, `duty_slots` 2, `games` 1 | **mittel** — Migration + Schreibpfad |
| **3 — teuer / später** | `game_attendances` (`videos`/`members` optional) | keine Zeitspalte | — | erst bei gemessenem Bedarf |

- **Tombstones sind Pflicht, nicht Kür:** jede Tier-1/2-Domäne hat harte `DELETE FROM …`-Pfade — ohne Tombstone-Log blieben Geister-Einträge. `training_sessions` hat mit **6** Delete-Stellen den höchsten Verdrahtungsaufwand.
- **Empfohlene Reihenfolge:** Tier 1 (kader/mitfahrten, kein `updated_at`-Aufwand) → Tier 2 (games/trainings/duty-slots) → Tier 3 nur bei Bedarf.

## Capabilities

### Added Capabilities

- `incremental-list-sync`: Pull-basierte Delta-Synchronisation schwerer Listen über einen Client-Cursor (`?since=`), inkl. Lösch-Marker (Tombstones) und Voll-Refetch-Fallback bei zu altem Cursor.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/games?since=` | `TestGamesSince_ReturnsOnlyChanged` | Nur Spiele mit `updated_at > cursor` in `items`; unveränderte fehlen. |
| `GET /api/games?since=` | `TestGamesSince_DeletedReportedAsTombstone` | Ein seit dem Cursor gelöschtes Spiel erscheint in `deleted_ids`, nicht in `items`. |
| `GET /api/games?since=` | `TestGamesSince_StaleCursorFallsBackToFull` | Cursor älter als Tombstone-Frist → Voll-Response + `full:true`. |
| `GET /api/games` (ohne `since`) | `TestGames_NoSinceUnchanged` | Ohne `?since` unverändertes (volles, paginiertes) Verhalten. |

**Garantierte Invariante:** Inkrementelle Synchronisation ändert **nie** die Sichtbarkeits-/Autorisierungsregeln. `?since=` liefert genau die Teilmenge der ohnehin sichtbaren Datensätze, die sich seit dem Cursor geändert haben (inkl. Löschungen als Tombstone); der lokal rekonstruierte Zustand ist identisch mit einem Voll-Refetch. Ein zu alter Cursor führt niemals zu still fehlenden Daten, sondern zum Voll-Refetch-Fallback.

## Mess-Anforderungen

Belegt über `payload-measurement-harness` (Voraussetzung), erweitert um einen Delta-Szenario-Lauf.

| Kennzahl | Werkzeug | Erwartung |
|---|---|---|
| Payload `GET /api/games?since=` nach Änderung EINES Spiels | `make measure` (Delta-Szenario) | Größe ≈ ein Datensatz, nicht die volle Liste (Baseline: volle Liste). |
| Reload-Payload nach Voll-Refetch-Fallback | `make measure` | Fallback liefert weiterhin vollständig (Regressionsschutz). |

## Impact

- **Schema/Migrationen:** neue nächste-freie-Nummer(n) für `updated_at` auf `games`/`duty_slots`/`training_sessions` (+ Setzen im Schreibpfad) und `sync_tombstones`-Log. **DB-Backup vor Migration.**
- **Backend:** `internal/games`, `internal/duties`, `internal/trainings`, `internal/kader` — Cursor-Filter, Tombstone-Auflösung, Fallback-Signal.
- **Frontend:** betroffene Seiten pflegen Cursor + kombinieren Bestand; `full:true` → neu aufbauen.
- **Abhängigkeit/Reihenfolge:** setzt `list-endpoint-pagination` (Cursor koexistiert mit `limit`/`offset`) und idealerweise `scoped-live-updates` voraus (Event als Trigger). Der Chat-Teil ist bereits unter `incremental-sync` erledigt.
- **Risiko:** Delete-Tracking ist der klassische Stolperstein inkrementeller Sync. Tombstone-Frist + Voll-Refetch-Fallback sind die Absicherung; ohne sie drohen „Geister-Einträge" beim Client. Bewusst konservativ: im Zweifel Fallback auf Voll-Refetch.
