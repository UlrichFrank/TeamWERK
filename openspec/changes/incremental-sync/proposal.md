## Why

Heute löst ein Live-Update in vielen Ansichten einen **vollständigen** Listen-Refetch aus (siehe `scoped-live-updates`, das den *Empfängerkreis* reduziert). Selbst nach Scoping überträgt jeder Reload die **komplette** Liste, obwohl sich meist nur ein Datensatz geändert hat: Bei 1 von 100 geänderten Spielen wandern 100 Datensätze zum Client.

Zusätzlich verschenkt der Chat ein bereits geliefertes Delta: Das SSE-Event trägt die Nachrichten-ID (`chat:new-message:123`), aber `ChatPage.tsx` ruft `loadMessages(convId)` — einen **Voll-Reload der ganzen Konversation** (bis 100 Nachrichten) — statt die eine bekannte Nachricht anzuhängen.

Dieser Change führt **pull-basierte inkrementelle Synchronisation** ein: der Client hält einen Cursor und fragt „was hat sich seit X geändert?". Kombiniert mit `scoped-live-updates` ergibt das das eigentliche Ziel: `O(betroffene Clients × geänderte Zeilen)` statt `O(alle Clients × ganze Liste)`.

**Warum pull-basiert (und nicht Push-Delta):** In `scoped-live-updates` wurden Push-Delta-Events bewusst verworfen, weil der SSE-Channel bei Bursts verwirft (Buffer 1) — ein verlorenes Delta hieße dauerhaft veralteter Client. Beim **Pull**-Modell besitzt der Client den Cursor: ein verworfenes Event ist harmlos, der nächste `?since=`-Aufruf holt die Lücke nach. Kein Resync-Protokoll nötig; das SSE-Event dient nur noch als „jetzt nachfragen"-Trigger.

## What Changes

- **Cursor-Parameter auf schweren Listen:** `GET /api/games`, `/api/duty-slots`, `/api/training-sessions`, `/api/kader` akzeptieren `?since=<cursor>` und liefern nur Datensätze mit `updated_at > cursor` (plus Lösch-Marker, siehe unten). Ohne `?since=` unverändertes Verhalten (voller, paginierter Abruf). Der Client kombiniert den Bestand lokal.
- **Chat inkrementell:** `GET /api/chat/conversations/{id}/messages?after=<msgId>` liefert nur neuere Nachrichten (append-only, id-basiert — **kein** `updated_at` nötig); `?before=<msgId>` liefert ältere für Verlaufs-Scrollen. `ChatPage` hängt bei `chat:new-message:<id>` gezielt an, statt die Konversation neu zu laden.
- **`updated_at` nachrüsten (nur wo nötig):** Migrationen ergänzen `updated_at` (App-seitiges Setzen bei INSERT/UPDATE) auf `games`, `duty_slots`, `training_sessions`, `videos`, `members` — dort existiert heute nur `created_at` (sauberer Backfill-Default). Auf `kader`, `mitfahrgelegenheiten`, `mitfahrt_paarungen` ist `updated_at` **bereits vorhanden** → dort ohne Spalten-Migration nutzbar. `messages` braucht kein `updated_at` (append-only → id-Cursor). `game_attendances` hat weder `created_at` noch `updated_at` und ist daher **kein** Billig-Kandidat.
- **Lösch-Marker (Tombstones):** Damit `?since=` auch Löschungen meldet (eine gelöschte Zeile taucht in „changed since" sonst nicht auf), liefert die Response neben `items` eine `deleted_ids`-Liste. Umsetzung über ein schlankes, append-only `sync_tombstones(entity, entity_id, deleted_at)`-Log (oder `deleted_at`-Spalte + Soft-Delete, wo fachlich vertretbar). Tombstones werden nach einer Aufbewahrungsfrist beschnitten; Clients ohne gültigen Cursor (älter als die Frist) fallen auf Voll-Refetch zurück.

## Geltungsbereich & Domänen-Priorität (schema-verifiziert)

Reihenfolge nach Umsetzungskosten, abgeleitet aus dem tatsächlichen Schema (`updated_at`-Bereitschaft) und den Delete-Pfaden:

| Tier | Domäne(n) | `updated_at` | Delete-Pfade (Tombstone) | Kosten |
|---|---|---|---|---|
| **0 — Chat** | `messages` | n/a (id-Cursor, append-only) | 0 harte Deletes gefunden (Soft-Delete-Flag) | **niedrig** — keine Migration |
| **1 — gratis** | `kader`, `mitfahrgelegenheiten`, `mitfahrt_paarungen` | **schon vorhanden** | je 1 (`kader`, `mitfahrgelegenheiten`) | **niedrig** — nur `?since=` + Tombstone |
| **2 — ADD updated_at** | `games`, `duty_slots`, `training_sessions`, `videos`, `members` | nur `created_at` → ADD + Backfill | `training_sessions` **6**, `duty_slots`/`videos` je 2, `games` 1 | **mittel** — Migration + Schreibpfad |
| **3 — teuer / später** | `game_attendances` | **keine Zeitspalte** | — | ADD created_at+updated_at nötig; RSVP-Status ändert sich häufig |

- **Tombstones sind Pflicht, nicht Kür:** jede Tier-1/2-Domäne hat harte `DELETE FROM …`-Pfade (verifiziert) — ohne Tombstone-Log blieben Geister-Einträge. `training_sessions` hat mit **6** Delete-Stellen den höchsten Verdrahtungsaufwand.
- **Empfohlene Reihenfolge:** Chat (Phase 1, keine Migration) → Tier 1 (kader/mitfahrten, kein `updated_at`-Aufwand) → Tier 2 (games/trainings/duty-slots) → Tier 3 nur bei Bedarf.
- **`videos`/`members`** sind bewusst optional: Video-Listen ändern sich selten (eher `lazy-rendering`), Member-CRUD ist niederfrequent (eher `scoped-live-updates` + Windowing). Beide nur aufnehmen, wenn Messung Bedarf zeigt.

## Capabilities

### Added Capabilities

- `incremental-sync`: Pull-basierte Delta-Synchronisation schwerer Listen über einen Client-Cursor (`?since=`/`?after=`/`?before=`), inkl. Lösch-Marker und Voll-Refetch-Fallback bei zu altem Cursor.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/games?since=` | `TestGamesSince_ReturnsOnlyChanged` | Nur Spiele mit `updated_at > cursor` in `items`; unveränderte fehlen. |
| `GET /api/games?since=` | `TestGamesSince_DeletedReportedAsTombstone` | Ein seit dem Cursor gelöschtes Spiel erscheint in `deleted_ids`, nicht in `items`. |
| `GET /api/games?since=` | `TestGamesSince_StaleCursorFallsBackToFull` | Cursor älter als Tombstone-Frist → Voll-Response + Signal „vollständig neu aufbauen". |
| `GET /api/games` (ohne `since`) | `TestGames_NoSinceUnchanged` | Ohne `?since` unverändertes (volles, paginiertes) Verhalten. |
| `GET /api/chat/.../messages?after=` | `TestMessagesAfter_ReturnsOnlyNewer` | Nur Nachrichten mit `id > after`; leere Liste wenn nichts Neues. |
| `GET /api/chat/.../messages?before=` | `TestMessagesBefore_ReturnsOlderPage` | Liefert die Seite älterer Nachrichten vor `before` (Verlaufs-Scroll). |

**Garantierte Invariante:** Inkrementelle Synchronisation ändert **nie** die Sichtbarkeits-/Autorisierungsregeln. `?since=` liefert genau die Teilmenge der ohnehin sichtbaren Datensätze, die sich seit dem Cursor geändert haben (inkl. Löschungen als Tombstone); der lokal rekonstruierte Zustand ist identisch mit einem Voll-Refetch. Ein zu alter Cursor führt niemals zu still fehlenden Daten, sondern zum Voll-Refetch-Fallback.

## Mess-Anforderungen

Belegt über `payload-measurement-harness` (Voraussetzung), erweitert um einen Delta-Szenario-Lauf.

| Kennzahl | Werkzeug | Erwartung |
|---|---|---|
| Payload `GET /api/games?since=` nach Änderung EINES Spiels | `make measure` (Delta-Szenario) | Größe ≈ ein Datensatz, nicht die volle Liste (Baseline: volle Liste). |
| Payload Chat-Reload nach einer neuen Nachricht | `make measure` | `?after=` liefert 1 Nachricht statt bis zu 100. |
| Reload-Payload nach `settings`/Voll-Refetch-Fallback | `make measure` | Fallback liefert weiterhin vollständig (Regressionsschutz). |

## Impact

- **Schema/Migrationen:** neue nächste-freie-Nummer(n) für `updated_at` auf `games`/`duty_slots`/`training_sessions` (+ Setzen im Schreibpfad) und `sync_tombstones`-Log (bzw. `deleted_at`). **DB-Backup vor Migration.**
- **Backend:** `internal/games`, `internal/duties`, `internal/trainings`, `internal/kader`, `internal/chat` — Cursor-Filter, Tombstone-Auflösung, Fallback-Signal.
- **Frontend:** betroffene Seiten pflegen Cursor + kombinieren Bestand; `ChatPage` hängt Delta an; Verlaufs-Scroll per `?before=`.
- **Abhängigkeit/Reihenfolge:** setzt `list-endpoint-pagination` (Cursor koexistiert mit `limit`/`offset`) und idealerweise `scoped-live-updates` voraus (Event als Trigger). Chat-Teil (id-Cursor, **ohne** Schema) kann als **Phase 1 vorgezogen** werden; die `updated_at`/Tombstone-Teile folgen.
- **Risiko:** Delete-Tracking ist der klassische Stolperstein inkrementeller Sync. Tombstone-Frist + Voll-Refetch-Fallback sind die Absicherung; ohne sie drohen „Geister-Einträge" beim Client. Bewusst konservativ: im Zweifel Fallback auf Voll-Refetch.
