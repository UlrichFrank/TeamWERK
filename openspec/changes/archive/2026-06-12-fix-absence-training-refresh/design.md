## Context

Beim Speichern einer Abwesenheit über `POST /api/absences` setzt das Backend korrekt:
- `training_responses` → `status = 'declined'`, `absence_id = <id>` für alle Trainings im Zeitraum
- Broadcasts: `"absences"`, `"trainings"`, `"games"`

Die KalenderPage:
- Reagiert in `useLiveUpdates` nur auf `"games"` und `"absences"`, nicht auf `"trainings"`
- Ruft in `doSaveAbsence()` nur `loadAbsences()` auf, nicht `loadTrainings()`
- Preview-Query zeigt nur bestätigte Training-Responses (`status = 'confirmed'`), nicht alle betroffenen Sessions

## Goals / Non-Goals

**Goals:**
- Training-Kacheln werden nach Absence-Save sofort neu geladen (lokaler Client)
- Andere geöffnete KalenderPage-Clients werden via SSE aktualisiert
- Preview zeigt alle Training-Sessions im Abwesenheitszeitraum, bei denen der Member Kader-Mitglied ist

**Non-Goals:**
- Kein Umbau des SSE-Mechanismus
- Keine Änderung an der Auto-Decline-Logik im Backend (korrekt)
- Kein Redesign des Kalender-Layouts

## Decisions

### 1. `loadTrainings()` nach Absence-Save aufrufen

In `doSaveAbsence()` wird nach erfolgreichem POST direkt `loadTrainings()` aufgerufen — parallel zu `loadAbsences()`.

*Alternative: nur auf SSE-Event verlassen* → abgelehnt, weil SSE leicht verloren gehen kann und der lokale Client die Änderung sofort sehen soll.

### 2. `useLiveUpdates` um `"trainings"` erweitern

```tsx
useLiveUpdates((event) => {
  if (event === 'games') loadGames()
  if (event === 'absences') loadAbsences()
  if (event === 'trainings') loadTrainings()   // neu
})
```

Konsistentes Pattern mit den anderen Events.

### 3. Preview-Query im Backend erweitern

Der `GET /api/absences/preview`-Endpoint gibt zwei Gruppen zurück:
1. **Bestätigte Trainings** (wie bisher): Sessions mit `tr.status = 'confirmed'` — werden als erstes aufgelistet mit Label "Bestätigt"
2. **Unbeantwortete Trainings**: Sessions ohne Antwort, aber im Kader des Members — mit Label "Offen"

Damit spiegelt der Preview korrekt wider, was die Auto-Decline-Logik tatsächlich tut.

Einfachste Umsetzung: separater Query-Block im `Preview`-Handler, der analog zur Auto-Decline-Query in `Create` formuliert ist:

```sql
SELECT ts.id, COALESCE(ts.title, ''), ts.date
FROM training_sessions ts
JOIN kader_members km ON km.member_id = ?
JOIN kader k ON k.id = km.kader_id AND k.team_id = ts.team_id
WHERE ts.date BETWEEN ? AND ?
AND NOT EXISTS (
  SELECT 1 FROM training_responses tr
  WHERE tr.training_id = ts.id AND tr.member_id = ?
)
```

Das `previewEvent`-Struct bekommt ein neues Feld `status string` (`"confirmed"` / `"pending"`), das das Frontend nutzt, um eine Unterscheidung anzuzeigen.

## Risks / Trade-offs

- [Doppelter `loadTrainings()`-Aufruf] Wenn SSE schnell ankommt, lädt die KalenderPage zweimal. Kein Bug — aber zwei identische API-Calls in kurzer Folge. Akzeptabel bei den Datenmengen.
- [Preview-Vollständigkeit] Der neue Pending-Block zeigt auch Sessions, die der Member möglicherweise explizit nicht besucht (z.B. anderes Team). Das ist gewollt — die Decline-Logik schreibt dasselbe.
