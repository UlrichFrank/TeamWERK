## ADDED Requirements

### Requirement: Draft-Erstellung durch den Slot-Owner ohne Rollen-Gate
Das System SHALL bei `POST /api/match-reports` mit Body `{ game_id, duty_slot_id }` einen
neuen Draft anlegen, wenn der authentifizierte User den referenzierten Duty-Slot besitzt
(`duty_slots.assigned_user_id = user.id`) und noch kein `match_report` für dieses Spiel
existiert. Ein Rollen-Gate SHALL es **nicht** mehr geben — der Besitz des
Spielbericht-Dienstes ist die Berechtigung. `admin` darf weiterhin für alle anlegen.
Response: HTTP 201 mit `{id}`. State-Initial: `draft`. `author_user_id = user.id`.

Dasselbe gilt für die übrigen Autoren-Routen (`GET /api/match-reports/my`,
`DELETE /api/match-reports/{id}`, `POST /api/match-reports/{id}/submit-for-review`): sie
liegen im Tier *Authenticated*; die fachliche Prüfung ist die Autorenschaft am Bericht.

#### Scenario: Standard-User mit eigenem Slot
- **WHEN** ein User mit Rolle `standard`, dem der Spielbericht-Slot gehört, `POST /api/match-reports` aufruft
- **THEN** liefert das System HTTP 201 und legt den Draft an

#### Scenario: Slot gehört anderem User
- **WHEN** ein User einen `duty_slot_id` referenziert, den er nicht besitzt
- **THEN** liefert das System HTTP 403 mit `{"error":"slot_not_owned"}`

#### Scenario: Zweiter Draft für dasselbe Spiel
- **WHEN** bereits ein `match_report` mit `game_id=X` existiert und ein weiterer Draft angelegt werden soll
- **THEN** liefert das System HTTP 409 mit `{"error":"report_exists"}`

## REMOVED Requirements

### Requirement: Draft-Erstellung durch Slot-Owner
**Reason**: Die Anforderung band die Draft-Erstellung an die System-Rolle `presseteam`, die
ersatzlos entfällt. Ersetzt durch „Draft-Erstellung durch den Slot-Owner ohne Rollen-Gate";
die einzige inhaltliche Änderung ist der Wegfall der Rollen-Bedingung.
**Migration**: Keine — der Besitz des Spielbericht-Dienstes war schon bisher die zweite,
fachlich tragende Bedingung und bleibt unverändert bestehen.
