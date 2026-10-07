# kalender-sprung Specification

## Purpose

Rückweg von den Listenseiten `/termine` und `/dienste` in den Monatskalender: jeder Termin bzw. Dienst-Block springt per Klick zum selben Termin in `/kalender`.

## Requirements

### Requirement: Aktion „Im Kalender öffnen“ auf Termin-Karten
Jede Termin-Karte auf `/termine` (Spiel, Event und Training, auch abgesagte) SHALL eine Icon-Aktion „Im Kalender öffnen“ (Icon `CalendarDays`, `aria-label="Im Kalender öffnen"`) tragen. Ein Klick SHALL zu `/kalender?date=<Termindatum YYYY-MM-DD>&focus=<game|training>-<id>` navigieren (Client-Navigation, kein Neuladen) und SHALL NICHT den Karten-Klick (Detailseite) auslösen. Die Aktion steht wie bei den Dienst-Blöcken ganz rechts oben in der Kopfzeile der Karte, rechts von den Zusage-Zählern, und bricht auf schmalen Bildschirmen nicht mit ihnen in die nächste Zeile um.

#### Scenario: Spiel-Karte springt in den Kalender
- **WHEN** ein Nutzer auf `/termine` bei der Karte von Spiel 17 am 2026-10-11 „Im Kalender öffnen“ klickt
- **THEN** navigiert die App zu `/kalender?date=2026-10-11&focus=game-17`
- **THEN** öffnet sich NICHT die Spiel-Detailseite

#### Scenario: Trainings-Karte springt in den Kalender
- **WHEN** ein Nutzer auf `/termine` bei der Karte von Training 42 am 2026-10-14 „Im Kalender öffnen“ klickt
- **THEN** navigiert die App zu `/kalender?date=2026-10-14&focus=training-42`

### Requirement: Aktion „Im Kalender öffnen“ auf Dienst-Blöcken
Jeder Block auf `/dienste`, der an einem Termin hängt, SHALL im Kopf des Blocks rechts dieselbe Icon-Aktion tragen. Ein Klick SHALL zu `/kalender?date=<Termindatum>&focus=game-<game_id>` navigieren. Blöcke ohne Termin (termin-lose Dienst-Slots) SHALL keine Aktion tragen. Die Aktion SHALL für alle Nutzer erscheinen, die den Block sehen — auch vergangene Blöcke.

#### Scenario: Dienst-Block eines Spiels
- **WHEN** ein Nutzer auf `/dienste` im Block des Spiels 17 am 2026-10-11 „Im Kalender öffnen“ klickt
- **THEN** navigiert die App zu `/kalender?date=2026-10-11&focus=game-17`

#### Scenario: Block ohne Termin
- **WHEN** `/dienste` einen Block ohne zugehörigen Termin anzeigt
- **THEN** trägt dieser Block keine Aktion „Im Kalender öffnen“
