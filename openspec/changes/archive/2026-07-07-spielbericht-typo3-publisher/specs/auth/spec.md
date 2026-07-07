## ADDED Requirements

### Requirement: users.role akzeptiert `presseteam`
Das System SHALL im `users.role`-CHECK-Constraint die Werte `admin`, `standard` und `presseteam` akzeptieren. Die Rolle ist hierarchisch: `admin ⊇ presseteam ⊇ standard`. Ein Presseteam-User kann alles, was ein Standard-User kann, plus die auf Presseteam eingeschränkten Match-Report-Aktionen. Ein Admin kann alles.

#### Scenario: Migration akzeptiert neuen Wert
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'presseteam')` ausgeführt wird
- **THEN** akzeptiert die Datenbank die Zeile

#### Scenario: Alter Wert weiterhin gültig
- **WHEN** eine Zeile mit `role='standard'` oder `role='admin'` besteht
- **THEN** bleibt sie unverändert und funktionsfähig

#### Scenario: Unzulässiger Wert
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'foo')` ausgeführt wird
- **THEN** lehnt der CHECK-Constraint mit Fehler ab

### Requirement: RequireRole akzeptiert Rollen-Liste
Das System SHALL die Middleware `auth.RequireRole(rollen...)` mit variabler Anzahl Rollen-Argumente erlauben. Ein Request mit `role IN rollen` läuft durch. Rolle `admin` fällt hierarchisch überall durch, wenn die Guard-Signatur `RequireRole("presseteam","admin")` lautet — Admin ist immer eine explizit erlaubte Alternative.

#### Scenario: Presseteam-User an Presseteam-Guard
- **WHEN** ein User mit `role='presseteam'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Admin an Presseteam-Guard
- **WHEN** ein User mit `role='admin'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Standard-User an Presseteam-Guard
- **WHEN** ein User mit `role='standard'` eine Route hinter `RequireRole("presseteam","admin")` aufruft
- **THEN** liefert das System HTTP 403
