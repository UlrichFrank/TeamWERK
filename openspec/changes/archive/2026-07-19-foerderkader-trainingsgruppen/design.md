# Design

## Kontext & Leitentscheidung

Trainings und Spiele hängen in TeamWERK an `team_id` (nie direkt am `kader`); der
teilnehmende Roster wird über den Kader mit passender `team_id`+`season_id`
abgeleitet. Dienste (`duty_slots`) hängen ausschließlich an `games`. Daraus folgt
die zentrale Design-Prämisse dieses Changes:

> **Ein Förderkader/Perspektivkader ist strukturell ein normaler Kader**
> (`team` + `kader` mit `dedicated_birth_year`), für den nie ein Spiel angelegt
> wird. Trainings, RSVP und Anwesenheit funktionieren dadurch **ohne
> Codeänderung**. Der Change fügt nur das hinzu, was heute die *Anlage* dieser
> Kader und die *Aufnahme der Kinder* verhindert.

Deshalb werden Trainings-, RSVP-, Anwesenheits- und Dienste-Capabilities
**bewusst nicht** angefasst.

## Entscheidung 1: Kein `kader.type`-Diskriminator (② „legt halt keiner an")

Verworfen: eine Spalte `kader.type`/`is_trainingsgruppe`, an der Spielplan/Dienste
gefiltert würden. Begründung:
- Spiele sind bereits opt-in (`games_per_season` Default 0, Dienste nur aus
  Spielen). „Keine Spiele" ist der Default, nicht ein zu erzwingender Zustand.
- Die vorhandene `qualifikations-kader`-Spec beschreibt ein `kader.type`, das im
  realen Schema **nicht existiert** (Drift). Wir zementieren diese Drift nicht,
  indem wir eine konkurrierende `type`-Semantik einführen.

Konsequenz: kein UI-Guard gegen versehentliches Spiel-Anlegen. Akzeptierter
Trade-off gemäß Nutzerentscheidung.

## Entscheidung 2: Getrennte Referenzliste statt Erweiterung von `age_class_game_rules` (B2)

`age_class_game_rules` ist **spielgebunden** (CHECK auf A–D-Jugend, trägt
Halbzeit-/Pausendauer). Trainingsgruppen dort einzutragen würde Spielregeln für
Gruppen erzeugen, die nie spielen, und die CHECK aufweichen.

Stattdessen neue, entkoppelte Referenztabelle:

```sql
CREATE TABLE training_group_categories (
    name       TEXT     PRIMARY KEY,        -- 'Förderkader', 'Perspektivkader'
    sort_order INTEGER  NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Seed (sort_order = gewünschte Kategorie-Reihenfolge: Perspektivkader vor Förderkader):
INSERT INTO training_group_categories (name, sort_order)
VALUES ('Perspektivkader', 1), ('Förderkader', 2);
```

- **Read:** `GET /api/training-group-categories` (authentifiziert, cachebar wie
  andere Referenzdaten). Datenressource → englischer Routenname (Konvention
  `04-api-db.md`).
- **Verwaltung:** `POST /api/training-group-categories` /
  `DELETE /api/training-group-categories/{name}` unter dem Vorstand-Tier, mit
  `h.hub.Broadcast("training-group-categories-changed")` (Broadcast-Gate). Löschen
  einer noch verwendeten Kategorie: die Kader behalten ihren Freitext-`age_class`
  (keine FK) — die Kategorie verschwindet nur aus dem Anlage-Dropdown. Das ist
  akzeptabel und wird in der Spec als Szenario festgehalten.

**Warum keine FK von `kader.age_class` auf diese Tabelle?** `age_class` ist heute
Freitext und trägt sowohl Spiel-Altersklassen (aus `age_class_game_rules`) als auch
künftig Trainingsgruppen. Eine FK würde beide Quellen zusammenzwingen. Die Liste
ist reine **Eingabe-Unterstützung** für die Anlage-Maske, keine referentielle
Wahrheit.

## Entscheidung 3: Kader-Anlage — Union + freier Jahrgang

`AdminKaderPage` lädt heute nur `/age-class-rules`. Neu:

```
Altersklasse-<select>:
  ── Wettkampf ──         (aus /age-class-rules; Jahrgang = ComputeAgeBrackets)
     A-Jugend … D-Jugend
  ── Trainingsgruppen ──  (aus /training-group-categories; Jahrgang FREI)
     Förderkader
     Perspektivkader

Jahrgang-<select>:
  - Spiel-Altersklasse gewählt → bracketYears (wie bisher)
  - Trainingsgruppe gewählt    → freie Jahresliste (z. B. Saison-Startjahr − 4 … − 14)
```

Der Backend-Anlagepfad (`POST /api/kader` → `createSingleKader` → `ensureTeam` +
`INSERT`) bleibt **unverändert**: `age_class` ist bereits Freitext, ein beliebiger
`dedicated_birth_year` wird bereits akzeptiert. Es ist also rein ein
Frontend-Datenquellen- + Backend-Referenzlisten-Change.

Label-Rendering (Kader-Admin): der Kartentitel ist `{age_class} [team_number]
{gender}` und der Jahrgang erscheint als **separater Badge** (`birthYearLabel`
aus `k.birth_years`), **nicht** im Namen. Diese bestehende Darstellung wird
übernommen — für eine Trainingsgruppe steht der Jahrgang also weiterhin im Badge
(„Förderkader gemischt" + Badge „2016"), nicht als „Förderkader 2016" im Titel.

## Entscheidung 4: `foerderkind` als `members.status` (nicht Flag, nicht Funktion)

Spiegelt exakt `anwaerter-member-status`:
- Ein reines Förderkind ist **kein** normales Mitglied → ein Status-Wert (exklusiv)
  ist die richtige Form, nicht ein orthogonales Flag.
- Ein Kind, das **zusätzlich** reguläres Mitglied ist, bleibt `status='aktiv'` und
  wird einfach zusätzlich in den Förderkader aufgenommen (①: „in Kader und
  erweitertem Kader wie bei den anderen auch"). Kein Sonderpfad, keine
  Doppel-Datensätze.

SQLite kennt kein `ALTER TABLE … ADD CHECK`, daher **Tabellen-Rebuild** von
`members` in Migration 034 (create `members_new` mit erweiterter CHECK → `INSERT
… SELECT` → `DROP`/`RENAME`, Indizes neu). Vorbild: Rebuilds in Migration 018.

CHECK neu:
```
status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv',
           'honorar','anwaerter','foerderkind')
```

Validierung: Mitglieder-Anlage/-Bearbeitung erzwingt für `foerderkind` (wie für
`anwaerter`) **nur Name + Geburtsdatum**; `join_date` ist nicht Pflicht (③ kein
Beitrag → kein Beitrags-relevantes Eintrittsdatum nötig).

## Entscheidung 5: Beitragslauf-Ausschluss (③ kein Beitrag)

Eine Zeile in `internal/beitragslauf/query.go`:
`WHERE m.status NOT IN ('honorar','anwaerter')` →
`… NOT IN ('honorar','anwaerter','foerderkind')`. Damit fällt `foerderkind` vor
jeder weiteren Prüfung (SEPA-Mandat, Adresse) aus dem Lauf. Spec-Delta in
`sepa-beitragslauf` (Ein-/Ausschlussregeln).

## Entscheidung 6: Kurzname ohne Änderung der geteilten Formel (Option C)

Der kanonische Team-Kurzname (`display_short`) wird an **einer** Stelle berechnet
und überall wiederverwendet: `web/src/lib/teamName.ts → buildTeamShortNames` und
spiegelbildlich `internal/db/team_display_short.go → TeamDisplayShort` (SQL). Formel:
`<gender: m/w/g> + erster Buchstabe(age_class) + team_number (nur wenn mehrere
Kader dieselbe age_class+gender in der aktiven Saison teilen)`.

**Diese Formel wird NICHT geändert.** Damit ist per Konstruktion ausgeschlossen,
dass A–D-Bezeichnungen sich ändern (harte Nutzer-Anforderung). Die neuen
Kategorien fügen sich von selbst ein:
- „Perspektivkader" → `ageInitial` matcht nur `[A-F]`, „P" fällt auf `charAt(0)`
  zurück → **`gP`**.
- „Förderkader" → „F" ∈ `[A-F]` → **`gF`**; bei zwei Förderkadern derselben
  gender in einer Saison → **`gF1` / `gF2`**.
- Keine Kollision mit A–D (dort `gA`/`wB`/`gD` …; kein F-Jugend-Eintrag in
  `age_class_game_rules`).

**Jahrgang bleibt sichtbar & jedes Jahr klar:** der Jahrgang steckt in
`dedicated_birth_year` (Badge in der Kader-Ansicht), nicht im Kurznamen. Pro Saison
gesetzt → diese Saison „gF1 = 2016", „gF2 = 2017", „gP = 2015".

**Deterministische Nummerierung (für stabile Namen):** damit `gF1`/`gF2` dem
Jahrgang folgen statt der Anlage-Reihenfolge, SHALL die `team_number` für Kader
einer Trainingsgruppen-Kategorie bei der Anlage nach **aufsteigendem
`dedicated_birth_year`** vergeben werden (älterer Jahrgang → niedrigere Nummer).
Verworfen wurden: Jahr in die Formel falten (Blast-Radius auf alle dedizierten
Kader inkl. A–D) und ein eigenes `teams.short_name`-Feld (unnötiger Aufwand, da C
das gewünschte Ergebnis liefert).

## Entscheidung 7: Kanonische Sortierreihenfolge (A–D, dann Trainingsgruppen)

Heute sortieren alle Listen nach rohem `age_class`-String alphabetisch
(`ORDER BY k.age_class, k.gender, k.team_number` bzw. `t.age_class …`, Frontend
`[...].sort()`). Das ergibt A,B,C,D,**Förderkader,Perspektivkader** — falsch, denn
alphabetisch steht `F` vor `P`. Gewünscht ist die feste Kategorie-Ordnung
**A–D → Perspektivkader → Förderkader** (logisch: ältester Jahrgang zuerst),
sekundär alphabetisch/nach `team_number`.

Statt N-fach dupliziertem `CASE` ein **geteilter Sortier-Schlüssel** (Sibling von
`TeamDisplayShort`):

- Go: `internal/db.AgeClassSortKey(col)` liefert einen SQL-Ausdruck
  ```sql
  (CASE WHEN <col> IN (SELECT name FROM training_group_categories)
        THEN '1' || printf('%04d', (SELECT sort_order FROM training_group_categories WHERE name = <col>))
        ELSE '0' END) || <col>
  ```
  Block `0` = alle Nicht-Trainingsgruppen (`*-Jugend`), alphabetisch → A,B,C,D
  unverändert. Block `1` = Trainingsgruppen nach `sort_order` → Perspektivkader
  vor Förderkader. Verwendung: `ORDER BY <sortkey>, k.gender, k.team_number`.
- TS: `compareAgeClass(a, b, categories)` in `web/src/lib/teamName.ts` mit
  derselben Logik (categories aus `GET /api/training-group-categories`, das
  `AdminKaderPage` ohnehin lädt).

Zu ersetzen (Audit aller Team-/Kader-Ordnungen):
`internal/kader/handler.go:236`; `internal/games/handler.go` (≈5×
`ORDER BY t.age_class, t.gender, k.team_number`); `internal/teams/handler.go:314`
u.a. (`ORDER BY t.name` → wo Kader-Ordnung gemeint ist, auf den Sortkey umstellen);
`web/src/pages/AdminKaderPage.tsx:117,318`.

**Single source of truth für „P vor F" ist `training_group_categories.sort_order`**
— keine hartkodierte Reihenfolge im Code, keine `LIKE '%-Jugend'`-Heuristik. A–D
bleiben per Konstruktion unberührt (Block 0, alphabetisch).

## Datenfluss (Ende-zu-Ende)

```
Vorstand legt „Förderkader 2016" an
  AdminKaderPage → POST /api/kader {age_class:"Förderkader", gender, season_id,
                                    dedicated_birth_year:2016}
  → ensureTeam("Förderkader …", …) → teams-Row
  → INSERT kader                      → kader-Row (games_per_season=0)

Vorstand nimmt Gastkind auf
  Mitglieder-Anlage → POST /api/members {first_name,last_name,date_of_birth,
                                         status:"foerderkind"}   (join_date optional)
  → in Mitgliederliste sichtbar (Badge „Förderkind", Filter)

Kind dem Kader zuordnen
  AdminKaderPage → PUT /api/kader/{id} (kader_members / kader_extended_members)

Training organisieren
  bestehende Trainings-Serie/-Session auf team_id  → RSVP + Anwesenheit
  (unverändert, keine Codeänderung)

Beitragslauf
  status='foerderkind' → NOT IN (…) → nie im Lauf
```

## Risiken / offene Punkte

- **`members`-Rebuild** ist der schwergewichtigste Task (viele Spalten, viele
  Indizes, `CREATE TABLE IF NOT EXISTS "members"`). Sorgfältiger `INSERT … SELECT`
  mit vollständiger Spaltenliste + Wiederherstellung aller Indizes; down-Migration
  spiegelbildlich. Vor Prod-Migration DB-Backup (Standard).
- **Mehrfach-Kader-Mitgliedschaft** (①) ist bereits heute erlaubt
  (`kader_members UNIQUE(kader_id, member_id)` pro Kader), Förderkader macht sie
  nur systematischer. Kein Code, der „ein Team pro Mitglied pro Saison" hart
  annimmt, wird durch diesen Change *neu* gebrochen — die `player_memberships`-View
  liefert schon heute potenziell mehrere Teams. Bei der Umsetzung dennoch
  Roster-/RSVP-Ableitungen gegen doppelte Team-Zuordnung sichten.
- **Kategorie-Löschung bei Verwendung:** bewusst kein Kaskaden-/Sperr-Verhalten
  (kein FK); betroffene Kader behalten ihren `age_class`-Text.
