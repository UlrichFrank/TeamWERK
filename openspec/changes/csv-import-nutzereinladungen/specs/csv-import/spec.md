## ADDED Requirements

### Requirement: Admin kann CSV mit E-Mail-Adressen importieren
Das System SHALL einen `POST /api/admin/invitations/import-csv`-Endpoint bereitstellen, der eine CSV-Datei entgegennimmt, die Spalten `Email` und `Email 2` liest, alle unique E-Mail-Adressen dedupliziert und für jede neue Adresse einen `invitation_token` anlegt, ohne eine E-Mail zu versenden. Nur Nutzer mit Rolle `admin` dürfen diesen Endpoint aufrufen.

#### Scenario: Erfolgreicher Import
- **WHEN** Admin lädt eine gültige CSV-Datei hoch
- **THEN** legt das System für jede unique E-Mail-Adresse (aus `Email` und `Email 2`), die weder in `users` noch in `invitation_tokens` existiert, einen `invitation_token` mit Rolle `standard` und Ablaufzeit +48h an
- **THEN** gibt das System `{ "created": N, "skipped": M }` zurück (200 OK)

#### Scenario: Bereits vorhandene E-Mails werden übersprungen
- **WHEN** eine E-Mail aus der CSV bereits in `users.email` existiert
- **THEN** legt das System keinen neuen Token für diese Adresse an und zählt sie als `skipped`

#### Scenario: Bereits eingeladene E-Mails werden übersprungen
- **WHEN** eine E-Mail aus der CSV bereits in `invitation_tokens.email` existiert (unabhängig ob abgelaufen oder nicht)
- **THEN** legt das System keinen neuen Token für diese Adresse an und zählt sie als `skipped`

#### Scenario: Leere E-Mail-Felder werden ignoriert
- **WHEN** eine Zeile in der CSV kein `Email`- oder kein `Email 2`-Feld hat (leer oder fehlt)
- **THEN** überspringt das System dieses Feld kommentarlos

#### Scenario: Ungültige CSV
- **WHEN** die hochgeladene Datei keine gültige CSV ist oder die Spalte `Email` fehlt
- **THEN** gibt das System 400 Bad Request zurück

### Requirement: CSV-Import-UI in der Nutzerverwaltung
Das Frontend SHALL den bisherigen „+ Einladung"-Button durch einen „CSV importieren"-Button ersetzen, der ein Modal öffnet.

#### Scenario: CSV-Upload-Modal
- **WHEN** Admin klickt auf „CSV importieren"
- **THEN** öffnet sich ein Modal mit einem Datei-Upload-Feld (akzeptiert `.csv`)
- **THEN** nach dem Upload zeigt das Modal das Ergebnis: „X Einladungen angelegt, Y übersprungen"

#### Scenario: Fehler beim Upload
- **WHEN** der Server einen Fehler zurückgibt (400 oder 500)
- **THEN** zeigt das Modal eine Fehlermeldung mit dem Fehlertext an

### Requirement: Einladungs-E-Mail on demand versenden
Das System SHALL einen `POST /api/admin/invitations/{id}/send`-Endpoint bereitstellen, der für eine bestehende Einladung die Einladungs-E-Mail versendet.

#### Scenario: E-Mail erfolgreich versendet
- **WHEN** Admin klickt „Einladung senden" im ActionMenu einer Einladungs-Zeile
- **THEN** sendet das System die Einladungs-E-Mail an die hinterlegte Adresse und gibt 204 zurück

#### Scenario: SMTP-Fehler
- **WHEN** der SMTP-Versand fehlschlägt
- **THEN** gibt das System 502 zurück und das Frontend zeigt eine Fehlermeldung an

#### Scenario: Einladung nicht gefunden
- **WHEN** die `id` keiner Einladung entspricht
- **THEN** gibt das System 404 zurück
