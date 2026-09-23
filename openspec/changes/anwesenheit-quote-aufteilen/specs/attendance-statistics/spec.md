## MODIFIED Requirements

### Requirement: Trainer- und Spieler-Sichten im Frontend

Das Frontend SHALL zwei Sichten bereitstellen:

- **Trainer-/SL-Sicht** unter `/team/:id/anwesenheit`: zeigt oben einen Banner mit der Anzahl offener Erfassungen (Link zur Detail-Liste), darunter eine Tabelle mit dem Stammkader (Spieler, drei Zähler + Quote je für Trainings und Spiele), darunter einen separat überschriebenen Block "Erweiterter Kader (N)" mit gleichem Layout und einer Team-Durchschnittszeile. Tabellen folgen den Projekt-Conventions (brand-Tokens, `lucide-react`-Icons, Mobile-Card-Layout, Touch-Targets ≥ 44px).
- **Spieler-Sicht (eigenes Mitglied)** als Tab in der eigenen Profil-Seite `/profil` (oder `/profil/anwesenheit`): zeigt für das eigene Mitglied die drei Zähler für Trainings und Spiele getrennt, jeweils mit ihrem Anteil am Gesamt der gezählten Termine (anwesend / entschuldigt / fehlt in Prozent) statt einer zusammengefassten Quote, plus eine tabellarische Liste aller Trainings und aller Spiele im Saisonzeitraum mit Datum, Titel, Status und Begründung.
- **Eltern-Sicht (verlinktes Kind)** als Tab auf der jeweiligen Kind-Detailseite `/profil/kind/:memberId`: dieselbe Statistik für genau dieses Kind. Die Anwesenheit eines Kindes liegt bewusst **auf dessen Kind-Seite**, nicht aggregiert im Eltern-Profil.

Die **Spieler-/Eltern-Sicht** SHALL nur Mitglieder mit der Vereinsfunktion `spieler` berücksichtigen:

- Der Tab „Anwesenheit" in `/profil` SHALL sichtbar sein, genau dann wenn `own_member.club_functions` `spieler` enthält. Andernfalls SHALL der Tab nicht in der Tab-Liste erscheinen. Der Tab zeigt ausschließlich die Statistik des eigenen Mitglieds (`ProfilAnwesenheitContent` mit `forcedMemberId=own_member.id`, keine Auswahl-Buttons).
- Der Tab „Anwesenheit" auf `/profil/kind/:memberId` SHALL sichtbar sein, genau dann wenn `member.club_functions` des Kindes `spieler` enthält, und die Statistik dieses Kindes zeigen (`ProfilAnwesenheitContent` mit `forcedMemberId=member.id`).
- Die eigenständige Seite `/profil/anwesenheit` behält für Nutzer mit mehreren eigenen Spieler-Bezügen die Auswahl-Buttons in `ProfilAnwesenheitContent`: `own_member` nur einschließen, wenn dessen `club_functions` `spieler` enthält; ein `children[i]` nur, wenn dessen `club_functions` `spieler` enthält. Default-`selectedId` ist das erste Mitglied dieser gefilterten Liste (Priorität: eigenes Mitglied vor Kindern).
- Der Trainer-Drilldown-Aufruf `/profil/anwesenheit?member=X` (Parameter `forcedMemberId` an `ProfilAnwesenheitContent`) SHALL den Spieler-Filter absichtlich umgehen — der aufrufende Nutzer (Trainer/SL) muss nicht selbst die Funktion `spieler` haben, um die Detailstatistik eines Spielers seines Kaders zu sehen.

Beide Sichten SHALL auf SSE-Event `attendance-changed` neu laden.

#### Scenario: Trainer sieht offene-Erfassungen-Banner

- **WHEN** ein Trainer `/team/:id/anwesenheit` öffnet und `GET /api/teams/{id}/attendance-open` mindestens einen Eintrag liefert
- **THEN** zeigt die Seite oben einen Banner "N offene Erfassungen" mit Link zur Detail-Liste

#### Scenario: Stammkader und erweiterter Kader sind visuell getrennt

- **WHEN** ein Team sowohl Stammkader- als auch erweiterte Kader-Mitglieder hat
- **THEN** zeigt die Trainer-Sicht zwei separate Tabellenblöcke mit eigenen Durchschnittszeilen

#### Scenario: Elternteil öffnet die Anwesenheit eines Spieler-Kindes auf dessen Kind-Seite

- **WHEN** ein Elternteil `/profil/kind/:memberId` eines Kindes mit `club_functions` `spieler` öffnet und den Tab „Anwesenheit" wählt
- **THEN** ist der Tab vorhanden und zeigt die Statistik genau dieses Kindes (ohne Auswahl-Buttons); der Anwesenheit-Tab im eigenen `/profil` des Elternteils bleibt davon unberührt

#### Scenario: Nicht-Spieler-Kind hat keinen Anwesenheit-Tab auf der Kind-Seite

- **WHEN** ein Elternteil `/profil/kind/:memberId` eines Kindes ohne `spieler` in `club_functions` öffnet
- **THEN** enthält die Tab-Liste der Kind-Seite kein „Anwesenheit"

#### Scenario: Elternteil mit mehreren Kindern wechselt das Kind (Standalone-Seite)

- **WHEN** ein Elternteil mit mehreren verlinkten Spieler-Kindern die Standalone-Seite `/profil/anwesenheit` öffnet und ein anderes Kind in der Kind-Auswahl wählt
- **THEN** lädt die Seite die Statistik für die neue `member_id` und ersetzt die Termin-Liste entsprechend

#### Scenario: Live-Update nach Erfassung

- **WHEN** ein Trainer auf der Trainer-Sicht ist und ein anderer Trainer im selben Browser-Cluster `POST /api/games/{id}/attendances` aufruft
- **THEN** sendet der Hub `attendance-changed` und die Seite lädt die Statistik automatisch neu

#### Scenario: Nutzer ohne eigene Spieler-Funktion sieht keinen Anwesenheit-Tab im eigenen Profil

- **WHEN** ein Nutzer, dessen eigenes Mitglied nur `trainer` (oder andere Nicht-Spieler-Funktionen) in `club_functions` hat, `/profil` öffnet
- **THEN** enthält die Tab-Liste kein „Anwesenheit" — unabhängig davon, ob Spieler-Kinder verknüpft sind (deren Anwesenheit liegt auf der jeweiligen Kind-Seite)

#### Scenario: Elternteil-Trainer sieht die Kind-Anwesenheit auf der Kind-Seite, nicht im eigenen Profil

- **WHEN** ein Nutzer mit `own_member.club_functions=[trainer]` und einem verlinkten Kind mit `club_functions=[spieler]` sein eigenes `/profil` öffnet
- **THEN** enthält die Tab-Liste des eigenen Profils kein „Anwesenheit"; die Anwesenheit des Kindes ist stattdessen als Tab auf `/profil/kind/:memberId` erreichbar

#### Scenario: Trainer-Drilldown funktioniert ohne eigene Spieler-Funktion

- **WHEN** ein Trainer über die Team-Sicht per `openMember`-Klick auf `/profil/anwesenheit?member=42` navigiert, obwohl sein eigenes Mitglied nicht `spieler` in `club_functions` führt
- **THEN** rendert die Seite die Statistik für Mitglied 42 direkt (kein 403, keine leere Auswahl)

#### Scenario: Profil zeigt drei Anteile statt einer Quote

- **WHEN** ein Spieler in der Saison 5 Trainings anwesend war, 3 entschuldigt und 2 unentschuldigt gefehlt hat und den Tab „Anwesenheit" öffnet
- **THEN** zeigt der Block „Trainings" `anwesend 50 %`, `entschuldigt 30 %` und `fehlt 20 %`
- **AND** es erscheint keine einzelne „Quote", die Entschuldigungen aus dem Nenner nimmt
