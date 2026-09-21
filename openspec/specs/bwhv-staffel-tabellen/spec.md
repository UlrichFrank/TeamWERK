# bwhv-staffel-tabellen Specification

## Purpose
Tabellenstand und vollständiger Spielplan jeder Staffel, der eine Mannschaft
zugeordnet ist — einschließlich der Begegnungen ohne eigene Beteiligung. Beides
fällt beim selben Abruf ab, der die Spielberichte holt, und braucht kein PDF.

Umfasst außerdem die Ansicht „Staffeln" mit Mannschafts-Umschalter.

## Requirements

### Requirement: Tabellenstand je zugeordneter Staffel

Das System SHALL den Tabellenstand jeder Staffel, die einem Kader der aktiven Saison
zugeordnet ist, aus demselben Abruf speichern, der den Spielplan liefert, und ohne
PDF-Abruf bereitstellen.

Das System SHALL den Tabellenstand allen eingeloggten Nutzern zugänglich machen.

#### Scenario: Eingeloggter Nutzer ruft die Tabelle ab
- **WHEN** ein eingeloggter Nutzer die Tabelle einer zugeordneten Staffel abruft
- **THEN** antwortet der Server mit HTTP 200 und den Tabellenzeilen des letzten Abrufs

#### Scenario: Nicht eingeloggter Zugriff
- **WHEN** die Tabelle ohne gültiges Zugangstoken abgerufen wird
- **THEN** antwortet der Server mit HTTP 401

#### Scenario: Unbekannte Staffel
- **WHEN** eine Staffel abgerufen wird, die keinem Kader der aktiven Saison zugeordnet ist
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Tabelle ist ohne Spielbericht aktuell
- **WHEN** ein Poll Ergebnisse geliefert hat, zu denen noch kein Spielbericht existiert
- **THEN** ist der Tabellenstand aktualisiert

### Requirement: Kompletter Staffel-Spielplan

Das System SHALL den vollständigen Spielplan einer Staffel bereitstellen, einschließlich
der Begegnungen ohne Beteiligung des eigenen Vereins, jeweils mit Datum, Anwurfzeit,
beiden Mannschaften, Spielort und — sofern vorhanden — Ergebnis und Halbzeitstand.

Das System SHALL kenntlich machen, welche Begegnungen mit einem eigenen Spieltermin
verknüpft sind.

#### Scenario: Spielplan enthält fremde Begegnungen
- **WHEN** der Spielplan einer Staffel abgerufen wird
- **THEN** enthält die Antwort auch Begegnungen ohne Verknüpfung zu einem eigenen Spieltermin

#### Scenario: Eigene Begegnung ist erkennbar
- **WHEN** eine Begegnung mit einem eigenen Spieltermin verknüpft ist
- **THEN** weist die Antwort diese Verknüpfung aus

### Requirement: Ansicht „Staffeln" mit Mannschafts-Umschalter

Das System SHALL eine Oberfläche bereitstellen, die über einen Mannschafts-Umschalter
zwischen den zugeordneten Staffeln wechselt und Tabelle, Spielplan und Ranglisten in
getrennten Reitern zeigt.

Das System SHALL diese Ansicht über einen eigenen Navigationseintrag erreichbar machen,
der allen eingeloggten Nutzern angezeigt wird.

Das System SHALL den Spielbericht einer einzelnen Begegnung NICHT in dieser Ansicht,
sondern in der Detailansicht des Spiels zeigen.

#### Scenario: Wechsel zwischen Mannschaften
- **WHEN** ein Nutzer im Umschalter eine andere Mannschaft wählt
- **THEN** zeigen Tabelle, Spielplan und Ranglisten die Staffel dieser Mannschaft

#### Scenario: Navigationseintrag ist für alle sichtbar
- **WHEN** ein eingeloggter Nutzer ohne besondere Vereinsfunktion die Anwendung öffnet
- **THEN** ist der Navigationseintrag „Staffeln" sichtbar

### Requirement: Live-Aktualisierung der Staffel-Ansichten

Das System SHALL in den Staffel-Ansichten auf das SSE-Ereignis des Polls reagieren und die
dargestellten Daten nachladen.

#### Scenario: Offene Ansicht lädt nach
- **WHEN** ein Poll neue Ergebnisse gespeichert und das SSE-Ereignis gesendet hat
- **THEN** lädt eine geöffnete Staffel-Ansicht die Daten nach, ohne dass der Nutzer neu lädt
