## MODIFIED Requirements

### Requirement: ACL-Einträge verwalten
Nutzer mit `can_write` auf einen Ordner SHALL Berechtigungseinträge via `POST /api/folders/:id/permissions` anlegen können. Bestehende Einträge SHALL via `DELETE /api/folders/:id/permissions/:permId` entfernt werden können.

Ein Eintrag besteht aus `principal_type` (`everyone` | `role` | `club_function` | `user` | `team` | `team_parents`), `principal_ref` (null bei `everyone`, sonst Rollenname / Funktionsname / `users.id` / `teams.id`) sowie `can_read` und `can_write`.

Der Wert `owner` ist **kein** gültiger `principal_type`: Er erscheint ausschließlich als synthetischer Eintrag in der Leseantwort und MUSS beim Anlegen mit HTTP 400 abgelehnt werden.

#### Scenario: Berechtigung anlegen
- **WHEN** ein Nutzer mit `can_write` einen neuen ACL-Eintrag anlegt
- **THEN** wird der Eintrag gespeichert und ab sofort bei der Auflösung berücksichtigt

#### Scenario: Berechtigung lesen
- **WHEN** ein Nutzer mit `can_write` `GET /api/folders/:id/permissions` aufruft
- **THEN** erhält er alle direkten ACL-Einträge des Ordners (ohne geerbte)

#### Scenario: Berechtigung entfernen
- **WHEN** ein berechtigter Nutzer `DELETE /api/folders/:id/permissions/:permId` aufruft
- **THEN** wird der Eintrag gelöscht; geerbte Rechte bleiben unberührt

#### Scenario: Unbekannter Principal-Typ wird abgelehnt
- **WHEN** ein Nutzer einen Eintrag mit `principal_type=owner` oder einem sonstigen unbekannten Wert anlegt
- **THEN** antwortet der Server mit HTTP 400 und speichert nichts

### Requirement: Principal-Typen
Das System SHALL sechs Principal-Typen unterstützen: `user`, `team`, `team_parents`, `club_function`, `role` und `everyone`. Alle auf einem Ordner definierten Einträge werden ausgewertet; ein Treffer bei einem beliebigen Typ reicht für die Gewährung des jeweiligen Rechts (`can_read` und `can_write` werden getrennt vereinigt).

`team` und `team_parents` tragen eine `teams.id` als `principal_ref` und werden gegen den Kader der aktiven Saison aufgelöst — die Auflösungsregeln stehen in der Capability `folder-permission-resolution`.

#### Scenario: Everyone-Berechtigung
- **WHEN** ein Ordner einen ACL-Eintrag mit `principal_type = 'everyone'` und `can_read = 1` hat
- **THEN** hat jeder authentifizierte Nutzer `can_read` auf diesen Ordner

#### Scenario: Rollen-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'role'`, `principal_ref = 'admin'` und `can_write = 1` hat
- **THEN** haben alle Nutzer mit `role = 'admin'` `can_write` auf diesen Ordner

#### Scenario: Vereinsfunktions-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'club_function'` und `principal_ref = 'kassierer'` hat
- **THEN** haben alle Nutzer mit der Funktion `kassierer` in `ClubFunctions[]` die entsprechenden Rechte

#### Scenario: User-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'user'` und `principal_ref = '42'` hat
- **THEN** hat der Nutzer mit `user_id = 42` die entsprechenden Rechte

#### Scenario: Team-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'team'` und `principal_ref = '7'` hat
- **THEN** haben alle Spieler, erweiterten Kadermitglieder und Trainer der Mannschaft mit `teams.id = 7` in der aktiven Saison die entsprechenden Rechte

#### Scenario: Eltern-Berechtigung
- **WHEN** ein Ordner einen Eintrag mit `principal_type = 'team_parents'` und `principal_ref = '7'` hat
- **THEN** haben alle über `family_links` verknüpften Elternteile der Spieler und erweiterten Kadermitglieder dieser Mannschaft in der aktiven Saison die entsprechenden Rechte

## REMOVED Requirements

### Requirement: Additive Berechtigungsvererbung

**Reason**: Fachlich abgelöst durch die Capability `folder-permission-resolution` (Nearest-Ancestor-Wins), eingeführt mit `archive/2026-06-13-folder-permissions-fix`. Die additive Auflösung war ein Sicherheitsloch — eine `everyone: read`-Freigabe in einem Vorfahren hebelte jede Einschränkung im Unterordner aus. Das Requirement wurde damals nur nicht gestrichen und widerspricht seither dem tatsächlichen Verhalten sowie der neueren Spec.

**Migration**: Keine. Das beschriebene Verhalten existiert seit Juni 2026 nicht mehr im Code; maßgeblich ist `folder-permission-resolution`. Die einzige verbliebene Form eines nicht entziehbaren Rechts ist der Eigentümer-Vorrang, der dort eigenständig spezifiziert ist.
