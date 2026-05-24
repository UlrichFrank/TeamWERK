## ADDED Requirements

### Requirement: Tagesweise Batch-Dienstgenerierung
Das System SHALL einen Endpoint bereitstellen, der alle Dienst-Slots für alle Spiele eines Kalendertags in einem Schritt generiert und dabei die konfigurierte same_day_behavior- und adjacent_day_behavior-Optimierung korrekt anwendet.

#### Scenario: Erfolgreiche Generierung für Spieltag mit zwei Heimspielen
- **WHEN** `POST /api/admin/games/regenerate-day?date=2026-05-23` aufgerufen wird
- **THEN** werden alle leeren Slots aller Spiele am 2026-05-23 gelöscht
- **THEN** werden für jedes Spiel neue Slots gemäß zugewiesenem Template erzeugt
- **THEN** werden Dienste mit `same_day_behavior=skip`, die zeitlich zwischen zwei Spielen liegen, nicht erzeugt
- **THEN** gibt der Endpoint `200 OK` mit Zusammenfassung je Spiel zurück

#### Scenario: Dienst mit same_day_behavior=skip wird zwischen Spielen ausgelassen
- **WHEN** zwei Heimspiele um 11:00 und 15:00 existieren und ein Diensttyp `same_day_behavior=skip` hat
- **THEN** wird der Abbau-Dienst nach dem ersten Spiel (zeitlich zwischen 11:00 und 15:00) nicht generiert
- **THEN** wird der Aufbau-Dienst vor dem zweiten Spiel (zeitlich zwischen 11:00 und 15:00) nicht generiert

#### Scenario: Bereits belegte Slots bleiben erhalten
- **WHEN** ein Slot am Spieltag `slots_filled > 0` hat
- **THEN** wird dieser Slot nicht gelöscht
- **THEN** wird in der Response `kept_slots` für das betroffene Spiel erhöht

#### Scenario: Spiel ohne Template wird übersprungen
- **WHEN** ein Spiel kein gespeichertes `template_id` hat und kein Standard-Template für den Event-Typ existiert
- **THEN** wird dieses Spiel ohne Fehler übersprungen
- **THEN** werden die anderen Spiele des Tages normal verarbeitet

#### Scenario: Kein Spiel an diesem Tag
- **WHEN** `POST /api/admin/games/regenerate-day?date=2030-01-01` aufgerufen wird und keine Spiele existieren
- **THEN** gibt der Endpoint `200 OK` mit leerer Spiel-Liste zurück

### Requirement: Frontend-Einstiegspunkt im Spielplan-Kalender
Das System SHALL im Spielplan-Kalender auf Tagesebene einen Button „Dienste generieren" anzeigen, wenn an dem Tag mindestens ein Spiel existiert.

#### Scenario: Button erscheint bei Klick auf Spieltag
- **WHEN** der Nutzer auf einen Kalendertag klickt, an dem mindestens ein Spiel vorhanden ist
- **THEN** wird ein Dialog geöffnet, der alle Spiele des Tages mit ihrem zugewiesenen Template auflistet
- **THEN** enthält der Dialog einen „Generieren"-Button und einen „Abbrechen"-Button

#### Scenario: Bestätigung löst Batch-Generierung aus
- **WHEN** der Nutzer im Dialog auf „Generieren" klickt
- **THEN** wird `POST /api/admin/games/regenerate-day` aufgerufen
- **THEN** wird nach Erfolg der Spielplan neu geladen
- **THEN** wird eine Erfolgsmeldung mit Anzahl erzeugter Slots angezeigt

#### Scenario: Warnung bei Konflikten nach Generierung
- **WHEN** die Generierung abgeschlossen ist und Dienste des gleichen Typs zur gleichen Zeit für verschiedene Spiele entstanden sind
- **THEN** wird eine Warnmeldung angezeigt mit dem Hinweis, die Optimierungsregeln zu prüfen

#### Scenario: Button nicht sichtbar ohne Spiele
- **WHEN** der Nutzer auf einen Kalendertag ohne Spiele klickt
- **THEN** wird kein „Dienste generieren"-Button angezeigt
