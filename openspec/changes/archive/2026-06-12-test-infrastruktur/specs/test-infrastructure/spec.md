## ADDED Requirements

### Requirement: Migrations-FS ist aus Tests zugänglich

Die Datenbankmigrationen SHALL als exportiertes `db.FS embed.FS` in `internal/db/migrations.go` verfügbar sein, damit Test-Helpers die vollständige Migrationskette ausführen können ohne den `cmd/teamwerk`-Package zu importieren.

#### Scenario: testDB läuft alle Migrations durch

- **WHEN** `testutil.NewDB(t)` aufgerufen wird
- **THEN** wird eine SQLite-In-Memory-Datenbank mit vollständig angewendetem Schema zurückgegeben (alle Migrations von 001 bis aktuell)

#### Scenario: main.go kompiliert nach dem Refactor weiterhin

- **WHEN** `db.FS` statt des inline-embeds in `main.go` verwendet wird
- **THEN** kompiliert `go build ./cmd/teamwerk` ohne Fehler und das Verhalten der Anwendung ändert sich nicht

---

### Requirement: testutil.NewDB liefert isolierte In-Memory-Datenbank

`testutil.NewDB(t)` SHALL eine frische SQLite-In-Memory-Datenbank mit angewendeten Migrations zurückgeben. Jeder Test-Aufruf MUSS eine vollständig isolierte DB-Instanz erhalten (kein geteilter State zwischen Tests).

#### Scenario: Zwei Tests laufen ohne gegenseitige Beeinflussung

- **WHEN** zwei Tests jeweils `testutil.NewDB(t)` aufrufen und beide Daten schreiben
- **THEN** sieht kein Test die Daten des anderen

#### Scenario: DB wird nach dem Test automatisch freigegeben

- **WHEN** ein Test mit `testutil.NewDB(t)` endet
- **THEN** wird die DB-Verbindung via `t.Cleanup` geschlossen (kein Leak)

---

### Requirement: testutil.NewServer baut einen partiellen Chi-Router

`testutil.NewServer(t, db, routes)` SHALL einen `*httptest.Server` zurückgeben, der nur die übergebenen Routen registriert und die Auth-Middleware (`auth.Middleware`) korrekt eingebunden hat.

#### Scenario: Unauthentifizierter Request wird abgelehnt

- **WHEN** ein Request ohne `Authorization`-Header an eine geschützte Route gesendet wird
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Authentifizierter Request mit gültigem Token wird durchgelassen

- **WHEN** ein Request mit einem via `testutil.Token()` erzeugten Bearer-Token gesendet wird
- **THEN** erreicht der Request den Handler und gibt keinen 401 zurück

---

### Requirement: testutil.Token erzeugt signierte JWT-Tokens für beliebige Rollen

`testutil.Token(userID, role, clubFunctions)` SHALL einen gültigen JWT-String zurückgeben, der von der `auth.Middleware` des Testservers akzeptiert wird.

#### Scenario: Token für Trainer-Rolle

- **WHEN** `testutil.Token(42, "standard", []string{"trainer"})` aufgerufen wird
- **THEN** enthält der resultierende JWT `uid: 42` und `club_functions: ["trainer"]`

---

### Requirement: Fixture-Helpers erstellen minimale Test-Datensätze

`testutil` SHALL Helper-Funktionen für häufig benötigte Fixtures bereitstellen: `CreateUser`, `CreateTeam`, `CreateSeason`, `CreateTrainingSeries`, `CreateTrainingSession`.

Jede Funktion MUSS via `t.Fatal` abbrechen wenn das Einfügen fehlschlägt, und die erzeugte Entität zurückgeben.

#### Scenario: CreateUser legt einen User mit Passwort-Hash an

- **WHEN** `testutil.CreateUser(t, db, "standard", teamID)` aufgerufen wird
- **THEN** existiert ein User-Datensatz in der DB mit bcrypt-gehashetem Passwort und der angegebenen Rolle

#### Scenario: CreateSeason erstellt eine aktive Saison

- **WHEN** `testutil.CreateSeason(t, db, "2025/26")` aufgerufen wird
- **THEN** existiert ein Saison-Datensatz mit `is_active = 1`
