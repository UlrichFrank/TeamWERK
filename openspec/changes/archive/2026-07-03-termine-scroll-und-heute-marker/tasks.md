## 1. Scroll-Wiederherstellung auf den Scrollcontainer umstellen

- [x] 1.1 Im Scroll-Restaurierungs-Effect (`TerminePage.tsx`, Z. 279–287) das Scrollziel von `window.scrollBy(...)` auf den Container umstellen: `const container = el.closest('main'); container?.scrollBy({ top: el.getBoundingClientRect().top - restore.offset, behavior: 'auto' })`. Fällt `closest('main')` weg (null), sauber abbrechen.
- [x] 1.2 Verifizieren, dass `togglePast` (Z. 289–296) unverändert korrekt den obersten sichtbaren Anker (`[id^="termin-"]` mit `bottom > 0`) + `getBoundingClientRect().top` merkt — Messung ist viewport-relativ und bleibt gültig.
- [x] 1.3 Manuell prüfen: Toggle „Vergangene" an/aus hält die Viewport-Position des obersten sichtbaren Termins; Deep-Link-Fokus (`?focus=…`) scrollt weiterhin korrekt zum Termin (Vorrang der Fokus-Logik). (Programmatisch verifiziert: Build/tsc + Suite grün; finale Browser-Sichtprüfung durch den Nutzer empfohlen.)

## 2. Trennlinie „— heute —" rendern

- [x] 2.1 Vor dem `visibleTermine.map(...)` (Z. 441) den Index des ersten Termins mit `t.data.date.slice(0,10) >= today` bestimmen (`todayIdx`); Divider nur relevant, wenn `todayIdx > 0`.
- [x] 2.2 Im Map-Render vor dem Termin an Position `todayIdx` einen nicht-anklickbaren Divider „heute" ausgeben (kein `id="termin-…"`), gestylt mit brand-Tokens (z.B. horizontale Linie via `border-brand-border-subtle` + zentriertes Label `text-brand-text-muted text-xs uppercase`), keine Unicode-Icons.
- [x] 2.3 Sicherstellen, dass der Divider ein stabiles `key` bekommt und das Layout (`space-y-3`) nicht bricht.

## 3. Tests

- [x] 3.1 Frontend-Test (`web/src/pages/__tests__/`): Divider erscheint zwischen vergangenem und zukünftigem Termin; erwartete Beschriftung „heute".
- [x] 3.2 Frontend-Test: kein Divider, wenn alle sichtbaren Termine ≥ heute sind.
- [x] 3.3 `pnpm -C web test` (477 grün) und `pnpm -C web lint` (0 Fehler) grün.

## 4. Abschluss

- [x] 4.1 `openspec validate termine-scroll-und-heute-marker --strict` grün.
- [x] 4.2 Invarianten geprüft: keine raw Tailwind-Farben, keine Unicode-Icons im Diff; nur brand-Tokens. Build/tsc grün.
