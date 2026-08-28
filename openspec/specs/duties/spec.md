# duties Specification

## Purpose

Diese Spezifikation beschreibt die Capability `duties`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Duty board (Dienstbörse)
Das System SHALL eine Dienstbörse mit allen Duty-Slots anzeigen. Jeder Slot enthält neben den bisherigen Informationen (event name, date, duty type, vacancies) auch die Liste der eingetragenen Personen mit privacy-gefiltertem Kontaktdaten-Payload. Beim Beanspruchen eines Slots MUSS für Elternteile mit verknüpften Kindern mit Proxy-Account ein „Für wen?"-Selektor erscheinen. Das Beanspruchen eines Slots MUSS race-frei implementiert sein: die Prüfung auf verfügbare Kapazität, das Eintragen des Nutzers und das Aktualisieren des Zählers MÜSSEN als eine atomare Operation erfolgen, die auch bei gleichzeitigen Anfragen korrekt funktioniert.

Die `GET /api/duty-board`-Response gruppiert Slots pro Spiel bzw. pro game-losem Termin. Jede Gruppe SHALL folgende Felder enthalten:

- `game_id` (Integer oder null)
- `team_id` (Integer oder null) — für den Frontend-Team-Filter
- `date`, `event_time`
- `opponent`, `event_type` — bei Spielen aus der `games`-Tabelle; bei game-losen Gruppen ist `opponent` leer und `event_type` SHALL `"generisch"` sein
- `venue` — der Name des Spielorts (`venues.name`) über `games.venue_id`. Bei game-losen Gruppen und bei Spielen ohne gesetztes `venue_id` SHALL das Feld leer sein bzw. entfallen (`omitempty`). Es SHALL nur der **Name** übertragen werden, nicht Straße, Stadt oder PLZ.
- `team_name`, `label`, `past`
- `slots[]` — Liste der Slot-Objekte mit den bisherigen Feldern

Das `venue`-Feld SHALL **keine** eigene Sichtbarkeitsregel einführen: wer eine Gruppe sieht, sieht ihren Ort. Die nachfolgenden Sichtbarkeits- und Audience-Regeln bleiben davon unberührt.

Die Sichtbarkeit der Gruppen wird wie folgt gefiltert:

- System-Rolle `admin`: alle Gruppen aller Teams der aktiven Saison.
- Vereinsfunktion `vorstand`: alle Gruppen aller Teams der aktiven Saison.
- Alle anderen Rollen (Trainer, Sportliche Leitung, Spieler, Eltern): nur Gruppen, deren `team_id` einem Team entspricht, in dem der Nutzer als Spieler (`player_memberships`) ODER als Trainer (`trainer_memberships`) eingetragen ist oder ein verknüpftes Familienmitglied (`family_links`) als Spieler eingetragen ist; zusätzlich game-lose Gruppen, die zu einem Spiel eines dieser Teams gehören.

Die Audience-Filterung auf Slot-Ebene (`audiences`-JSON-Array mit `eltern`/Vereinsfunktionen) erfolgt nach folgender Regel:

- System-Rolle `admin`: kein Audience-Filter (Bypass), unabhängig vom Query-Parameter.
- Privilegierte Vereinsfunktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung`: standardmäßig Audience-Filter aktiv (nur Slots mit NULL-Audience oder Audience-Match zur eigenen Funktion); per Query-Parameter `?audience=all` deaktivierbar.
- Alle anderen Rollen: Audience-Filter immer aktiv, nicht abschaltbar (Query-Parameter `?audience=all` wird ignoriert).

Der Audience-Match prüft pro Slot, ob das `audiences`-Array eines der folgenden Elemente enthält:
- eine der Vereinsfunktionen des Nutzers (`mcf.function`)
- den Wert `'eltern'`, falls der Nutzer mindestens ein verknüpftes Kind (`family_links`) hat, das **im Team des Slots** spielt (`player_memberships.team_id = ds.team_id`); bei game-losen Slots reicht es, wenn das Kind in einem der teilnehmenden Teams des Spiels spielt.

#### Scenario: Gruppe trägt den Spielort
- **WHEN** ein berechtigter Nutzer `GET /api/duty-board` aufruft und die Gruppe gehört zu einem Spiel mit gesetztem `venue_id`
- **THEN** enthält das Gruppen-Objekt `venue` mit dem Wert von `venues.name`

#### Scenario: Spiel ohne Spielort
- **WHEN** die Gruppe zu einem Spiel ohne gesetztes `venue_id` gehört
- **THEN** ist `venue` leer bzw. nicht enthalten, und die Antwort ist im Übrigen unverändert

#### Scenario: Game-lose Gruppe hat keinen Spielort
- **WHEN** ein Dienst-Slot ohne `game_id` (z. B. Vereinsfest) existiert
- **THEN** ist `venue` der zugehörigen Gruppe leer bzw. nicht enthalten

#### Scenario: Der Spielort ändert keine Sichtbarkeit
- **WHEN** derselbe Nutzer `GET /api/duty-board` vor und nach Einführung des `venue`-Feldes aufruft
- **THEN** ist die Menge der zurückgegebenen Gruppen- und Slot-IDs identisch

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, remaining vacancies, and the list of assignees (name + conditionally photo URL, phones, address)

#### Scenario: Vorstand sieht Dienste fremder Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` (System-Rolle `standard`) `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort auch Gruppen für Teams, in denen der Nutzer kein Mitglied ist

#### Scenario: Spieler sieht nur eigene Team-Dienste
- **WHEN** ein Spieler ohne privilegierte Vereinsfunktion `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort nur Gruppen für Teams, in denen der Spieler oder ein Familienmitglied über Kader eingetragen ist

#### Scenario: Trainer sieht Dienste seines trainierten Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer`, der als Trainer (via `kader_trainers`) im Kader von Team A der aktiven Saison eingetragen ist, aber **nicht** als Spieler dort spielt, `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort die Gruppen für Team A
- **AND** enthält **nicht** Gruppen anderer Teams, in denen er weder als Spieler noch als Trainer eingetragen ist

#### Scenario: Sportliche Leitung sieht Dienste ihrer trainierten Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung`, der als Trainer in mehreren Kadern eingetragen ist, `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort Gruppen für alle Teams, in deren Kader der Nutzer als Trainer steht

#### Scenario: Game-lose Gruppe trägt event_type=generisch
- **WHEN** ein Dienst-Slot ohne `game_id` (z. B. Vereinsfest) existiert
- **THEN** enthält die zugehörige Gruppe `event_type: "generisch"` in der API-Response

#### Scenario: Gruppe enthält team_id
- **WHEN** eine team-spezifische Gruppe in der Response erscheint
- **THEN** enthält das Gruppen-Objekt ein numerisches `team_id`-Feld

#### Scenario: Trainer sieht standardmäßig nur Audience-Match
- **WHEN** ein Trainer ohne `?audience`-Query-Parameter `GET /api/duty-board` aufruft, und Team A enthält sowohl Slots mit `audiences=["trainer"]` als auch Slots mit `audiences=["spieler"]`
- **THEN** enthält die Antwort nur die Slots mit `audiences=["trainer"]` (und Slots mit NULL-Audience)
- **AND** enthält **nicht** die Slots mit `audiences=["spieler"]`

#### Scenario: Trainer deaktiviert Audience-Filter
- **WHEN** ein Trainer `GET /api/duty-board?audience=all` aufruft, und Team A enthält Slots mit verschiedenen Audiences
- **THEN** enthält die Antwort **alle** Slots der sichtbaren Gruppen, unabhängig von ihrem Audience-Array

#### Scenario: Vorstand deaktiviert Audience-Filter
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `GET /api/duty-board?audience=all` aufruft
- **THEN** enthält die Antwort alle Slots aller Teams ohne Audience-Filterung

#### Scenario: Spieler kann Audience-Filter nicht deaktivieren
- **WHEN** ein Spieler ohne privilegierte Funktion `GET /api/duty-board?audience=all` aufruft
- **THEN** wird der Query-Parameter ignoriert und der Audience-Filter bleibt aktiv (nur Slots mit Match zur Spieler-Audience oder NULL)

#### Scenario: Eltern-Audience ist team-gescoped
- **WHEN** ein Trainer (gleichzeitig Elternteil eines Kindes in Team B) `GET /api/duty-board` ohne `?audience=all` aufruft, und Team A hat einen Slot mit `audiences=["eltern"]`
- **THEN** ist der Slot **nicht** in der Antwort enthalten — der Eltern-Match greift nur, wenn ein Kind im Slot-Team spielt
- **WHEN** der gleiche Nutzer `GET /api/duty-board?audience=all` aufruft
- **THEN** ist der Slot in der Antwort sichtbar (über die Trainer-Team-Quelle, Audience-Filter deaktiviert)

#### Scenario: Admin sieht immer alle Audiences
- **WHEN** ein Admin `GET /api/duty-board` (ohne Query-Param) aufruft
- **THEN** enthält die Antwort alle Slots ohne Audience-Filterung
- **AND** das Ergebnis ist identisch mit `GET /api/duty-board?audience=all`

#### Scenario: Claim a duty slot — kein Familienmitglied vorhanden
- **WHEN** a user without linked children with proxy accounts claims an open slot
- **THEN** the system records the assignment directly for that user, decrements the vacancy count, updates the claimant's duty account, and the claimant's name appears in the assignee list

#### Scenario: Claim a duty slot — Elternteil mit Kind-Auswahl
- **WHEN** ein Elternteil mit mindestens einem verknüpften Kind mit Proxy-Account auf „Eintragen" klickt
- **THEN** erscheint ein „Für wen?"-Dialog mit dem eigenen Namen als Default und je einem Eintrag pro Kind mit Proxy-Account
- **WHEN** das Elternteil sich selbst auswählt und bestätigt
- **THEN** wird der Dienst dem Elternteil zugebucht (Verhalten wie bisher)
- **WHEN** das Elternteil ein Kind auswählt und bestätigt
- **THEN** wird der Dienst der `user_id` des Kindes zugebucht und das Dienstkonto des Kindes aktualisiert

#### Scenario: Concurrent claim — letzter freier Slot
- **WHEN** zwei Nutzer gleichzeitig den letzten freien Slot beanspruchen
- **THEN** gelingt genau einem der Claim (HTTP 204), der andere erhält HTTP 409
- **THEN** ist `slots_filled` danach exakt gleich `slots_total` (kein Überlauf)

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer shows vacancies but the assignee names remain visible

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

#### Scenario: Unclaim — atomare Gegenbuchung
- **WHEN** ein Nutzer seinen Dienst-Claim aufhebt
- **THEN** wird die `duty_assignments`-Zeile gelöscht UND `slots_filled` dekrementiert in einer einzigen Transaktion
- **THEN** bei einem Datenbankfehler während der Transaktion bleibt der Zähler konsistent (kein partieller State)

#### Scenario: Privacy-gefilterte Assignee-Daten im API-Response
- **WHEN** der `/duty-board`-Endpoint einen Slot mit Assignees zurückgibt
- **THEN** enthält jeder Assignee-Eintrag: `name` (immer), `photo_url` (nur wenn `photo_visible=1`), `phones` (nur wenn `phones_visible=1`, sonst leeres Array), `address` (nur wenn `address_visible=1`, sonst null)
- **THEN** haben Proxy-Account-Assignees keine `phones` und keine `address` (da Proxy-Accounts diese Daten nicht haben)

### Requirement: Audience-Filter-Pille auf Dienste-Seite
Die Dienstbörse-UI (`/dienste`, `web/src/pages/DutyPage.tsx`) SHALL eine zusätzliche Filter-Pille „Nur meine Audience" mit `Filter`-Icon enthalten, die ausschließlich für Nutzer mit mindestens einer der Vereinsfunktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung` sichtbar ist. Die Pille SHALL standardmäßig aktiv sein und ihren Zustand in den URL-Search-Params persistieren: aktiv = kein Param (Default), inaktiv = `?audience=all`. Beim Aufruf von `/api/duty-board` SHALL der Query-Parameter `?audience=all` exakt dann angehängt werden, wenn die Pille deaktiviert ist.

#### Scenario: Pille für Trainer sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille in der Filter-Leiste sichtbar
- **AND** ist sie standardmäßig im aktiven Zustand (gelb hinterlegt)

#### Scenario: Pille für Vorstand sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille in der Filter-Leiste sichtbar

#### Scenario: Pille für Spieler nicht sichtbar
- **WHEN** ein Nutzer ohne privilegierte Vereinsfunktion (nur Spieler oder Elternteil) die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille **nicht** sichtbar

#### Scenario: Default-Zustand erzeugt keine URL-Params
- **WHEN** ein Trainer die Seite ohne Filter-Änderung lädt
- **THEN** enthält die URL keinen `audience`-Parameter
- **AND** wird `GET /api/duty-board` ohne `audience`-Query aufgerufen

#### Scenario: Deaktivierte Pille schreibt audience=all in URL
- **WHEN** ein Trainer die „Nur meine Audience"-Pille deaktiviert
- **THEN** enthält die URL `?audience=all`
- **AND** wird `GET /api/duty-board?audience=all` aufgerufen

#### Scenario: Deep-Link mit audience=all
- **WHEN** ein Trainer eine URL `/dienste?audience=all` öffnet
- **THEN** ist die Audience-Pille im inaktiven Zustand dargestellt
- **AND** zeigt die Liste alle Slots seiner Teams unabhängig vom Audience-Array

### Requirement: Spielbericht-Duty-Slot nur für Presseteam sichtbar
Das System SHALL Duty-Slots des Typs „Spielbericht" (identifiziert per `duty_types.name='Spielbericht'` oder dedizierter Flag) im `GET /api/duty-board`-Response nur für User mit `role IN ('presseteam','admin')` ausliefern. Für andere User wird der Slot herausgefiltert, als wäre er nicht vorhanden.

#### Scenario: Standard-User sieht Slot nicht
- **WHEN** ein User mit `role='standard'` `GET /api/duty-board` aufruft
- **THEN** enthält die Response keinen Slot des Typs „Spielbericht"

#### Scenario: Presseteam sieht Slot
- **WHEN** ein User mit `role='presseteam'` `GET /api/duty-board` aufruft
- **THEN** enthält die Response Slots des Typs „Spielbericht"

### Requirement: Spielbericht-Slot-Ziehen prüft Rolle
Das System SHALL beim `POST /api/duty-slots/{id}/take` prüfen: wenn der Slot vom Typ „Spielbericht" ist, MUSS der Requester `role IN ('presseteam','admin')` haben. Andernfalls HTTP 403 mit `{"error":"role_required"}`.

#### Scenario: Standard-User versucht Ziehen
- **WHEN** ein Standard-User einen „Spielbericht"-Slot per direktem API-Call ziehen will
- **THEN** liefert das System HTTP 403 (Backend-Guard, nicht nur UI-Filter)

### Requirement: Spielbericht-Slot wird auto-regeneriert
Das System SHALL bei jedem Anlegen oder Update eines Spiels mit `event_type IN ('heim','auswärts')` und gesetztem `template_id` automatisch einen Duty-Slot vom Typ „Spielbericht" erzeugen, wenn noch keiner existiert. Slot-`due_at` wird als `game.end_time + 24h` gesetzt (oder `game.date 23:59 + 24h` falls kein end_time). Custom-editierte Slots (`is_custom=1`) werden nicht überschrieben.

#### Scenario: Neues Heimspiel
- **WHEN** ein neues Heim-Spiel mit `template_id` angelegt wird
- **THEN** existiert danach genau ein Duty-Slot vom Typ „Spielbericht" für dieses Spiel

#### Scenario: Kein Slot bei template_id=NULL
- **WHEN** ein generisches Event ohne `template_id` angelegt wird
- **THEN** wird kein Spielbericht-Slot erzeugt

### Requirement: Slot-Erledigung durch Publish
Das System SHALL beim erfolgreichen `POST /api/match-reports/{id}/publish` den zugehörigen Duty-Slot (`match_reports.duty_slot_id`) als erledigt markieren und dem Slot-Owner die Dienstkonto-Gutschrift geben — analog zum manuellen „erledigt"-Klick.

#### Scenario: Publish zählt aufs Dienstkonto
- **WHEN** ein Presseteam-User seinen Bericht erfolgreich publisht
- **THEN** ist der Slot in `duty_slots` erledigt UND das Dienstkonto des Users um den Slot-Wert erhöht

### Requirement: Rotations-Slots ohne Team-Zuordnung bleiben sichtbar und übernehmbar

Ein Duty-Slot aus einem rotations-aktivierten Vorlagen-Item, der wegen erschöpfter Team-Warteschlange ohne Team-Zuordnung entsteht (`duty_slots.team_id = NULL`, `game_id` gesetzt), SHALL über den bestehenden `team_id IS NULL`-Fallback (Sichtbarkeit anhand der Teams des referenzierten Spiels, siehe `GET /api/duty-board`) für Mitglieder, Trainer und Eltern der am Spiel beteiligten Teams sichtbar und übernehmbar bleiben — ohne dass dafür eine gesonderte Sichtbarkeitsregel nötig ist.

#### Scenario: Unzugeordneter Rotations-Slot ist für das Spielteam sichtbar

- **WHEN** ein Regen-Lauf für ein Heimspiel von Team X einen Rotations-Slot ohne Team-Zuordnung erzeugt (Cap-Überlauf)
- **AND** ein Elternteil eines Spielers von Team X ruft `GET /api/duty-board` auf
- **THEN** erscheint der Slot in der Gruppe dieses Spiels, für den Elternteil sichtbar und übernehmbar (vorbehaltlich der bestehenden Audience-Filterung des Items)

### Requirement: Ein Diensttyp kann eine dynamische Dauer haben

Ein Diensttyp SHALL einen Dauer-Modus tragen: `absolut` oder `dynamisch`. Im Modus
`absolut` SHALL sich nichts gegenüber dem bisherigen Verhalten ändern — die Dauer ist die
gepflegte Stundenzahl.

Im Modus `dynamisch` SHALL das Ende des Dienstes über einen **End-Anker** und einen
**End-Versatz** bestimmt werden, mit denselben zwei Ankern wie der Start: `start`
(Anpfiff) und `end` (Spielende). Der Versatz SHALL in beide Richtungen zulässig sein.
Die Dauer SHALL dort ausschließlich die Differenz aus aufgelöster End- und Startzeit
sein; die gepflegte Stundenzahl SHALL keine Rolle spielen. Trägt der Diensttyp zusätzlich
das Kennzeichen `end_at_next_duty`, SHALL das so aufgelöste Ende als **Deckel** wirken und
durch eine Ablösung nach vorn gezogen werden können (siehe Requirement „Ein Dienst kann
bei Ablösung enden").

Die Auflösung des End-Ankers SHALL identisch zur Auflösung des Start-Ankers sein:
`end` verwendet die gepflegte Endzeit des Termins, und andernfalls den Anpfiff zuzüglich
der ermittelten Spieldauer.

Die Vorlagen-Zeile SHALL Modus, End-Anker und End-Versatz wie die übrigen Item-Felder
per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig führen.

Die Oberfläche SHALL die beiden Modi nach der Entscheidung benennen, die sie treffen
lassen — `absolut` als **„Startzeit + Dauer"**, `dynamisch` als **„Startzeit + Endzeit"** —
und die Felder in der Reihenfolge der Rechnung zeigen: Modus, dann **Start-Anker** und
**Start-Versatz**, darunter je nach Modus die **Dauer** oder **End-Anker** und
**End-Versatz**. Im Modus `dynamisch` SHALL kein Dauer-Feld angeboten werden. Die
gespeicherten Werte (`absolut`/`dynamisch`) SHALL die Umbenennung nicht berühren.

#### Scenario: Dienst dauert so lang wie das Spiel

- **WHEN** ein Diensttyp im Modus `dynamisch` bei Anpfiff −30 min startet und bei
  Spielende +15 min endet
- **AND** zwei Termine unterschiedlicher Altersklassen dieselbe Vorlage nutzen
- **THEN** tragen die erzeugten Slots **unterschiedliche** Dauern
- **AND** entspricht jede der Spieldauer des jeweiligen Termins zuzüglich der Versätze

#### Scenario: Gepflegte Endzeit hat Vorrang

- **WHEN** der Termin eine eigene Endzeit trägt und der End-Anker `end` ist
- **THEN** rechnet die Dauer gegen diese Endzeit
- **AND** nicht gegen Anpfiff zuzüglich der Spieldauer

#### Scenario: Halbzeit-Dienst über zwei Anpfiff-Versätze

- **WHEN** Start und Ende beide am Anker `start` hängen, mit +25 min und +40 min
- **THEN** ist die Dauer des Slots 15 Minuten
- **AND** bleibt sie unabhängig von der Spieldauer des Termins

#### Scenario: Absoluter Modus bleibt unverändert

- **WHEN** ein Diensttyp im Modus `absolut` steht
- **THEN** trägt der erzeugte Slot exakt die gepflegte Stundenzahl
- **AND** spielen End-Anker und End-Versatz keine Rolle

#### Scenario: Kein Dauer-Feld im Modus „Startzeit + Endzeit"

- **WHEN** ein Vorstand in der Diensttyp- oder Vorlagen-Maske auf „Startzeit + Endzeit"
  umschaltet
- **THEN** verschwindet das Dauer-Feld
- **AND** erscheinen stattdessen End-Anker und End-Versatz unter den Start-Feldern

### Requirement: Die Dauer eines Diensttyps ist positiv

`POST /api/duty-types` und `PUT /api/duty-types/{id}` SHALL eine explizit gesendete
`hours_value` ≤ 0 mit HTTP 400 abweisen, **vor** jedem Schreibvorgang. Fehlt das Feld,
SHALL derselbe Default wie in der DB-Spalte gelten (1.0) — dieselbe Regel, die beide
Routen für `default_anchor`, `duration_mode` und die Verhaltensfelder schon anwenden.

Begründung: Die Zusage „ein Slot trägt nach jedem Regen-Lauf eine Dauer > 0" war über den
Diensttyp umgehbar. Slot- und Vorlagen-Routen prüfen jeweils nur ihre **eigene** Eingabe;
eine per Copy-on-pick geerbte Dauer sendet niemand explizit, sie wandert stumm vom Typ in
die Vorlagen-Zeile und von dort in den Slot. Der Diensttyp war die letzte Schreibstelle
ohne diese Prüfung.

#### Scenario: Dauer null wird abgewiesen

- **WHEN** ein Vorstand einen Diensttyp mit `hours_value: 0` anlegt oder speichert
- **THEN** antwortet der Server mit HTTP 400
- **AND** ist nichts persistiert — bei `PUT` bleibt auch der mitgesendete Name unverändert

#### Scenario: Fehlendes Feld ergibt den Spalten-Default

- **WHEN** ein Vorstand einen Diensttyp ohne `hours_value` anlegt
- **THEN** trägt der Diensttyp die Dauer 1,0 Stunden
- **AND** nicht die Null aus dem leeren Request

### Requirement: Manuell angelegte Dienste bleiben absolut

Ein per `POST /api/duty-slots` angelegter Dienst SHALL eine absolute Dauer tragen. Der
Dauer-Modus SHALL für solche Slots nicht angeboten werden, da sie `is_custom=1` tragen und
vom Regen nie erneuert werden — eine als dynamisch etikettierte Dauer würde dort nie
nachgeführt.

Das Anlege-Formular SHALL die Dauer aus einer dynamischen Typ-Definition jedoch berechnen
und vorbelegen dürfen.

Der Termin-Dialog SHALL beim Anlegen **und** beim Bearbeiten eines Dienstes darauf
hinweisen, dass der Dienst dadurch als manuell gepflegt gilt und von der automatischen
Regeneration nicht mehr angefasst wird. Beim Bearbeiten eines bisher automatisch
erzeugten Dienstes ist das eine Nebenwirkung des Speicherns und SHALL vor dem Speichern
sichtbar sein.

#### Scenario: Dienst mit dynamischem Typ von Hand hinzufügen

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Diensttyp im Modus `dynamisch`
  auswählt
- **THEN** wird die Dauer aus Anker und Versätzen gegen diesen Termin berechnet vorbelegt
- **AND** ist der so entstandene Slot danach eine feste Zahl, die der Regen nicht anrührt

#### Scenario: Hinweis auf die Herausnahme aus der Regeneration

- **WHEN** ein Vorstand einen automatisch erzeugten Dienst im Termin-Dialog bearbeitet
- **THEN** steht im Dialog, dass der Dienst danach manuell gepflegt ist und nicht mehr
  automatisch regeneriert wird

### Requirement: Ein Dienst-Slot trägt seine eigene Dauer

Jeder `duty_slots`-Eintrag SHALL eine Spalte `hours_value` (`REAL`, Stunden, `NOT NULL`,
`> 0`) tragen. Dieser Wert SHALL zugleich die angezeigte Dauer **und** die Gutschrift auf
das Dienstkonto sein — es SHALL keine zweite Zahl für dieselbe Größe geben.

Der Wert SHALL beim Erzeugen des Slots **materialisiert** werden, wie `event_time`,
`slots_total` und `audiences` es bereits werden. Ein Slot SHALL zur Laufzeit weder die
Vorlage noch den Diensttyp nach seiner Dauer befragen.

`duty_accounts.ist` SHALL aus `SUM(duty_slots.hours_value)` der `fulfilled`-Zuweisungen
aggregiert werden, nicht aus `duty_types.hours_value`.

#### Scenario: Dauer eines Slots weicht vom Diensttyp ab

- **WHEN** ein Vorstand die Dauer eines Slots von 1,0 auf 2,0 Stunden ändert
- **THEN** zeigt die Dienstbörse für diesen Slot die längere Spanne
- **AND** rechnet eine Neuberechnung von `duty_accounts.ist` mit 2,0 Stunden für diesen Slot
- **AND** bleibt die Dauer des zugrundeliegenden Diensttyps unverändert

#### Scenario: Spätere Änderung am Diensttyp erreicht bestehende Slots nicht

- **WHEN** die Dauer eines Diensttyps nach dem Erzeugen von Slots geändert wird
- **THEN** behalten die bestehenden Slots ihre materialisierte Dauer
- **AND** tragen erst neu erzeugte Slots den geänderten Wert

### Requirement: Dienstbörse zeigt Start- und Endzeit

Wo heute der Startzeitpunkt eines Dienst-Slots angezeigt wird, SHALL eine Zeitspanne
`Start–Ende` erscheinen, gebildet aus `event_time` und `hours_value`. Das betrifft die
Dienstbörse (`/dienste`) und die Dienst-Liste im Spieltag-Detail-Modal (`/kalender`).

Die Spanne SHALL mit einem Halbgeviertstrich getrennt und **ohne** nachgestelltes „Uhr"
dargestellt werden. Läuft ein Dienst über Mitternacht, SHALL die Endzeit als Uhrzeit des
Folgetags **ohne** Datumszusatz erscheinen. Trägt ein Slot **kein** `event_time`, SHALL die
bisherige Platzhalter-Darstellung erhalten bleiben.

Die Anstoßzeit des Spiels in der Spieltag-Kopfzeile SHALL unverändert als Zeitpunkt
angezeigt werden — sie ist keine Dienstzeit.

#### Scenario: Dienst mit Startzeit und Dauer

- **WHEN** ein Slot `event_time = 08:00` und eine Dauer von 1,0 Stunden trägt
- **THEN** zeigt die Dienstbörse `8:00–9:00`

#### Scenario: Dienst läuft über Mitternacht

- **WHEN** ein Slot `event_time = 23:30` und eine Dauer von 1,0 Stunden trägt
- **THEN** zeigt die Dienstbörse `23:30–00:30`
- **AND** erscheint kein Datumszusatz

#### Scenario: Dienst ohne Startzeit

- **WHEN** ein Slot kein `event_time` trägt
- **THEN** bleibt die Darstellung der bisherige Platzhalter
- **AND** wird keine Spanne konstruiert

### Requirement: Dauer ist beim Anlegen und Bearbeiten eines Slots editierbar

Die Modale „Dienst hinzufügen" und „Dienst bearbeiten" SHALL neben Uhrzeit und Personenzahl
ein Feld für die Dauer anbieten. `POST /api/duty-slots` und `PUT /api/duty-slots/{id}` SHALL
`hours_value` entgegennehmen.

Beide Routen SHALL `hours_value <= 0` mit HTTP 400 ablehnen. Fehlt `hours_value` im Request
von `POST /api/duty-slots`, SHALL der Server die Dauer des angegebenen `duty_type_id`
einsetzen statt 0 zu speichern.

`PUT /api/duty-slots/{id}` SHALL wie bisher `is_custom=1` setzen — auch dann, wenn
ausschließlich die Dauer geändert wurde.

#### Scenario: Dauer beim Anlegen mitgeben

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Slot mit Dauer anlegt
- **THEN** antwortet die Route mit 201
- **AND** trägt der Slot die angegebene Dauer und `is_custom = 1`

#### Scenario: Alt-Client sendet keine Dauer

- **WHEN** `POST /api/duty-slots` ohne `hours_value` aufgerufen wird
- **THEN** trägt der angelegte Slot die Dauer seines Diensttyps
- **AND** wird nicht 0 gespeichert

#### Scenario: Unzulässige Dauer

- **WHEN** `hours_value` als 0 oder negativ gesendet wird
- **THEN** antwortet die Route mit HTTP 400
- **AND** bleibt der Slot unverändert

### Requirement: Dienst hinzufügen belegt Dauer und Uhrzeit aus dem Diensttyp vor

Wählt ein Vorstand im Modal „Dienst hinzufügen" einen Diensttyp, SHALL das Formular Dauer
und Uhrzeit aus diesem Typ vorbelegen — die Uhrzeit berechnet aus `default_anchor` und
`default_offset_minutes` gegen die Zeit des Termins. Beide Werte SHALL editierbar bleiben.
Die bestehende Vorbelegung der Zielgruppen SHALL erhalten bleiben.

#### Scenario: Diensttyp auswählen

- **WHEN** ein Vorstand im Modal einen Diensttyp mit Anker „Start" und Versatz −30 Minuten
  an einem Termin um 10:00 auswählt
- **THEN** steht im Uhrzeit-Feld 09:30
- **AND** steht im Dauer-Feld die Dauer dieses Diensttyps
- **AND** sind beide Felder weiterhin überschreibbar

### Requirement: Vorlagen-Zeile trägt eine eigene Dauer (Copy-on-pick)

`game_template_items` SHALL eine Spalte `hours_value` (`REAL`, `NOT NULL`) tragen. Der
Vorlagen-Editor SHALL sie beim Auswählen des Diensttyps aus dessen Wert vorbelegen — auf
demselben Weg, auf dem er heute `anchor` und `offset_minutes` vorbelegt. Danach SHALL die
Zeile eigenständig sein: ein leeres Feld als „erben" SHALL es **nicht** geben.

Ein aus der Vorlage erzeugter Slot SHALL die Dauer der **Vorlagen-Zeile** übernehmen.

#### Scenario: Diensttyp in der Vorlage auswählen

- **WHEN** ein Vorstand in einer Vorlagen-Zeile einen Diensttyp auswählt
- **THEN** wird das Dauer-Feld der Zeile mit der Dauer dieses Typs gefüllt
- **AND** kann es anschließend abweichend gesetzt werden

#### Scenario: Vorlage weicht vom Diensttyp ab

- **WHEN** eine Vorlagen-Zeile eine von ihrem Diensttyp abweichende Dauer trägt
- **THEN** tragen die daraus erzeugten Slots die Dauer der Vorlagen-Zeile

### Requirement: Bei reduzierter Variante bestimmt der Varianten-Typ die Dauer

Ersetzt die Varianten-Logik (`same_day_behavior` oder `adjacent_day_behavior` = `reduced`)
den Diensttyp eines Vorlagen-Items durch den hinterlegten Varianten-Typ, SHALL der erzeugte
Slot die Dauer des **Varianten-Typs** tragen. Die in der Vorlagen-Zeile hinterlegte Dauer
SHALL in diesem Fall verworfen werden.

Begründung: Die Vorlage bestimmt Position und Umfang eines Dienstes, der Diensttyp bestimmt
die Art der Arbeit — die Dauer ist eine Eigenschaft der Arbeit, nicht der Platzierung.

Position (`anchor`, `offset_minutes`) und Personenzahl (`slots_count`) SHALL unverändert aus
der Vorlagen-Zeile stammen.

#### Scenario: Mehrfachspieltag löst Variante aus

- **WHEN** an einem Mehrfachspieltag `same_day_behavior = 'reduced'` greift und den
  Diensttyp durch seine Variante ersetzt
- **THEN** trägt der erzeugte Slot die Dauer des Varianten-Typs
- **AND** nicht die Dauer der Vorlagen-Zeile
- **AND** stammen Uhrzeit und Personenzahl weiterhin aus der Vorlagen-Zeile

### Requirement: Eine Dauer-Änderung erhält bestehende Zusagen

Eine geänderte Dauer SHALL den Zeitpunkt eines Slots nicht verschieben und damit den
Wiederherstellungs-Schlüssel `(duty_type_id, event_time, team_id)` nicht berühren. Ein
Regen-Lauf nach einer Dauer-Änderung SHALL alle Zusagen wiederherstellen.

#### Scenario: Regen nach Dauer-Änderung in der Vorlage

- **WHEN** die Dauer einer Vorlagen-Zeile geändert und ein Regen-Lauf ausgelöst wird
- **THEN** tragen die neu erzeugten Slots die geänderte Dauer
- **AND** sind alle zuvor bestehenden Zusagen wiederhergestellt
- **AND** erscheint keine „entfernt"-Benachrichtigung

### Requirement: Eine unmögliche Zeitspanne wird abgewiesen

`POST /api/duty-types`, `PUT /api/duty-types/{id}` und `PUT /api/duty-templates/{id}`
SHALL im Modus `dynamisch` eine Kombination abweisen, deren Ende **an keinem Termin**
nach dem Start liegen kann: Start-Anker und End-Anker sind gleich **und** der End-Versatz
ist kleiner oder gleich dem Start-Versatz. Die Antwort SHALL HTTP 400 sein, geprüft
**vor** jedem Schreibvorgang — bei `PUT` bleibt der Bestand vollständig unverändert.

Bei **verschiedenen** Ankern SHALL nicht validiert werden. Die Dauer hängt dort von der
Spieldauer des konkreten Termins ab, die zum Pflegezeitpunkt nicht feststeht: „Start bei
Anpfiff, Ende 15 min vor Spielende" ist eine gültige Definition, die bei jedem
hinreichend langen Spiel eine positive Dauer ergibt.

Im Modus `absolut` SHALL die Prüfung nicht greifen — End-Anker und End-Versatz sind dort
bedeutungslos.

Das Frontend SHALL dieselbe Regel anwenden und das Speichern mit einer Meldung am Feld
blockieren, statt den Server antworten zu lassen.

#### Scenario: Gleicher Anker, Ende nicht nach dem Start

- **WHEN** ein Vorstand einen Diensttyp im Modus `dynamisch` mit Start-Anker `start`
  +40 min und End-Anker `start` +25 min speichert
- **THEN** antwortet der Server mit HTTP 400
- **AND** ist nichts persistiert

#### Scenario: Verschiedene Anker bleiben erlaubt

- **WHEN** ein Vorstand Start-Anker `start` +0 min und End-Anker `end` −15 min speichert
- **THEN** wird die Definition angenommen
- **AND** ergibt sie bei jedem Termin, dessen Spieldauer 15 Minuten übersteigt, eine
  positive Dauer

#### Scenario: Absoluter Modus ist von der Prüfung nicht betroffen

- **WHEN** dieselbe unmögliche Anker-/Versatz-Kombination im Modus `absolut` gespeichert
  wird
- **THEN** wird sie angenommen
- **AND** trägt der erzeugte Slot die gepflegte Stundenzahl

#### Scenario: Vorlagen-Zeile mit unmöglicher Spanne

- **WHEN** eine Vorlage mit einem Eintrag gespeichert wird, dessen Spanne unmöglich ist
- **THEN** antwortet der Server mit HTTP 400
- **AND** bleiben die bestehenden Einträge der Vorlage unverändert

### Requirement: Eine unauflösbare dynamische Dauer erzeugt keinen Slot

Ergibt die dynamische Auflösung gegen einen konkreten Termin keine positive Dauer, SHALL
für diesen Termin **kein** Slot entstehen. Es SHALL keinen Rückfall auf die gepflegte
Stundenzahl geben.

Der Ausfall SHALL in der Regen-Zusammenfassung als eigener Eintrag mit Datum und
Diensttyp erscheinen (`invalid_span`), getrennt von `skipped` — `skipped` meint eine
gewollte Auslassung der Varianten-Logik, `invalid_span` einen Definitionsfehler. Die
Oberfläche SHALL ihn entsprechend als Fehler ausweisen.

Eine Zusage auf einem dadurch entfallenen Bestandsslot SHALL wie jeder andere entfallene
Dienst behandelt werden: Der Helfer wird über die Entfernung benachrichtigt.

#### Scenario: Spieldauer lässt die Spanne zusammenschrumpfen

- **WHEN** ein Dienst bei Spielende −0 min startet und bei Anpfiff +30 min enden soll
- **AND** das Spiel 60 Minuten dauert
- **THEN** entsteht kein Slot
- **AND** meldet die Zusammenfassung einen `invalid_span`-Eintrag mit Datum und Diensttyp

#### Scenario: Kein Rückfall auf die gepflegte Stundenzahl

- **WHEN** derselbe Termin regeneriert wird und der Diensttyp eine `hours_value` von 2
  Stunden trägt
- **THEN** entsteht kein Slot mit 2 Stunden Dauer

#### Scenario: Helfer auf einem entfallenen Dienst wird benachrichtigt

- **WHEN** auf dem betroffenen Bestandsslot eine Zusage lag
- **THEN** erhält der Helfer die reguläre „Dienst wurde entfernt"-Meldung

### Requirement: Ein Dienst kann bei Ablösung enden

Ein Diensttyp und eine Vorlagen-Zeile SHALL ein Kennzeichen `end_at_next_duty` tragen
können. Ist es gesetzt **und** steht der Dauer-Modus auf `dynamisch`, SHALL das Ende des
erzeugten Slots der **frühere** der beiden folgenden Zeitpunkte sein:

- der Start des nächsten gleichartigen Dienstes am selben Spieltag, oder
- das aus End-Anker und End-Versatz aufgelöste Ende (der **Deckel**).

Existiert kein solcher Nachfolger, SHALL der Deckel unverändert gelten. Die Ablösung
SHALL ein Ende nur nach vorn ziehen können, nie nach hinten.

**Gleichartig** SHALL bedeuten: derselbe Diensttyp, unter dem der Slot tatsächlich
entsteht — also nach einer eventuellen Varianten-Reduktion. Ein Termin ohne Slot dieses
Diensttyps SHALL den Vorgänger **nicht** verlängern.

Als Nachfolger SHALL ausschließlich ein Slot in Frage kommen, dessen Startzeit **nach**
der eigenen liegt. Ein zeitgleicher oder früher liegender Slot SHALL nicht ablösen; in
diesem Fall greift der Deckel.

Die Ablösung SHALL sich auf **alle** an diesem Tag bestehenden Dienst-Slots stützen,
unabhängig davon, wie sie entstanden sind: manuell angelegte Dienste (`is_custom=1`) und
Dienste an Terminen, die von einer Massenregeneration ausgenommen wurden, lösen ebenso ab
wie automatisch erzeugte. Gekappt SHALL dagegen nur werden, was das Kennzeichen selbst
trägt — manuell angelegte Dienste behalten ihre Dauer.

Greift die Varianten-Logik, SHALL das Kennzeichen des **Varianten-Diensttyps** gelten, wie
bei Modus, End-Anker und End-Versatz auch.

Im Modus `absolut` SHALL das Kennzeichen bedeutungslos sein — es SHALL gespeichert, aber
nicht angewendet werden, damit ein Moduswechsel hin und zurück den Wert nicht verliert.

Ein gesetztes Kennzeichen SHALL **keine** zusätzliche Eingabe-Validierung nach sich
ziehen: Es kann eine Definition nicht unmöglich machen, weil es eine Dauer ausschließlich
verkürzt und nur Nachfolger nach dem eigenen Start berücksichtigt. Die gekappte Dauer
SHALL daher immer positiv bleiben, und die Kappung SHALL nie dazu führen, dass ein bereits
entstandener Slot wieder entfällt.

Die Oberfläche SHALL das Kennzeichen als Häkchen unter End-Anker und End-Versatz
anbieten und nur im Modus „Startzeit + Endzeit" zeigen. Die Vorlagen-Zeile SHALL es wie
die übrigen Item-Felder per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig
führen.

Die Einzeltermin-Vorschau SHALL den Deckel zeigen. Sie rechnet gegen einen einzelnen
Termin und kann die Kette nicht kennen; ihre Angabe ist damit eine obere Schranke.

#### Scenario: Der Nachfolger löst ab

- **WHEN** an einem Spieltag zwei Slots desselben Diensttyps entstehen, der erste mit
  einem Deckel um 12:00 Uhr, der zweite mit Startzeit 11:15 Uhr
- **AND** der erste Diensttyp trägt das Kennzeichen
- **THEN** endet der erste Slot um 11:15 Uhr
- **AND** entspricht seine Dauer der Spanne von seiner Startzeit bis 11:15 Uhr

#### Scenario: Der Letzte in der Kette behält den Deckel

- **WHEN** kein weiterer Slot desselben Diensttyps an diesem Tag nach diesem beginnt
- **THEN** endet der Slot zum aufgelösten Ende aus End-Anker und End-Versatz
- **AND** ist seine Dauer identisch zu der ohne gesetztes Kennzeichen

#### Scenario: Nachfolger liegt hinter dem Deckel

- **WHEN** der nächste gleichartige Dienst erst nach dem aufgelösten Ende beginnt
- **THEN** bleibt es beim Deckel
- **AND** wird die Dauer nicht verlängert

#### Scenario: Rückwärts liegender Slot löst nicht ab

- **WHEN** ein Slot desselben Diensttyps am selben Tag **vor** dem eigenen Start beginnt
- **THEN** wird er als Ablösung nicht berücksichtigt
- **AND** greift der Deckel
- **AND** bleibt der Slot bestehen

#### Scenario: Nur derselbe Diensttyp löst ab

- **WHEN** zwischen zwei Bewirtungsdiensten ein Slot eines anderen Diensttyps beginnt
- **THEN** kappt dieser den Bewirtungsdienst nicht

#### Scenario: Heimspiel ohne gleichartigen Dienst verlängert nicht

- **WHEN** auf den letzten Bewirtungsdienst eines Tages noch zwei Heimspiele folgen, an
  denen die Rotation keinen Bewirtungsdienst zugeteilt hat
- **THEN** endet der Bewirtungsdienst an seinem Deckel
- **AND** wird er nicht bis zu diesen Spielen verlängert

#### Scenario: Manuell angelegter Dienst löst ab, wird aber nicht gekappt

- **WHEN** am selben Tag ein Dienst desselben Diensttyps von Hand angelegt wurde
  (`is_custom=1`)
- **THEN** löst er einen davorliegenden automatisch erzeugten Dienst ab
- **AND** behält er selbst seine gepflegte Dauer

#### Scenario: Ausgenommener Termin löst trotzdem ab

- **WHEN** ein Massenlauf einen Termin ausnimmt, an dem ein Slot desselben Diensttyps
  bestehen bleibt
- **THEN** zählt dieser Slot als Ablösung für die neu erzeugten Dienste des Tages

#### Scenario: Variante bestimmt das Kennzeichen

- **WHEN** die Varianten-Logik greift und der Varianten-Diensttyp das Kennzeichen **nicht**
  trägt
- **THEN** wird der erzeugte Slot nicht gekappt
- **AND** gilt allein sein aufgelöstes Ende

#### Scenario: Absoluter Modus ist nicht betroffen

- **WHEN** ein Diensttyp mit gesetztem Kennzeichen im Modus `absolut` steht
- **THEN** trägt der erzeugte Slot exakt die gepflegte Stundenzahl
- **AND** bleibt das Kennzeichen gespeichert

#### Scenario: Kappung lässt nie einen Dienst entfallen

- **WHEN** ein Slot durch die Ablösung gekappt wird
- **THEN** ist seine Dauer weiterhin größer als null
- **AND** entsteht kein `invalid_span`-Eintrag
- **AND** wird keine darauf liegende Zusage als entfernt gemeldet

#### Scenario: Häkchen nur im dynamischen Modus

- **WHEN** ein Vorstand in der Diensttyp- oder Vorlagen-Maske auf „Startzeit + Dauer"
  umschaltet
- **THEN** verschwindet das Häkchen
- **AND** bleibt der gespeicherte Wert beim Zurückschalten erhalten

### Requirement: Ein spielgebundener Dienst-Slot trägt kein Team

Ein Dienst-Slot mit gesetztem `game_id` SHALL `team_id = NULL` tragen — unabhängig davon,
wie viele Teams dem Termin zugeordnet sind und über welchen Weg er entsteht (Vorlagen-Regen,
Bewirtungsrotation, `POST /api/duty-slots`, Massenlauf, H4A-Import). `POST /api/duty-slots`
SHALL ein mitgeschicktes `team_id` bei gesetztem `game_id` ignorieren statt es abzulehnen;
das Frontend SHALL es nicht mehr senden.

Die Sichtbarkeit eines spielgebundenen Slots SHALL ausschließlich über `game_id` →
`game_teams` aufgelöst werden. Die Leseabfragen SHALL `ds.team_id` bei gesetztem `game_id`
nicht auswerten — auch nicht, wenn dort noch ein Bestandswert steht. Damit sind migrierte
und nicht migrierte Zeilen ununterscheidbar sichtbar.

`duty_slots.team_id` SHALL für Slots **ohne** `game_id` unverändert der Geltungsbereich
bleiben (Vereinsfest o. ä.): dort bestimmt es allein, wer den Slot sieht, und ein Slot ohne
Team und ohne Spiel bleibt nur für Vorstand/Admin sichtbar.

#### Scenario: Slot an Termin mit drei Teams ist für alle drei sichtbar
- **WHEN** ein Vorstand an einem Termin mit den Teams A, B und C einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` und gesetztem `game_id` gespeichert
- **AND** erscheint er in `GET /api/duty-board` für Spieler, Trainer und Eltern aller drei Teams

#### Scenario: Slot an Termin mit genau einem Team trägt ebenfalls kein Team
- **WHEN** ein Vorstand an einem Termin mit nur Team A einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` gespeichert
- **AND** sehen Mitglieder von Team A ihn, Mitglieder anderer Teams nicht

#### Scenario: Mitgeschicktes team_id wird ignoriert
- **WHEN** ein Client `POST /api/duty-slots` mit `game_id` UND `team_id` aufruft
- **THEN** antwortet die API mit 201 und der gespeicherte Slot trägt `team_id = NULL`

#### Scenario: Bestands-Slot mit alter team_id bleibt für alle Teams des Termins sichtbar
- **WHEN** ein vor der Migration angelegter Slot noch `team_id` eines der Teams trägt
- **THEN** sehen ihn Mitglieder **aller** Teams seines Termins, nicht nur die dieses einen Teams

#### Scenario: Slot ohne Spiel behält seinen Team-Geltungsbereich
- **WHEN** ein Dienst-Slot ohne `game_id`, aber mit `team_id = A` angelegt wird
- **THEN** sehen ihn nur Mitglieder von Team A und deren Eltern

