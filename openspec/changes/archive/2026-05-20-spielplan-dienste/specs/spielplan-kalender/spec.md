## ADDED Requirements

### Requirement: Spielplan-Kalender zeigt Monatsübersicht mit Besetzungsampel
Das System SHALL eine Monatsansicht aller Heimspiele der aktiven Saison anzeigen. Jeder Spieltag zeigt Mannschaft, Gegner und einen farbkodierten Besetzungsgrad (Ampel) der verknüpften Dienste.

#### Scenario: Kalender-Seite aufrufen
- **WHEN** ein authentifizierter Nutzer die Route `/spielplan` aufruft
- **THEN** wird eine Monatsansicht des aktuellen Monats angezeigt mit allen Heimspielen und ihrer Besetzungsampel (rot = unbesetzt, gelb = teilbesetzt, grün = voll besetzt)

#### Scenario: Monat wechseln
- **WHEN** ein Nutzer auf „Vorheriger Monat" oder „Nächster Monat" klickt
- **THEN** lädt der Kalender die Spiele des entsprechenden Monats ohne Seiten-Reload

#### Scenario: Kein Spiel im Monat
- **WHEN** der ausgewählte Monat keine Heimspiele enthält
- **THEN** wird der leere Kalender angezeigt mit dem Hinweis „Keine Heimspiele in diesem Monat"

### Requirement: Besetzungsampel ist korrekt berechnet
Die Ampelfarbe pro Spieltag SHALL aus dem Verhältnis `slots_filled / slots_total` aller verknüpften Duty Slots berechnet werden.

#### Scenario: Vollständig besetzt
- **WHEN** alle verknüpften Duty Slots eines Spiels `slots_filled = slots_total` haben
- **THEN** wird der Spieltag grün dargestellt

#### Scenario: Teilweise besetzt
- **WHEN** mindestens ein Slot besetzt und mindestens ein Slot unbesetzt ist
- **THEN** wird der Spieltag gelb dargestellt

#### Scenario: Kein Slot besetzt
- **WHEN** alle verknüpften Duty Slots `slots_filled = 0` haben oder keine Slots existieren
- **THEN** wird der Spieltag rot dargestellt

### Requirement: Kalender ist für Admin und Trainer zugänglich
Die Seite `/spielplan` und die API `GET /api/games` SHALL für eingeloggte Nutzer mit Rolle `admin` oder `trainer` zugänglich sein.

#### Scenario: Trainer sieht Kalender
- **WHEN** ein Nutzer mit Rolle `trainer` `/spielplan` aufruft
- **THEN** wird der Kalender angezeigt

#### Scenario: Elternteil oder Spieler hat keinen Zugriff
- **WHEN** ein Nutzer mit Rolle `elternteil` oder `spieler` `GET /api/games` aufruft
- **THEN** antwortet das System mit HTTP 403
