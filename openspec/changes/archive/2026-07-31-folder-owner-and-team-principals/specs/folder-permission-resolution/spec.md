## ADDED Requirements

### Requirement: Eigentümer-Vorrang vor der Pfad-Auflösung

Das System SHALL einem Nutzer `can_read` **und** `can_write` auf einen Ordner gewähren, wenn seine
`user_id` in `file_folders.created_by` des Ordners **oder eines beliebigen Vorfahren** im Pfad zur
Wurzel steht. Diese Prüfung MUSS **vor** der Nearest-Ancestor-Auflösung erfolgen und diese
kurzschließen; sie ist unabhängig davon, welche Berechtigungseinträge auf dem Ordner oder seinen
Vorfahren existieren.

Das Recht ist absolut und unbefristet: Es besteht unabhängig von System-Rolle, Vereinsfunktionen
und ACL-Einträgen fort und endet nur, wenn `file_folders.created_by` geändert wird.

Da `file_folders.created_by` `NOT NULL` ist und für jeden bestehenden Ordner gefüllt, wirkt diese
Regel ohne Datenmigration rückwirkend auf alle Bestandsordner.

#### Scenario: Ersteller vergibt Rechte an eine andere Person und behält seine eigenen
- **WHEN** Nutzer F einen Unterordner anlegt (dessen Elternordner ihm `can_write` gewährt), auf
  diesem Ordner keine weiteren Berechtigungseinträge existieren, und er anschließend
  `POST /api/folders/{id}/permissions` mit `principal_type=user, principal_ref=<B>, can_read=1,
  can_write=1` aufruft
- **THEN** liefert die Auflösung für Nutzer F weiterhin `can_read=true, can_write=true`
- **THEN** erscheint der Ordner weiterhin in `GET /api/folders/{parent}/contents` für Nutzer F
- **THEN** darf Nutzer F `GET /api/folders/{id}/permissions` weiterhin aufrufen (200)

#### Scenario: Ersteller ohne jeden passenden ACL-Eintrag behält Vollzugriff
- **WHEN** der Ordner ausschließlich Berechtigungseinträge besitzt, die auf den Ersteller nicht
  zutreffen (weder `everyone`, noch seine Rolle, Vereinsfunktion oder User-ID)
- **THEN** liefert die Auflösung für den Ersteller `can_read=true, can_write=true`

#### Scenario: Eigentümerrecht gilt im gesamten Unterbaum
- **WHEN** Nutzer F Ordner X angelegt hat und Nutzer B darin den Unterordner X/Y anlegt, dessen
  einziger Berechtigungseintrag `user=<B>` lautet
- **THEN** liefert die Auflösung für Nutzer F auch auf X/Y `can_read=true, can_write=true`

#### Scenario: Eigentümerrecht überlebt den Verlust der Vereinsfunktion
- **WHEN** Nutzer F einen Ordner angelegt hat, während er die Vereinsfunktion `trainer` besaß, und
  ihm diese Funktion später entzogen wird
- **THEN** liefert die Auflösung für Nutzer F auf diesem Ordner weiterhin
  `can_read=true, can_write=true`

#### Scenario: Fremder Nutzer erhält kein Eigentümerrecht
- **WHEN** Nutzer G weder Ersteller des Ordners noch eines Vorfahren ist und kein
  Berechtigungseintrag auf ihn zutrifft
- **THEN** liefert die Auflösung `can_read=false, can_write=false`

#### Scenario: Anti-Eskalation erlaubt dem Eigentümer das Vergeben von Rechten
- **WHEN** der Ersteller eines Ordners `POST /api/folders/{id}/permissions` mit `can_write=1`
  aufruft, obwohl kein ACL-Eintrag ihm selbst Rechte gewährt
- **THEN** wird der Eintrag gespeichert (201), weil die Eskalationsprüfung auf derselben
  Auflösung basiert

### Requirement: Auflösung der Principal-Typen `team` und `team_parents`

Das System SHALL die Principal-Typen `team` und `team_parents` unterstützen. `principal_ref` ist in
beiden Fällen eine `teams.id` als Text. Die Zugehörigkeit MUSS **bei jedem Zugriff** gegen den
Kader der **aktiven Saison** (`seasons.is_active = 1`) aufgelöst werden; es darf keine
Personenliste zum Zeitpunkt der Rechtevergabe eingefroren werden.

- `team` matcht einen Nutzer, wenn sein Mitglieds-Datensatz in der aktiven Saison in einem Kader
  dieser Mannschaft als Spieler (`kader_members`), als erweiterter Kader
  (`kader_extended_members`) **oder** als Trainer (`kader_trainers`) geführt wird.
- `team_parents` matcht einen Nutzer, wenn er über `family_links` mit einem Mitglied verknüpft ist,
  das in der aktiven Saison in einem Kader dieser Mannschaft als Spieler oder erweiterter Kader
  geführt wird. Trainer-Verknüpfungen zählen für `team_parents` nicht.

Existiert keine aktive Saison, matchen `team`- und `team_parents`-Einträge niemanden
(Fail-Closed). Kader ohne `team_id` (Trainingsgruppen) sind über diese Typen nicht adressierbar.

#### Scenario: Spieler der Mannschaft erhält Zugriff
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1` hat und Nutzer S in der aktiven Saison
  über `kader_members` im Kader der Mannschaft T geführt wird
- **THEN** darf Nutzer S den Ordner lesen (200)

#### Scenario: Trainer der Mannschaft erhält Zugriff über den Team-Eintrag
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1, can_write=1` hat und Nutzer C in der
  aktiven Saison über `kader_trainers` im Kader der Mannschaft T geführt wird
- **THEN** darf Nutzer C den Ordner lesen und beschreiben

#### Scenario: Erweiterter Kader zählt als Team-Mitglied
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1` hat und Nutzer E in der aktiven Saison
  über `kader_extended_members` im Kader der Mannschaft T geführt wird
- **THEN** darf Nutzer E den Ordner lesen (200)

#### Scenario: Elternteil erhält keinen Zugriff über den Team-Eintrag
- **WHEN** ein Ordner ausschließlich den Eintrag `team=<T>: can_read=1` hat und Nutzer P lediglich
  über `family_links` mit einem Spieler der Mannschaft T verknüpft ist
- **THEN** erhält Nutzer P HTTP 403

#### Scenario: Elternteil erhält Zugriff über den Eltern-Eintrag
- **WHEN** ein Ordner den Eintrag `team_parents=<T>: can_read=1` hat und Nutzer P über
  `family_links` mit einem Mitglied verknüpft ist, das in der aktiven Saison im Kader der
  Mannschaft T geführt wird
- **THEN** darf Nutzer P den Ordner lesen (200)

#### Scenario: Spieler erhält keinen Zugriff über den Eltern-Eintrag
- **WHEN** ein Ordner ausschließlich den Eintrag `team_parents=<T>: can_read=1` hat und Nutzer S
  Spieler der Mannschaft T ist, ohne selbst Elternteil eines Kadermitglieds zu sein
- **THEN** erhält Nutzer S HTTP 403

#### Scenario: Mitglied einer anderen Mannschaft erhält keinen Zugriff
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1` hat und Nutzer S ausschließlich im Kader
  der Mannschaft U geführt wird
- **THEN** erhält Nutzer S HTTP 403

#### Scenario: Kaderwechsel wirkt ohne Änderung der Berechtigung
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1` hat und Nutzer S nachträglich aus dem
  Kader der Mannschaft T entfernt wird
- **THEN** erhält Nutzer S beim nächsten Zugriff HTTP 403, ohne dass der Berechtigungseintrag
  geändert wurde

#### Scenario: Kader einer inaktiven Saison zählt nicht
- **WHEN** ein Ordner den Eintrag `team=<T>: can_read=1` hat und Nutzer S nur im Kader der
  Mannschaft T einer **nicht** aktiven Saison geführt wird
- **THEN** erhält Nutzer S HTTP 403

### Requirement: Einheitliche Auflösungsfunktion

Das System SHALL die Ordner-Rechteauflösung an genau einer Stelle implementieren
(`policy.FolderAccess`). Alle Routen unter `/api/folders/*` und `/api/files/*` MÜSSEN diese
Funktion verwenden; eine zweite, parallele Implementierung darf nicht existieren.

#### Scenario: Alle Ordner-Routen liefern dieselbe Entscheidung
- **WHEN** derselbe Nutzer denselben Ordner über `GET /api/folders/{id}/contents` und über
  `GET /api/folders` (Wurzelliste) bzw. eine Mutations-Route erreicht
- **THEN** ist die zugrundeliegende Lese-/Schreib-Entscheidung in allen Fällen identisch

## Test-Anforderungen

**Eigentümer-Vorrang** (`internal/policy/folders_test.go`, `internal/files/handler_test.go`)

- `TestFolderAccess_OwnerKeepsRightsAfterGrantingToOthers` — Ordner mit einziger `user`-Zeile für
  Dritte → Ersteller `can_read=true, can_write=true`. Garantierte Invariante: eine ACL-Zeile für
  Dritte entzieht dem Ersteller nichts.
- `TestFolderAccess_OwnerRightsSpanSubtree` — Ersteller von X hat Vollzugriff auf X/Y, das ein
  anderer angelegt und nur für sich freigegeben hat.
- `TestFolderAccess_OwnerWithoutClubFunction` — Ersteller ohne passende Vereinsfunktion oder Rolle
  → Vollzugriff.
- `TestFolderAccess_NonOwnerNoAccess` — Nicht-Ersteller ohne passenden Eintrag → `false, false`
  (Absicherung, dass der Kurzschluss nicht zu breit greift).
- Route `GET /api/folders/{id}/permissions`: `TestListPermissions_OwnerNotLockedOut` → 200 für den
  Ersteller, obwohl kein ACL-Eintrag auf ihn zutrifft.
- Route `POST /api/folders/{id}/permissions`: `TestAddPermission_OwnerMayGrant` → 201.

**Team-Principals** (`internal/policy/folders_test.go`, `internal/files/handler_test.go`)

- `TestFolderAccess_TeamPlayerMatches` — `kader_members` in aktiver Saison → Zugriff.
- `TestFolderAccess_TeamTrainerMatches` — `kader_trainers` in aktiver Saison → Zugriff.
- `TestFolderAccess_TeamExtendedMemberMatches` — `kader_extended_members` → Zugriff.
- `TestFolderAccess_TeamParentNotMatchedByTeam` — Elternteil gegen `team`-Zeile → kein Zugriff.
- `TestFolderAccess_TeamParentsMatches` — Elternteil gegen `team_parents`-Zeile → Zugriff.
- `TestFolderAccess_TeamParentsDoesNotMatchPlayer` — Spieler gegen `team_parents`-Zeile → kein
  Zugriff.
- `TestFolderAccess_TeamOtherTeamNoAccess` — Mitglied einer anderen Mannschaft → kein Zugriff.
- `TestFolderAccess_TeamInactiveSeasonNoAccess` — Kader nur in inaktiver Saison → kein Zugriff.
- `TestFolderAccess_TeamNoActiveSeasonFailsClosed` — keine aktive Saison → kein Zugriff.
- Route `GET /api/folders/{id}/contents`: `TestFolderContents_TeamPermission` → 200 für
  Kadermitglied, 403 für Nutzer einer anderen Mannschaft.
- Route `POST /api/folders/{id}/permissions`: `TestAddPermission_TeamType` → 201 mit gespeicherter
  Zeile; `TestAddPermission_OwnerPseudoTypeRejected` (`principal_type=owner`) → 400.

**Nicht-Regression**

- Die bestehenden Szenarien der Requirements „Nearest-Ancestor-Wins Berechtigungsauflösung" und
  „Family-Context" MÜSSEN unverändert grün bleiben — insbesondere
  `TestResolveAccess_NearestAncestorWins` mit einem Ordner, dessen Ersteller **nicht** der
  anfragende Nutzer ist (sonst maskiert der Eigentümer-Vorrang die Einschränkung).
