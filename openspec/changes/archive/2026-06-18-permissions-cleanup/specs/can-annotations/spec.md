## ADDED Requirements

### Requirement: Listen- und Detail-DTOs tragen ein _can-Objekt

Das System SHALL in API-Responses für Resources ein `can`-Objekt mitliefern, das dem
aufrufenden Client angibt, welche Aktionen auf dieser Resource zulässig sind. Das Frontend
MUSS Button-Sichtbarkeit ausschließlich aus `can.*` ableiten und DARF NICHT `hasFunction`
oder `user.role`-Vergleiche für diesen Zweck verwenden.

#### Scenario: Vorstand erhält edit=true und delete=true für Members
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/members` aufruft
- **THEN** enthält jedes Member-Objekt `"can": { "edit": true, "delete": true }`

#### Scenario: Spieler erhält edit=false für fremdes Member
- **WHEN** ein User mit Vereinsfunktion `spieler` `GET /api/members/{id}` für ein fremdes Member aufruft
- **THEN** enthält die Response `"can": { "edit": false, "delete": false }`

#### Scenario: Eigentümer erhält edit=true für eigenes Profil
- **WHEN** ein Nutzer `GET /api/members/{id}` für sein eigenes Member aufruft
- **THEN** enthält die Response `"can": { "edit": true, "delete": false }`

---

### Requirement: _can-Felder sind additiv und rückwärtskompatibel

Das System SHALL `can`-Objekte additiv zu bestehenden Response-Feldern hinzufügen, ohne
bestehende Felder zu entfernen oder umzubenennen. Ein fehlendes `can`-Feld in einem älteren
Client MUSS nicht zu einem Fehler führen.

#### Scenario: Response-Struktur bleibt kompatibel
- **WHEN** ein Client `GET /api/members` aufruft und das `can`-Feld ignoriert
- **THEN** funktionieren alle bestehenden Felder unverändert

---

### Requirement: _can-Schema verwendet snake_case

Das System SHALL `can`-Objekte mit snake_case-Keys ausgeben, konsistent mit den übrigen
DTO-Feldern. Erlaubte Aktions-Keys pro Domäne:

- Members: `edit`, `delete`
- Games: `edit`, `delete`, `manage_lineup`
- Duties/Slots: `edit`, `delete`, `fulfill`
- Kader: `edit`, `delete`

#### Scenario: snake_case im JSON-Output
- **WHEN** ein Client `GET /api/members` aufruft
- **THEN** enthält jedes Item `"can": { "edit": true }` (snake_case, nicht camelCase)
