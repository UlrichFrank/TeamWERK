## Context

Die `MitfahrgelegenheitenPage` (`web/src/pages/MitfahrgelegenheitenPage.tsx`) zeigt heute Spiele in drei Tabs (Auswärts / Heim / Events). Die parallel existierende `TerminePage` zeigt alle Termine (Trainings + Spiele + Events) in einer chronologischen Liste mit Filter-Pills und farblicher Codierung. Das Pattern in TerminePage ist bewährt, kommt mit Compact-Header für Mobile und URL-Persistierung der Filter — alles bereits in dedizierten Helfern (`getEventColors`, `useCompactHeader`, `buildTeamShortNames`).

Die Mitfahrgelegenheiten-Seite soll auf das gleiche visuelle Pattern umgestellt werden — ohne Training-Filter (Mitfahrt gibt es nur für Spiele/Events).

**Wiederverwendbare Bausteine** (existieren bereits):
- `web/src/lib/eventColors.ts` — `getEventColors(type)` liefert pro Event-Typ Klassen-Strings für Card, Filter-Pill und Pill-Variante
- `web/src/lib/teamName.ts` — `buildTeamShortNames(teams)` bildet kompakte Team-Kürzel (z. B. `mA2`)
- `web/src/hooks/useCompactHeader.ts` — Viewport-Listener mit konfigurierbarer Schwelle

**Backend-Daten** liefert `GET /api/mitfahrgelegenheiten` bereits:
- `game.eventType` ∈ `{heim, auswärts, generisch}` — Spalte für Farbcodierung & Pill-Filter
- `game.date`, `game.team` — bereits vorhanden, aber `team` ist nur der lange Name

**Wichtige Entdeckung — Team-Kürzel-Lookup**:
- `GET /api/teams/my` liefert nur `{id, name, isExtended}` — **nicht** die für `buildTeamShortNames` nötigen Felder (`age_class`, `gender`, `team_number`, `group_count`).
- `GET /api/teams` liefert die vollen Felder, ist aber rollenabhängig gefiltert (Vorstand/Admin → alle).

## Goals / Non-Goals

**Goals:**
- Konsistente visuelle Sprache zwischen `TerminePage` und `MitfahrgelegenheitenPage`.
- Chronologische Übersicht statt Tab-Wechsel.
- Mehrfach-Auswahl von Event-Typen möglich (z. B. nur Heim + Auswärts).
- Filter-State per URL teilbar.
- Compact-Header für Mobile ohne Layout-Bruch.

**Non-Goals:**
- Backend-Änderungen — der API-Vertrag bleibt unverändert.
- Vergangenheits-Toggle — bewusst nicht übernommen (Mitfahrt für vergangenen Termin sinnlos).
- Änderung an Biete/Suche-Karten-Inhalt, an Paarungs-Workflow oder an `FormModal`.
- Migrationspfad zu `/teams` für den Filter-Dropdown — bleibt bei `/teams/my` (eigene Teams).
- Ein einheitlicher Backend-Endpoint, der direkt `team_shortname` mitliefert. (Optional zukünftig — wir lösen das im Frontend.)

## Decisions

### 1. Filter-Bar-Aufbau (analog `TerminePage`)

```tsx
<h1>Mitfahrgelegenheiten</h1>
<select> Alle Teams | Team X | Team Y …  ← aus /teams/my
<Pill type="heim">      ← getEventColors('heim').filter
<Pill type="auswärts">  ← getEventColors('auswärts').filter
<Pill type="generisch"> ← getEventColors('generisch').filter
<Pill type="mine">      ← brand-yellow active state, UserCheck-Icon
```

Pills toggeln je einen `Set<string>`-Eintrag. "Meine" hat einen eigenen Boolean-State, wird aber visuell wie eine Pill gerendert.

**Alternative verworfen**: Pills als Checkbox-Liste mit Labels. → Pill-Variante ist bereits etabliert und kompakt; Konsistenz schlägt Innovation.

### 2. Team-Kürzel-Lookup ohne Backend-Änderung

Wir laden **zwei** Endpoints im Mount-Hook:
- `/teams/my` → für die Dropdown-Optionen (User sieht nur eigene Teams im Filter)
- `/teams` → für die Kürzel-Map (`buildTeamShortNames`), gefiltert über die User-Rolle

Das ergibt eine kleine Redundanz. Akzeptabel, weil:
- Beide Calls sind cached und billig.
- `/teams` ist bereits ein Standard-Endpoint, der von vielen Seiten genutzt wird.
- Alternative wäre, `/teams/my` um `age_class`, `gender`, `team_number`, `group_count` zu erweitern — größerer Eingriff für minimalen Nutzen.

**Fallback**: Wenn `/teams` keine Daten für eine `team_id` liefert (z. B. weil das Team nicht in der aktiven Saison ist), wird der lange `game.team`-Name als Sortierschlüssel verwendet.

### 3. Sortierung

Primär: `game.date + 'T' + game.time` (lexikographisch, da ISO-Format).
Sekundär: Team-Kürzel über die ShortName-Map; Fallback `game.team`-Langname.

```ts
function sortKey(d: GameCarpoolData): string {
  const dateKey = d.game.date.slice(0, 10) + 'T' + (d.game.time ?? '00:00')
  const teamKey = teamShortNames.get(d.game.teamId) ?? d.game.team
  return dateKey + '|' + teamKey
}
```

**Problem**: Der heutige `GameCarpoolData.game` enthält **keinen** `teamId` — nur den `team` als String. → **Backend-Anpassung leichtgewichtig nötig**: Im Response `team_id` (oder `team_ids` für generische Multi-Team-Events) ergänzen.

**Alternative verworfen**: Sortierung rein über Long-Team-Name. → Verfehlt das vom User explizit gewünschte "Kürzel".

**Entscheidung**: Backend-Response um `team_id: number` (Single-Team-Events) bzw. `team_ids: number[]` (Multi-Team generisch) erweitern. Bestehende Felder bleiben unangetastet. **Damit verlässt das Change die "reine Frontend"-Klassifizierung leicht — der Eingriff ist aber minimal (ein zusätzliches Feld in der Response).**

### 4. Card-Farbcodierung

`getEventColors(eventType)` liefert direkt:
- `card.border` → `border-brand-yellow` / `border-brand-text-muted` / `border-brand-blue`
- `card.bg` → leichter Tint (10–20 % Opacity)

Die GameCard wird umgebaut: statt `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow` (uniform gelb) wird die Border-Color und Background-Tint dynamisch je nach `eventType` gesetzt:

```tsx
<div className={`rounded-xl shadow border-t-4 overflow-hidden ${colors.card.border} ${colors.card.bg}`}>
```

Das Card-Innenleben (Biete/Suche, Paarungen) wird nicht verändert — der farbige Streifen + Tint sitzt drumherum.

### 5. URL-Search-Params

Pattern aus `TerminePage` übernommen — drei Parameter:
- `?team=<id>` (numerisch, optional, default: alle eigenen Teams)
- `?types=<heim,auswärts,generisch>` (CSV, default: alle drei aktiv = kein Parameter)
- `?mine=1` (default: Team-Modus = kein Parameter)

**Default-State** (keine Params): alle Teams, alle Typen, Team-Modus. URL bleibt clean.

### 6. Compact-Header

`useCompactHeader(950)` — Threshold gleich wie TerminePage. Bei `compact === true`:
- Pills zeigen nur das Icon (Label `hidden`)
- Padding reduziert auf `px-2` statt `px-3`

### 7. "Meine"-Pill-Icon

`<UserCheck>` aus `lucide-react`. Bewusst nicht `<User>`, da die Pill ausdrückt "Spiele, an denen *ich* beteiligt bin (bestätigt/angefragt)".

## Risks / Trade-offs

- **[Backend-Touchpoint kleiner als angekündigt]** → Der Proposal nennt die Änderung "rein Frontend". Tatsächlich braucht die Sortierung nach Team-Kürzel ein zusätzliches `team_id`-Feld in `GET /api/mitfahrgelegenheiten`. Mitigation: Im Proposal/Impact und in den Tasks explizit ausweisen; Tests für die erweiterte Response ergänzen.

- **[Zwei Team-Endpoints werden geladen]** → `/teams/my` (Filter) + `/teams` (Kürzel-Map) ergeben einen zusätzlichen Roundtrip. Mitigation: Beide Calls parallel im Mount-Hook; bei modernen Verbindungen vernachlässigbar.

- **[Bestehende Lesezeichen]** → Wer aktuell `?tab=heim` bookmarkt, verliert das. Mitigation: Die `?tab`-Param war nie persistiert (der State war `useState`), Lesezeichen darauf existieren also nicht.

- **[Visuelle Veränderung deutlich]** → Heutiges Layout zeigt alle Karten gelb; künftig nur Heimspiele. Mitigation: Konsistenz mit `TerminePage`/`KalenderPage` schafft das visuelle Vokabular über die ganze App.

- **[GameCard-Wrapper hat heute eigene Border-Top-Klasse]** → Die innere Komponente `GameCard` rendert den Wrapper selbst. Beim Umbau muss die Border-Logik mit übergeben werden (Prop `eventType`) oder die `getEventColors`-Lookup wandert in `GameCard`.

## Migration Plan

1. Backend-Response um `team_id` (bzw. `team_ids`) erweitern, Test hinzufügen.
2. Frontend: `GameCarpoolData.game.teamId`/`teamIds` in Typdefinition aufnehmen.
3. Filter-Header umbauen, alte Tab-Logik entfernen.
4. Sortierung implementieren, ShortName-Lookup verdrahten.
5. `GameCard`-Wrapper farblich kodiert; Innenleben unverändert.
6. URL-Search-Params verdrahten.
7. Manuell testen (Mobile / Desktop / mehrere Teams / `mine`-Modus / Filter-Kombinationen).

**Rollback**: Die Änderung ist rein UI-seitig (plus ein additives Response-Feld, das von keinem alten Client gelesen wird). Ein Revert des Commits reicht.

## Open Questions

- **Sollen Trainer/Sportliche-Leitung im Team-Filter weiterhin nur die eigenen Teams sehen?** Aktuell: ja, `/teams/my`. Ein Vorstand kann via `/teams` alle Teams einsehen, müsste aber für Mitfahrgelegenheiten alle Teams im Filter zeigen können. Für den Erstwurf bleibt es bei `/teams/my` für alle Rollen — falls später Bedarf besteht, kann der Endpoint je nach Rolle gewechselt werden.

- **Was passiert bei generischen Multi-Team-Events?** Heute kombiniert das Backend die Team-Namen kommagetrennt. Für die Kürzel-Sortierung nehmen wir den ersten Team-Kürzel (alphabetisch); alternativ alle Kürzel kommagetrennt. → **Entscheidung**: alphabetisch sortieren über die kommagetrennten Kürzel; falls keine Kürzel auflösbar sind, Fallback auf den Long-Name.
