## MODIFIED Requirements

### Requirement: Upload-Initialisierung mit Pre-Disk-Check

Nutzer mit Vereinsfunktion `trainer`, `sportliche_leitung`, `vorstand` oder Rolle `admin` SHALL via `POST /api/videos` einen Upload initialisieren können. Zusätzlich SHALL ein Nutzer, der für `game_id` eine Dienst-Zuweisung (`duty_assignments`, beliebiger `status`) auf einen Diensttyp mit `grants_video_upload = 1` hat, für genau dieses `game_id` einen Upload initialisieren können — unabhängig von Vereinsfunktion oder Rolle. Diese dienst-basierte Berechtigung SHALL an das konkrete `game_id` der Dienst-Zuweisung gebunden sein, nicht an das Team: sie gilt nicht für andere Spiele desselben Teams. Die Anfrage MUST `title`, `team_id`, `season_id` und die erwartete `size_bytes` enthalten; optional `description`, `game_id`. Bei einer dienst-basierten Berechtigung MUST `game_id` gesetzt und die Dienst-Zuweisung dafür vorhanden sein. Der Server MUST vor der Annahme prüfen, dass im Storage-Verzeichnis mindestens `size_bytes × 2.5 + 2 GiB` frei sind.

#### Scenario: Erfolgreiche Initialisierung
- **WHEN** ein Trainer mit ausreichender Berechtigung für `team_id` und ausreichend freiem Speicher `POST /api/videos` aufruft
- **THEN** legt der Server eine DB-Zeile mit `status='uploading'` an und liefert `{ video_id, upload_url }` mit HTTP 201

#### Scenario: Unzureichender Speicher
- **WHEN** die geforderte Größe das freie Speicherbudget übersteigt
- **THEN** antwortet der Server mit HTTP 507 ohne DB-Eintrag anzulegen

#### Scenario: Fehlende Upload-Berechtigung
- **WHEN** ein Nutzer ohne Trainer-/Vorstand-/Admin-Rolle und ohne qualifizierende Dienst-Zuweisung für das angegebene `game_id` `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Trainer fremdes Team
- **WHEN** ein Trainer für ein `team_id` hochlädt, in dem er nicht Trainer ist und nicht admin/vorstand ist und keine qualifizierende Dienst-Zuweisung für das angegebene `game_id` hat
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst-basierter Upload für das zugewiesene Spiel
- **WHEN** ein Nutzer ohne Trainer-/sportliche-Leitung-/Vorstand-/Admin-Berechtigung eine Dienst-Zuweisung auf einen als `grants_video_upload` markierten Diensttyp für `game_id = 42` hat und mit `game_id = 42` `POST /api/videos` aufruft
- **THEN** legt der Server eine DB-Zeile mit `status='uploading'` an und liefert `{ video_id, upload_url }` mit HTTP 201

#### Scenario: Dienst-Zuweisung für ein anderes Spiel
- **WHEN** derselbe Nutzer mit `game_id = 99` (einem Spiel, für das er keine qualifizierende Dienst-Zuweisung hat) `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst-basierte Berechtigung ohne `game_id`
- **WHEN** ein Nutzer ohne Rollen-Berechtigung `POST /api/videos` ohne `game_id` aufruft, auch wenn er irgendeine qualifizierende Dienst-Zuweisung in der Saison hat
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst auf nicht-markiertem Diensttyp
- **WHEN** ein Nutzer eine Dienst-Zuweisung für `game_id = 42` hat, deren Diensttyp `grants_video_upload = 0` trägt, und keine sonstige Berechtigung
- **THEN** antwortet der Server mit HTTP 403

## ADDED Requirements

### Requirement: Eigene upload-berechtigte Spiele abrufen

Der Server SHALL einen authentifizierten Endpoint bereitstellen, der die Menge der Spiele liefert, für die der aufrufende Nutzer aktuell ein Video hochladen darf: Spiele der Teams, für die eine Rollen-Berechtigung (Trainer des Teams, sportliche Leitung, Vorstand, Admin) besteht, vereinigt mit den Spielen, für die eine Dienst-Zuweisung auf einen `grants_video_upload`-Diensttyp existiert. Diese Liste SHALL ausschließlich der Client-seitigen Vorauswahl dienen — die tatsächliche Berechtigungsprüfung MUST weiterhin serverseitig bei `POST /api/videos` erfolgen.

#### Scenario: Nutzer mit reiner Dienst-Berechtigung
- **WHEN** ein Nutzer ohne Trainer-/sportliche-Leitung-/Vorstand-Funktion, aber mit einer Dienst-Zuweisung auf einen `grants_video_upload`-Diensttyp für Spiel `42`, den Endpoint aufruft
- **THEN** enthält die Antwort Spiel `42` und keine anderen Spiele, für die keine Berechtigung besteht

#### Scenario: Trainer sieht alle Spiele seines Teams
- **WHEN** ein Trainer von Team A den Endpoint aufruft
- **THEN** enthält die Antwort alle Spiele von Team A, unabhängig von eigenen Dienst-Zuweisungen

#### Scenario: Nutzer ohne jede Berechtigung
- **WHEN** ein Nutzer ohne Rollen-Berechtigung und ohne qualifizierende Dienst-Zuweisung den Endpoint aufruft
- **THEN** liefert die Antwort eine leere Liste mit HTTP 200 (kein 403 — Abrufen der eigenen leeren Berechtigungsmenge ist kein Fehler)
