# mannschaftsstrafen Specification

## Purpose

Team-interne Strafenverwaltung („Strichliste") mit per-Kader-Rolle `Strafenwart`, editierbarem Betrags-Catalog, Vergabe/Storno/Reset. Bewusst ohne Zahlungs-Historie: nur der aktuell offene Kassenstand ist die Wahrheit. Sichtbarkeit ist team-intern: Spieler + Trainer + Erweiterter Kader sehen; **Eltern sehen bewusst nicht**.
## Requirements
### Requirement: Strafenwart-Appointment pro Kader
Das System SHALL die Rolle „Strafenwart" als per-Kader-Appointment führen (`kader_strafenwarte`, Sibling von `kader_trainers`), nicht als globalen `member_club_functions`-Wert. Der Trainer des Kaders SHALL Strafenwarte ernennen und abberufen (`admin` passt immer). Ein Member kann in verschiedenen Kadern unabhängig Strafenwart sein.

#### Scenario: Trainer ernennt einen Strafenwart
- **WHEN** ein Trainer des Kaders einen Spieler zum Strafenwart ernennt
- **THEN** antwortet das System mit 200/201 und der Spieler ist Strafenwart dieses Kaders

#### Scenario: Kein neuer globaler Vereinsfunktions-Wert
- **WHEN** der CHECK-Constraint von `member_club_functions` geprüft wird
- **THEN** enthält er keinen Wert `strafenwart` (die Rolle lebt ausschließlich in `kader_strafenwarte`)

#### Scenario: Nicht-Trainer darf nicht ernennen
- **WHEN** ein Spieler ohne Trainer-Rolle versucht, einen Strafenwart zu ernennen
- **THEN** antwortet das System mit HTTP 403

### Requirement: Strafen-Catalog pro Kader
Das System SHALL pro Kader einen Catalog von Strafen (`penalty_types`) mit Grund und Default-Betrag (in Cent) führen, den der Trainer des Kaders pflegt. Der Catalog dient als Vorschlag; der Betrag ist bei der Vergabe editierbar.

#### Scenario: Trainer pflegt Strafen-Catalog
- **WHEN** ein Trainer die Strafe „Trikot vergessen" mit Default-Betrag 500 Cent anlegt
- **THEN** antwortet das System mit 200/201 und die Strafe steht als Vorschlag bereit

#### Scenario: Nicht-Trainer darf Catalog nicht ändern
- **WHEN** ein Spieler ohne Trainer-Rolle versucht, den Strafen-Catalog zu ändern
- **THEN** antwortet das System mit HTTP 403

### Requirement: Strafe vergeben
Das System SHALL es dem Strafenwart des Kaders erlauben, einem Teammitglied eine Strafe mit Betrag (Cent) und Grund zu vergeben (`team_penalties`). Grund und Betrag werden als Snapshot gespeichert und SHALL bei späterem Catalog-Edit unverändert bleiben. Der Betrag SHALL editierbar sein (Default aus Catalog, unterschiedliche Höhe erlaubt). Nur der Strafenwart des betreffenden Kaders SHALL vergeben dürfen; alle anderen erhalten HTTP 403.

#### Scenario: Strafenwart vergibt eine Strafe
- **WHEN** der Strafenwart dem Spieler eine Strafe „Harztopf vergessen" über 1000 Cent vergibt
- **THEN** antwortet das System mit 200/201 und die Strafe erscheint in der Strafenliste des Teams

#### Scenario: Editierbarer Betrag abweichend vom Default
- **WHEN** der Strafenwart eine Catalog-Strafe mit abweichendem Betrag vergibt
- **THEN** wird der abweichende Betrag als Snapshot gespeichert, nicht der Catalog-Default

#### Scenario: Nicht-Strafenwart darf nicht vergeben
- **WHEN** ein Spieler oder Trainer, der nicht Strafenwart des Kaders ist, eine Strafe zu vergeben versucht
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Strafenwart kann nur das eigene Team bestrafen
- **WHEN** ein Strafenwart von Team A eine Strafe für ein Mitglied von Team B zu vergeben versucht
- **THEN** antwortet das System mit HTTP 403 und es wird keine Strafe angelegt

#### Scenario: Catalog-Edit ändert bestehende Strafe nicht
- **WHEN** der Trainer den Default-Betrag einer Catalog-Strafe ändert, nachdem eine Strafe dieses Grundes vergeben wurde
- **THEN** bleibt der Betrag der bereits vergebenen Strafe unverändert

### Requirement: Strafe stornieren und je Spieler zurücksetzen
Das System SHALL dem Strafenwart des Kaders zwei Lösch-Operationen bieten, beide als echtes Löschen (kein Status, kein Strikethrough): **Storno** einer einzelnen Strafe und **Zurücksetzen** aller Strafen eines einzelnen Spielers. Es SHALL keine Zahlungs-Historie geführt werden; nur der aktuell offene Kassenstand bleibt bestehen.

#### Scenario: Storno einer einzelnen Strafe
- **WHEN** der Strafenwart eine einzelne Strafe storniert
- **THEN** wird die Strafe hart gelöscht und verschwindet aus der Liste (kein durchgestrichener Rest)

#### Scenario: Zurücksetzen je Spieler
- **WHEN** der Strafenwart die Strafen eines Spielers zurücksetzt (weil abgegolten)
- **THEN** werden alle Strafen dieses Spielers im Kader hart gelöscht und sein Kassenstand ist 0

#### Scenario: Nicht-Strafenwart darf nicht löschen
- **WHEN** ein Nutzer, der nicht Strafenwart des Kaders ist, eine Strafe zu stornieren oder zurückzusetzen versucht
- **THEN** antwortet das System mit HTTP 403

### Requirement: Teaminternes Read-Gate für Strafen
Das System SHALL die Strafenliste über einen eigenen Endpoint (`GET /api/teams/{id}/penalties`) ausliefern, getrennt von der Roster-Response. Lesen SHALL nur erlaubt sein, wenn der Caller-Member Spieler (`kader_members`), Trainer (`kader_trainers`) oder Erweiterter Kader (`kader_extended_members`) des Kaders der aktiven Saison ist. Eltern (`family_links`) und alle Außenstehenden SHALL HTTP 403 erhalten. Strafen SHALL NICHT als Feld der Roster-Response ausgeliefert werden.

#### Scenario: Spieler des Teams sieht die Strafen
- **WHEN** ein Spieler des Kaders `GET /api/teams/{id}/penalties` aufruft
- **THEN** antwortet das System mit 200 und der Strafenliste inkl. Kassenstand-Summe pro Spieler

#### Scenario: Trainer des Teams sieht die Strafen
- **WHEN** ein Trainer des Kaders die Strafenliste abruft
- **THEN** antwortet das System mit 200 und der Strafenliste

#### Scenario: Erweiterter Kader darf lesen
- **WHEN** ein Mitglied des Erweiterten Kaders die Strafenliste abruft
- **THEN** antwortet das System mit 200 und der Strafenliste

#### Scenario: Elternteil erhält 403
- **WHEN** ein Elternteil mit Team-Zugriff `GET /api/teams/{id}/penalties` aufruft
- **THEN** antwortet das System mit HTTP 403 und liefert keine Strafendaten

#### Scenario: Außenstehender erhält 403
- **WHEN** ein eingeloggter Nutzer ohne Zugehörigkeit zum Kader die Strafenliste abruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Strafen nicht auf der Roster-Response
- **WHEN** ein beliebiger berechtigter Nutzer die Roster-Response lädt
- **THEN** enthält sie keine Strafendaten

### Requirement: Live-Update bei Strafen-Änderung
Jede Strafen-Mutations-Route (vergeben, stornieren, zurücksetzen, Catalog, Ernennung) SHALL `h.hub.Broadcast("penalties")` (oder einen äquivalenten Broadcast-Helfer) aufrufen, und das Frontend SHALL via `useLiveUpdates` reagieren.

#### Scenario: Broadcast nach Vergabe
- **WHEN** der Strafenwart eine Strafe vergibt, storniert oder zurücksetzt
- **THEN** sendet der Handler einen `penalties`-Broadcast und offene, berechtigte Ansichten aktualisieren sich ohne Reload

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

