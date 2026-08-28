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

### Requirement: Manuell angelegter Dienst-Slot an einem Mehr-Team-Termin trägt kein Team

Wird ein Dienst-Slot über das Spieltag-Detail-Modal (`POST /api/duty-slots` mit gesetztem
`game_id`) an einem Termin angelegt, dem **mehr als ein Team** zugeordnet ist
(`game_teams`), SHALL das Frontend `team_id: null` senden. Nur bei genau einem
zugeordneten Team SHALL dessen ID übertragen werden.

Begründung: `duty_slots.team_id` ist das Sichtbarkeits-Feld, nicht eine
Zuordnungs-Notiz. Ein gesetztes Team schränkt die Dienstbörse auf genau dieses Team ein
(`ds.team_id IN (…)`), während `team_id IS NULL` zusammen mit `game_id` über den
bestehenden Fallback auf **alle** Teams des Termins auflöst — inklusive deren Eltern über
den `eltern`-Audience-Zweig. Ein Slot ohne Team und ohne Spiel bleibt unverändert nur für
Vorstand/Admin sichtbar.

#### Scenario: Slot an Termin mit drei Teams ist für alle drei sichtbar
- **WHEN** ein Vorstand an einem Termin mit den Teams A, B und C einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` und gesetztem `game_id` gespeichert
- **AND** erscheint er in `GET /api/duty-board` für Spieler, Trainer und Eltern aller drei Teams

#### Scenario: Slot an Termin mit genau einem Team bleibt team-gebunden
- **WHEN** ein Vorstand an einem Termin mit nur Team A einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = A` gespeichert
- **AND** sehen Mitglieder anderer Teams ihn nicht

### Requirement: Ein Diensttyp kann eine dynamische Dauer haben

Ein Diensttyp SHALL einen Dauer-Modus tragen: `absolut` oder `dynamisch`. Im Modus
`absolut` SHALL sich nichts gegenüber dem bisherigen Verhalten ändern — die Dauer ist die
gepflegte Stundenzahl.

Im Modus `dynamisch` SHALL das Ende des Dienstes über einen **End-Anker** und einen
**End-Versatz** bestimmt werden, mit denselben zwei Ankern wie der Start: `start`
(Anpfiff) und `end` (Spielende). Der Versatz SHALL in beide Richtungen zulässig sein.

Die Auflösung des End-Ankers SHALL identisch zur Auflösung des Start-Ankers sein:
`end` verwendet die gepflegte Endzeit des Termins, und andernfalls den Anpfiff zuzüglich
der ermittelten Spieldauer.

Die Vorlagen-Zeile SHALL Modus, End-Anker und End-Versatz wie die übrigen Item-Felder
per Copy-on-pick vom Diensttyp übernehmen und danach eigenständig führen.

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

### Requirement: Eine unauflösbare dynamische Dauer fällt auf die absolute zurück

Ergibt die dynamische Auflösung keine positive Dauer — weil die errechnete Endzeit vor der
Startzeit liegt —, SHALL der Slot trotzdem entstehen und die gepflegte absolute Dauer
tragen. Die Stundenzahl SHALL deshalb auch im Modus `dynamisch` pflegbar bleiben.

Ein Slot SHALL nach jedem Regen-Lauf eine Dauer größer als null tragen, unabhängig davon,
wie Anker und Versätze gesetzt sind.

#### Scenario: Endzeit läge vor der Startzeit

- **WHEN** ein dynamischer Diensttyp bei Anpfiff −30 min startet und bei Anpfiff −60 min
  enden würde
- **THEN** entsteht der Slot
- **AND** trägt er die gepflegte absolute Dauer
- **AND** fällt kein Dienst aus dem Plan

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

#### Scenario: Dienst mit dynamischem Typ von Hand hinzufügen

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Diensttyp im Modus `dynamisch`
  auswählt
- **THEN** wird die Dauer aus Anker und Versätzen gegen diesen Termin berechnet vorbelegt
- **AND** ist der so entstandene Slot danach eine feste Zahl, die der Regen nicht anrührt

