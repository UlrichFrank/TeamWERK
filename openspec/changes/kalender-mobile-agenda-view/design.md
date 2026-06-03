## Context

`KalenderPage.tsx` lädt Spiele und Trainings via `api.get('/kalender?year=&month=')` und gruppiert sie in `gamesByDate` und `trainingsByDate` (je ein Record<string, Game[]> bzw. Record<string, Training[]>). Diese Datenstrukturen können direkt für die Agenda-View wiederverwendet werden — kein neuer API-Call nötig.

Das Dual-Render-Pattern (`sm:hidden` / `hidden sm:block`) ist bereits an zwei Stellen im Projekt etabliert:
- `AdminSettingsPage`: Mobile Cards + Desktop-Tabelle
- `AdminDutyTemplateDetailPage`: Mobile Accordion + Desktop-Zeilen

Der Desktop-Kalender hat Swipe-Gesten via Pointer-Events (`onPointerDown/Move/Up`). Diese bleiben auf Desktop aktiv; auf Mobile werden sie durch natürliches Scrollen der Agenda ersetzt.

## Goals / Non-Goals

**Goals:**
- Lesbare, tippbare Darstellung aller Monats-Events auf Mobile
- FAB für Admins/Trainer zum Anlegen neuer Events (ersetzt group-hover-Button)
- Keine neuen API-Aufrufe, keine neue State-Verwaltung

**Non-Goals:**
- Keine Wochenansicht oder andere Kalender-Paradigmen
- Kein Offline-Support / Caching
- Keine Änderung am Desktop-Grid

## Decisions

**Agenda-Liste statt scrollbarem Grid:** Ein `overflow-x: auto` mit fixierter Grid-Mindestbreite wäre die einfachste Lösung, erzeugt aber schlechte Touch-UX (horizontales Scrollen in einer vertikal scrollenden App). Eine Agenda-Liste ist das etablierte Mobile-Muster und nutzt die bestehenden Datenstrukturen direkt.

**FAB statt Inline-Button:** Auf Mobile gibt es keinen Hover-State. Ein FAB (`fixed bottom-6 right-6`) ist touch-freundlich (44px+ Touch-Target), immer sichtbar, und ist ein etabliertes Mobile-Pattern für primäre Aktionen. Er öffnet denselben Wizard wie der Desktop-„+"-Button.

**Datum-Gruppierung aus `gamesByDate` + `trainingsByDate`:** Beide Maps sind bereits befüllt. Die Agenda-View iteriert über alle Tage des Monats (1..daysInMonth), zeigt nur Tage mit mindestens einem Event, und rendert Spiele vor Trainings pro Tag.

**Kein separates State für Agenda/Grid-Toggle:** Die View-Auswahl erfolgt ausschließlich über CSS (`sm:hidden` / `hidden sm:block`). Kein `useState` für den View-Modus nötig.

## Risks / Trade-offs

- [Langer Monat mit vielen Events] → Liste wird lang, User muss scrollen. Mitigation: Tage ohne Events werden übersprungen; zukünftig könnte ein „Nur zukünftige" Filter ergänzt werden (out of scope).
- [FAB überdeckt letzten Listeneintrag] → Padding-bottom am Listencontainer (`pb-20`) verhindert Überlappung.

## Migration Plan

Kein Migrations- oder Rollback-Aufwand — reine Frontend-Ergänzung, kein Backend berührt.
