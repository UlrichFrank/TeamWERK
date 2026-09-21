## MODIFIED Requirements

### Requirement: Ranglisten je Staffel

Das System SHALL je Staffel Ranglisten bereitstellen — mindestens Torschützen,
Siebenmeter-Quote und eine Fair-Play-Wertung aus Zeitstrafen und Karten — und diese über
alle ausgewerteten Berichte der Saison bilden.

Das System SHALL zu jedem Spieler die Zahl der Spiele ausweisen, in denen er in einer
Mannschaftsliste geführt wurde, damit seine Werte einordbar sind.

Das System SHALL die Siebenmeter-Wertung als eigene Rangliste bereitstellen, nach
getroffenen Siebenmetern sortiert, und darin neben Versuchen und Treffern auch die
Fehlversuche ausweisen.

Das System SHALL in die Siebenmeter-Rangliste nur Spieler mit mindestens einem Versuch
aufnehmen, damit „nie geworfen" nicht wie „immer verworfen" erscheint.

Das System SHALL eigene und fremde Spieler gleichermaßen in die Ranglisten aufnehmen.

Das System SHALL die Ranglisten allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Torschützenliste
- **WHEN** ein eingeloggter Nutzer die Ranglisten einer Staffel abruft
- **THEN** antwortet der Server mit HTTP 200 und einer nach Toren absteigend sortierten Liste

#### Scenario: Spiele sind ausgewiesen
- **WHEN** ein Spieler in drei ausgewerteten Berichten in der Mannschaftsliste steht
- **THEN** weist seine Zeile drei Spiele aus

#### Scenario: Siebenmeter-Rangliste ist eigenständig sortiert
- **WHEN** die Siebenmeter-Rangliste einer Staffel abgerufen wird
- **THEN** ist sie nach getroffenen Siebenmetern absteigend sortiert und enthält je Spieler die Fehlversuche

#### Scenario: Spieler ohne Siebenmeter-Versuch fehlt
- **WHEN** ein Spieler in keinem Bericht einen Siebenmeter geworfen hat
- **THEN** erscheint er nicht in der Siebenmeter-Rangliste

#### Scenario: Fremde Spieler sind enthalten
- **WHEN** eine Rangliste gebildet wird
- **THEN** enthält sie auch Spieler von Mannschaften anderer Vereine

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** Ranglisten ohne gültiges Zugangstoken abgerufen werden
- **THEN** antwortet der Server mit HTTP 401
