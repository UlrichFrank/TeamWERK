## Context

`/termine` (`web/src/pages/TerminePage.tsx`) ist die einheitliche Termin-Liste für Spieler/Eltern/Trainer und bereits in `termine-unified-view` spezifiziert. Ihre Filter (`filterTeamId`, `filterTypes`, `showPast`) leben heute ausschließlich in `useState` und sind weder über URL teilbar noch über Browser-Back navigierbar.

Push-Notifications werden vom Backend über `notify.Send(db, cfg, userIDs, category, title, body, url)` (`internal/notify/notify.go`) verschickt — das `url`-Feld landet als `data.url` in der Service-Worker-Notification und wird beim Klick als Ziel verwendet. Aktuell verlinken:
- `internal/games/handler.go` (3 Stellen) → `/kalender`
- `internal/trainings/handler.go` (3 Stellen) → `/training`
- `internal/duties/handler.go` (2 Stellen) → `/dienste` (bleibt unverändert)

Das `EventInfoModal` (`event-info-modal`) zeigt heute schreibgeschützt Game-/Training-Daten und hat absichtlich keine Aktion-Buttons außer Schließen.

## Goals / Non-Goals

### Goals
- Push- und E-Mail-Klick führt direkt zum richtigen Termin in der Liste, ohne dass der Empfänger sucht.
- Filter sind URL-driven und damit linkbar/back-bar.
- Sprung von Kalender-Modal in die Termine-Liste ist ein Klick weit.

### Non-Goals
- Keine neuen API-Endpunkte. Die `/termine`-Daten kommen weiterhin aus `/api/games/my` + `/api/training-sessions`.
- Keine Persistenz der Filter pro User in der DB (URL als „Persistenz" reicht).
- Kein Drittsystem (Linear, Slack, etc.).
- Dienste werden in diesem Change nicht in `/termine` integriert. `push-duties` bleibt unangetastet.

## URL-Schema

```
/termine
/termine?team=<int>
/termine?types=training,heim,auswaerts
/termine?past=1
/termine?focus=game-<int>
/termine?focus=training-<int>
/termine?team=2&types=heim&past=1&focus=game-17
```

**Defaults** (= entspricht `/termine`):
- `team` fehlt → kein Team-Filter
- `types` fehlt → alle drei Typen aktiv (heutige Default-Set)
- `past` fehlt/`0` → nur zukünftige Termine

**Robustheit:** Unbekannte/leere Werte werden ignoriert. Der `focus`-Parameter wird gegen Regex `^(training|game)-\d+$` validiert; alles andere wird ignoriert.

## Frontend-Implementierung

### State ↔ URL-Sync (TerminePage)

`useSearchParams()` von `react-router-dom` v6 ersetzt die rein lokalen `useState`-Filter:

```ts
const [searchParams, setSearchParams] = useSearchParams()

const filterTeamId = parseInt(searchParams.get('team') ?? '') || null
const filterTypes  = parseTypes(searchParams.get('types'))    // Set<string>
const showPast     = searchParams.get('past') === '1'
const focus        = parseFocus(searchParams.get('focus'))    // { kind, id } | null
```

Setter:
```ts
const update = (patch: Partial<Filters>) => {
  const next = new URLSearchParams(searchParams)
  // mutate next based on patch, drop keys whose value matches default
  setSearchParams(next, { replace: true })
}
```

`replace: true` vermeidet einen History-Eintrag pro Filteränderung. Die existierenden RSVP-`useState`s (`rsvpLoading`, `modalReason`, etc.) bleiben lokal — sie haben nichts mit Filtern zu tun.

### Focus-Scroll + Highlight

Jede Termin-Karte erhält eine deterministische `id`: `id={\`termin-${t.kind}-${t.data.id}\`}`. Nach Datenladen läuft ein Effect:

```ts
useEffect(() => {
  if (!focus || loading) return
  const exists = visibleTermine.some(t => t.kind === focus.kind && t.data.id === focus.id)
  if (!exists) {
    // ggf. showPast aktivieren, falls Termin in der Vergangenheit liegt
    // oder Hinweis „nicht verfügbar" anzeigen
    return
  }
  const el = document.getElementById(`termin-${focus.kind}-${focus.id}`)
  el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el?.classList.add('ring-2', 'ring-brand-yellow', 'transition-all')
  const t = setTimeout(() => el?.classList.remove('ring-2', 'ring-brand-yellow'), 2000)
  return () => clearTimeout(t)
}, [focus?.kind, focus?.id, loading, visibleTermine.length])
```

**Edge-Case Vergangenheit:** Wenn der Termin-Datum < heute und `showPast=0`, aktivieren wir `showPast=1` einmalig (in der URL und im Reload-Effect), damit der Termin gerendert wird. Nutzer kann es danach manuell wieder ausschalten.

**Edge-Case Filter blendet Focus aus:** Wenn `?team=2&focus=game-17` und Game 17 gehört zu Team 5, ignorieren wir den Team-Filter für die Sichtbarkeit dieses einen Termins (Highlight + Auto-Expand) **nicht** — wir akzeptieren das Verhalten „wird angezeigt, sofern sichtbar". Begründung: Push-Links setzen typischerweise keinen Team-Filter; ein manueller Filter-Plus-Focus-Konflikt ist user-getrieben und außerhalb des typischen Push-Flows.

→ **Korrektur in der Spec:** Das Scenario „Focus kombiniert mit Filter" oben sagt: einschränkende Filter, die den Focus-Termin verbergen würden, werden ignoriert. Wir entscheiden uns dafür, weil das im Push-Flow am erwartbarsten ist. Implementierung: `visibleTermine` ignoriert `team`/`types`-Filter für genau die Karte mit der `focus`-ID.

### EventInfoModal-Button

Ein zusätzlicher Button im Footer-Bereich des Modals:

```tsx
<button
  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
  onClick={() => {
    onClose()
    navigate(`/termine?focus=${kind}-${id}`)
  }}
>
  In Terminen öffnen
</button>
```

Sichtbar nur für `kind ∈ {game, training}`. `kind` und `id` ergeben sich aus dem bereits durchgereichten Modal-Prop.

## Backend-Implementierung

In `internal/games/handler.go` die 3 `notify.Send(...)`-Aufrufe anpassen:

```go
// Vorher
notify.Send(..., "games", "Neues Spiel", body, "/kalender")
// Nachher
notify.Send(..., "games", "Neues Spiel", body, fmt.Sprintf("/termine?focus=game-%d", gameID))
```

Analog in `internal/trainings/handler.go` für die 3 `notify.Send(...)`-Aufrufe. Beim Delete-Pfad gibt es keinen Termin mehr → fallback auf `/termine`.

E-Mail-Templates (sofern Hardlinks enthalten — zu prüfen unter `internal/mailer/templates` und `internal/scheduler` für Reminder-Mails) auf dasselbe Schema umstellen. **Tasks** sehen den Check explizit vor.

## Decisions

### Decision: `replace` statt `push` bei URL-Updates
**What:** Filter-Änderungen ersetzen den History-Eintrag.
**Why:** Sonst füllt jedes Klicken auf einen Filter-Toggle die Back-Button-History — der „Zurück"-Button wird unbrauchbar. Trade-off: Nutzer kann nicht per Back zum vorigen Filter — akzeptabel, da Filter trivial neu einzustellen sind.
**Alternatives considered:** `push` — verworfen wegen Back-Button-Spam.

### Decision: `focus` als kombinierter String, nicht zwei Params
**What:** `focus=game-17` statt `focus_type=game&focus_id=17`.
**Why:** Kürzere URL, weniger Validierungslogik, klar erkennbar in Logs/Bookmarks.
**Alternatives considered:** Zwei Parameter — overhead ohne Gewinn.

### Decision: Auto-Expand Past bei Focus auf vergangenen Termin
**What:** Wenn `focus`-Termin in Vergangenheit liegt, wird `past=1` automatisch gesetzt.
**Why:** Push-Notifications werden oft kurz vor oder während eines Spiels gelesen. Wenn die Spielzeit gerade vorbei ist, würde der Termin sonst unsichtbar bleiben.
**Risk:** Nutzer öffnet den Link, sieht plötzlich vergangene Termine. Akzeptabel — er hat aktiv auf einen Termin geklickt.

### Decision: Focus-Termin umgeht restriktive Filter
**What:** Wenn `team`/`types`-Filter den fokussierten Termin verbergen würden, wird er trotzdem angezeigt + hervorgehoben.
**Why:** Push-Klick darf nicht in einer leeren Liste landen, nur weil ein vorheriger Bookmark-Filter aktiv ist.
**Alternatives considered:** Filter strikt durchsetzen + Hinweis „durch Filter ausgeblendet" → schlechtere UX im häufigsten Fall (Push-Klick).

## Risks / Trade-offs

- **Falscher Termin-Typ in URL:** Reine Validierung über Regex — Backend liefert garantiert `training` oder `game`. Risiko = niedrig.
- **Backend-URL-Drift:** Wenn `/termine`-Route je umbenannt wird, brechen alle hist. Push-Notifications. Wahrscheinlichkeit niedrig (existierende Push-URLs verwenden bereits hartkodierte Pfade wie `/kalender`).
- **PWA-Cache:** Falls der Service Worker den alten Bundle hält und das neue URL-Schema nicht versteht, könnte ein Focus-Parameter ignoriert werden — Default-Liste wird gerendert (kein Fehler). Nach SW-Update wirkt die neue Logik.

## Migration Plan

Schrittweise, keine DB-Migration nötig:

1. Frontend: `TerminePage` auf `useSearchParams` umstellen + Focus-Logik. **Kompatibel rückwärts**: Aufrufe ohne Query-Params verhalten sich identisch.
2. Frontend: `EventInfoModal`-Button. Standalone, kein Risiko.
3. Backend: `notify.Send`-URLs ändern. Nur neue Notifications nutzen das neue Schema; bereits ausgelieferte Push-Nachrichten verlinken weiter auf die alten Pfade — die alten Routen existieren weiterhin und liefern den Kalender/Termine wie bisher.
4. Optional cleanup nach Beobachtungsphase: keine.

Kein Feature-Flag nötig — Änderung ist klein, additiv und rückwärtskompatibel.

## Open Questions

- Sollen E-Mail-Reminder (`duty-reminder-emails`, `notification_log`-basierte Reminder) ebenfalls auf `/termine?focus=...` umgestellt werden? **Antwort für diesen Change:** Ja, sofern sie Spiele oder Trainings betreffen; Dienst-Reminder bleiben bei `/dienste`. Wird in `tasks.md` mit einem expliziten Audit-Task abgedeckt.
- Brauchen wir einen Toast „Termin nicht verfügbar" oder reicht eine Inline-Info? **Entschieden:** Inline-Info oberhalb der Liste (siehe Spec), weil Toasts in der App bisher nicht etabliert sind.
