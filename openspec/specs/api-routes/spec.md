# api-routes Specification

## Purpose

Diese Spezifikation beschreibt die Capability `api-routes`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Teams-Endpoint ist rollenabhängig

`GET /api/teams` SHALL die Mannschaften liefern, die der aufrufende Nutzer im
Mannschafts-Filter auswählen können muss. Die Antwort ist die **Vereinigung** aller Wege, auf
denen er an einer Mannschaft hängt — nie die Menge eines einzelnen Wegs:

- `admin` oder Vereinsfunktion `vorstand`: alle Teams inklusive inaktiver, ohne
  Kader-Filterung.
- Vereinsfunktion `sportliche_leitung`: alle Teams mit aktivem Kader.
- Alle übrigen Nutzer: alle Teams mit aktivem Kader, an denen sie über **mindestens einen**
  der folgenden Wege hängen — eigener Stammkader, eigener Trainer-Eintrag, eigener
  erweiterter Kader, Stammkader eines Kindes (`family_links`), erweiterter Kader eines
  Kindes. Das ist die Menge des Views `user_accessible_teams`.

Eine Vereinsfunktion SHALL die Zugehörigkeiten eines Nutzers **erweitern, nie ersetzen**: wer
Trainer ist und zugleich Elternteil, erhält sein Trainer-Team **und** die Mannschaften seiner
Kinder.

Der Query-Parameter `?scope` engt die Antwort für bestimmte Aufrufer ein:

- `?scope=duties` (Dienstbörse): Stammkader von Nutzer und Kindern, vereinigt mit den
  Trainer-Teams des Nutzers. Der erweiterte Kader entfällt — er schuldet keine Dienststunden,
  eine Filter-Option dafür bliebe immer leer.
- `?scope=attendance-stats` und `?scope=diary-stats`: nur die eigenen Trainer-Teams, für
  Nutzer ohne `admin`/`sportliche_leitung` (bei `diary-stats` zusätzlich ohne `vorstand`).
  Diese beiden Scopes speisen eine **Navigation** statt einer Filterung: die Statistikseiten
  springen auf das erste zurückgegebene Team, dessen Abruf danach gegen
  `attendance.canSeeTeamStats` bzw. `trainingdiary.canSeeTeamDiary` läuft. Eine weitere
  Menge führte dort zu einem Selektor, dessen Auswahl mit HTTP 403 endet.

#### Scenario: Vorstand ruft Teams ab

- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/teams` aufruft
- **THEN** liefert der Endpoint alle Teams inklusive inaktiver zurück

#### Scenario: Trainer ruft Teams ab

- **WHEN** ein User mit Vereinsfunktion `trainer` ohne weitere Zugehörigkeiten
  `GET /api/teams` aufruft
- **THEN** liefert der Endpoint nur Teams zurück, in deren Kader der aktiven Saison er als
  Trainer eingetragen ist

#### Scenario: Trainer, der zugleich Elternteil ist

- **WHEN** ein User mit Vereinsfunktion `trainer` eine Mannschaft trainiert und über
  `family_links` Kinder im Stammkader einer zweiten und im erweiterten Kader einer dritten
  Mannschaft hat und `GET /api/teams` aufruft
- **THEN** enthält die Antwort alle drei Mannschaften

#### Scenario: Derselbe Nutzer ruft den Dienst-Filter ab

- **WHEN** derselbe User `GET /api/teams?scope=duties` aufruft
- **THEN** enthält die Antwort seine Trainer-Mannschaft und die Stammkader-Mannschaft seines
  Kindes
- **AND** enthält sie nicht die Mannschaft, in der das Kind nur im erweiterten Kader steht

#### Scenario: Spieler ruft Teams ab

- **WHEN** ein User ohne Vereinsfunktion `GET /api/teams` aufruft
- **THEN** liefert der Endpoint die Teams mit aktivem Kader, an denen er oder eines seiner
  Kinder über Stamm- oder erweiterten Kader hängt

#### Scenario: Teamselektor der Anwesenheitsstatistik

- **WHEN** ein User mit Vereinsfunktion `trainer`, der zugleich Elternteil ist,
  `GET /api/teams?scope=attendance-stats` aufruft
- **THEN** liefert der Endpoint nur seine Trainer-Teams

### Requirement: Keine /admin-Präfix-Routen

Die API SHALL keine Routen unter `/api/` mehr exponieren. Alle bisherigen `/api/*`-Routen MÜSSEN unter ihrem kanonischen Ressource-Pfad `/api/{ressource}` erreichbar sein. Zugriffskontrolle erfolgt ausschließlich über Middleware-Gruppen.

#### Scenario: Vorstand ruft Vereinskonfiguration ab
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/club` aufruft
- **THEN** liefert der Endpoint die Vereinskonfiguration zurück (bisher: `GET /api/club`)

#### Scenario: Unautorisierter Zugriff auf privilegierte Route
- **WHEN** ein User ohne ausreichende Berechtigung eine privilegierte Route aufruft (z.B. `GET /api/club`)
- **THEN** antwortet der Server mit HTTP 403
