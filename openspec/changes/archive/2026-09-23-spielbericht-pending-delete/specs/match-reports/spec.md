# Spec Delta

## MODIFIED Requirements

### Requirement: State `pending_review` als Review-Gate
Das System SHALL zwischen `draft` und `publishing` den State `pending_review` einführen. Der Vorwärtsflow lautet ausschließlich `draft → pending_review → publishing → published`, mit `publish_failed` als Retry-Punkt aus `publishing`. **Es gibt keinen Rückweg zum Autor** — kein Übergang `pending_review → draft`, kein Reject, kein Zurückschicken zur Bearbeitung durch den Autor. Ein **endgültiger Abbruch per Löschen** durch Freigeber ist davon ausgenommen (siehe Requirement „Löschen im State `pending_review` durch Freigeber") — das ist kein Rückweg, sondern das Ende des Berichts. Der State-Wert `pending_review` MUSS im DB-CHECK-Constraint auf `match_reports.state` akzeptiert werden.

#### Scenario: State-Wert ist gültig
- **WHEN** `UPDATE match_reports SET state='pending_review' WHERE id=?` ausgeführt wird
- **THEN** akzeptiert die Datenbank den Wert

#### Scenario: Kein Rückweg per SQL
- **WHEN** ein Client versucht, `pending_review → draft` über irgendeine Route herbeizuführen
- **THEN** existiert keine solche Route — der Aufruf liefert HTTP 404 oder 405

## ADDED Requirements

### Requirement: Löschen im State `pending_review` durch Freigeber
Das System SHALL bei `DELETE /api/match-reports/{id}` zusätzlich zu `draft`/`publish_failed` auch den State `pending_review` löschbar machen. Berechtigt sind dafür ausschließlich Freigeber (Vereinsfunktion `medien` ODER `vorstand` ODER Rolle `admin`) — der Autor selbst darf einen bereits eingereichten Bericht NICHT löschen, da er mit `submit-for-review` die Verfügung über den Bericht an die Freigeber abgegeben hat. Beim Löschen werden alle zugehörigen Bild-Dateien entfernt (analog zum bestehenden Verhalten bei `draft`). Response: HTTP 204. Broadcast `match-report-event`.

#### Scenario: Freigeber löscht eingereichten Bericht
- **WHEN** ein Nutzer mit Vereinsfunktion `medien` `DELETE /api/match-reports/{id}` auf einen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 204, der Bericht samt Bildern ist entfernt

#### Scenario: Vorstand löscht eingereichten Bericht
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `DELETE /api/match-reports/{id}` auf einen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 204

#### Scenario: Autor ohne Freigeber-Funktion versucht Löschen
- **WHEN** der Autor (ohne `medien`/`vorstand`) `DELETE /api/match-reports/{id}` auf seinen eigenen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 403

#### Scenario: Löschen im State published bleibt verboten
- **WHEN** ein Freigeber `DELETE /api/match-reports/{id}` auf einen `published`-Bericht ruft
- **THEN** liefert das System HTTP 409 mit `{"error":"already_published"}`
