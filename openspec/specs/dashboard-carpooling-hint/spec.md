# dashboard-carpooling-hint Specification

## Purpose

Diese Spezifikation beschreibt die Capability `dashboard-carpooling-hint`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Personalisierten Fahrtgemeinschafts-Status anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingConfirmed` auch bestätigte Paarungen der Kinder des eingeloggten Nutzers anzeigen — zusätzlich zu den eigenen Paarungen.

`carpoolingConfirmed` enthält die nächsten max. 3 Auswärtsspiele, bei denen entweder der Nutzer selbst ODER eines seiner Kinder eine `confirmed`-Paarung hat. `partnerName` zeigt die Gegenseite aus Sicht des Nutzers/Kindes.

Jeder Paarungs-Eintrag SHALL zusätzlich das Feld `partnerTreffpunkt` (string) enthalten. `partnerTreffpunkt` ist der Treffpunkt der **Gegenseite** der Paarung aus Sicht des Nutzers (bzw. des Kindes):

- Ist die eigene Seite der **Bieter**-Eintrag der Paarung, dann ist `partnerTreffpunkt` der Treffpunkt des verknüpften **Sucher**-Eintrags.
- Ist die eigene Seite der **Sucher**-Eintrag, dann ist `partnerTreffpunkt` der Treffpunkt des verknüpften **Bieter**-Eintrags.

Wenn der Partner-Eintrag keinen Treffpunkt gesetzt hat (`mitfahrgelegenheiten.treffpunkt` NULL oder leer), SHALL `partnerTreffpunkt` als leerer String (`""`) zurückgegeben werden — nicht weggelassen und nicht `null`.

#### Scenario: User mit bestätigter Paarung

- **WHEN** der User eine Paarung mit `status='confirmed'` für ein kommendes Auswärtsspiel hat
- **THEN** enthält `carpoolingConfirmed` dieses Spiel mit der entsprechenden Paarung

#### Scenario: Kind mit bestätigter Paarung

- **WHEN** ein Kind des eingeloggten Elternteils eine `confirmed`-Paarung für ein kommendes Auswärtsspiel hat
- **THEN** erscheint diese Paarung ebenfalls in `carpoolingConfirmed` des Elternteils

#### Scenario: Kein Auswärtsspiel mit Paarung

- **WHEN** weder der Nutzer noch ein Kind eine bestätigte Paarung für kommende Auswärtsspiele haben
- **THEN** ist `carpoolingConfirmed` ein leeres Array

#### Scenario: Nutzer ohne Kinder — unverändert

- **WHEN** ein Nutzer ohne `family_links`-Einträge das Dashboard lädt
- **THEN** enthält `carpoolingConfirmed` ausschließlich seine eigenen Paarungen (Verhalten wie bisher)

#### Scenario: partnerTreffpunkt bei eigener Bieter-Paarung

- **WHEN** die eigene Seite Bieter ist und der verknüpfte Sucher-Eintrag den Treffpunkt `"Bahnhof Mitte"` hat
- **THEN** liefert die Paarung `partnerTreffpunkt = "Bahnhof Mitte"`

#### Scenario: partnerTreffpunkt bei eigener Sucher-Paarung

- **WHEN** die eigene Seite Sucher ist und der verknüpfte Bieter-Eintrag den Treffpunkt `"Marktplatz"` hat
- **THEN** liefert die Paarung `partnerTreffpunkt = "Marktplatz"`

#### Scenario: partnerTreffpunkt leer, wenn Partner keinen gesetzt hat

- **WHEN** der Partner-Eintrag keinen Treffpunkt gesetzt hat
- **THEN** liefert die Paarung `partnerTreffpunkt = ""` (leerer String, nicht null und nicht weggelassen)

#### Scenario: partnerTreffpunkt bei Kind-Paarung

- **WHEN** das Kind des Eltern-Users Bieter ist und der verknüpfte Sucher-Eintrag den Treffpunkt `"Schule"` hat
- **THEN** liefert die Paarung im Eltern-Dashboard `partnerTreffpunkt = "Schule"`
