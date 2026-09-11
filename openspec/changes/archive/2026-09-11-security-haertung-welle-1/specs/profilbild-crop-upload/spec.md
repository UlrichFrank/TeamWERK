## ADDED Requirements

### Requirement: Foto-Upload vertraut nur dem Dateiinhalt

Der Server MUST den Typ jeder hochgeladenen Datei aus ihren Bytes bestimmen und die vom Client gesendete Typangabe sowie die Dateiendung des Originalnamens ignorieren. Die gespeicherte Dateiendung MUST aus dem erkannten Typ abgeleitet werden. Bei der Auslieferung MUST der Server den Typ aus dem gespeicherten Wert setzen und jede Datei, die kein Bild ist, als Download (`Content-Disposition: attachment`) ausliefern.

#### Scenario: HTML mit Bild-Typangabe wird abgelehnt
- **WHEN** ein Nutzer eine HTML-Datei mit `Content-Type: image/png` und Dateiname `x.html` hochlädt
- **THEN** antwortet der Server mit HTTP 400 und speichert nichts

#### Scenario: Endung folgt dem Inhalt
- **WHEN** ein Nutzer ein gültiges PNG mit Dateiname `foto.html` hochlädt
- **THEN** wird die Datei mit Endung `.png` gespeichert und als `image/png` ausgeliefert

#### Scenario: Nicht-Bild wird nie inline gerendert
- **WHEN** eine gespeicherte Datei ausgeliefert wird, deren Typ nicht `image/*` ist
- **THEN** trägt die Antwort `Content-Disposition: attachment`
