## ADDED Requirements

### Requirement: Wählbare Einheit für Strafen pro Kader
Das System SHALL pro Kader die Einheit für Strafen wählbar machen: `euro` oder `striche` (`penalty_settings.unit`). Default für alle bestehenden und neu angelegten Kader SHALL `euro` sein. Die Einheit SHALL für den gesamten Kader gelten — sowohl für den Strafen-Katalog als auch für alle einzelnen Strafen; gemischte Einheiten innerhalb eines Kaders SHALL nicht möglich sein.

#### Scenario: Neuer Kader startet mit Einheit Euro
- **WHEN** ein Kader angelegt wird
- **THEN** ist die Einheit seiner Strafen auf `euro` gesetzt

#### Scenario: Trainer setzt Einheit auf Striche
- **WHEN** ein Trainer die Einheit des Kaders auf `striche` setzt
- **THEN** antwortet das System mit 200 und alle nachfolgenden Katalog- und Strafen-Beträge werden als Striche-Anzahl interpretiert

#### Scenario: Einheit ist team-intern lesbar
- **WHEN** ein Spieler, Trainer oder Member des Erweiterten Kaders `GET /api/teams/{id}/penalty-settings` aufruft
- **THEN** antwortet das System mit 200 und liefert die aktuelle Einheit

#### Scenario: Nicht-Trainer darf Einheit nicht setzen
- **WHEN** ein Spieler oder Elternteil versucht, die Einheit zu ändern
- **THEN** antwortet das System mit HTTP 403

### Requirement: Einheiten-Wechsel rechnet Beträge um
Beim Wechsel der Einheit (`PUT /api/teams/{id}/penalty-settings`) SHALL das System in einer Transaktion alle Katalog-Default-Beträge und alle vorhandenen `team_penalties`-Beträge dieses Kaders umrechnen. Rate: **`1 € = 1 Strich`** (fest, nicht konfigurierbar). Euro → Striche SHALL Kommazahlen **aufrunden** (`ceil(amount_cent / 100)`). Striche → Euro SHALL exakt multiplizieren (`n * 100`). Der Wechsel SHALL nicht scheitern, weil Strafen existieren — er rechnet um.

#### Scenario: Euro nach Striche rundet auf
- **GIVEN** eine Strafe über 5,50 € (550 Cent) im Kader
- **WHEN** der Trainer die Einheit von `euro` auf `striche` wechselt
- **THEN** wird die Strafe auf 6 Striche aktualisiert (aufgerundet)

#### Scenario: Ganze Euro-Beträge werden verlustfrei umgerechnet
- **GIVEN** eine Strafe über 5,00 € (500 Cent)
- **WHEN** der Trainer die Einheit auf `striche` wechselt
- **THEN** wird die Strafe auf 5 Striche aktualisiert (keine Rundung nötig)

#### Scenario: Striche nach Euro ist exakt
- **GIVEN** eine Strafe über 6 Striche
- **WHEN** der Trainer die Einheit auf `euro` wechselt
- **THEN** wird die Strafe auf 6,00 € (600 Cent) aktualisiert

#### Scenario: Katalog wird mit umgerechnet
- **GIVEN** ein Katalog-Eintrag „Trikot vergessen" mit Default 5,50 €
- **WHEN** die Einheit auf `striche` wechselt
- **THEN** hat der Katalog-Eintrag den neuen Default 6 Striche

#### Scenario: Preview zeigt Delta ohne Mutation
- **WHEN** ein Trainer `GET /api/teams/{id}/penalty-settings/preview?to=striche` aufruft
- **THEN** antwortet das System mit 200 und liefert eine Delta-Liste (betroffene Rows, alte/neue Werte, Anzahl aufgerundet), ohne dass sich Katalog oder Strafen ändern

#### Scenario: Wechsel ist atomar
- **WHEN** die Umrechnungs-Transaktion fehlschlägt (z. B. bei einer inkonsistenten Row)
- **THEN** bleiben Einheit, Katalog und Strafen unverändert (keine Halb-Umrechnung)

### Requirement: Ganzzahl-Erzwingung bei Einheit Striche
Bei aktueller Einheit `striche` SHALL das System sowohl beim Anlegen von Katalog-Einträgen (`POST /api/teams/{id}/penalty-types`) als auch beim Vergeben von Strafen (`POST /api/teams/{id}/penalties`) verlangen, dass der Betrag eine ganze Anzahl Striche ist (d. h. das Feld `amount_cent` durch 100 teilbar ist, weil ein Strich als 100 Cent-Einheiten gespeichert wird). Kommazahlen SHALL mit HTTP 400 abgelehnt werden.

#### Scenario: Ganze Anzahl Striche wird akzeptiert
- **GIVEN** Einheit `striche`
- **WHEN** der Strafenwart eine Strafe über 3 Striche vergibt
- **THEN** antwortet das System mit 200/201

#### Scenario: Kommazahl bei Striche wird abgelehnt
- **GIVEN** Einheit `striche`
- **WHEN** der Strafenwart eine Strafe über „2,5 Striche" (250 Cent) vergibt
- **THEN** antwortet das System mit HTTP 400 und keine Row wird angelegt

#### Scenario: Kommazahl bei Euro bleibt zulässig
- **GIVEN** Einheit `euro`
- **WHEN** der Strafenwart eine Strafe über 2,50 € (250 Cent) vergibt
- **THEN** antwortet das System mit 200/201

### Requirement: Live-Update bei Einheiten-Änderung
Das System SHALL beim Setzen der Einheit einen SSE-Event `penalty-settings` broadcasten. Da die Massen-Umrechnung auch die Beträge in `team_penalties` mutiert, SHALL zusätzlich ein `penalties`-Event gesendet werden, damit Team-interne Clients ihre Strafen-Liste aktualisieren.

#### Scenario: Wechsel triggert beide Events
- **WHEN** ein Trainer die Einheit setzt
- **THEN** werden die SSE-Events `penalty-settings` und `penalties` gesendet
