## Context

Siehe `proposal.md — Why`. Ausgangslage im Code:

- `kader` trägt die Saison-Mitgliedschaft, `teams` die saisonübergreifende Identität.
  `kader.team_id` wird über `ensureTeam(age_class, gender, team_number)` abgeleitet — der
  Name eines Teams ist berechnet, nie eingegeben (`internal/kader/handler.go:99,110`).
- `training_sessions.team_id` und `training_series.team_id` sind `NOT NULL REFERENCES
  teams(id)`. Alle 63 `.team_id`-Vorkommen in `internal/trainings/handler.go` sind
  Varianten desselben Joins auf `kader`.
- Die vier Views (`player_memberships`, `team_memberships`, `trainer_memberships`,
  `user_accessible_teams`) filtern bereits `WHERE k.team_id IS NOT NULL`.
- `internal/teams/access.go` und `penalties.go` lösen jede Berechtigung über
  `WHERE k.team_id = ?` auf.

## Goals / Non-Goals

**Goals**

- Eine benannte Gruppe ohne Altersklasse, Geschlecht, Jahrgang und Teamnummer.
- Volle Trainings-Funktionalität: Termine, Serien, RSVP, Anwesenheit, Erinnerungen.
- Kommunikation über die Chat-Standardgruppen.
- Der Ausschluss aller übrigen Flächen ist **strukturell**, nicht als Filterliste.

**Non-Goals**

- **Mitteilungen/Broadcasts an eine Übungsgruppe.** `broadcast_targets` trägt einen CHECK
  `(kind LIKE 'team\_%') = (team_id IS NOT NULL)` und eine `teams`-Referenz. Das zu
  erweitern ist ein eigener, kleiner Change. „Kommunikation per Nachrichten" ist mit den
  Chat-Standardgruppen erfüllt.
- **Zusammenführung mit Förderkader/Perspektivkader** — siehe Entscheidung 5.
- **Saisonübergreifende Gruppen.** `kader.season_id` ist NOT NULL und bleibt es.
- **Kopieren von Übungsgruppen in die Folgesaison.** Bewusst entschieden, nicht offen —
  siehe Entscheidung 7. `CopyFromSeason` überspringt Übungsgruppen; sie werden zu jeder
  Saison neu angelegt und neu besetzt.
- Spiele, Dienste, Kasse, Strafen, Aufgaben, Videos, Ordner-Rechte, Statistiken,
  iCal-Feed, Abwesenheitskalender.

## Entscheidung 1 — Übungsgruppe ist eine `kader`-Zeile ohne `teams`-Zwilling

Drei Alternativen standen zur Wahl:

| | Ansatz | verworfen weil |
|---|---|---|
| A | `age_class` trägt den Typ (wie Migration `034`) | Die Anforderung lautet ausdrücklich „keine Altersgruppe, keine Jahrgangsbindung". Eine Altersklasse als Typträger für etwas ohne Altersklasse ist ein Kategorienfehler, und `AgeClassSortKey`, `compareAgeClass` sowie die Jahrgangs-Renummerierung hängen daran. |
| B | Eigene Tabelle `training_groups`, `training_sessions` bekommt einen **zweiten** Owner (`team_id` XOR `training_group_id`) | 63 Stellen müssten eine zweite Zugehörigkeitsquelle lernen, darunter der dichteste SQL-Block des Repos (RSVP-Sichtbarkeit, `handler.go:1110–1222`). Polymorpher Besitz für etwas, das einen gemeinsamen Besitzertyp hat. |
| C | `kader`-Zeile **mit** `teams`-Zwilling und `kind`-Spalte | Der Ausschluss müsste in ~12 Dateien als negativer Filter nachgezogen werden — und **fails open**: ein vergessener Filter lässt die Gruppe still in Videos, Ordnern und Statistiken auftauchen. Ein Arch-Test kann „dieses SQL joint `teams` und müsste filtern" nicht prüfen; die mechanische Sicherung dieses Repos greift dort nicht. Jeder künftige `JOIN teams` erbt die Pflicht stillschweigend. |

**Gewählt: `kader`-Zeile ohne `teams`-Zeile, `training_sessions.kader_id` als
einheitlicher Besitzer.** Damit gibt es *einen* Besitzertyp statt zweier, die 63 Joins
werden kürzer statt länger, und die Abwesenheit der `teams`-Zeile ist das Gate.

Der Ansatz **fails closed**: wer einen Consumer nicht über Übungsgruppen belehrt, bekommt
dort keine Übungsgruppen zu sehen — genau der gewünschte Default.

## Entscheidung 2 — `team_id` bleibt als nullable Projektion

`training_sessions.team_id` wird nicht entfernt, sondern nullable:

```
  Mannschaftstraining:  kader_id = 42,  team_id = 7
  Übungsgruppentraining: kader_id = 99, team_id = NULL
```

`kader_id` ist die Wahrheit, `team_id` die abgeleitete Antwort auf „gehört das einer
Mannschaft?". Die 28 externen Referenzen (`internal/attendance`, `absences`, `calendar`,
`dashboard`, `videos`, `scheduler`) bleiben unverändert und schließen Übungsgruppen
dadurch von selbst aus.

**Das ist eine Zusage, kein Zufall.** `team_id IS NULL` heißt „gehört einer
Übungsgruppe" und darf nie durch einen „Reparatur"-Backfill gefüllt werden — das risse
sämtliche Ausschlüsse auf einmal ein. Deshalb steht es als Requirement in
`specs/trainings/spec.md` und nicht nur als Kommentar im Schema.

Die Redundanz `training_sessions.season_id` neben `kader.season_id` bleibt bestehen
(dutzendfach in Queries verwendet); ihre Auflösung wäre reine Churn.

## Entscheidung 3 — genau ein Gate, und es liegt bei `kader_extended_members`

Der erweiterte Kader ist die **einzige** unerwünschte Fläche, die an `kader_id` statt an
`teams.id` hängt und deshalb über `PUT /api/kader/{id}` erreichbar bliebe. Alles andere
(Strafen, Kasse, Aufgaben, Warte) adressiert `/api/teams/{id}/…` und löst mit
`WHERE k.team_id = ?` auf — für eine Übungsgruppe nicht adressierbar.

Antwort ist **409**, nicht 404: der Kader existiert, nur passt die Operation nicht zur
Variante. Ein 404 würde eine Existenz verschleiern, die hier nichts zu schützen hat.

Weil es genau ein Gate ist, braucht es **keine Allowlist und keinen Arch-Test** im Stil
von `broadcastAllowlist`/`audienceAllowlist`. Ein einzelner Charakterisierungstest
(`TestPracticeGroup_KeineTeamRoute`) hält die strukturelle Aussage fest.

## Entscheidung 4 — Begriffe: „Übungsgruppe" neu, „Sonderkader" alt

Die `optgroup` in `AdminKaderPage.tsx:561` und `:725` heißt heute „Trainingsgruppen" und
listet Förderkader/Perspektivkader. Bliebe der Name, stünden zwei verschiedene Dinge in
derselben Maske unter demselben Wort.

- neue Entität → **„Übungsgruppe"**
- bestehende `optgroup` → **„Sonderkader"**

Danach ist „Trainingsgruppe" in der Oberfläche nicht mehr belegt. Tabelle
(`training_group_categories`), Route (`/api/training-group-categories`) und Spaltennamen
bleiben unverändert — reine Label-Änderung, keine Migration, jederzeit revidierbar.

Im Code und in der API heißt die Entität **`practice`** (`kader.kind='practice'`,
`/api/practice-groups`, `internal/practicegroups`). Bewusst nicht `training_group`: das
kollidierte mit der bestehenden Tabelle `training_group_categories` und würde die
Verwechslung, die Entscheidung 4 in der UI auflöst, im Code neu aufmachen. Backend-Routen
sind laut Konvention englisch, die Frontend-Route bleibt deutsch (`/uebungsgruppen`).

## Entscheidung 5 — Förderkader bleibt, bewusst vertagt

Der Bestandsmechanismus (`age_class` aus `training_group_categories`, Migration `034`)
bleibt unverändert. Förderkader und Perspektivkader sind weiterhin reguläre Kader mit
`kind='team'`, `teams`-Zwilling, Jahrgangsbadge und erweitertem Kader.

Die Zusammenführung ist **vertagt, nicht vergessen**: sie erfordert eine fachliche
Entscheidung über die Jahrgangsbindung der Bestands-Förderkader („Förderkader 2016" und
„Förderkader 2017" würden zu zwei Gruppen gleichen Namens und kollidierten mit dem
Partial-Unique-Index auf `(season_id, name)`), die zum Zeitpunkt dieses Changes nicht
getroffen werden konnte.

Der Nutzen der Vertagung: der Change wird **rein additiv**. Kein Bestandsdatensatz ändert
seinen Inhalt, der Backfill in Migration `058` läuft für A–D-Jugend und Förderkader über
dieselbe Ableitung, und ein Rollback ist ein reiner Schema-Rollback.

## Entscheidung 6 — CHECK erzwingt das Modell, nicht die Anwendung

```sql
CHECK (
     (kind = 'team'     AND age_class IS NOT NULL AND gender IS NOT NULL
                        AND name IS NULL)
  OR (kind = 'practice' AND age_class IS NULL AND gender IS NULL
                        AND name IS NOT NULL AND team_id IS NULL
                        AND dedicated_birth_year IS NULL)
)
```

Alles-oder-nichts, in der Datenbank. Eine Übungsgruppe kann dadurch nicht versehentlich
einen `teams`-Zwilling bekommen — auch nicht über einen künftigen Codepfad, der die
Variante nicht kennt. Das ist die härteste verfügbare Form der Invariante 2.

Der bestehende `CHECK (gender IN ('m','f','mixed'))` bleibt **unverändert**: in SQLite
wertet `NULL IN (…)` zu NULL aus, und ein CHECK gilt bei NULL als erfüllt.

`team_number` bleibt `NOT NULL DEFAULT 1` und ist bei `kind='practice'` bedeutungslos. Der
bestehende `UNIQUE(season_id, age_class, gender, team_number)` braucht keine Anpassung:
SQLite behandelt NULLs in UNIQUE-Indizes als distinkt, `(season, NULL, NULL, 1)`
kollidiert also nie. Die Eindeutigkeit des Namens trägt ein eigener Partial-Index:

```sql
CREATE UNIQUE INDEX idx_kader_practice_name
  ON kader(season_id, name) WHERE kind = 'practice';
```

## Entscheidung 7 — Lebensdauer und Saisonende

Der Wechsel des Besitzers von `teams` (saisonübergreifend) auf `kader` (saisongebunden)
wirft die Frage auf, was am Saisonende passiert. Drei Antworten:

**1. Für Mannschaften ändert sich nichts.** `internal/trainings/handler.go` enthält kein
einziges `is_active=1`; die Auflösung läuft heute schon über `ts.season_id`, nicht über die
aktive Saison. `k.team_id=ts.team_id AND k.season_id=ts.season_id` und `k.id=ts.kader_id`
sind exakt äquivalent — auch für Trainings abgelaufener Saisons. Alte Kader-Zeilen werden
beim Saisonwechsel nicht gelöscht, `CopyFromSeason` legt neue an. Da `team_id` bei
Mannschaftstrainings gesetzt bleibt, sind saisonübergreifende Auswertungen weiter über
`team_id` möglich.

**2. `ON DELETE RESTRICT`, nicht CASCADE.** `teams` werden nie gelöscht, nur über
`is_active` stillgelegt — deshalb war das bisherige `ON DELETE CASCADE` auf `team_id`
folgenlos. Kader **sind** löschbar (`DELETE /api/kader/{id}`), und die einzige Guard dort
ist `COUNT(*) FROM kader_members > 0 → 409` (`handler.go:737`). Ein **leerer** Altkader —
genau das Ergebnis eines Saisonaufräumens — wäre löschbar, und ein CASCADE nähme seine
Trainings samt Anwesenheits- und RSVP-Historie still mit.

Deshalb `ON DELETE RESTRICT` plus eine zweite Guard derselben Form: 409 mit
`training_count`, solange Termine am Kader hängen. Das ist bewusst dieselbe Gestalt wie die
bestehende Mitglieder-Guard, kein neues Muster.

**3. Übungsgruppen haben keine saisonübergreifende Identität.** Ohne `team_id` sind
„Torwarttraining 25/26" und „Torwarttraining 26/27" unverbundene Zeilen. Das ist die
konsequente Folge von Entscheidung 1 und mit den Non-Goals verträglich (keine
Statistiken, keine Historie über Saisons). Praktische Konsequenz: die Gruppe muss zu jeder
Saison neu angelegt werden.

`CopyFromSeason` SHALL Übungsgruppen **überspringen**. Der Kopierer keyt auf
`ageClass+"|"+gender` (`copy.go:60`) und ruft `ensureTeam(a.AgeClass, a.Gender, 1)` — bei
zwei NULLs kollabieren alle Übungsgruppen auf denselben Schlüssel `"|"` und es entstünde
ein Team ohne Altersklasse und Geschlecht.

Ein **Kopierpfad** für Übungsgruppen (Name, Mitglieder und Trainer in die Folgesaison
übernehmen) wurde erwogen und **verworfen**. Die Gruppe wird zu jeder Saison neu angelegt
und neu besetzt. Das ist bewusst akzeptierte Handarbeit, kein Versäumnis: eine Übungsgruppe
ist ein situatives Gefäß (Sichtung, Athletikblock, Torwartrunde), dessen Besetzung ohnehin
zur Saison neu zu entscheiden ist — ein Kopierpfad würde eine Kontinuität suggerieren, die
das Modell ohne `team_id` gar nicht trägt.

## Bekannter Rest — Selbst-RSVP ohne Kaderprüfung

Beim Umstellen der Trainings auf `kader_id` ist aufgefallen: **`Respond` prüft für die
Selbst-RSVP keine Kaderzugehörigkeit.** Jeder eingeloggte Nutzer mit Mitglieds-Datensatz
kann `POST /api/training-sessions/{id}/respond` auf eine beliebige Session-ID schicken; die
Zeile wird gespeichert (HTTP 204) und zählt in `confirmed_count` mit — der Trainer sieht
eine Zusage von jemandem, der nicht in seinem Kader ist. Nachgemessen, nicht abgeleitet.

Das ist **Bestandsverhalten und variantenunabhängig**: Mannschaftstrainings sind exakt
gleich betroffen, der Change verursacht es nicht. Es hier mitzubeheben hieße, das Verhalten
der Mannschaftsvariante zu ändern (es gibt nur einen Codepfad — das ist der Punkt von
Entscheidung 1) und Bestandstests fallen zu lassen, die sich ausdrücklich darauf stützen
(`TestRespond_CreatesRSVP` nutzt ein Mitglied ohne jeden Kader und erwartet 204). Beides
widerspräche der Zusage „rein additiv" und dem Regressionsschutz aus Task 6.3.

Deshalb bleibt die Lücke hier stehen und wird im eigenen Change **`rsvp-kader-gate`**
geschlossen — mit eigenen Tests für die Mannschaftsseite, wo das Risiko liegt.

## Risiken

**Backfill von `kader_id` (Migration 058).** `kader_id` ist NOT NULL; ein
`training_session`, dessen `(team_id, season_id)` keinen Kader findet, lässt die Migration
**abbrechen**. Das ist gewollt (laut scheitern statt still NULL), verlangt aber eine
Zählabfrage auf Prod **vor** dem Deploy — Task 6.1. Ein solcher Waisen-Datensatz ist
möglich, weil `kader` gelöscht werden kann, während `training_sessions` an `teams` hängt.

**Regressionsrisiko liegt in Task 3.** Vitest und Playwright sehen SQL-Semantik nicht. Die
Umstellung der RSVP-Sichtbarkeit (`handler.go:1110–1222`) muss über Go-Tests abgesichert
werden, und zwar für alle vier Betroffenengruppen (Spieler, Eltern, Trainer, erweiterter
Kader der Mannschaftsvariante) — die bestehenden Trainings-Tests müssen dabei **grün
bleiben**, sie sind der eigentliche Regressionsschutz.

**Zwei Tabellen-Rebuilds.** `kader` (Migration 057) und `training_sessions`/
`training_series` (058). Muster ist etabliert (`018`, `034`); `PRAGMA
legacy_alter_table=ON` verhindert, dass SQLite beim DROP/RENAME die vier auf `kader`
verweisenden Views validiert. `migrate` setzt beim Up ohnehin `PRAGMA foreign_keys=OFF`
(`internal/db/db.go`), die eingehenden FKs sind damit unkritisch. Alle Indizes müssen
danach neu angelegt werden.

**SSE-Zielmenge.** `internal/hub/audience.go:80` löst die Empfänger eines Trainings über
`SELECT team_id FROM training_sessions WHERE id = ?` auf. Bei NULL bliebe die Menge leer
und Live-Updates erreichten niemanden — stumm, ohne Fehler. Muss auf `kader_id` umgestellt
werden (Task 3.5).
