## ADDED Requirements

### Requirement: PII-tragende Routen sind fail-closed autorisiert und mechanisch getestet
Jede Route, die personenbezogene Daten ausliefert oder verändert (Ordner/Dateien,
Abwesenheiten, Anwesenheits-Recording, Spielbericht-Bilder, Spielbericht-Slots), SHALL einen
Autorisierungs-Test besitzen, der sowohl einen berechtigten Zugriff (Owner/Eltern/Staff) als
auch einen unberechtigten Fremdzugriff prüft. Fremdzugriff SHALL fail-closed enden (401/403/404),
nie mit stiller Auslieferung fremder Daten.

#### Scenario: Fremdzugriff wird abgewiesen
- **WHEN** ein Nutzer ohne Berechtigung eine PII-Route dieses Clusters aufruft
- **THEN** die Route SHALL mit 401/403/404 antworten und keine fremden Daten preisgeben

#### Scenario: Berechtigter Zugriff funktioniert
- **WHEN** der Owner, ein berechtigtes Elternteil oder das zuständige Staff-Mitglied die Route aufruft
- **THEN** die Route SHALL den Zugriff gewähren (2xx)

### Requirement: files — schreibende und ausliefernde Ordner-/Datei-Routen sind route-getestet
`CreateFolder`, `DeleteFolder`, `UploadFile`, `AddPermission`, `DeletePermission` und
`HandleDownloadToken` SHALL je einen Route-Ebenen-Test für Erfolg und Fremd-/Nicht-Berechtigt-Fall
haben. Die vorhandenen Unit-Tests (`resolveAccess`, `checkAntiEscalation`) SHALL nicht dupliziert,
sondern um den HTTP-Pfad ergänzt werden.

#### Scenario: Upload ohne can_write
- **WHEN** ein Nutzer ohne `can_write` `POST /api/folders/{id}/files` (multipart) aufruft
- **THEN** die Route SHALL mit 403 antworten und keine Datei speichern

#### Scenario: Permission-Grant über eigene Rechte hinaus
- **WHEN** ein Nutzer via `POST /api/folders/{id}/permissions` ein Recht vergibt, das er selbst nicht hält
- **THEN** die Route SHALL mit 403 antworten (HTTP-Ebene, ergänzend zu den `checkAntiEscalation`-Units)

#### Scenario: Download-Token fail-closed
- **WHEN** ein Nutzer ohne Leserecht `GET /api/files/{id}/download-token` aufruft
- **THEN** die Route SHALL keinen gültigen Token ausgeben (403), nicht still einen Token liefern

### Requirement: matchreports — ServeImage liefert Bilder nur an Autor oder Reviewer
`GET /api/match-reports/{id}/images/{imgId}/blob` SHALL das Bild nur an den Autor des Berichts
oder einen Reviewer (medien/vorstand/admin) ausliefern.

#### Scenario: Fremder Nutzer
- **WHEN** ein eingeloggter Nutzer, der weder Autor noch Reviewer ist, ein Bild abruft
- **THEN** die Route SHALL mit 403 antworten

#### Scenario: Unbekannter Bericht oder Bild
- **WHEN** eine unbekannte Report- oder Image-ID abgerufen wird
- **THEN** die Route SHALL mit 404 antworten (keine Existenz-Preisgabe über 403 vs 404 hinaus)

#### Scenario: Autor und Reviewer
- **WHEN** der Autor oder ein Reviewer sein/ein Bild abruft
- **THEN** die Route SHALL das Bild ausliefern (200)

### Requirement: duties — Spielbericht-Slot-Guard beschränkt auf Presseteam/Admin
Das Ziehen eines Duty-Slots vom Typ `"Spielbericht"` SHALL nur `presseteam`/`admin` erlaubt sein;
andere Rollen SHALL mit `role_required` (403) abgewiesen werden. Nicht-Spielbericht-Slots SHALL
unberührt bleiben.

#### Scenario: Nicht-Presseteam zieht Spielbericht-Slot
- **WHEN** ein Nutzer ohne Rolle `presseteam`/`admin` einen Spielbericht-Slot claimt
- **THEN** der Guard SHALL mit 403 (`role_required`) abweisen

#### Scenario: Proxy-Claim durch Elternteil (Rollenverschiebung)
- **WHEN** ein Elternteil ohne `presseteam`-Rolle einen Spielbericht-Slot für sein Kind claimt
- **THEN** der Guard SHALL die Rolle des **handelnden** Elternteils werten und mit 403 abweisen

#### Scenario: Nicht-Spielbericht-Slot
- **WHEN** ein beliebiger berechtigter Nutzer einen Slot anderen Typs claimt
- **THEN** der Guard SHALL nicht eingreifen (regulärer Claim-Pfad)

### Requirement: attendance-Recording — nur Staff des zuständigen Teams darf speichern
`POST /api/training-sessions/{id}/attendances` (`Training.SaveAttendances`) und
`POST /api/games/{id}/attendances` (`Games.SaveAttendances`) SHALL nur Trainer/sportliche Leitung
des zuständigen Teams erlauben. Ein Trainer eines fremden Teams und ein Nicht-Staff SHALL mit 403
abgewiesen werden.

#### Scenario: Trainer des falschen Teams
- **WHEN** ein Trainer, der nicht dem Team der Session/des Spiels zugeordnet ist, Anwesenheiten speichert
- **THEN** die Route SHALL mit 403 antworten

#### Scenario: Zuständiger Trainer
- **WHEN** der Trainer des zuständigen Teams Anwesenheiten speichert
- **THEN** die Route SHALL die Speicherung durchführen (2xx)

### Requirement: absences — Sichtbarkeit und Mutation sind auf Owner/Eltern/Staff beschränkt
`GET /api/absences/calendar?show_team=true` SHALL Team-Abwesenheiten nur an vorstand/trainer-like
Nutzer liefern. `PUT`/`DELETE /api/absences/{id}` durch einen fremden Nutzer SHALL 403 ergeben.
`GET /api/absences` SHALL keine fremden Abwesenheiten preisgeben.

#### Scenario: show_team ohne Berechtigung
- **WHEN** ein einfaches Mitglied/Elternteil `calendar?show_team=true` aufruft
- **THEN** die Antwort SHALL keine Team-Abwesenheiten anderer enthalten

#### Scenario: Fremde Abwesenheit ändern/löschen
- **WHEN** ein Nutzer eine Abwesenheit ändert oder löscht, die nicht ihm/seinem Kind gehört und die er nicht als Staff verwalten darf
- **THEN** die Route SHALL mit 403 antworten
