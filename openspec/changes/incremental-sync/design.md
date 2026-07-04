# Design — incremental-sync

## Pull statt Push — warum das die Buffer-Drop-Falle umgeht

`scoped-live-updates` verwarf Push-Delta, weil der SSE-Channel bei Bursts verwirft (Buffer 1, `hub.go:36`) → ein verlorenes Delta = dauerhaft veralteter Client. Pull dreht die Verantwortung um:

```
   Push-Delta (verworfen):  Server sagt "Zeile 42 geändert" → geht verloren → Client stale, merkt es nie
   Pull-Cursor (dieser CH): Server sagt "irgendwas geändert" (Trigger) → Client fragt "seit X?" → Lücke wird nachgeholt
```

Das SSE-Event wird zum bloßen **Trigger** („jetzt nachfragen"), nicht zum Datenträger. Ein verlorenes Event verzögert höchstens; Korrektheit hängt allein am Cursor + Tombstones.

## Cursor-Semantik

```
GET /api/games?since=<cursor>[&limit=&offset=]
→ { items: [ …updated_at > cursor… ], deleted_ids: [ … ], cursor: <new>, full: false }

GET /api/games?since=<zu-alt>
→ { items: [ …volle Seite… ], deleted_ids: [], cursor: <new>, full: true }   ← Fallback
```

- `cursor` = server-vergebener Wert (max. `updated_at` bzw. eine monotone Sequenz). Der Client sendet ihn beim nächsten Aufruf zurück.
- `full: true` signalisiert dem Client „verwirf lokalen Bestand, baue neu auf" — greift, wenn `cursor` älter als die Tombstone-Aufbewahrung ist.
- Chat nutzt **id** als Cursor (append-only, monoton) → **kein** `updated_at` nötig; `?after=`/`?before=`.

## Das harte Problem: Löschungen

Eine gelöschte Zeile hat kein `updated_at > cursor` mehr — sie ist einfach weg. Ohne Gegenmaßnahme behält der Client „Geister-Einträge". Optionen:

| Ansatz | Vorteil | Nachteil |
|---|---|---|
| **`sync_tombstones(entity, entity_id, deleted_at)`** (append-only Log) | eine zentrale Stelle, keine Fach-Tabellen anfassen | zusätzliche Schreib-Stelle je Delete; Pruning nötig |
| `deleted_at`-Spalte + Soft-Delete je Tabelle | fachlich oft ohnehin nützlich | jede Query muss `WHERE deleted_at IS NULL`; invasiver |

Empfehlung: **Tombstone-Log** als generischer, wenig invasiver Weg; Soft-Delete nur dort, wo es fachlich ohnehin sinnvoll ist. Delete-Handler schreiben zusätzlich einen Tombstone.

**Aufbewahrung + Fallback:** Tombstones werden nach Frist `T` (z. B. 30 Tage) beschnitten. Ein Client mit Cursor älter als `T` kann Löschungen verpasst haben → Server antwortet `full: true`, Client baut neu auf. So gibt es **nie** still fehlende/übrige Daten, nur gelegentlich einen teureren Voll-Refetch.

## `updated_at` nachrüsten

Fehlt heute auf `games`, `duty_slots`, `training_sessions`. Migration:
- Spalte `updated_at` (Default = `created_at`/jetzt beim Backfill).
- Schreibpfad setzt `updated_at` bei jedem `UPDATE`/`INSERT` (App-seitig explizit — SQLite-Trigger sind eine Alternative, aber App-seitig bleibt es sichtbar und testbar).
- Nächste freie Migrationsnummer (`ls internal/db/migrations/`), nie ≤ aktueller DB-Version.

## Phasenschnitt (Chat zuerst, weil billig)

1. **Chat (ohne Schema):** `?after=`/`?before=` (id-Cursor), `ChatPage` hängt Delta an. Sofort-Win, keine Migration.
2. **`updated_at`-Migrationen** für games/duty_slots/training_sessions + Schreibpfad.
3. **Tombstone-Log** + Delete-Handler + Pruning.
4. **`?since=`** auf games/duty_slots/training_sessions/kader + Frontend-Cursor-Handling.

## Zusammenspiel mit anderen Changes

```
   scoped-live-updates:  weniger Clients werden getriggert
   incremental-sync:     jeder Trigger holt nur das Delta
   list-endpoint-pagination:  Cursor koexistiert mit limit/offset (erste Vollseite paginiert, dann Deltas)
   ─────────────────────────────────────────────────────────────
   Netto: O(alle × ganze Liste)  →  O(betroffene × geänderte Zeilen)
```

## Invariante

`?since=` liefert exakt die seit dem Cursor geänderten Datensätze der ohnehin sichtbaren Menge, plus Tombstones für Löschungen. Der lokal rekonstruierte Zustand ist identisch mit einem Voll-Refetch. Sichtbarkeit/Autorisierung bleiben unverändert; ein zu alter Cursor führt zum Voll-Refetch, nie zu stillem Datenverlust.
