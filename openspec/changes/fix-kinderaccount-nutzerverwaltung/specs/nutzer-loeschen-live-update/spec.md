## ADDED Requirements

### Requirement: Löschen eines Nutzers spiegelt sich ohne manuellen Reload
Das Löschen eines Nutzers über `DELETE /api/users/{id}` SHALL sich unmittelbar in der Nutzerverwaltung widerspiegeln, ohne dass der Vorstand die Seite manuell neu laden muss. Der Backend-Handler MUSS nach erfolgreichem Commit `h.hub.Broadcast("users")` aufrufen; das Frontend MUSS nach dem `DELETE` die Nutzerliste neu laden (direkter `refreshUsers()`-Aufruf) und zusätzlich auf das SSE-Event `users` reagieren.

#### Scenario: Backend broadcastet nach erfolgreicher Löschung
- **WHEN** ein berechtigter Aufrufer `DELETE /api/users/{id}` für einen existierenden, fremden Nutzer sendet
- **THEN** löscht das Backend den Nutzer (HTTP 204)
- **THEN** wird nach dem erfolgreichen Commit genau ein `Broadcast("users")` ausgelöst

#### Scenario: Auslösender Tab aktualisiert die Liste sofort
- **WHEN** der Vorstand in der Nutzerverwaltung "Löschen" bestätigt und das Backend mit 204 antwortet
- **THEN** ruft das Frontend `refreshUsers()` auf und die gelöschte Zeile verschwindet ohne manuellen Reload

#### Scenario: Andere offene Sessions aktualisieren per SSE
- **WHEN** in einer anderen offenen Session der Nutzerverwaltung das SSE-Event `users` eintrifft
- **THEN** lädt diese Session die Nutzerliste neu

#### Scenario: Löschen eines Kinder-Kontos hinterlässt keinen FK-Fehler
- **WHEN** `DELETE /api/users/{id}` für ein Kinder-Konto mit verknüpftem `members`-Datensatz (`members.user_id`) gesendet wird
- **THEN** antwortet das Backend mit HTTP 204
- **THEN** bleibt der `members`-Datensatz mit `user_id = NULL` erhalten (FK `ON DELETE SET NULL`)

#### Scenario: Selbst-Löschung bleibt abgelehnt
- **WHEN** ein Aufrufer `DELETE /api/users/{id}` mit der eigenen userId sendet
- **THEN** antwortet das Backend mit HTTP 400 und es wird nicht gelöscht
