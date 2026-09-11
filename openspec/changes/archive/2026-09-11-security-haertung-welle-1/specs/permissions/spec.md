## ADDED Requirements

### Requirement: Objekt-Gates auf Detail- und Mutationsrouten

Routen, die ein einzelnes Objekt lesen oder verändern, MUST die Berechtigung am Objekt prüfen, nicht nur am Auth-Tier: `GET /api/training-sessions/{id}` folgt dem Kader-Zugriff; `POST /api/games/{id}/lineup` erlaubt nur Trainer eines beteiligten Teams (admin und sportliche Leitung vereinsweit); `POST /api/chat/conversations/{id}/read` und `DELETE /api/chat/conversations/{id}/members/me` verlangen aktive Mitgliedschaft; `POST /api/duty-assignments/{id}/fulfill` und `/cash-substitute` verlangen admin, trainer oder sportliche Leitung (das bestehende Recht, nun auch im Handler geprüft) und MUST mit 404 antworten, wenn die Zuweisung nicht existiert.

#### Scenario: Trainer eines fremden Teams speichert keine Aufstellung
- **WHEN** ein Trainer, dessen Kader an dem Spiel nicht beteiligt ist, `POST /api/games/{id}/lineup` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Nicht-Mitglied kann keine Lesebestätigung setzen
- **WHEN** ein Nutzer ohne aktive Mitgliedschaft `POST /api/chat/conversations/{id}/read` aufruft
- **THEN** antwortet der Server mit HTTP 403 und schreibt keine `message_reads`-Zeile

#### Scenario: Nicht-Mitglied hinterlässt keine System-Nachricht
- **WHEN** ein Nutzer ohne aktive Mitgliedschaft `DELETE /api/chat/conversations/{id}/members/me` aufruft
- **THEN** antwortet der Server mit HTTP 403 und es entsteht keine Nachricht

#### Scenario: Spieler sieht fremde Trainingseinheit nicht
- **WHEN** ein Spieler ohne Kader-Zugriff `GET /api/training-sessions/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst-Erfüllung nur durch Berechtigte und nur für existierende Zuweisungen
- **WHEN** ein Spieler `POST /api/duty-assignments/{id}/fulfill` aufruft
- **THEN** antwortet der Server mit HTTP 403
- **WHEN** ein Trainer dieselbe Route für eine nicht existierende ID aufruft
- **THEN** antwortet der Server mit HTTP 404
