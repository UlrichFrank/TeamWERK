## ADDED Requirements

### Requirement: Einheitliche Paginierung schwerer Listen-Endpoints

Das System SHALL für die Listen-Endpoints `GET /api/kader`, `GET /api/duty-slots`, `GET /api/games`, `GET /api/games/{id}/participants` und `GET /api/training-sessions` eine Paginierung über `?limit=` und `?offset=` anbieten und mit `{ "items": [...], "total": <int> }` antworten, wobei `total` die Gesamtzahl der der Sichtbarkeit entsprechenden Elemente (ohne `LIMIT`) ist. Ohne `limit` SHALL ein routen-spezifisches Default-Limit gelten; keine dieser Routen SHALL unbeschränkte Ergebnismengen liefern.

#### Scenario: Limit und Offset begrenzen die Ergebnismenge

- **WHEN** ein Client `GET /api/kader?limit=2&offset=0` aufruft und mehr als zwei Kader existieren
- **THEN** enthält `items` genau zwei Einträge
- **AND** `total` gibt die Gesamtzahl der sichtbaren Kader an

#### Scenario: Default-Limit ohne expliziten Parameter

- **WHEN** ein Client eine der Listen-Routen ohne `limit` aufruft
- **THEN** wird das routen-spezifische Default-Limit angewandt
- **AND** `total` bleibt die vollständige Gesamtzahl

#### Scenario: Offset erreicht weitere Seiten

- **WHEN** ein Client dieselbe Route mit steigendem `offset` aufruft
- **THEN** erhält er disjunkte, aufeinanderfolgende Ausschnitte derselben sortierten Gesamtmenge

### Requirement: Sichtbarkeit bleibt bei Paginierung invariant

Das System SHALL durch Paginierung oder Feld-Trimming die Autorisierungs- und Sichtbarkeitsregeln einer Route NICHT verändern. `items` und `total` SHALL exakt dieselben `WHERE`-Sichtbarkeitsbedingungen verwenden.

#### Scenario: Kein Element wird durch Paginierung neu sichtbar

- **WHEN** ein Nutzer eine paginierte Route über alle `offset`-Seiten hinweg abruft
- **THEN** erhält er genau die Elemente, die er auch unpaginiert sehen dürfte — keine zusätzlichen, keine fehlenden

### Requirement: Serverseitiger Filter statt clientseitigem Filtern

Das System SHALL Filter, die das Frontend bisher clientseitig über volle Listen anwendet, als Query-Parameter serverseitig anbieten. Insbesondere SHALL `GET /api/training-sessions?exclude_series=1` nur Sessions ohne Serienbezug (`series_id IS NULL`) liefern.

#### Scenario: exclude_series filtert serverseitig

- **WHEN** ein Client `GET /api/training-sessions?exclude_series=1` aufruft
- **THEN** enthält die Antwort ausschließlich Sessions mit `series_id IS NULL`

#### Scenario: Ohne Filter unverändert

- **WHEN** ein Client `GET /api/training-sessions` ohne `exclude_series` aufruft
- **THEN** enthält die Antwort Sessions mit und ohne Serienbezug (im Rahmen von Paginierung/Zeitfenster)

### Requirement: Body-Preview für Nachrichtenlisten

Das System SHALL in `GET /api/chat/conversations/{id}/messages` je Nachricht einen gekürzten Body-Preview (höchstens ~280 Zeichen) samt `truncated`-Flag liefern; der Volltext SHALL nur über den Einzel-Nachrichten-Pfad abrufbar sein. Gelöschte Nachrichten SHALL keinen Body/Preview enthalten.

#### Scenario: Langer Body wird gekürzt

- **WHEN** eine Nachricht einen Body länger als die Preview-Grenze hat
- **THEN** enthält das Listen-Item einen gekürzten `preview`
- **AND** `truncated` ist `true`

#### Scenario: Gelöschte Nachricht ohne Body

- **WHEN** eine Nachricht als gelöscht markiert ist
- **THEN** enthält das Listen-Item weder Body noch Preview
