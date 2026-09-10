## MODIFIED Requirements

### Requirement: Dynamische soll-Berechnung für Elternteil
Der `/api/dashboard`-Endpoint MUST `dutyAccount` für Elternteile als Liste liefern
— eine Position pro verknüpftem Kind mit aktiver Kader-Mitgliedschaft in der
aktiven Saison — statt eines einzelnen aggregierten Werts. Ist der Nutzer selbst
Spieler ohne verknüpftes Kind, enthält die Liste genau eine Position für ihn
selbst.

Für jede Position gilt:
```
soll(Kind) = Gesamtsumme(Kader des Kindes) / Anzahl Spieler im Kader
```
wobei `Gesamtsumme(Kader)` die Summe aller team-gebundenen `duty_slots` der Saison
(über `game_id`→`game_teams` oder direkte `team_id`) zuzüglich des anteiligen,
proportional zur Spieleranzahl verteilten Anteils generischer `duty_slots` (ohne
`game_id` und ohne `team_id`) ist. Geschwister im selben Kader werden NICHT
dedupliziert — jedes Kind erhält seinen eigenen vollen Anteil.

Diese Formel ersetzt die bisherige Schätzung aus
`games_per_season × avg_slots_per_game / player_count / parent_count`: die neue
Basis sind die tatsächlich existierenden Slots der Saison statt einer am Kader
gepflegten Spielanzahl-Schätzung.

#### Scenario: Kind mit einem Elternteil, 20 Spiele, 6 Slots/Spiel, 20 Spieler
- **WHEN** Kader K hat 20 bekannte Spiele mit je 6 team-gebundenen Slots
  (`Gesamtsumme(K) = 120`), 20 Spieler, und ein Elternteil ist mit einem dieser
  Spieler verknüpft
- **THEN** `soll = 120 / 20 = 6` für die Position dieses Kindes — dasselbe Ergebnis
  wie zuvor die Schätzformel für den Einzel-Elternteil-Fall lieferte, jetzt aber
  aus den tatsächlich bekannten Slots statt aus `games_per_season` hergeleitet

#### Scenario: Kind mit zwei Elternteilen
- **WHEN** zwei Elternteile sind mit demselben Kind verknüpft, `soll` dieses
  Kindes beträgt 6
- **THEN** liefert `/api/dashboard` für JEDES der beiden Elternteile eine Position
  mit `soll = 6` — die Formel teilt NICHT mehr durch die Anzahl Elternteile
  (bisheriges Verhalten: 3 pro Elternteil bei zweien); der Fair-Anteil gehört dem
  Kind, nicht dem einzelnen Elternteil-Account

#### Scenario: Zwei Kinder im selben Kader, ein Elternteil
- **WHEN** ein Elternteil ist mit zwei Kindern desselben Kaders verknüpft
- **THEN** liefert `/api/dashboard` ZWEI separate Positionen in `dutyAccount`,
  eine pro Kind, beide mit demselben `soll` dieses Kaders — bisheriges Verhalten
  (Summierung zu einem einzigen Wert) entfällt, da `dutyAccount` jetzt eine Liste
  ist

#### Scenario: Kind in keinem aktiven Kader
- **WHEN** ein verknüpftes Kind hat keinen `kader_members`-Eintrag für die aktive
  Saison
- **THEN** erscheint für dieses Kind keine Position in `dutyAccount`

#### Scenario: games_per_season = 0
- **WHEN** `Gesamtsumme(Kader) = 0` (keine Slots bekannt — der Nachfolgezustand
  von vormals `games_per_season = 0`)
- **THEN** `soll = 0` für alle Kinder dieses Kaders, kein Fehler

#### Scenario: Zwei Kinder desselben Elternteils in unterschiedlichen Kadern
- **WHEN** ein Elternteil hat zwei Kinder, je eines in Kader K1 und Kader K2
- **THEN** liefert `/api/dashboard` zwei Positionen in `dutyAccount`, eine pro
  Kind, mit je eigenem `soll` aus dem jeweiligen Kader

### Requirement: Erklärtext im Frontend
Die Dashboard-Kachel SHALL für jedes Kind (bzw. für den Spieler selbst) eine eigene
Zeile mit einem Segment-Balken zeigen statt eines reinen Zahlentexts:
- ein Segment für `geleistet` (Termin in der Vergangenheit)
- ein Segment für `vorhersage` (Termin in der Zukunft, bereits eingeteilt)
- der verbleibende Rest bis `soll` bleibt unausgefüllt

Die Kachel SHALL auf die Rangliste-Seite des jeweiligen Kaders verlinken. Keine
Formel-Details sind im UI sichtbar.

#### Scenario: soll = 0 (nicht konfiguriert)
- **WHEN** `soll = 0` für ein Kind
- **THEN** zeigt das UI keinen Fortschrittsbalken für dieses Kind, nur den Zähler
  „0 Dienste"

#### Scenario: Balken zeigt beide Segmente
- **WHEN** ein Kind hat `geleistet = 2`, `vorhersage = 1`, `soll = 4`
- **THEN** zeigt der Balken zwei sichtbar unterschiedene Segmente (geleistet,
  vorhersage) und einen Rest von 1 Einheit bis `soll`
