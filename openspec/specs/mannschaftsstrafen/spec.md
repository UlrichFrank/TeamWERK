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
