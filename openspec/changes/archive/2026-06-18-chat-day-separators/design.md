## Context

Die Chat-Liste rendert in `ChatPage.tsx:545` direkt `messages.map(msg => <MessageBubble ... />)`. Jede Bubble hat unter sich ein `<span>` mit `toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })` (Z. 908 + 1011). Datum taucht aktuell nirgends auf — weder an der Bubble noch zwischen Bubbles.

Messages haben das Feld `sentAt` (ISO-Timestamp-String, vom Backend gesetzt). System-Messages (`msg.isSystem === true`) werden bereits als zentrierte „Pill" gerendert (Z. 548–553).

## Goals / Non-Goals

**Goals:**
- Tageswechsel im Chat-Verlauf jederzeit visuell erkennbar.
- Nahegelegene Tage menschenfreundlich („Heute", „Gestern").
- Weiter zurückliegende Tage mit vollem Datum inkl. Wochentag.
- Inline-Bubble-Timestamp bleibt schlank (`HH:MM`), das Datum kommt vom Separator.

**Non-Goals:**
- Server-seitige Vorberechnung, Aggregation oder Caching.
- Datumssprünge in der Konversations-Übersicht/Sidebar.
- Mehrere Separatoren über kalendarische Lücken (es gibt immer nur _einen_ Separator zwischen zwei Nachrichten, mit dem Datum der neueren).
- Sticky-Header beim Scrollen (denkbar als spätere Iteration, aber nicht jetzt).

## Decisions

### Decision 1: Reine Funktion `daySeparatorLabel(date, now)`, kein React-Hook

`daySeparatorLabel` ist eine reine `Date × Date → string`-Funktion ohne React-Abhängigkeit. Vorteile:
- Trivial testbar (Vitest ohne DOM).
- Wiederverwendbar (theoretisch auch andernorts, z.B. Push-Notification-Text).
- Komponente bleibt dumm.

**Alternative verworfen:** Hook `useDaySeparatorLabel(date)` mit eigenem Tick-Timer für „Heute" → „Gestern" um Mitternacht. Verkompliziert ohne Mehrwert: Wer den Chat um Mitternacht offen lässt, kann auch mit einer veralteten Anzeige leben — beim nächsten Live-Update (SSE) wird ohnehin neu gerendert.

### Decision 2: Tagesschlüssel auf lokale Mitternacht, nicht 24h-Distanz

`dayKey(d)` gibt `YYYY-MM-DD` aus den lokalen Datumsteilen zurück (`d.getFullYear()`, `d.getMonth()`, `d.getDate()`). Vergleich erfolgt über Stringgleichheit. Konsequenz: Eine Nachricht von 23:30 und eine um 00:30 am Folgetag erzeugen einen Separator, selbst wenn weniger als 24h dazwischen liegen. Das ist das gewünschte Verhalten — Tageswechsel bedeutet kalendarisch, nicht „vor 24h".

Distanzberechnung für das Label nutzt dieselbe Logik:
```
diffDays = (dayKey(now) - dayKey(date)) als kalendarische Tage
  0 → "Heute"
  1 → "Gestern"
  >= 2 → "{Wochentag}, {Tag}. {Monat} {Jahr}"
```

Implementierung der Diff-Berechnung: Beide Daten auf lokale Mitternacht setzen (`new Date(y, m, d)`), Differenz in ms, durch `86400000`, gerundet. Sommerzeit/Winterzeit verfälscht die ms-Diff um ±1h, deswegen `Math.round` (nicht `Math.floor`).

### Decision 3: Insertion über `for`-Loop mit Akkumulator, kein verschachteltes `.map`

```tsx
const nodes: ReactNode[] = []
let lastDayKey: string | null = null
const now = new Date()
for (const msg of messages) {
  const k = dayKey(new Date(msg.sentAt))
  if (k !== lastDayKey) {
    nodes.push(<DaySeparator key={`sep-${msg.id}`} label={daySeparatorLabel(new Date(msg.sentAt), now)} />)
    lastDayKey = k
  }
  nodes.push(msg.isSystem ? <SystemRow ... /> : <MessageBubble ... />)
}
return <>{nodes}</>
```

`now` wird einmal pro Render festgelegt — keine Re-Renders deshalb (`now` ändert sich nicht zwischen Renders ohne State-Change).

**Alternative verworfen:** `messages.flatMap(...)`. Funktional eleganter, aber `lastDayKey` müsste extern gehalten werden — Closure über `let` ist nicht schöner als der `for`-Loop und schlechter lesbar.

**Alternative verworfen:** Memoisieren via `useMemo`. Die Anzahl Messages im offenen Chat liegt typischerweise unter 200. Der Loop ist <1ms. Premature Optimization.

### Decision 4: Hairline-Style ohne Pille

```
──────────  Mittwoch, 15. April 2026  ──────────
```

Implementierung als Flex-Row:
```tsx
<div className="flex items-center gap-3 my-3 text-xs text-brand-text-muted">
  <div className="flex-1 h-px bg-brand-border-subtle" />
  <span>{label}</span>
  <div className="flex-1 h-px bg-brand-border-subtle" />
</div>
```

Linien `bg-brand-border-subtle` (statt z.B. `brand-border`), Text `brand-text-muted`. Beide aus dem etablierten Token-Set in `tailwind.config.js`.

**Alternative verworfen:** Pille analog zur System-Message (`bg-brand-surface-card px-3 py-1 rounded-full`). Wäre konsistent mit dem System-Message-Stil, aber visuell aufdringlich — der User hat ausdrücklich „dünn und wenig aufdringlich" gefordert. Die Hairline-Variante ist dezenter.

### Decision 5: Vitest als Frontend-Test-Runner

Das Frontend hat aktuell keine Tests. Statt diese eine Funktion „manuell durchdacht zu validieren", wird Vitest minimal eingerichtet:
- `vitest` + `@vitest/ui` als Devdependencies
- `vitest.config.ts` mit Jsdom-Environment (für künftige Component-Tests, jetzt aber nicht nötig)
- `pnpm test` als Script in `web/package.json`

Vitest fügt sich nahtlos in Vite ein und ist die kanonische Wahl. Setup-Aufwand ~10min, ermöglicht künftige Frontend-Tests ohne erneuten Setup-Aufwand.

**Alternative verworfen:** Jest. Mehr Konfiguration, schlechter mit ESM, kein Vorteil hier.

**Alternative verworfen:** Tests komplett weglassen, weil „nur eine Funktion". Die Logik (Mitternacht-Edge-Case, Sommerzeit-Round, Jahreswechsel) hat genug Fallstricke, dass Tests ihr Geld wert sind.

### Decision 6: Locale-Formatierung über `Intl.DateTimeFormat`

```ts
const longDate = new Intl.DateTimeFormat('de-DE', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
}).format(date)
// → "Mittwoch, 15. April 2026"
```

Konsistent mit dem bereits genutzten `toLocaleTimeString('de-DE', ...)` im selben Modul.

## Risks / Tradeoffs

| Risiko | Mitigation |
|---|---|
| Lange Chat-Verläufe mit vielen Tageswechseln werden „luftiger" und User muss mehr scrollen. | Akzeptiert. `my-3` ist bewusst nicht riesig. Lesbarkeit gewinnt. |
| Mitternacht-Drift: User hat Chat offen, „Heute" wird bei 00:00 nicht zu „Gestern". | Akzeptiert. Bei nächster Live-Update-Aktualisierung korrekt. |
| Sommerzeit-Übergang verfälscht ms-Diff. | `Math.round` statt `Math.floor` in `diffDays`. Im Test mit DST-Datum abgedeckt. |
| Neuer Vitest-Setup könnte CI brechen falls dort `pnpm test` automatisch läuft. | Vorher prüfen, ob CI ein Test-Step hat. Falls nein: kein Risiko. Falls ja: Step muss `web/`-Tests laufen lassen. |

## Migration / Rollout

- Reines Frontend-Feature, kein Datenbank-Touch, kein Backend-Touch.
- Deploy via normalem `make deploy` (bündelt `web/dist`).
- Kein Feature-Flag nötig — Änderung ist additiv-visuell, kein Risiko für bestehende Flows.

## Open Questions

Keine offenen Punkte — Spec ist nach Klarstellungen vollständig.
