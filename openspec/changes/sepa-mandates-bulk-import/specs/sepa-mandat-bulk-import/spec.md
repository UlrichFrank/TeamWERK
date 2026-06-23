## ADDED Requirements

### Requirement: Vorstand/Kassierer/Admin kann SEPA-Mandat-PDFs als Verzeichnis bulk-importieren
Das System SHALL einen Bulk-Import-Endpoint `POST /api/members/sepa-mandates/import` anbieten, der eine Multipart-Liste von PDF-Dateien entgegennimmt und sie via Filename-Match den Mitgliedern zuordnet. Der Endpoint SHALL für die Vereinsfunktionen `vorstand` und `kassierer` sowie für `admin` zugänglich sein und für alle anderen Rollen HTTP 403 liefern.

#### Scenario: Vorstand importiert Mandate-Verzeichnis
- **WHEN** ein Vorstandsmitglied auf `/mitglieder` über das Aktions-Dropdown „Import SEPA-Mandate" ein Verzeichnis mit PDFs auswählt und absendet
- **THEN** wird `POST /api/members/sepa-mandates/import` mit allen PDFs als Multipart-Parts aufgerufen und die Antwort ist HTTP 200 mit einem Report-JSON

#### Scenario: Kassierer importiert Mandate-Verzeichnis
- **WHEN** ein Kassierer den Bulk-Import auslöst
- **THEN** antwortet der Server mit HTTP 200 (Berechtigungs-Parität zum Einzel-Upload `POST /api/upload/sepa-mandat/{id}`)

#### Scenario: Spieler ohne Vorstandsfunktion ruft Bulk-Import
- **WHEN** ein Nutzer mit ausschließlich `spieler`-Vereinsfunktion `POST /api/members/sepa-mandates/import` aufruft
- **THEN** antwortet der Server mit HTTP 403 Forbidden, ohne Dateien zu lesen

### Requirement: Filename-Match per normalisiertem Vor-/Nachnamen
Das System SHALL pro PDF den Basename (ohne `.pdf`) normalisieren (lowercase, Umlaut-Substitution `ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`, Entfernen von Leerzeichen/Bindestrichen/Apostrophen/Unterstrichen/Punkten) und gegen die normalisierte Konkatenation `first_name+last_name` und `last_name+first_name` jedes Mitglieds vergleichen.

#### Scenario: Eindeutiger Match in Vorname-Nachname-Reihenfolge
- **WHEN** Datei `MaxMustermann.pdf` hochgeladen wird und genau ein Mitglied `first_name='Max', last_name='Mustermann'` existiert
- **THEN** wird das PDF diesem Mitglied zugeordnet (Eintrag in `imported`)

#### Scenario: Eindeutiger Match in Nachname-Vorname-Reihenfolge
- **WHEN** Datei `MustermannMax.pdf` hochgeladen wird und genau ein Mitglied `first_name='Max', last_name='Mustermann'` existiert
- **THEN** wird das PDF diesem Mitglied zugeordnet

#### Scenario: Umlaute werden normalisiert
- **WHEN** Datei `JuergenMueller.pdf` hochgeladen wird und ein Mitglied `first_name='Jürgen', last_name='Müller'` existiert
- **THEN** matcht der normalisierte Dateiname das normalisierte Mitglied und das PDF wird importiert

#### Scenario: Kein passendes Mitglied
- **WHEN** Datei `Unbekannt.pdf` hochgeladen wird und kein Mitglied normalisiert auf diesen Basename matcht
- **THEN** wandert der Eintrag in `no_match`, **keine Datei** wird unter `<uploadDir>/sepa-mandats/` abgelegt, kein DB-Update

#### Scenario: Mehrere Mitglieder matchen denselben Basename
- **WHEN** Datei `MaxMueller.pdf` hochgeladen wird und zwei Mitglieder normalisiert darauf matchen (z.B. Vater & Sohn)
- **THEN** wandert der Eintrag in `ambiguous` mit beiden Kandidaten-`member_id`s; **keine Datei** wird abgelegt, kein DB-Update

### Requirement: Bestehende Mandate werden nie überschrieben
Das System SHALL den Bulk-Import idempotent halten: Mitglieder mit nicht-leerem `sepa_mandat_path` werden übersprungen — weder die Datei noch der Pfad noch das `sepa_mandat`-Flag ändern sich.

#### Scenario: Mitglied hat bereits ein Mandat hinterlegt
- **WHEN** Datei `MaxMustermann.pdf` einem Mitglied zugeordnet wird, dessen `sepa_mandat_path` bereits gefüllt ist
- **THEN** wandert der Eintrag in `already_exists`, die alte Datei bleibt unverändert auf dem Filesystem, `members.sepa_mandat_path` und `members.sepa_mandat` werden **nicht** geändert

#### Scenario: Mehrfacher Import desselben Verzeichnisses
- **WHEN** der Bulk-Import zweimal hintereinander mit identischem Inhalt aufgerufen wird
- **THEN** sind beim zweiten Aufruf **alle** Einträge in `already_exists`, die DB-Zustände identisch zum Ergebnis des ersten Aufrufs

### Requirement: Erfolgreicher Match setzt Pfad und Mandat-Flag
Das System SHALL pro erfolgreich gematchter Datei (a) die Datei unter `<uploadDir>/sepa-mandats/<uuid>.pdf` ablegen, (b) `members.sepa_mandat_path` auf den relativen Pfad setzen und (c) `members.sepa_mandat = 1` setzen. Das Feld `members.sepa_mandat_date` SHALL **nicht** verändert werden.

#### Scenario: Match setzt Pfad und Flag
- **WHEN** eine PDF erfolgreich gematcht und importiert wird
- **THEN** ist `members.sepa_mandat_path` mit dem neuen Pfad gefüllt, `members.sepa_mandat = 1`, und `members.sepa_mandat_date` unverändert (NULL oder altem Wert)

#### Scenario: Datei wird physisch gespeichert
- **WHEN** ein Match erfolgt
- **THEN** existiert die Datei unter `<uploadDir>/sepa-mandats/<uuid>.pdf` und ist via `GET /api/members/{id}/sepa-mandat/download` (mit gültigem Token) abrufbar

### Requirement: Per-File-Atomarität bei Schreib-Fehlern
Das System SHALL beim Bulk-Import jede Datei atomar verarbeiten: schlägt der DB-`UPDATE` nach erfolgreichem Filesystem-Write fehl, MUSS die soeben geschriebene Datei via `os.Remove` wieder entfernt werden. Andere Dateien des Requests SHALL davon unberührt bleiben.

#### Scenario: DB-Fehler bei einer von mehreren Dateien
- **WHEN** während des Bulk-Imports ein einzelnes `UPDATE` fehlschlägt
- **THEN** wird die zugehörige Datei vom Filesystem gelöscht, der Eintrag landet in einer Fehler-Sektion des Reports, alle anderen Dateien werden weiter verarbeitet

### Requirement: Validierung pro Datei (PDF-MIME, Größenlimit)
Das System SHALL pro Datei `application/pdf` als MIME-Type erzwingen (Header-basiert mit Magic-Byte-Fallback) und Dateien > 10 MB ablehnen. Verletzungen MÜSSEN nicht den Gesamt-Request abbrechen, sondern als Report-Eintrag in `no_match` mit Begründung gemeldet werden.

#### Scenario: Nicht-PDF wird abgelehnt
- **WHEN** eine `.jpg`-Datei im Multipart-Body landet
- **THEN** wird der Eintrag in `no_match` mit Begründung „kein PDF" gemeldet, keine Datei gespeichert

#### Scenario: PDF überschreitet Größenlimit
- **WHEN** eine PDF > 10 MB im Multipart-Body landet
- **THEN** wird der Eintrag in `no_match` mit Begründung „zu groß (>10 MB)" gemeldet, keine Datei gespeichert

### Requirement: Report-Response strukturiert nach Status
Das System SHALL eine JSON-Antwort mit vier Listen liefern: `imported`, `already_exists`, `no_match`, `ambiguous`. Pro Eintrag SHALL mindestens `filename` enthalten sein; bei `imported`/`already_exists` zusätzlich `member_id` und `member_name`; bei `ambiguous` `candidates: [{member_id, member_name}, …]`; bei `no_match` optional `reason` (`"kein PDF"`, `"zu groß"`, oder leer für „kein Match").

#### Scenario: Report enthält alle vier Sektionen
- **WHEN** ein Bulk-Import mit gemischten Files (1 match, 1 already-exists, 1 no-match, 1 ambiguous) verarbeitet wird
- **THEN** enthält die Response je einen Eintrag in `imported`, `already_exists`, `no_match`, `ambiguous` mit den erwarteten Feldern

### Requirement: SSE-Broadcast nach erfolgreichem Import
Das System SHALL nach mindestens einem erfolgreichen Match `hub.Broadcast("members")` aufrufen, damit offene Mitgliederlisten-Tabs sich automatisch neu laden. Hat kein einziger Match stattgefunden, MUSS kein Broadcast erfolgen (kein Live-Trigger ohne Datenänderung).

#### Scenario: Broadcast nach mindestens einem Match
- **WHEN** der Bulk-Import mindestens eine Datei in `imported` aufnimmt
- **THEN** wird `hub.Broadcast("members")` exakt einmal aufgerufen

#### Scenario: Kein Broadcast bei reinem No-Match-Import
- **WHEN** der Bulk-Import keine einzige Datei importiert (alle in `no_match`/`already_exists`/`ambiguous`)
- **THEN** wird `Broadcast` **nicht** aufgerufen

### Requirement: Multipart-Body-Limit
Das System SHALL den Multipart-Body auf 500 MB Gesamtgröße beschränken (`http.MaxBytesReader`). Überschreitungen MÜSSEN mit HTTP 413 (Request Entity Too Large) inklusive JSON-Body `{error, limit}` abgewiesen werden, ohne dass Dateien geschrieben werden.

#### Scenario: Body überschreitet 500 MB
- **WHEN** ein Multipart-Request > 500 MB eingeht
- **THEN** antwortet der Server mit HTTP 413 und einem JSON-Body mit `limit`-Feld, und keine Datei wird im Upload-Verzeichnis abgelegt

### Requirement: Frontend zeigt Import-Modal mit Verzeichnis-Picker
Das Frontend SHALL auf `/mitglieder` im Aktions-Dropdown den Eintrag „Import SEPA-Mandate" anzeigen, ein Modal mit `<input type="file" webkitdirectory multiple>` öffnen, die ausgewählten Dateien clientseitig auf `*.pdf` filtern und nach dem Submit den Report rendern. Der Eintrag SHALL nur sichtbar sein, wenn der eingeloggte Nutzer `admin`, `vorstand` oder `kassierer` ist.

#### Scenario: Eintrag im Dropdown
- **WHEN** ein Vorstandsmitglied das Aktions-Dropdown öffnet
- **THEN** ist „Import SEPA-Mandate" zwischen „Import CSV" und „Export CSV" sichtbar

#### Scenario: Spieler sieht Eintrag nicht
- **WHEN** ein Nutzer mit ausschließlich `spieler`-Vereinsfunktion das Aktions-Dropdown öffnet
- **THEN** ist „Import SEPA-Mandate" **nicht** sichtbar

#### Scenario: Verzeichnis-Auswahl filtert auf PDFs
- **WHEN** der Nutzer ein Verzeichnis mit gemischten Dateien (PDF, JPG, .DS_Store) auswählt
- **THEN** zeigt das Modal in der Vorschau nur die PDF-Dateien an und sendet beim Submit nur diese

#### Scenario: Report-Anzeige nach Import
- **WHEN** der Server eine Report-Response liefert
- **THEN** rendert das Modal vier Sektionen (Importiert / Bereits vorhanden / Nicht zugeordnet / Mehrdeutig) mit Zähler und Dateilisten
