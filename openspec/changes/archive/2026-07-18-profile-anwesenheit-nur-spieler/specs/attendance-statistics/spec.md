## MODIFIED Requirements

### Requirement: Trainer- und Spieler-Sichten im Frontend

Das Frontend SHALL zwei Sichten bereitstellen:

- **Trainer-/SL-Sicht** unter `/team/:id/anwesenheit`: zeigt oben einen Banner mit der Anzahl offener Erfassungen (Link zur Detail-Liste), darunter eine Tabelle mit dem Stammkader (Spieler, drei Zähler + Quote je für Trainings und Spiele), darunter einen separat überschriebenen Block "Erweiterter Kader (N)" mit gleichem Layout und einer Team-Durchschnittszeile. Tabellen folgen den Projekt-Conventions (brand-Tokens, `lucide-react`-Icons, Mobile-Card-Layout, Touch-Targets ≥ 44px).
- **Spieler-/Eltern-Sicht** als Tab in der Profil-Seite (oder `/profil/anwesenheit`): zeigt für das eigene Mitglied (bzw. das ausgewählte Kind bei Eltern mit mehreren Kindern) die drei Zähler + Quote für Trainings und Spiele getrennt, plus eine tabellarische Liste aller Trainings und aller Spiele im Saisonzeitraum mit Datum, Titel, Status und Begründung.

Die **Spieler-/Eltern-Sicht** SHALL im Profil-Tab und in der Auswahl innerhalb der Sicht nur Mitglieder mit der Vereinsfunktion `spieler` berücksichtigen:

- Der Tab „Anwesenheit" in `/profil` SHALL sichtbar sein, wenn `own_member.club_functions` `spieler` enthält ODER mindestens ein `children[i].club_functions` `spieler` enthält. Andernfalls SHALL der Tab nicht in der Tab-Liste erscheinen.
- Die Auswahl-Buttons in `ProfilAnwesenheitContent` SHALL `own_member` nur einschließen, wenn dessen `club_functions` `spieler` enthält; ein `children[i]` SHALL nur eingeschlossen sein, wenn dessen `club_functions` `spieler` enthält.
- Der Default-`selectedId` beim Laden SHALL das erste Mitglied dieser gefilterten Liste sein (Priorität: eigenes Mitglied vor Kindern).
- Der Trainer-Drilldown-Aufruf `/profil/anwesenheit?member=X` (Parameter `forcedMemberId` an `ProfilAnwesenheitContent`) SHALL diesen Spieler-Filter absichtlich umgehen — der aufrufende Nutzer (Trainer/SL) muss nicht selbst die Funktion `spieler` haben, um die Detailstatistik eines Spielers seines Kaders zu sehen.

Beide Sichten SHALL auf SSE-Event `attendance-changed` neu laden.

#### Scenario: Trainer sieht offene-Erfassungen-Banner

- **WHEN** ein Trainer `/team/:id/anwesenheit` öffnet und `GET /api/teams/{id}/attendance-open` mindestens einen Eintrag liefert
- **THEN** zeigt die Seite oben einen Banner "N offene Erfassungen" mit Link zur Detail-Liste

#### Scenario: Stammkader und erweiterter Kader sind visuell getrennt

- **WHEN** ein Team sowohl Stammkader- als auch erweiterte Kader-Mitglieder hat
- **THEN** zeigt die Trainer-Sicht zwei separate Tabellenblöcke mit eigenen Durchschnittszeilen

#### Scenario: Elternteil mit mehreren Kindern wechselt das Kind

- **WHEN** ein Elternteil mit mehreren verlinkten Kindern die Spieler-Sicht öffnet und ein anderes Kind in der Kind-Auswahl wählt
- **THEN** lädt die Seite die Statistik für die neue `member_id` und ersetzt die Termin-Liste entsprechend

#### Scenario: Live-Update nach Erfassung

- **WHEN** ein Trainer auf der Trainer-Sicht ist und ein anderer Trainer im selben Browser-Cluster `POST /api/games/{id}/attendances` aufruft
- **THEN** sendet der Hub `attendance-changed` und die Seite lädt die Statistik automatisch neu

#### Scenario: Trainer ohne Spieler-Funktion sieht keinen Anwesenheit-Tab im Profil

- **WHEN** ein Nutzer, dessen verlinktes Mitglied nur `trainer` (oder andere Nicht-Spieler-Funktionen) in `club_functions` hat und der keine Spieler-Kinder verknüpft hat, `/profil` öffnet
- **THEN** enthält die Tab-Liste kein „Anwesenheit"

#### Scenario: Elternteil-Trainer sieht nur das Spieler-Kind in der Auswahl

- **WHEN** ein Nutzer mit `own_member.club_functions=[trainer]` und einem verlinkten Kind mit `club_functions=[spieler]` den Anwesenheit-Tab öffnet
- **THEN** ist der Tab sichtbar, und die Auswahl-Buttons enthalten nur das Kind (nicht das eigene Mitglied)

#### Scenario: Trainer-Drilldown funktioniert ohne eigene Spieler-Funktion

- **WHEN** ein Trainer über die Team-Sicht per `openMember`-Klick auf `/profil/anwesenheit?member=42` navigiert, obwohl sein eigenes Mitglied nicht `spieler` in `club_functions` führt
- **THEN** rendert die Seite die Statistik für Mitglied 42 direkt (kein 403, keine leere Auswahl)
