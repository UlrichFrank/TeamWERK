## MODIFIED Requirements

### Requirement: Upload-Initialisierung mit Pre-Disk-Check

Nutzer mit Vereinsfunktion `trainer`, `sportliche_leitung`, `vorstand` oder Rolle `admin` SHALL via `POST /api/videos` einen Upload initialisieren können. Zusätzlich SHALL ein Nutzer, der für `game_id` eine Dienst-Zuweisung (`duty_assignments`, beliebiger `status`) auf einen Diensttyp mit `grants_video_upload = 1` hat, für genau dieses `game_id` einen Upload initialisieren können — unabhängig von Vereinsfunktion oder Rolle. Diese dienst-basierte Berechtigung SHALL an das konkrete `game_id` der Dienst-Zuweisung gebunden sein, nicht an das Team: sie gilt nicht für andere Spiele desselben Teams. Bei einer ausschließlich dienst-basierten Berechtigung MUST das angegebene `team_id` eine der am Spiel beteiligten Mannschaften sein; ein fremdes `team_id` MUST mit HTTP 400 abgewiesen werden, damit ein Dienst-Upload nicht bei einer unbeteiligten Mannschaft sichtbar wird. Die Anfrage MUST `title`, `team_id`, `season_id` und die erwartete `size_bytes` enthalten; optional `description`, `game_id`. Bei einer dienst-basierten Berechtigung MUST `game_id` gesetzt und die Dienst-Zuweisung dafür vorhanden sein. Der Server MUST vor der Annahme prüfen, dass im Storage-Verzeichnis mindestens `size_bytes × 2.5 + 2 GiB` frei sind.

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
- **WHEN** ein Nutzer ohne Trainer-/sportliche-Leitung-/Vorstand-/Admin-Berechtigung eine Dienst-Zuweisung auf einen als `grants_video_upload` markierten Diensttyp für `game_id = 42` hat und mit `game_id = 42` und einem `team_id` einer am Spiel beteiligten Mannschaft `POST /api/videos` aufruft
- **THEN** legt der Server eine DB-Zeile mit `status='uploading'` an und liefert `{ video_id, upload_url }` mit HTTP 201

#### Scenario: Dienst-basierter Upload an eine unbeteiligte Mannschaft
- **WHEN** derselbe Nutzer mit `game_id = 42`, aber einem `team_id`, das nicht zu den Mannschaften von Spiel 42 gehört, `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 400 ohne DB-Eintrag anzulegen

#### Scenario: Dienst-Zuweisung für ein anderes Spiel
- **WHEN** derselbe Nutzer mit `game_id = 99` (einem Spiel, für das er keine qualifizierende Dienst-Zuweisung hat) `POST /api/videos` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst-basierte Berechtigung ohne `game_id`
- **WHEN** ein Nutzer ohne Rollen-Berechtigung `POST /api/videos` ohne `game_id` aufruft, auch wenn er irgendeine qualifizierende Dienst-Zuweisung in der Saison hat
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Dienst auf nicht-markiertem Diensttyp
- **WHEN** ein Nutzer eine Dienst-Zuweisung für `game_id = 42` hat, deren Diensttyp `grants_video_upload = 0` trägt, und keine sonstige Berechtigung
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Eigene upload-berechtigte Spiele abrufen

Der Server SHALL einen authentifizierten Endpoint bereitstellen, der beschreibt, wofür der aufrufende Nutzer aktuell ein Video hochladen darf. Die Antwort MUST drei Mengen enthalten: (1) die Spiel-IDs (`game_ids`) — Spiele der Teams, für die eine Rollen-Berechtigung (Trainer des Teams, sportliche Leitung, Vorstand, Admin) besteht, vereinigt mit den Spielen, für die eine Dienst-Zuweisung auf einen `grants_video_upload`-Diensttyp existiert; (2) dieselben Spiele als Datensätze (`games`) mit Datum, Gegner, Saison und den IDs der beteiligten Mannschaften, damit ein Client sie anzeigen kann, ohne auf die allgemeine Termin-Sichtbarkeit angewiesen zu sein; (3) die Mannschaften (`teams`), für die ein Upload möglich ist — die Rollen-Teams mit Kader in der aktiven Saison vereinigt mit den Mannschaften der Dienst-Spiele —, jede mit dem Kennzeichen `upload_without_game`, das genau dann wahr ist, wenn der Nutzer für diese Mannschaft auch ohne Spielbezug hochladen darf (Rollen-Berechtigung). Diese Antwort SHALL ausschließlich der Client-seitigen Vorauswahl dienen — die tatsächliche Berechtigungsprüfung MUST weiterhin serverseitig bei `POST /api/videos` erfolgen, und die Antwort MUST NOT großzügiger sein als jene Prüfung.

#### Scenario: Nutzer mit reiner Dienst-Berechtigung
- **WHEN** ein Nutzer ohne Trainer-/sportliche-Leitung-/Vorstand-Funktion, aber mit einer Dienst-Zuweisung auf einen `grants_video_upload`-Diensttyp für Spiel `42` von Mannschaft A, den Endpoint aufruft
- **THEN** enthält `game_ids` Spiel `42` und keine anderen Spiele, `games` den Datensatz zu Spiel `42` mit Mannschaft A unter den Team-IDs, und `teams` genau Mannschaft A mit `upload_without_game = false` — auch wenn der Nutzer mit Mannschaft A sonst in keiner Weise verbunden ist

#### Scenario: Trainer sieht alle Spiele seines Teams
- **WHEN** ein Trainer von Team A den Endpoint aufruft
- **THEN** enthält `game_ids` alle Spiele von Team A, unabhängig von eigenen Dienst-Zuweisungen, und `teams` enthält Team A mit `upload_without_game = true`

#### Scenario: Trainer mit zusätzlichem Dienst bei einer fremden Mannschaft
- **WHEN** ein Trainer von Team A zusätzlich eine qualifizierende Dienst-Zuweisung für ein Spiel von Team B hat und den Endpoint aufruft
- **THEN** enthält `teams` Team A mit `upload_without_game = true` und Team B mit `upload_without_game = false`, und `games` enthält das Dienst-Spiel von Team B

#### Scenario: Vorstand oder Admin
- **WHEN** ein Nutzer mit Rolle `admin` oder Vereinsfunktion `vorstand`/`sportliche_leitung` den Endpoint aufruft
- **THEN** enthält `teams` alle Mannschaften mit Kader in der aktiven Saison, jede mit `upload_without_game = true`

#### Scenario: Nutzer ohne jede Berechtigung
- **WHEN** ein Nutzer ohne Rollen-Berechtigung und ohne qualifizierende Dienst-Zuweisung den Endpoint aufruft
- **THEN** liefert die Antwort drei leere Listen mit HTTP 200 (kein 403 — Abrufen der eigenen leeren Berechtigungsmenge ist kein Fehler)
