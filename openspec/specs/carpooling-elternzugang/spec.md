# carpooling-elternzugang Specification

## Purpose

Diese Spezifikation beschreibt die Capability `carpooling-elternzugang`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Elternteil kann Eintrag für Kind anlegen
Das System SHALL es Elternteilen erlauben, einen Mitfahreintrag (biete oder suche) für ein Kind anzulegen, indem sie `forUserId` (die `users.id` des Kindes) im Request-Body mitgeben. Die Berechtigung wird ausschließlich über `family_links` geprüft — unabhängig von `can_login` des Kindes. Der Eintrag wird mit der `user_id` des Kindes gespeichert.

#### Scenario: Elternteil legt Suche-Eintrag für Kind an
- **WHEN** ein Elternteil `POST /api/mitfahrgelegenheiten` mit `forUserId = <kind-user-id>` und `typ='suche'` sendet
- **THEN** wird ein Eintrag mit `user_id = <kind-user-id>` angelegt und erscheint in der Liste mit dem Namen des Kindes

#### Scenario: Elternteil gibt fremde userId an
- **WHEN** ein Elternteil `forUserId` einer Person angibt, die NICHT ihr Kind ist
- **THEN** antwortet die API mit 403 Forbidden

#### Scenario: Elternteil überschreibt bestehenden Biete-Eintrag des Kindes (Upsert)
- **WHEN** das Kind bereits einen Biete-Eintrag für ein Spiel hat und der Elternteil denselben Eintrag via `forUserId` aktualisiert
- **THEN** wird der bestehende Eintrag des Kindes überschrieben (Upsert-Semantik bleibt erhalten)

### Requirement: Elternteil kann Kind-Eintrag löschen
Das System SHALL es Elternteilen erlauben, Mitfahreinträge zu löschen, deren `user_id` eines ihrer Kinder ist.

#### Scenario: Elternteil löscht Kind-Eintrag
- **WHEN** ein Elternteil `DELETE /api/mitfahrgelegenheiten/{id}` für einen Eintrag aufruft, der einem seiner Kinder gehört
- **THEN** wird der Eintrag gelöscht

#### Scenario: Elternteil löscht fremden Eintrag (kein Kind)
- **WHEN** ein Elternteil versucht, einen Eintrag zu löschen, der weder ihm noch einem seiner Kinder gehört
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Elternteil kann Paarungsanfrage für Kind stellen
Ein Elternteil SHALL eine Paarungsanfrage stellen können, wenn entweder der Biete-Eintrag oder der Suche-Eintrag einem seiner Kinder gehört (und der jeweils andere Eintrag dem Elternteil selbst oder ebenfalls einem Kind gehört).

#### Scenario: Elternteil stellt Anfrage für Kind (Kind sucht)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` mit `sucheId` (Kind-Eintrag) und `bieteId` (fremder Eintrag) sendet
- **THEN** wird eine Paarung mit `initiiert_von='suche'` angelegt

#### Scenario: Elternteil lädt Sucher ein (Kind bietet)
- **WHEN** ein Elternteil `POST /api/mitfahrt-paarungen` mit `bieteId` (Kind-Eintrag) und `sucheId` (fremder Eintrag) sendet
- **THEN** wird eine Paarung mit `initiiert_von='biete'` angelegt

#### Scenario: Elternteil greift auf Paarung ohne Bezug zum Kind zu
- **WHEN** weder bieteId noch sucheId einem Kind des Elternteils gehört (und auch nicht dem Elternteil selbst)
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Elternteil kann Paarung für Kind bestätigen oder ablehnen
Das System SHALL es Elternteilen erlauben, `confirm` und `reject` für Paarungen aufzurufen, in denen ein ihrer Kinder beteiligt ist — analog zu den Rechten des Kindes selbst.

#### Scenario: Elternteil bestätigt offene Anfrage für Kind
- **WHEN** eine `pending`-Paarung existiert, bei der das Kind die Gegenseite bestätigen müsste, und der Elternteil `POST /api/mitfahrt-paarungen/{id}/confirm` aufruft
- **THEN** wird `status='confirmed'` gesetzt

#### Scenario: Elternteil lehnt Paarung für Kind ab
- **WHEN** eine Paarung existiert, bei der ein Kind beteiligt ist, und der Elternteil `POST /api/mitfahrt-paarungen/{id}/reject` aufruft
- **THEN** wird `status='rejected'` gesetzt

### Requirement: isOwn für Kind-Einträge
Die `GET /api/mitfahrgelegenheiten`-Antwort SHALL `isOwn=true` (bzw. `bieteIsOwn`/`sucheIsOwn`) setzen, wenn der Eintrag einem Kind des eingeloggten Elternteils gehört — zusätzlich zum eigenen Eintrag.

#### Scenario: Elternteil sieht Kind-Eintrag als eigenen
- **WHEN** ein Elternteil `GET /api/mitfahrgelegenheiten` aufruft und ein Kind einen Biete-Eintrag hat
- **THEN** hat dieser Eintrag `isOwn: true` in der Antwort

### Requirement: ListResponse enthält Kind-Nutzer
`GET /api/mitfahrgelegenheiten` SHALL im Response-Objekt ein `children`-Array enthalten mit `userId` und `name` aller verknüpften Kinder des eingeloggten Nutzers. Für Nutzer ohne Kinder ist es ein leeres Array.

#### Scenario: Elternteil mit zwei Kindern
- **WHEN** ein Elternteil mit zwei verknüpften Kindern `GET /api/mitfahrgelegenheiten` aufruft
- **THEN** enthält `children` zwei Einträge mit `userId` und `name`

#### Scenario: Nutzer ohne Kinder
- **WHEN** ein Nutzer ohne `family_links`-Einträge die Liste abruft
- **THEN** ist `children` ein leeres Array `[]`
