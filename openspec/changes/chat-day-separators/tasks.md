## 1. Vitest-Setup (falls noch nicht vorhanden)

- [x] 1.1 Prüfen ob `web/` bereits Vitest enthält: `grep -E "vitest" web/package.json`. Falls vorhanden, Tasks 1.2–1.4 überspringen.
- [x] 1.2 Devdependencies installieren: `cd web && pnpm add -D vitest @vitest/ui jsdom`
- [x] 1.3 `web/vitest.config.ts` anlegen mit `test.environment = 'jsdom'`, `globals: true`
- [x] 1.4 Scripts in `web/package.json` ergänzen: `"test": "vitest run"`, `"test:watch": "vitest"`
- [x] 1.5 CI-Konfiguration prüfen (`.github/workflows/*`, `Makefile`): wenn ein Test-Step für Frontend existiert oder nachgezogen werden soll, Eintrag ergänzen. Sonst dokumentieren, dass Tests nur lokal/manuell laufen.

## 2. Pure Funktionen in `web/src/lib/chatDateFormat.ts`

- [x] 2.1 Datei `web/src/lib/chatDateFormat.ts` neu anlegen
- [x] 2.2 Internen Helper `dayKey(d: Date): string` implementieren — Format `YYYY-MM-DD` aus `getFullYear`/`getMonth`/`getDate` (nicht aus `toISOString`, das wäre UTC)
- [x] 2.3 Internen Helper `diffDays(later: Date, earlier: Date): number` — beide Daten auf lokale Mitternacht setzen (`new Date(y, m, d)`), ms-Diff bilden, `Math.round(ms / 86400000)`
- [x] 2.4 `daySeparatorLabel(date: Date, now: Date): string` exportieren:
  - `diffDays(now, date) === 0` → `"Heute"`
  - `=== 1` → `"Gestern"`
  - `>= 2` → `new Intl.DateTimeFormat('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(date)`
- [x] 2.5 `shouldRenderSeparator(prev: Message | null, current: Message): boolean` exportieren — `prev === null || dayKey(...) !== dayKey(...)`. Nur die `sentAt`-Strings vergleichen, kein `now` nötig.

## 3. Unit-Tests `web/src/lib/chatDateFormat.test.ts`

- [x] 3.1 Datei anlegen, `import { describe, it, expect } from 'vitest'`
- [x] 3.2 `daySeparatorLabel` — Heute: `now = new Date(2026, 5, 18, 14, 0)`, `date = new Date(2026, 5, 18, 9, 30)` → `"Heute"`
- [x] 3.3 `daySeparatorLabel` — Gestern: `now = new Date(2026, 5, 18, 14, 0)`, `date = new Date(2026, 5, 17, 23, 50)` → `"Gestern"`
- [x] 3.4 `daySeparatorLabel` — Vorgestern (≥2 Tage): `now = new Date(2026, 5, 18)`, `date = new Date(2026, 5, 16, 12, 0)` → `"Dienstag, 16. Juni 2026"`
- [x] 3.5 `daySeparatorLabel` — Vorjahr: `now = new Date(2026, 5, 18)`, `date = new Date(2025, 11, 24, 18, 0)` → `"Mittwoch, 24. Dezember 2025"`
- [x] 3.6 `daySeparatorLabel` — Mitternacht-Edge: `now = new Date(2026, 5, 18, 0, 30)`, `date = new Date(2026, 5, 17, 23, 30)` → `"Gestern"` (nicht „Heute", obwohl Differenz <1h)
- [x] 3.7 `daySeparatorLabel` — DST-Übergang (Frühjahr): Daten um den März-DST-Sonntag wählen und prüfen dass `diffDays` korrekt zählt (Test schützt `Math.round`-Entscheidung)
- [x] 3.8 `shouldRenderSeparator` — `prev = null` → `true`
- [x] 3.9 `shouldRenderSeparator` — Gleicher Tag → `false`
- [x] 3.10 `shouldRenderSeparator` — Tageswechsel → `true`

## 4. Komponente `web/src/components/DaySeparator.tsx`

- [x] 4.1 Datei anlegen, Props `{ label: string }`
- [x] 4.2 Markup gemäß Design Decision 4:
  ```tsx
  <div className="flex items-center gap-3 my-3 text-xs text-brand-text-muted">
    <div className="flex-1 h-px bg-brand-border-subtle" />
    <span>{label}</span>
    <div className="flex-1 h-px bg-brand-border-subtle" />
  </div>
  ```
- [x] 4.3 `aria-label` oder `role="separator"` ergänzen für Screenreader

## 5. Integration in `ChatPage.tsx`

- [x] 5.1 Imports ergänzen: `daySeparatorLabel`, `shouldRenderSeparator`, `DaySeparator`
- [x] 5.2 Render-Block ab Z. 545 umbauen: statt direktem `.map` ein `for`-Loop, der `nodes: ReactNode[]` füllt; `lastDayKey` wird im Loop getrackt, `now` einmal vor dem Loop instanziiert
- [x] 5.3 System-Messages weiterhin als zentrierte Pille rendern (Z. 548–553) — nur das Wrapper-Element ändert sich, der innere Aufbau bleibt
- [x] 5.4 `MessageBubble`-Aufruf bleibt unverändert
- [x] 5.5 `<div ref={messagesEndRef} />` am Ende des Containers belassen
- [x] 5.6 Build prüfen: `cd web && pnpm build` ohne TS-Fehler

## 6. Manuelle Verifikation

- [ ] 6.1 Lokal `make dev` starten, in einen Chat mit Verlauf über mehrere Tage navigieren
- [ ] 6.2 Separator über erster Nachricht, jeweils zwischen Tagen, nicht zwischen Nachrichten am gleichen Tag
- [ ] 6.3 Label-Werte stichprobenartig prüfen: heutige Konv. → „Heute", gestrige → „Gestern", ältere → volles Datum mit Wochentag
- [ ] 6.4 Tageswechsel über System-Message (z.B. „X hat die Gruppe verlassen") prüfen: Separator erscheint, wenn die System-Message an einem anderen Tag liegt
- [ ] 6.5 Eine neue Nachricht senden — Separator vor heutigen Nachrichten zeigt weiter „Heute", keine Duplikate
- [ ] 6.6 Visueller Check: Hairline ist dezent, lenkt nicht ab; Bubble-Timestamps zeigen weiterhin nur `HH:MM`

## 7. Commit & Archiv

- [ ] 7.1 Pro Task-Block einen Commit nach Conventional-Commits-Format:
  - `chore(web): vitest setup`
  - `feat(chat): daySeparatorLabel und shouldRenderSeparator`
  - `test(chat): Tests für daySeparatorLabel + shouldRenderSeparator`
  - `feat(chat): DaySeparator-Komponente`
  - `feat(chat): Tageswechsel-Separatoren im Verlauf rendern`
- [ ] 7.2 Abschlusscommit, der die OpenSpec-Proposal-Datei archiviert (`opsx:archive`)
