# Design — list-endpoint-pagination

## Einheitlicher Paginierungs-Vertrag

Wiederverwendung der bestehenden Konvention von `/api/members`/`/api/users`:

```
Query:    ?limit=<n>&offset=<m>[&search=…][&season_id=…][&from=…&to=…]
Response: { "items": [...], "total": <int> }
```

- `total` kommt aus einem separaten `COUNT(*)` mit denselben `WHERE`-Bedingungen (ohne `LIMIT`).
- Default-Limit pro Route, damit „ohne Parameter" nicht mehr unbeschränkt ist:

| Route | Default-Limit | Sortierung |
|---|---|---|
| `/api/kader` | 50 | wie bisher (Saison, Name) |
| `/api/duty-slots` | 100 | `event_date DESC` |
| `/api/games` | 50 | `date`, `time` |
| `/api/games/{id}/participants` | 200 | Roster-Reihenfolge |
| `/api/training-sessions` | 100 | `date`, `time` |

**Kompatibilitäts-Falle:** Der Wechsel von „nacktem Array" auf `{items,total}` ist ein Breaking Change für jeden Aufrufer. Alle betroffenen Frontend-Seiten müssen im selben Change umgestellt werden (siehe `tasks.md`). Es gibt keine parallele Legacy-Antwort.

## duty-board: Namen behalten, schwere Felder aufschieben

Die Anforderung `duty-assignee-display` („Namen unter jedem Slot sichtbar") bleibt **erfüllt** — Namen sind billige Strings. Aufgeschoben werden nur die schweren/optionalen Felder:

```
Board-Assignee (neu):  { user_id, name }              // inline, wie bisher sichtbar
On-Demand (bei Klick):  { photo_url, phones, emails }  // pro Slot/Assignee nachgeladen
```

- Avatar + Kontakt-Tooltip laden erst, wenn der Nutzer einen Slot/Assignee öffnet.
- Das reduziert die Board-Payload dort am stärksten, wo sie am größten ist (viele Slots × viele Assignees × `photo_url`), ohne die sichtbare Namensliste zu entfernen.
- Falls es noch keinen geeigneten Detail-Endpoint gibt, wird ein schlanker `GET`-Pfad für Assignee-Kontaktdaten eines Slots ergänzt (mit denselben Sichtbarkeits-/`*_visible`-Regeln wie heute im Tooltip).

## chat messages: Body-Preview

```
List-Item:  { id, author, created_at, preview: body[:280], truncated: len(body) > 280, deleted }
Detail:     voller body (nur wenn nicht deleted)
```

- Gelöschte Nachrichten liefern `deleted: true` und **keinen** Body/Preview.
- Reaktionen werden wie bisher geladen; nur der Body wird gekürzt.
- Der 280-Zeichen-Schnitt ist ein reiner Serialisierungs-Trim, kein DB-Change.

## Sichtbarkeit bleibt invariant

Paginierung/Trim ist eine reine Transport-Optimierung. Jede Route behält ihre `WHERE`-Sichtbarkeitsfilter und Auth-Tier unverändert; `COUNT(*)` und `items` verwenden **dieselben** Bedingungen. Kein Element wird durch die Umstellung neu sichtbar oder unsichtbar — nur die pro Request übertragene Teilmenge ändert sich.
