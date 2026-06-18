## ADDED Requirements

### Requirement: Policy-Package konsolidiert alle Berechtigungs-Predicates

Das System SHALL ein Package `internal/policy/` bereitstellen, das alle
Berechtigungs-Predicates enthält. Handler MÜSSEN ihre Inline-`claims.Role`/`claims.HasFunction`-
Checks durch Aufrufe in dieses Package ersetzen.

#### Scenario: Admin umgeht alle ClubFunction-Checks
- **WHEN** ein User mit `claims.Role == "admin"` eine beliebige Policy-Funktion aufruft
- **THEN** gibt die Funktion `true` zurück ohne auf ClubFunctions zu prüfen

#### Scenario: Trainer-ähnliche Personas werden einheitlich geprüft
- **WHEN** ein Handler `policy.IsTrainerLike(claims)` aufruft
- **THEN** liefert die Funktion `true` für Nutzer mit ClubFunction `trainer` ODER `sportliche_leitung`

#### Scenario: Keine direkten HasFunction-Aufrufe in Handlern
- **WHEN** der Code nach `claims.HasFunction(` oder `claims.HasAnyFunction(` gesucht wird
- **THEN** sind keine Treffer mehr in `internal/*/handler*.go` zu finden (nach vollständiger Migration)

---

### Requirement: ScopeMembersQuery liefert personas-gerechtes SQL-WHERE-Fragment

Das System SHALL eine Funktion `policy.ScopeMembersQuery(claims)` bereitstellen, die ein
SQL-WHERE-Fragment zurückgibt, das Members-Abfragen auf das für den Nutzer sichtbare Set
einschränkt.

#### Scenario: Vorstand sieht alle Members
- **WHEN** `policy.ScopeMembersQuery` mit Vorstand-Claims aufgerufen wird
- **THEN** gibt die Funktion `"1=1"` (keine Einschränkung) zurück

#### Scenario: Trainer sieht nur Team-Members
- **WHEN** `policy.ScopeMembersQuery` mit Trainer-Claims (teamID = 5) aufgerufen wird
- **THEN** enthält das zurückgegebene WHERE-Fragment eine Einschränkung auf team_id = 5

#### Scenario: Admin sieht alle Members
- **WHEN** `policy.ScopeMembersQuery` mit Admin-Claims aufgerufen wird
- **THEN** gibt die Funktion `"1=1"` zurück

---

### Requirement: Folder-ACL-Checks nutzen data-driven Policy

Das System SHALL eine Funktion `policy.CanReadFolder(ctx, db, claims, folderID)` bereitstellen,
die Folder-Leserechte via DB-JOIN auf `folder_permissions` und `member_club_functions` prüft.

#### Scenario: Vorstand darf alle Folder lesen
- **WHEN** `policy.CanReadFolder` mit Vorstand-Claims aufgerufen wird
- **THEN** gibt die Funktion `true` zurück ohne DB-Lookup

#### Scenario: Spieler ohne ACL-Eintrag darf Folder nicht lesen
- **WHEN** `policy.CanReadFolder` mit Spieler-Claims für einen Folder ohne passenden ACL-Eintrag aufgerufen wird
- **THEN** gibt die Funktion `false` zurück

#### Scenario: Spieler mit ACL-Eintrag darf Folder lesen
- **WHEN** `policy.CanReadFolder` mit Spieler-Claims für einen Folder aufgerufen wird und die DB einen passenden `folder_permissions`-Eintrag enthält
- **THEN** gibt die Funktion `true` zurück
