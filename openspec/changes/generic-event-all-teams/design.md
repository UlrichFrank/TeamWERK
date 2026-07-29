## Context

Der Mannschafts-Picker auf `/kalender` (Wizard und `GameEditModal`) lädt seine Liste aus
`GET /api/teams` (`games.ListTeamsForUser`). Dieser Endpoint ist bewusst **nutzergefiltert**:

| Aufrufer | Ergebnis |
|---|---|
| `admin` / `vorstand` | alle Teams der aktiven Saison |
| `sportliche_leitung` | alle Teams der aktiven Saison |
| reiner `trainer` | nur Teams mit eigenem `kader_trainers`-Eintrag |
| `spieler` / Elternteil | nur Teams via `user_accessible_teams` |

Die Filterung ist an vielen Stellen richtig (Kalender-Filter-Dropdown, Trainingsanlage,
Video-Upload-Ziele) — beim generischen Event ist sie es nicht. Ein Vereinsfest oder
Trainingslager betrifft naturgemäß mehrere Mannschaften, und der Trainer, der es organisiert,
kann die anderen heute nicht anhaken.

Zwei Fakten aus dem Bestandscode, die das Design tragen:

1. **`GET /api/teams/names` ist bereits vereinsweit lesbar** (`games.ListTeamNames`, Spec
   `team-names-endpoint`) und liefert für **jeden** eingeloggten Nutzer alle aktiven Teams der
   aktiven Saison mit `id`, `age_class`, `gender`, `team_number`, `group_count`. `KalenderPage`
   lädt diese Liste ohnehin schon (`allTeamNames`) für die Kurznamen-Berechnung. Die Existenz
   und Benennung fremder Mannschaften ist im Verein also keine schützenswerte Information.
2. **`POST /api/games` prüft den Trainer-Scope für *alle* Event-Typen** (`handler.go`, Block mit
   `claims.HasFunction("trainer") && !claims.HasFunction("sportliche_leitung")`) —
   `PUT /api/games/{id}` und `DELETE /api/games/{id}` prüfen **gar nichts** außer dem
   Router-Tier `RequireClubFunction("vorstand","trainer","sportliche_leitung")`.

Punkt 2 ist eine echte Lücke: heute darf ein reiner Trainer per PUT jedes fremde Spiel umdatieren
oder per DELETE löschen. In der UI ist das nicht erreichbar (`ScopeGamesQuery` blendet fremde
Spiele aus), per API aber sehr wohl.

## Goals / Non-Goals

**Goals:**
- Bei `event_type='generisch'` kann jeder, der Events anlegen/bearbeiten darf, **alle aktiven
  Mannschaften** auswählen — im Wizard und im Bearbeiten-Dialog.
- Die Berechtigung eines Trainers hängt an einer **Server-Prüfung pro Event**, nicht an der
  Länge einer Dropdown-Liste.
- Die bestehende Sichtbarkeits- und Rechte-Matrix bleibt sonst unangetastet.

**Non-Goals:**
- Heim-/Auswärtsspiele: Mannschaftsauswahl bleibt auf eigene Teams beschränkt.
- Keine Änderung an `ScopeGamesQuery` (Event-Sichtbarkeit), `canRecordGameAttendance`
  (Anwesenheiten), `canEditGameNote` (Hinweisfeld) oder Kader-/Dienst-Rechten.
- Kein neuer Endpoint, keine neue Vereinsfunktion, keine Migration.
- Kein Sichtbarkeits-Effekt für die eingeladenen Mannschaften über das hinaus, was
  `event-team-visibility` heute schon leistet (Team am Event ⇒ Team sieht Event).

## Decisions

### D1 — Picker-Quelle: `GET /api/teams/names` statt neuem `?scope=all`

Bei `event_type='generisch'` speist sich die Checkbox-Liste aus `GET /api/teams/names`; alle
anderen Picker (Filter-Dropdown, Heim-/Auswärts-Single-Select, Training) bleiben auf
`GET /api/teams`.

*Warum:* Der Endpoint existiert, ist für jeden Eingeloggten freigegeben, ist per Spec
(`team-names-endpoint`) genau als „vereinsweite Team-Identität" definiert und liefert exakt die
Felder, die `buildTeamShortNames` braucht. Er filtert bereits auf `t.is_active = 1` und die
aktive Saison. Null neue Authz-Oberfläche.

*Alternativen:*
- `GET /api/teams?scope=all` — neuer Query-Param auf einem nutzergefilterten Endpoint. Erzeugt
  einen zweiten Rechte-Pfad in derselben Funktion und lädt zur Verwechslung ein
  („welcher Aufruf war nochmal gefiltert?"). Verworfen.
- Neuer Endpoint `GET /api/teams/selectable` — dritte Team-Liste mit dritter Semantik, ohne dass
  eine der beiden bestehenden fehlt. Verworfen.

*Konsequenz:* `/api/teams/names` liefert kein `name`-Feld. Das ist unkritisch, weil
`team-names-endpoint` ohnehin fordert, dass die UI überall den berechneten Kurznamen zeigt
(„Kein Fallback auf rohe DB-Namen"). `GameEditModal` muss `/teams/names` zusätzlich laden — es
baut seine `teamShortNames` heute aus der gefilterten `availableTeams`-Liste, was für fremde
Teams keinen Namen ergäbe.

### D2 — Scope-Prüfung typabhängig, mit „mindestens ein eigenes Team"-Anker

Ein zentraler Helper in `internal/games` prüft für **reine Trainer** (nicht
`sportliche_leitung`, nicht `vorstand`, nicht `admin`):

| Operation | Bedingung |
|---|---|
| `POST` `heim`/`auswärts` | **alle** `team_ids` sind eigene Teams (Bestandsverhalten) |
| `POST` `generisch` | **mindestens ein** `team_id` ist ein eigenes Team |
| `PUT` / `DELETE` | Trainer ist Trainer mindestens einer **aktuell** beteiligten Mannschaft |
| `PUT` mit neuen `team_ids`, Ziel-Typ `heim`/`auswärts` | zusätzlich: **alle** neuen `team_ids` sind eigene Teams |
| `PUT` mit neuen `team_ids`, Ziel-Typ `generisch` | zusätzlich: **mindestens ein** neues `team_id` ist ein eigenes Team |

*Warum der „mindestens ein eigenes Team"-Anker:* Ohne ihn könnte ein Trainer ein Event anlegen
oder umhängen, das er anschließend selbst nicht mehr sieht (`ScopeGamesQuery` filtert über
`game_teams`) — er hätte es weder korrigieren noch löschen können. Ein 403 ist die ehrlichere
Antwort als ein verwaistes Event. Er ist zugleich die Klammer, die verhindert, dass „darf fremde
Teams einladen" zu „darf fremde Events verwalten" wird.

*Alternative:* Für `PUT`/`DELETE` gar nicht prüfen und sich auf `ScopeGamesQuery` verlassen
(„der Trainer sieht das Event ja nicht"). Verworfen — Sichtbarkeitsfilter in Lese-Queries sind
keine Autorisierung von Schreib-Routen.

*Rollen ohne Einschränkung:* `admin`, `vorstand`, `sportliche_leitung` durchlaufen die Prüfung
nicht — konsistent zur bestehenden `CreateGame`-Logik und zu `CanViewAllGames`.

### D3 — Prüfung im Handler, nicht in `internal/policy`

Die Prüfung braucht einen DB-Zugriff (`kader_trainers` × `game_teams` in der aktiven Saison).
`internal/policy` ist bewusst rein funktional (`Principal` rein, `bool` raus) und hat keinen
`*sql.DB`. Der Helper landet daher als Methode auf `games.Handler` neben
`canEditGameNote`/`canRecordGameAttendance`, die exakt dasselbe Muster verwenden.

### D4 — Kein Frontend-Guard für „letztes eigenes Team entfernen"

`GameEditModal` verhindert das Abwählen der letzten eigenen Mannschaft **nicht** clientseitig;
der Server antwortet mit 403 und der Dialog zeigt die vorhandene Fehlermeldung. Der Client
kennt die Trainer-Zuordnung des Nutzers nicht (die Team-Liste sagt nur „aktiv", nicht „meins"),
und dafür eine zusätzliche Rechte-Information ins Frontend zu schieben, wäre teurer als der
seltene 403.

## Risks / Trade-offs

- **[Verhaltensänderung: 403 auf bisher erlaubten PUT/DELETE]** → Betrifft nur reine Trainer auf
  Events ohne eigene beteiligte Mannschaft. Diese Events sind für den Trainer bereits heute in
  keiner Liste sichtbar; ein UI-Pfad dorthin existiert nicht. Regression-Risiko praktisch null,
  Sicherheitsgewinn real.
- **[Trainer lädt versehentlich die falsche Mannschaft ein]** → Die eingeladene Mannschaft sieht
  das Event dann im Kalender und bekommt ggf. Push (`push-games`). Mitigation: keine technische,
  sondern eine soziale — dasselbe Risiko besteht heute für Vorstand und sportliche Leitung. Die
  Korrektur (Team wieder abwählen) ist verlustfrei möglich.
- **[Dienst-Slots pro Team]** → `event-wizard` legt für jede gewählte Mannschaft einen Satz Slots
  an. Bei vielen eingeladenen Mannschaften entstehen entsprechend viele Slots. Kein neues
  Verhalten (gilt heute für Vorstand genauso), aber für Trainer neu erreichbar. Der bestehende
  `regen_summary` im Response macht das Ergebnis sichtbar.
- **[Zwei Team-Listen im selben Dialog]** → `GameEditModal` hält künftig `availableTeams`
  (gefiltert, für Spiele) und die vereinsweite Liste (für generische Events) parallel. Trade-off
  gegen einen zusätzlichen Request; akzeptiert, weil beide Endpoints `private, no-cache` mit
  ETag-Revalidierung ausliefern und die Payload klein ist.
- **[`/api/teams/names` ohne `name`-Feld]** → Fällt die Kurznamen-Berechnung aus (leere Liste),
  zeigt der Picker keine brauchbaren Labels. Mitigation: `buildTeamShortNames` bekommt dieselbe
  Liste, aus der der Picker gerendert wird — beide können nicht auseinanderlaufen.
