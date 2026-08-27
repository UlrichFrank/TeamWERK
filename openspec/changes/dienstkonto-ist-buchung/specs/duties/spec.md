## ADDED Requirements

### Requirement: Das Dienstkonto folgt dem Status der Zuweisungen

`duty_accounts.ist` SHALL zu jedem Zeitpunkt die Summe der Dauern derjenigen Dienst-Slots
sein, auf denen der Nutzer in der jeweiligen Saison eine als geleistet geltende Zuweisung
hat. Der Wert SHALL nicht davon abhängen, ob zufällig ein Termin gelöscht wurde.

Jeder Statuswechsel einer Zuweisung, der sie in diese Menge hinein- oder aus ihr
herausbewegt, SHALL das Konto nachziehen — mindestens: Erledigung
(`POST /api/duty-assignments/{id}/fulfill`), Ersatzzahlung
(`POST /api/duty-assignments/{id}/cash-substitute`), Zurückgeben einer Zuweisung und das
Löschen eines einzelnen Dienst-Slots (`DELETE /api/duty-slots/{id}`).

Die Bestandswerte SHALL einmalig über alle `(user_id, season_id)`-Paare neu berechnet
werden — die heute gespeicherten Zahlen sind nicht rekonstruierbar korrekt.

#### Scenario: Dienst wird als erledigt markiert

- **WHEN** ein Vorstand eine Dienst-Zuweisung als erledigt markiert
- **THEN** steigt `duty_accounts.ist` des Zugewiesenen um die Dauer dieses Slots
- **AND** zeigt die Dienstkonten-Ansicht den neuen Stand ohne weiteren Eingriff

#### Scenario: Erledigter Dienst wird entfernt

- **WHEN** ein einzelner Dienst-Slot mit einer erledigten Zuweisung gelöscht wird
- **THEN** sinkt `duty_accounts.ist` des Zugewiesenen um die Dauer dieses Slots
- **AND** bleiben die übrigen erledigten Dienste der Saison unverändert angerechnet

#### Scenario: Konto ohne erledigte Dienste

- **WHEN** ein Nutzer in einer Saison keine erledigte Zuweisung hat
- **THEN** ist `duty_accounts.ist` für diese Saison 0
