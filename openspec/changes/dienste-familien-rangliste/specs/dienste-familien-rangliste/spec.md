## Purpose

Macht sichtbar, wie viele Dienste eine Familie bezogen auf ein Kind in der aktiven
Saison bereits geleistet hat, und stellt diese Zahl in einen fairen, aus den
tatsächlich bekannten Terminen der Saison hergeleiteten Vergleichsrahmen.

## ADDED Requirements

### Requirement: Geleistet- und Vorhersage-Zählung je Kind

Das System SHALL für jedes Kind mit aktiver Kader-Mitgliedschaft in der aktiven
Saison zwei Zahlen aus den `duty_assignments` des Kindes zählen:
- `geleistet`: Anzahl Zuweisungen mit `event_date` in der Vergangenheit
- `vorhersage`: Anzahl Zuweisungen mit `event_date` in der Zukunft (heute
  eingeschlossen)

Der `status` einer Zuweisung ist für diese Zählung ohne Bedeutung — die Existenz der
Zuweisung genügt.

Da eine Zuweisung nur einen Account (`user_id`) trägt, SHALL das System sie wie
folgt Kindern zurechnen — die erste nicht-leere Stufe gewinnt und wird
gleichmäßig auf ihre Mitglieder geteilt:
1. eigene Mitglieder des Accounts (eigener Login oder Proxy-Account), deren
   Team zum Slot passt
2. Kinder des Accounts via `family_links`, deren Team zum Slot passt
3. eigene Mitglieder des Accounts, unabhängig vom Team

Ein generischer Slot passt zu jedem Team. Eine Eltern-Zuweisung an einem Slot, zu
dem kein Kind passt, zählt für niemanden.

#### Scenario: Zuweisung an vergangenem Termin zählt als geleistet
- **WHEN** ein Kind eine Zuweisung zu einem `duty_slot` hat, dessen `event_date`
  vor dem heutigen Datum liegt
- **THEN** zählt diese Zuweisung zu `geleistet`, unabhängig vom `status`-Feld

#### Scenario: Zuweisung an zukünftigem Termin zählt als Vorhersage
- **WHEN** ein Kind eine Zuweisung zu einem `duty_slot` hat, dessen `event_date`
  heute oder in der Zukunft liegt
- **THEN** zählt diese Zuweisung zu `vorhersage`, nicht zu `geleistet`

#### Scenario: Eltern-Zuweisung wird zwischen passenden Geschwistern geteilt
- **WHEN** ein Elternteil mit zwei Kindern im selben Kader eine Zuweisung auf
  dem eigenen Account an einem Slot dieses Kaders hat
- **THEN** zählt diese Zuweisung für jedes der beiden Kinder zu 0,5

#### Scenario: Eltern-Zuweisung ohne passendes Kind zählt nicht
- **WHEN** ein Elternteil eine Zuweisung an einem Slot einer Mannschaft hat, in
  deren Kader keines seiner Kinder steht
- **THEN** zählt diese Zuweisung für keines seiner Kinder

#### Scenario: Eigene Zuweisung des Kindes zählt voll
- **WHEN** ein Kind über seinen eigenen (Proxy-)Account eine Zuweisung hat, auch
  an einem Slot einer fremden Mannschaft
- **THEN** zählt diese Zuweisung voll für das Kind

### Requirement: Gesamtsumme aus bekannten Dienst-Slots je Kader

Das System SHALL die `Gesamtsumme` eines Kaders in der aktiven Saison aus den
tatsächlich existierenden `duty_slots` berechnen, nicht aus einer Schätzung:
- Team-gebundene Slots (verknüpft über `game_id`→`game_teams` oder direkt über
  `team_id`) zählen voll in die `Gesamtsumme` des zugehörigen Kaders.
- Generische Slots (weder `game_id` noch `team_id` gesetzt, z.B. ein
  Vereinsfest-Dienst) werden anteilig auf alle Kader der Saison verteilt,
  proportional zur Spieleranzahl je Kader. Der resultierende Anteil je Kader darf
  eine Bruchzahl sein.

Die `Gesamtsumme` SHALL live berechnet werden (kein gespeicherter/gecachter Wert)
und wächst über die Saison, sobald neue Termine oder Slots hinzukommen.

#### Scenario: Team-gebundener Slot zählt voll
- **WHEN** ein `duty_slot` über `game_id` einem Spiel zugeordnet ist, das über
  `game_teams` zu Kader K gehört
- **THEN** zählt `slots_total` dieses Slots vollständig in `Gesamtsumme(K)`

#### Scenario: Generischer Slot wird proportional verteilt
- **WHEN** ein generischer `duty_slot` (kein `game_id`, kein `team_id`) mit
  `slots_total = 10` existiert, und Kader K hat 8 von insgesamt 40 Spielern der
  Saison
- **THEN** trägt dieser Slot `10 × 8 / 40 = 2.0` zu `Gesamtsumme(K)` bei

#### Scenario: Neuer Termin erhöht die Gesamtsumme
- **WHEN** für Kader K ein zusätzliches Spiel mit Dienst-Slots angelegt wird
  (z.B. durch H4A-Import oder manuelle Anlage)
- **THEN** steigt `Gesamtsumme(K)` bei der nächsten Berechnung entsprechend, ohne
  dass ein manueller Neuberechnungsschritt nötig ist

### Requirement: Fair-Anteil je Kind

Das System SHALL für jedes Kind eines Kaders denselben Fair-Anteil ausweisen:

```
Fair-Anteil(Kind) = Gesamtsumme(Kader) / Anzahl Spieler im Kader
```

Geschwister im selben Kader werden NICHT dedupliziert — jedes Kind zieht seinen
eigenen vollen Anteil, auch wenn dieselben Eltern dahinterstehen. Ein Spieler ohne
verknüpften Elternteil (`family_links`) gilt für diese Berechnung als eigene
Familie.

#### Scenario: Zwei Geschwister im selben Kader
- **WHEN** Kader K hat 20 Spieler inkl. zweier Geschwister mit denselben Eltern,
  `Gesamtsumme(K) = 60`
- **THEN** beträgt der Fair-Anteil für JEDES der beiden Geschwister-Kinder 3
  (nicht gemeinsam 3, sondern je 3)

#### Scenario: Erwachsener Spieler ohne Elternverknüpfung
- **WHEN** ein Kader-Mitglied hat keinen Eintrag in `family_links`
- **THEN** wird für dieses Mitglied dennoch ein Fair-Anteil berechnet, und das
  Mitglied selbst (statt eines Elternteils) sieht die eigene Zeile

### Requirement: Rangliste — Sortierung und Team-Filter

Die Rangliste-Seite SHALL pro ausgewähltem Team einen eigenen Block mit je einer
Zeile pro Kind dieses Kaders zeigen, absteigend sortiert nach
`geleistet + vorhersage`. Ist mehr als ein Team ausgewählt (Mehrfachauswahl über
den bestehenden Team-Filter), SHALL die Seite für jedes ausgewählte Team einen
separaten Block zeigen — Kinder unterschiedlicher Kader werden NICHT in einer
gemeinsamen Liste vermischt, da ihr Fair-Anteil unterschiedlich ist.

#### Scenario: Absteigende Sortierung innerhalb eines Kaders
- **WHEN** Kader K hat drei Kinder mit `geleistet+vorhersage` = 6, 1, 3
- **THEN** erscheinen sie in der Reihenfolge 6, 3, 1

#### Scenario: Mehrfachauswahl zeigt getrennte Blöcke
- **WHEN** ein Nutzer im Team-Filter zwei Teams auswählt
- **THEN** zeigt die Seite zwei separate, jeweils für sich absteigend sortierte
  Ranglisten-Blöcke, einen pro Team

### Requirement: Rangliste — Sichtbarkeit für Standard-Nutzer

Ein Nutzer ohne Rolle `admin` und ohne Vereinsfunktion `vorstand` SHALL im
Team-Filter der Rangliste ausschließlich Teams zur Auswahl angeboten bekommen, zu
denen er selbst eine Verbindung hat: ein eigenes Kind ist über `family_links` im
Kader dieses Teams, oder er ist selbst als Spieler im Kader dieses Teams.

Innerhalb einer für ihn sichtbaren Rangliste SHALL die Zeile des eigenen Kindes
(bzw. die eigene Zeile, falls er selbst Spieler ist) den echten Namen zeigen. Alle
anderen Zeilen SHALL ausschließlich mit ihrer Platzierung beschriftet werden (z.B.
„Platz 3") — kein Name, kein erfundenes Pseudonym.

#### Scenario: Elternteil sieht nur Teams der eigenen Kinder
- **WHEN** ein Elternteil hat ein Kind im Kader der wCJ, aber kein Kind oder
  eigene Mitgliedschaft in einem anderen Kader
- **THEN** bietet der Team-Filter der Rangliste ausschließlich die wCJ zur Auswahl

#### Scenario: Eigene Zeile ist benannt, fremde nicht
- **WHEN** ein Elternteil öffnet die Rangliste des Kaders seines Kindes
- **THEN** zeigt die Zeile des eigenen Kindes dessen echten Namen
- **THEN** zeigen alle anderen Zeilen ausschließlich eine Platzierung, keinen Namen

#### Scenario: Zugriff auf fremdes Team wird verweigert
- **WHEN** ein Standard-Nutzer versucht, die Rangliste eines Teams abzurufen, zu
  dem er keine Verbindung hat
- **THEN** antwortet das System mit HTTP 403

### Requirement: Rangliste — Sichtbarkeit für Vorstand

Ein Nutzer mit Rolle `admin` oder Vereinsfunktion `vorstand` SHALL im Team-Filter
der Rangliste alle Teams des Vereins zur Auswahl angeboten bekommen, unabhängig von
eigener `family_links`- oder Kader-Zugehörigkeit. Alle Zeilen jeder für ihn
sichtbaren Rangliste SHALL mit dem echten Namen beschriftet werden — keine
Anonymisierung.

#### Scenario: Vorstand sieht alle Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` öffnet den Team-Filter der
  Rangliste
- **THEN** stehen alle aktiven Teams des Vereins zur Auswahl, auch ohne eigene
  Kader-Verbindung

#### Scenario: Vorstand sieht alle Namen
- **WHEN** ein Vorstandsmitglied öffnet die Rangliste eines beliebigen Kaders
- **THEN** zeigt jede Zeile den echten Namen des zugehörigen Kindes/der Familie
