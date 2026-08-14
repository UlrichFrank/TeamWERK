## Why

Der globale „Zurück"-Button (`AppShell.tsx`, `navigate(-1)`, echter Browser-History-Pop) bringt Nutzer auf `/termine`, `/dienste` und `/kalender` aktuell nicht dorthin zurück, wo sie vorher waren: Listen springen an den Anfang statt zum zuletzt betrachteten Termin/Dienst-Slot, und der Kalender fällt immer auf den aktuellen Monat zurück statt auf den zuvor durchblätterten. Ursache ist, dass der Scroll-Container überall das eigene `<main class="overflow-auto">` ist (nicht `window`), wofür es keinerlei Restaurierungs-Mechanismus gibt, und dass der Kalender-Monat nie in der URL landet. Das macht wiederholtes Vergleichen einzelner Termine/Dienste und das Navigieren in der Zukunft/Vergangenheit im Kalender umständlich.

## What Changes

- `TerminePage.tsx`: Der bereits vorhandene `focus=<kind>-<id>`-URL-Mechanismus (bisher nur für eingehende Deep-Links aus Push-Notifications verdrahtet) wird beim eigenen Karten-Klick selbst gesetzt, bevor zur Detail-Seite navigiert wird — der `/termine`-History-Eintrag trägt den Fokus dann schon in sich.
- `DutyPage.tsx` / `DutySlotList.tsx`: Exakt analoger Fokus-Mechanismus wie bei Termine wird neu gebaut (existiert dort noch nicht) — Slot-Zeilen bekommen eine stabile DOM-ID, der Anleitungs-Link setzt vor der Navigation einen `focus=slot-<id>`-Param auf `/dienste`.
- `useWindowedList.ts`: Bugfix im `scroll:'window'`-Modus — der Re-Measure-Listener hängt an `window`, obwohl real `<main>` scrollt; dadurch friert das Zeilenfenster nach dem ersten Render ein. Voraussetzung dafür, dass der Fokus-Mechanismus bei Dienste zuverlässig eine tatsächlich gerenderte Zeile findet.
- `KalenderPage.tsx`: `year`/`month` werden bei Monatswechsel/„Heute" zusätzlich in `?date=` gespiegelt (analog zum bereits etablierten Filter-in-URL-Muster bei Termine/Dienste), damit der besuchte Monat einen eigenen History-Eintrag bekommt.

## Capabilities

### New Capabilities
- `zurueck-navigation-restore`: Der globale Zurück-Button stellt auf `/termine`, `/dienste` und `/kalender` die zuvor betrachtete Position wieder her (fokussierter Termin/Dienst-Slot bzw. besuchter Kalendermonat), basierend auf URL-State statt einem separaten Speicher-Mechanismus.

### Modified Capabilities
(keine — der `useWindowedList`-Fix stellt lediglich die bestehende Anforderung „Scrollen tauscht Zeilen aus" aus `lazy-rendering` wieder her, ohne deren Verhalten zu ändern)

## Impact

- **Frontend:** `web/src/pages/TerminePage.tsx`, `web/src/pages/DutyPage.tsx`, `web/src/components/DutySlotList.tsx`, `web/src/hooks/useWindowedList.ts`, `web/src/pages/KalenderPage.tsx`.
- **Keine Backend-/API-/DB-Änderungen.**
- **Kein neuer State-Speicher** (kein sessionStorage-Scroll-Hook) — die Lösung nutzt ausschließlich die bereits vorhandene URL-Such-Parameter-Infrastruktur.
