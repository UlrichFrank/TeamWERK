## Context

`duty-assignee-visibility` hat das Datenschutz-Modell implementiert und einen `AssigneeChip` in `DutySlotList` gebaut — aber mit Inline-Daten im Board-Response (Phones + Address direkt im Query aggregiert). Diese Change refaktoriert das auf Lazy-Fetch und extrahiert eine geteilte `PersonChip`-Komponente.

Vorhandene Infrastruktur:
- `user_visibility` (phones_visible, address_visible, photo_visible) ✓
- `user_phones` (label, number per user_id) ✓
- `users.photo_path`, `users.street/zip/city` ✓
- `AssigneeChip` in `DutySlotList` mit Hover/Tap-Logik ✓ — wird zur Vorlage

## Goals / Non-Goals

**Goals:**
- Eine Komponente (`PersonChip`) für alle Personen-Darstellungen in der App
- Lazy-Fetch der Kontaktdaten on first hover/tap — kein Overhead beim Seitenaufbau
- Session-scoped Cache verhindert wiederholte Requests für dieselbe Person
- Cache-Invalidierung bei Logout (kein Datenleck bei geteiltem Gerät)
- Graceful degradation: Persons ohne `user_id` → Plain-Text, kein Tooltip

**Non-Goals:**
- Neue Privacy-Einstellungen oder Datenschutz-Logik — das bestehende Modell wird nur angewendet
- Echtzeit-Aktualisierung der Kontaktdaten im Tooltip
- Neue Orte erfinden, wo Personen erscheinen — nur bestehende Stellen umstellen

## Decisions

### Entscheidung 1: Lazy-Fetch statt Inline

**Gewählt:** Neuer Endpoint `GET /api/users/:id/contact`; `PersonChip` fetcht on first hover.

**Begründung:** Duty-Board ist der einzige Ort mit mehreren Personen gleichzeitig (10–30 Slots). Inline war dort vertretbar. Aber für die Ausweitung auf Mitglieder-Liste (100+ Einträge), Kader (Trainer) und Dashboard würde Inline bedeuten, dass jede dieser Seiten ihre API erweitern muss. Lazy zieht eine gemeinsame Linie und skaliert besser.

**Trade-off:** Latenz beim ersten Hover (~100ms auf LAN). Akzeptabel, da Tooltips kein Zero-Latency-UI sind.

### Entscheidung 2: PersonContactContext als session-scoped Cache

**Gewählt:** React Context mit `Map<userId, PersonContact | 'loading' | 'error'>`.

**Begründung — Kein Datenleck:**
- Der Cache enthält ausschließlich das, was der Server nach Privacy-Filterung zurückgibt
- Server-seitige Filterung ist die Quelle der Wahrheit — der Client cacht nur das Ergebnis, nicht mehr
- Bei Logout: `AuthContext.logout()` triggert `clearCache()` im PersonContactContext → kein residuales Daten-Leck bei Shared Devices
- Wenn User A und User B denselben Browser teilen: Cache wird bei Login/Logout-Wechsel geleert

**Alternative:** Pro-Chip `useState` — einfacher, aber dieselbe Person triggert N Requests wenn sie auf N Stellen sichtbar ist (z.B. Trainer als Assignee + Kader-Eintrag auf derselben Seite).

**Alternative:** `localStorage`-Persistenz — abgelehnt, da Kontaktdaten nicht persistent im Browser gespeichert werden sollen.

### Entscheidung 3: Wo der neue Endpoint lebt

**Gewählt:** Handler-Methode in `internal/members/handler.go` (oder eigenem Package `internal/users/`), registriert als `GET /api/users/:id/contact`.

**Begründung:** Die SQL-Logik (`CASE WHEN uv.photo_visible ...`) ist identisch mit dem, was `duty-assignee-visibility` in `GetBoard` implementiert hat. Keine Code-Duplikation — stattdessen als eigene Funktion extrahieren.

**Response-Shape:**
```json
{
  "name": "Max Mustermann",
  "photo_url": "/api/uploads/...",
  "phones": [{ "label": "Mobil", "number": "+49..." }],
  "address": "Musterstr. 1, 70123 Stuttgart"
}
```
`photo_url`, `phones`, `address` sind optional — fehlen wenn nicht freigegeben.

### Entscheidung 4: PersonChip Props-Interface

```typescript
interface PersonChipProps {
  userId: number       // user_id für den Lazy-Fetch
  name: string         // sofort anzeigen, kein Fetch nötig
  photoUrl?: string    // optional: Avatar in Chip (kommt aus Board/Kader-Response)
}
```

Ohne `userId` kein Tooltip. Für Stellen die `userId` nicht haben (theoretisch): weiterhin Plain-Text. In der Praxis haben alle aktuellen Rollout-Orte entweder direkte `user_id` oder können sie mit minimalem Backend-Aufwand bekommen.

### Entscheidung 5: Kader-Trainers userId-Quelle

`kader_trainers.member_id → members.user_id` per LEFT JOIN. `user_id` ist NULL wenn der Trainer keinen Account hat (unwahrscheinlich aber möglich). `PersonChip` rendert dann `name` ohne interaktiven Tooltip.

### Entscheidung 6: Board-Response vereinfachen

`boardSlot.assignees[]` bisher: `[{ name, photo_url?, phones?, address? }]`
Nach Refactoring: `[{ user_id, name, photo_url? }]`

`photo_url` bleibt inline (nötig für Avatar im Chip selbst, kein Hover nötig). Phones/Address werden lazy gefetcht.

## Risks / Trade-offs

- **Cache-Staleness:** Wenn jemand seine Visibility-Einstellungen ändert, zeigt ein bereits gecachter Tooltip alte Daten. Akzeptabel — TTL ist implizit die Session-Dauer; bei Reload ist der Cache leer
- **N+1 auf Mitglieder-Liste:** 100 Members → bei allen hovern 100 Requests. Der Cache macht das idempotent (jede Person nur 1x). Kein Batching nötig für diesen Use-Case
- **Kein loading-Indikator wenn bereits gecacht:** PersonChip zeigt gecachte Daten sofort — kein Flackern. Beim ersten Fetch: Tooltip öffnet sich mit Spinner, Daten erscheinen
