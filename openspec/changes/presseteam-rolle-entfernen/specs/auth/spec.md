## ADDED Requirements

### Requirement: users.role akzeptiert ausschließlich `admin` und `standard`
Das System SHALL im `users.role`- und im `invitation_tokens.role`-CHECK-Constraint nur die
Werte `admin` und `standard` akzeptieren. Die Rolle bleibt hierarchisch: `admin ⊇ standard`.
Bestandszeilen mit `role='presseteam'` SHALL die Migration auf `standard` überführen — die
Rolle erlaubte nichts, was ein Standard-Nutzer nicht künftig ebenfalls darf.

`PUT /api/users/{id}/role` und die Einladung (`POST /api/invitations`) SHALL jeden anderen
Wert — `presseteam` eingeschlossen — mit HTTP 400 `"invalid role"` ablehnen, ohne die Rolle
zu ändern.

#### Scenario: Bestands-Presseteam wird Standard
- **WHEN** die Migration auf einer Datenbank mit einer Zeile `users.role='presseteam'` läuft
- **THEN** trägt die Zeile danach `role='standard'`
- **AND** enthält weder `users` noch `invitation_tokens` danach eine Zeile mit `role='presseteam'`

#### Scenario: presseteam wird vom CHECK abgelehnt
- **WHEN** `INSERT INTO users (…, role) VALUES (…, 'presseteam')` ausgeführt wird
- **THEN** lehnt der CHECK-Constraint mit Fehler ab

#### Scenario: Rollenvergabe lehnt presseteam ab
- **WHEN** ein Admin `PUT /api/users/{id}/role` mit Body `{"role":"presseteam"}` sendet
- **THEN** liefert das System HTTP 400 mit Body `"invalid role"` und ändert `users.role` nicht

#### Scenario: Bestehende Werte bleiben gültig
- **WHEN** eine Zeile mit `role='standard'` oder `role='admin'` besteht
- **THEN** bleibt sie unverändert und funktionsfähig

### Requirement: RequireRole gated ohne impliziten Admin-Bypass
Das System SHALL die Middleware `auth.RequireRole(rollen...)` mit variabler Anzahl
Rollen-Argumente erlauben. Ein Request mit `role IN rollen` läuft durch. Es gibt **keinen**
impliziten Admin-Bypass: `admin` kommt nur durch, wo die Guard-Signatur ihn explizit
aufführt — bei einem reinen `RequireRole("admin")` ist er die einzige erlaubte Rolle.

#### Scenario: Admin an Admin-Guard
- **WHEN** ein User mit `role='admin'` eine Route hinter `RequireRole("admin")` aufruft
- **THEN** wird der Request durchgelassen

#### Scenario: Standard-User an Admin-Guard
- **WHEN** ein User mit `role='standard'` eine Route hinter `RequireRole("admin")` aufruft
- **THEN** liefert das System HTTP 403

## REMOVED Requirements

### Requirement: users.role akzeptiert `presseteam`
**Reason**: Die Rolle trug genau eine Unterscheidung — wer einen Spielbericht schreiben darf
— und die ist über den Besitz des Spielbericht-Dienstes bereits abgedeckt. Sie entfällt
ersatzlos; alles, was sie erlaubte, darf künftig jeder eingeloggte Nutzer.
**Migration**: Bestandszeilen mit `role='presseteam'` werden von Migration `056` auf
`standard` gesetzt; der CHECK-Constraint kennt den Wert danach nicht mehr.

### Requirement: RequireRole akzeptiert Rollen-Liste
**Reason**: Die Anforderung war vollständig am Beispiel `RequireRole("presseteam","admin")`
formuliert; mit dem Wegfall der Rolle beschreiben ihre Szenarien eine Guard-Signatur, die es
nicht mehr gibt. Ersetzt durch „RequireRole gated ohne impliziten Admin-Bypass" — die
Mechanik der Middleware ist unverändert.
**Migration**: Keine. Kein Verhalten der Middleware ändert sich; entfallen ist nur die
Rolle, die in den Beispielen stand.
