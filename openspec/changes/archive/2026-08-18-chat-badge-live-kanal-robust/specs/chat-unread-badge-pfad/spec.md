## ADDED Requirements

### Requirement: Aktualität des Ungelesen-Zählers

Die Zahl, die alle Anzeigestellen tragen (Nav-Modul-Header, Hinweis-Punkt am Hamburger,
Tabs auf `/chat`), SHALL sich nicht ausschließlich auf den Live-Kanal verlassen. Sie SHALL
zusätzlich neu geladen werden, wenn

- die Anwendung nach einer Unterbrechung wieder sichtbar wird, und
- der Browser nach einem Ausfall wieder online geht.

Grund: Eine Homescreen-PWA wird beim Wechsel in den Hintergrund eingefroren und später
**ohne erneutes Mounten** fortgesetzt. Ein Zähler, der nur beim Mounten und bei
Live-Ereignissen geladen wird, hat in diesem Zustand keinen Weg zurück zur Wahrheit; er
zeigt den zuletzt geladenen Wert unbegrenzt weiter — in aller Regel `0`, weil beim letzten
Blick in den Chat alles gelesen war.

Ein fehlgeschlagener Ladeversuch SHALL folgenlos bleiben für die Anzeige (kein Fehlerdialog,
kein Sprung auf `0` — der zuletzt bekannte Wert bleibt stehen), aber NICHT folgenlos für den
Zustand: der Versuch SHALL beim nächsten Auslöser nachgeholt werden. Ein Kaltstart ohne Netz
darf den Zähler nicht dauerhaft auf `0` festschreiben.

Solange der Zähler noch nie erfolgreich geladen wurde, SHALL keine Anzeigestelle eine `0`
als gesicherte Aussage darstellen; „unbekannt" und „nichts ungelesen" sind verschiedene
Zustände.

#### Scenario: PWA kehrt aus dem Hintergrund zurück

- **WHEN** die Anwendung im Hintergrund war, währenddessen Nachrichten eingetroffen sind und sie ohne Neu-Mount wieder in den Vordergrund kommt
- **THEN** wird der Ungelesen-Zähler neu geladen
- **THEN** zeigen die Anzeigestellen die aktuelle Zahl

#### Scenario: Live-Kanal ist ausgefallen

- **WHEN** der Chat-Ereigniskanal getrennt ist und eine neue Nachricht eintrifft
- **THEN** ist die Zahl spätestens beim nächsten Sichtbarwerden der Anwendung wieder korrekt

#### Scenario: Verbindung kehrt zurück

- **WHEN** das Gerät offline war und wieder online geht
- **THEN** wird der Ungelesen-Zähler neu geladen

#### Scenario: Fehlgeschlagener Start-Load wird nachgeholt

- **WHEN** der erste Ladeversuch beim Start scheitert (kein Netz)
- **THEN** bleibt die Anzeige ohne Fehlermeldung
- **THEN** wird der Ladeversuch beim nächsten Sichtbarwerden wiederholt
- **THEN** erscheint die Zahl, sobald ein Versuch erfolgreich war

#### Scenario: Erfolgreiche Nullzählung bleibt eine Aussage

- **WHEN** der Zähler erfolgreich geladen wurde und das Ergebnis `0` ist
- **THEN** zeigt keine Anzeigestelle einen Badge
- **THEN** wird dieser Zustand nicht als „unbekannt" behandelt
