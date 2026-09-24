## MODIFIED Requirements

### Requirement: State `pending_review` als Review-Gate
Das System SHALL zwischen `draft` und `publishing` den State `pending_review` einführen. Der Vorwärtsflow lautet `draft → pending_review → publishing → published`, mit `publish_failed` als Retry-Punkt aus `publishing`. Einziger Rückweg zum Autor ist die **Rückgabe** durch einen Freigeber (`pending_review → draft`, siehe Requirement „Rückgabe an den Autor mit Kommentar"); der Autor selbst kann einen eingereichten Bericht nicht zurückziehen. Ein **endgültiger Abbruch per Löschen** durch Freigeber bleibt daneben bestehen (siehe Requirement „Löschen im State `pending_review` durch Freigeber"). Der State-Wert `pending_review` MUSS im DB-CHECK-Constraint auf `match_reports.state` akzeptiert werden.

#### Scenario: State-Wert ist gültig
- **WHEN** `UPDATE match_reports SET state='pending_review' WHERE id=?` ausgeführt wird
- **THEN** akzeptiert die Datenbank den Wert

#### Scenario: Kein Rückweg per SQL
- **WHEN** ein Client versucht, `pending_review → draft` über eine andere Route als `POST /api/match-reports/{id}/return` herbeizuführen (etwa `PUT /api/match-reports/{id}` mit einem `state`-Feld)
- **THEN** bleibt der State unverändert — `PUT` kennt kein `state`-Feld, die Rückgabe ist der einzige Übergang

#### Scenario: Autor kann nicht selbst zurückziehen
- **WHEN** der Autor ohne Freigeber-Funktion `POST /api/match-reports/{id}/return` auf seinen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 403 und der State bleibt `pending_review`

## ADDED Requirements

### Requirement: Rückgabe an den Autor mit Kommentar
Das System SHALL unter `POST /api/match-reports/{id}/return` mit Body `{"comment": "…"}` einem Freigeber (Vereinsfunktion `medien` ODER `vorstand` ODER Rolle `admin`) erlauben, einen Bericht im State `pending_review` an den Autor zurückzugeben. Der Kommentar ist Pflicht (nach Trimmen 1–2000 Zeichen). Der Übergang setzt `state='draft'`, `review_comment`, `returned_at` und `reviewer_user_id`, broadcastet `match-report-event` und benachrichtigt den Autor (Kategorie `operativ`) mit dem Kommentar. Danach gelten die Draft-Regeln: nur der Autor (oder Admin) darf bearbeiten und erneut einreichen. Der Kommentar bleibt beim erneuten Einreichen gespeichert und ist über `GET /api/match-reports/{id}` sichtbar; eine weitere Rückgabe überschreibt ihn.

#### Scenario: Freigeber gibt zurück
- **WHEN** ein Nutzer mit Vereinsfunktion `medien` `POST /api/match-reports/{id}/return` mit Kommentar auf einen `pending_review`-Bericht ruft
- **THEN** liefert das System HTTP 200, der Bericht steht auf `draft`, `review_comment` enthält den getrimmten Kommentar, und der Autor findet die Meldung in seinem Event-Log

#### Scenario: Autor bearbeitet nach Rückgabe
- **WHEN** der Autor nach der Rückgabe `PUT /api/match-reports/{id}` und `POST /api/match-reports/{id}/submit-for-review` ruft
- **THEN** liefern beide Aufrufe HTTP 200 und der Bericht steht wieder auf `pending_review`

#### Scenario: Rückgabe ohne Kommentar
- **WHEN** ein Freigeber `POST /api/match-reports/{id}/return` mit leerem oder nur aus Leerzeichen bestehendem Kommentar ruft
- **THEN** liefert das System HTTP 400 `comment_required` und der State bleibt unverändert

#### Scenario: Rückgabe im falschen State
- **WHEN** ein Freigeber einen Bericht im State `draft`, `publishing`, `published` oder `publish_failed` zurückgeben will
- **THEN** liefert das System HTTP 409 `not_pending_review`

#### Scenario: Zurückgegebener Bericht in der eigenen Liste
- **WHEN** der Autor `GET /api/match-reports/my` ruft und ein Bericht im State `draft` einen `review_comment` trägt
- **THEN** trägt dieser Eintrag `returned=true`
