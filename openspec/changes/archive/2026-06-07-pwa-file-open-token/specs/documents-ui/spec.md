## MODIFIED Requirements

### Requirement: Datei öffnen (alle Plattformen)
Das System MUSS beim Klick auf eine Datei zuerst ein kurzlebiges Download-Token vom Backend anfordern und anschließend die Datei-URL mit Token via `window.open(url, '_blank')` öffnen. Es DARF kein Blob heruntergeladen oder eine Blob-URL erzeugt werden. Der Browser entscheidet anhand des `Content-Type` selbst, ob er die Datei anzeigt (PDF, Bild) oder herunterlädt (DOCX, ZIP). Dies MUSS sowohl im iOS-PWA-Standalone-Modus als auch im Desktop-Browser funktionieren.

#### Scenario: PDF-Klick in iOS PWA
- **WHEN** ein Nutzer in der installierten iOS PWA auf eine PDF-Datei klickt
- **THEN** öffnet Safari die Datei in der nativen PDF-Ansicht

#### Scenario: Bild-Klick im Desktop-Browser
- **WHEN** ein Nutzer im Desktop-Browser auf eine Bilddatei klickt
- **THEN** öffnet ein neuer Tab die Datei direkt im Browser

#### Scenario: DOCX-Klick (kein nativer Viewer)
- **WHEN** ein Nutzer auf eine DOCX-Datei klickt
- **THEN** triggert der Browser einen Download-Dialog

#### Scenario: Token-Fehler beim Öffnen
- **WHEN** die Token-Anfrage fehlschlägt (z.B. Netzwerkfehler)
- **THEN** zeigt die UI einen Fehlerhinweis (kein stiller Fehler)
