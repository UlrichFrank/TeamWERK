## Why

Mehrere GET-Listen-Endpoints liefern unbeschränkte Ergebnismengen bzw. schwere Pro-Element-Felder an den Client — unabhängig davon, wie viel die aufrufende Seite tatsächlich braucht. Die etablierte Paginierungs-Konvention (`?search=&limit=&offset=` → `{ items, total }`, siehe `docs/agent/04-api-db.md`) ist bisher nur auf `/api/members`, `/api/users` und `/api/videos` umgesetzt. Andere schwere Listen fehlen:

| Route | Handler | Problem | grobe Größe |
|---|---|---|---|
| `GET /api/kader` | `internal/kader/handler.go` | keine Paginierung; je Kader volle Member-/Trainer-/Extended-Arrays | 50–100 KB |
| `GET /api/duty-slots` | `internal/duties/handler.go:306` | keine Paginierung/Filter; alle historischen + künftigen Slots | 50–100 KB |
| `GET /api/games` | `internal/games/handler.go:454` | ohne `season_id` unbeschränkt (aktive Saison + alle künftigen) | 30–50 KB |
| `GET /api/games/{id}/participants` | `internal/games/handler.go:2268` | volle Roster ohne Limit; Mehrfach-Team-Spiele 300+ Einträge | 20–50 KB |
| `GET /api/training-sessions` | `internal/trainings/handler.go:910` | 3-Monats-Fenster ohne `limit`/`offset` | 20–40 KB |
| `GET /api/duty-board` | `internal/duties/handler.go:412` | je Slot volle Assignee-Records inkl. `photo_url`/Kontakt | 100–200 KB |
| `GET /api/chat/.../messages` | `internal/chat/handler.go:426` | `LIMIT 100`, aber volle Bodies aller (auch gelöschter) Nachrichten | 50–200 KB |

Diese Änderungen brechen den Response-Vertrag (Array → `{items,total}` bzw. Feld-Verschiebung) und erfordern koordinierte Backend-+Frontend-Anpassung — deshalb bewusst getrennt von den additiven Quick-Wins (`efficient-data-loading-quickwins`).

## What Changes

- **Einheitliche Paginierung** (`?limit=&offset=` → `{ items, total }`, optional `?search=`/`?season_id=`/Datumsfenster) für `GET /api/kader`, `GET /api/duty-slots`, `GET /api/games`, `GET /api/games/{id}/participants`, `GET /api/training-sessions`. Sinnvolle Default-Limits pro Route; das Frontend nutzt „Mehr laden" statt clientseitigem `filter()`.
- **`GET /api/duty-board` — schwere Pro-Assignee-Felder aufschieben:** die Board-Antwort behält die **Namen** (Anforderung `duty-assignee-display` bleibt erfüllt), liefert aber `photo_url` und Kontakt-Tooltip-Daten NICHT mehr inline; diese werden bei Bedarf pro Slot/Assignee nachgeladen. Optional Datumsfenster-Filter (`?from=&to=`).
- **`GET /api/chat/.../messages` — Body-Preview:** die Nachrichtenliste liefert einen gekürzten Body-Preview (erste ~280 Zeichen) plus `truncated: bool`; der Volltext einer Nachricht wird bei Bedarf über den Einzel-Pfad geladen. Gelöschte Nachrichten liefern keinen Body.
- **Serverseitige Filter statt Client-`filter()`:** wo das Frontend heute volle Listen holt und filtert (z. B. `training-sessions` → `series_id === null`), bekommt die Route einen Query-Parameter (`?exclude_series=1`).

## Capabilities

### Added Capabilities

- `list-endpoint-pagination`: Standard-Paginierungs-/Feld-Trim-Vertrag (`{items,total}`, `limit`/`offset`, Default-Limits, serverseitige Filter) über die genannten schweren Listen-Endpoints.

### Modified Capabilities

- `duty-assignee-display`: Board-Liste behält Assignee-Namen, lädt Avatar/Kontaktdaten aber on-demand statt inline.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/kader` | `TestListKader_PaginationLimitOffset` | `?limit=2&offset=0` → genau 2 `items`, `total` = Gesamtzahl. |
| `GET /api/kader` | `TestListKader_DefaultLimitApplied` | Ohne `limit` wird das Default-Limit angewandt, `total` bleibt vollständig. |
| `GET /api/duty-slots` | `TestListDutySlots_Paginated` | Ergebnis ist begrenzt + `total`; ältester/neuester Slot über `offset` erreichbar. |
| `GET /api/games` | `TestListGames_PaginatedAndSeasonFilter` | `?season_id=&limit=` begrenzt; ohne `season_id` gilt Default-Limit statt unbeschränkt. |
| `GET /api/games/{id}/participants` | `TestParticipants_Paginated` | begrenzte `items` + `total`; Autorisierung unverändert. |
| `GET /api/training-sessions` | `TestListSessions_ExcludeSeriesFilter` | `?exclude_series=1` liefert nur `series_id IS NULL`; ohne Param unverändert. |
| `GET /api/duty-board` | `TestDutyBoard_NamesWithoutHeavyFields` | Assignees tragen `name`, aber KEIN `photo_url`/Kontaktfeld inline. |
| `GET /api/chat/.../messages` | `TestListMessages_BodyPreviewTruncated` | Body > Grenze → gekürzter Preview + `truncated=true`; gelöschte Nachricht ohne Body. |

**Garantierte Invariante:** Paginierung/Feld-Trim ändert **nur** Umfang/Form der Nutzlast, **nie** die Sichtbarkeits-/Autorisierungsregeln einer Route. Ein Element, das ein Nutzer unpaginiert sehen durfte, bleibt über `limit`/`offset` erreichbar; keines wird neu sichtbar.

## Mess-Anforderungen

Verglichen wird gegen `metrics/payload-baseline.md` aus `payload-measurement-harness` (Voraussetzung). Das Seeding dort ist auf realistische Größen ausgelegt (200 Members, 100 Games, 500 Slots …), damit die Payload-Deltas sichtbar werden.

| Kennzahl | Werkzeug | Erwartung nach diesem Change |
|---|---|---|
| Payload `GET /api/kader` (Default-Limit) | `make measure` | deutlich kleiner als unpaginiert (nur `limit` Einträge + `total`). |
| Payload `GET /api/duty-board` | `make measure` | kleiner (kein `photo_url`/Kontakt inline pro Assignee). |
| Payload `GET /api/duty-slots`, `/api/games`, `/api/training-sessions`, `/api/games/{id}/participants` | `make measure` | jeweils auf Default-Limit begrenzt statt Voll-Liste. |
| Payload `GET /api/chat/.../messages` | `make measure` | kleiner (Body-Preview statt Volltext). |

**Baseline-Regel:** Für jede oben genannte Route die Vorher-Bytes aus der Baseline notieren; nach Umsetzung `make measure` erneut laufen lassen und die Zeilen in `metrics/payload-baseline.md` mit den Nachher-Bytes aktualisieren. Der Payload-Rückgang je Route ist der Wirkungsnachweis.

## Impact

- **Backend:** `internal/kader/handler.go`, `internal/duties/handler.go` (ListSlots + Board), `internal/games/handler.go` (ListGames + GetParticipants), `internal/trainings/handler.go`, `internal/chat/handler.go` — Query um `LIMIT`/`OFFSET`/`COUNT(*)` + Filter erweitern, Response auf `{items,total}` umstellen; Board-/Message-Serialisierung trimmen. Ein neuer On-Demand-Pfad für Assignee-Kontaktdaten (falls nicht schon vorhanden).
- **Frontend:** `AdminKaderPage`, `DutyPage`/`DutySlotList`, `TerminePage`, `KalenderPage`, Game-Detail/Participants, `AdminTrainingsPage` (Client-`filter()` → Query-Param), Chat-Ansicht — „Mehr laden", `{items,total}`-Handling, Lazy-Load für Assignee-Avatar/Kontakt und Message-Volltext.
- **Kein** Schema-/Migrations-Change zwingend nötig (nur Query-Änderungen); falls für Message-Preview eine berechnete Spalte sinnvoll ist, separat prüfen.
- **Abhängigkeit/Reihenfolge:** unabhängig von `efficient-data-loading-quickwins` umsetzbar; sinnvollerweise **nach** den Quick-Wins, damit der Client-Cache die neuen `{items,total}`-Shapes berücksichtigt.
