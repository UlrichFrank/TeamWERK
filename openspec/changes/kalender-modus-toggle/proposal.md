## Why

Der Kalender (`/kalender`) dient heute zwei unterschiedlichen Zwecken — Dienstplanung (Wer hat welchen Slot bei welchem Spiel?) und Terminverwaltung (Trainings und Spieltermine bearbeiten) — ohne dass der Nutzer explizit zwischen diesen Absichten wechseln kann. Ein klares Modus-Konzept reduziert die kognitive Last und macht den richtigen Klick-Pfad pro Use-Case offensichtlich.

## What Changes

- Neuer segmentierter Toggle **[Dienste | Termine]** oben rechts auf `/kalender` (gleiche CSS-Struktur wie der "Team | Meine"-Toggle auf `/mitfahrgelegenheiten`)
- **Dienste-Modus** (Standard): Spieltag-Klick → `/kalender/:id` (SpieltagDetailPage, unverändert); Training-Klick → keine Aktion (nicht klickbar)
- **Termine-Modus**: Spieltag-Klick → neues `GameEditModal` (Felder: Datum, Uhrzeit, Gegner, Event-Typ); Training-Klick → bestehendes `TrainingEditModal`
- Rollenbasierter Zugang im Termine-Modus: Bearbeiten nur für admin/trainer/vorstand/sportliche_leitung; alle anderen sehen ein schreibgeschütztes `EventInfoModal` (Datum, Zeit, Ort/Gegner, RSVP-Zahlen)
- Neues Frontend-Komponent `GameEditModal` mit Anbindung an `PUT /api/admin/games/{id}` (API existiert bereits)
- Neues Frontend-Komponent `EventInfoModal` (read-only, kein Backend-Aufruf)

## Capabilities

### New Capabilities

- `kalender-modus-toggle`: Segmentierter Modus-Wechsler auf `/kalender` mit unterschiedlichem Klick-Verhalten pro Modus
- `game-edit-modal`: In-Calendar-Bearbeitungs-Modal für Spieltermine (Datum, Zeit, Gegner, Typ) für berechtigte Rollen
- `event-info-modal`: Schreibgeschütztes Detail-Modal für Kalendereinträge (Spieltage + Trainings) für Spieler/Elternteile

### Modified Capabilities

- `games`: Frontend-Klick-Verhalten auf Spieltage im Kalender ändert sich modus-abhängig (kein Backend-Änderung)

## Impact

- `web/src/pages/KalenderPage.tsx`: Neuer State `kalenderMode`, konditionaler Click-Handler für Spieltage und Trainings
- `web/src/components/GameEditModal.tsx`: Neues Komponent (analog zu `TrainingEditModal`)
- `web/src/components/EventInfoModal.tsx`: Neues Komponent (schreibgeschützte Detailansicht)
- API `PUT /api/admin/games/{id}`: Bereits vorhanden, wird vom `GameEditModal` konsumiert
- Keine Backend-Änderungen, keine Migrationen, keine neuen Abhängigkeiten
