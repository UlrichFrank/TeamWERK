## ADDED Requirements

### Requirement: Welcome-Mail-Anhänge im Dokumente-Bereich wählbar
Der Vorstand MUST im Dokumente-Bereich (`/dokumente`) Dateien als Begrüßungs-Anhang markieren können. Die Begrüßungs-E-Mail MUST genau die so markierten Dateien aus dem Dokumenten-Store anhängen — nicht länger hartcodierte, eingebettete PDFs. Die Markierungs-Route MUST `vorstand` (bzw. `admin`) verlangen und einen `Broadcast` auslösen.

#### Scenario: Vorstand markiert Datei als Begrüßungs-Anhang
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` eine Datei als Welcome-Anhang markiert
- **THEN** wird die Markierung persistiert, `Broadcast` ausgelöst und der Erfolg (200/204) zurückgegeben

#### Scenario: Nicht-Vorstand darf nicht markieren
- **WHEN** ein Nutzer ohne `vorstand`/`admin` die Markierungs-Route aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Begrüßungsmail hängt genau die markierten Dokumente an
- **WHEN** eine Begrüßungs-E-Mail versendet wird und Dateien als Welcome-Anhang markiert sind
- **THEN** enthält die Mail genau diese Dateien als Anhang (aus dem Dokumenten-Store geladen)

#### Scenario: Keine Markierung → keine Anhänge
- **WHEN** keine Datei als Welcome-Anhang markiert ist
- **THEN** wird die Begrüßungs-E-Mail ohne Anhang versendet, ohne Fehler
