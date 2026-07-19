## Why

Neben den regulären Wettkampf-Kadern (A–D-Jugend) organisiert der Verein
jahrgangsbezogene Talent-/Trainingsgruppen — konkret **Förderkader 2016**,
**Förderkader 2017** und **Perspektivkader 2015**. Diese Gruppen trainieren,
haben aber **keine Spiele** und brauchen entsprechend weniger Funktionalität
(kein Spielplan, keine Dienste, keine Aufstellung).

Das Datenmodell trägt das bereits: Trainings (`training_series` /
`training_sessions`) und Spiele hängen beide an `team_id`, nie direkt am `kader`;
der Roster wird über den Kader mit passender `team_id`+`season_id` abgeleitet.
`kader.age_class` ist Freitext, `kader.dedicated_birth_year` existiert
(Jahrgangskader), `games_per_season` ist Default 0, und Dienste entstehen
**ausschließlich** aus Spielen. Wer ein `team`+`kader` hat, kann also **ohne
Codeänderung** Trainings, RSVP und Anwesenheit nutzen; „keine Spiele" ist der
Default-Zustand, solange niemand ein Spiel anlegt.

Zwei Dinge fehlen aber:

1. **Die Kader-Anlage-Maske bietet die Kategorien nicht an.** Das
   Altersklasse-Dropdown wird ausschließlich aus `age_class_game_rules`
   (`GET /api/age-class-rules`) gefüllt — per CHECK auf `A/B/C/D-Jugend`
   festgelegt und mit Spielregeln (Halbzeit-/Pausendauer) verknüpft. „Förderkader"
   / „Perspektivkader" sind dort bewusst **nicht** einzuordnen. Auch der Jahrgang
   ist nur aus den Spiel-Alterbrackets wählbar (für „Förderkader" existiert kein
   Bracket → 2015/2016/2017 nicht auswählbar).
2. **Die Kinder müssen aufnehmbar sein, ohne normale Mitglieder zu sein.**
   Förderkinder sind häufig Gastkinder aus dem Stammverein — analog zu Trainern
   „nicht notwendigerweise normale Mitglieder". Sie müssen in der Mitgliederliste
   erscheinen und einem Kader zuordenbar sein, aber **keinen Beitrag** zahlen.

## What Changes

- **Neuer Member-Status `foerderkind`** (spiegelt `anwaerter`/`honorar`): gültiger
  Wert für `members.status`, in der Mitgliederliste sichtbar, einem Kader /
  erweiterten Kader wie jedes andere Mitglied zuordenbar. Nur Name + Geburtsdatum
  sind Pflicht (kein `join_date`-Zwang wie bei `anwaerter`). Ein Kind, das
  **zusätzlich** reguläres Mitglied ist, bleibt `status='aktiv'` und zahlt normal —
  `foerderkind` markiert ausschließlich das reine Gast-/Förderkind.
- **Beitragslauf-Ausschluss:** `foerderkind` wird — wie `honorar`/`anwaerter` —
  aus dem SEPA-Beitragslauf ausgeschlossen (`status NOT IN (...)`), zahlt also
  nichts.
- **Trainingsgruppen-Kategorien (B2):** neue, gepflegte Referenzliste
  (`training_group_categories`, z. B. „Förderkader", „Perspektivkader") **getrennt**
  von den spielgebundenen Altersklassen. Die Kader-Anlage-Maske unioniert beide
  Quellen; für eine Trainingsgruppen-Kategorie ist der **Jahrgang frei wählbar**
  (nicht aus Spiel-Brackets abgeleitet). `age_class`/`team` bleiben unverändert
  Freitext — kein Schema-Umbau an `kader`/`teams`, keine neue `kader.type`-Spalte.
- **Anzeige-Konvention:** ein Förderkader wird als `age_class="Förderkader"` +
  `dedicated_birth_year=2016` angelegt; der Jahrgang erscheint als Badge, nicht im
  Namen. Der kanonische **Kurzname** entsteht aus der bestehenden, **unveränderten**
  Formel → `gP`, `gF1`, `gF2` (Nummer nach aufsteigendem Jahrgang). Die Formel wird
  bewusst nicht angefasst, damit die A–D-Jugend-Bezeichnungen garantiert unberührt
  bleiben.
- **Kanonische Sortierreihenfolge:** Kader-/Team-Listen sortieren nicht mehr
  alphabetisch nach `age_class` (das stellte `Förderkader` vor `Perspektivkader`),
  sondern nach fester Kategorie-Ordnung **A–D → Perspektivkader → Förderkader**
  (sekundär `gender`, `team_number`). Umgesetzt über einen geteilten Sortier-Schlüssel
  (`internal/db.AgeClassSortKey` + TS-`compareAgeClass`); die „P vor F"-Ordnung lebt
  einzig in `training_group_categories.sort_order`. A–D-Jugend-Sortierung bleibt
  unverändert.
- **Bewusst NICHT enthalten (② „legt halt keiner an"):** kein UI-Guard, der
  Spiele/Dienste für diese Kader technisch verhindert; keine Änderung an
  Trainings-, RSVP-, Anwesenheits- oder Dienste-Capabilities. Diese funktionieren
  unverändert über `team_id`.

## Capabilities

### New Capabilities
- `foerderkind-member-status`: `foerderkind` als valider `members.status`, mit
  Sichtbarkeit in der Mitgliederliste (Badge/Filter), gelockerter
  Pflichtfeld-Validierung (nur Name + Geburtsdatum) und Zuordenbarkeit zu
  Kader/erweitertem Kader — analog zu `anwaerter-member-status`.
- `trainingsgruppen-kategorien`: gepflegte Referenzliste nicht-spielgebundener
  Kader-Kategorien (Förderkader, Perspektivkader), getrennt von
  `age_class_game_rules`; Lese-Endpoint + Vorstand-Verwaltung; Kader-Anlage
  unioniert beide Quellen und erlaubt freie Jahrgangswahl für diese Kategorien;
  definiert die kanonische Sortierreihenfolge (A–D → Trainingsgruppen nach
  `sort_order`) als Single Source of Truth für alle Kader-/Team-Listen.

### Modified Capabilities
- `sepa-beitragslauf`: Ein-/Ausschlussregel erweitert — `status = 'foerderkind'`
  wird zusätzlich zu `honorar`/`anwaerter` aus dem Lauf ausgeschlossen.

## Impact

- **Backend:** neue Migration `internal/db/migrations/033_*` — (a) `members`-Rebuild
  zur CHECK-Erweiterung um `foerderkind` (SQLite kennt kein ALTER … CHECK), (b)
  Tabelle `training_group_categories` + Seed („Förderkader", „Perspektivkader").
  Beitragslauf-Query (`internal/beitragslauf/query.go`) um `foerderkind` ergänzen.
  Mitglieder-Anlage/-Validierung (`internal/members`) analog `anwaerter` lockern.
  Neuer Config-Endpoint für die Kategorienliste (`internal/config`) mit
  Vorstand-CRUD + Broadcast (Broadcast-Gate); Read authentifiziert, cachebar wie
  andere Referenzdaten.
- **Sortierung:** neuer geteilter Sortier-Schlüssel `internal/db.AgeClassSortKey`
  (Sibling von `TeamDisplayShort`) + TS-`compareAgeClass`; Umstellung der
  Kader-/Team-`ORDER BY`-Stellen (`internal/kader`, `internal/games` ≈5×,
  `internal/teams`) und der `.sort()`-Aufrufe in `AdminKaderPage`.
- **Frontend:** `AdminKaderPage` — Altersklasse-Dropdown unioniert
  `/age-class-rules` + `/training-group-categories`, Jahrgang für Trainingsgruppen
  frei wählbar; Jahrgang als Badge (nicht im Namen); Kurzname `gP`/`gF1`/`gF2` aus
  unveränderter Formel. Mitglieder-Anlage/-Liste:
  `foerderkind` als Status (Badge + Filter, analog `anwaerter`). Ggf. kleine
  Vorstand-Verwaltung der Kategorien unter Einstellungen.
- **Kein** neuer externer Dienst, vernachlässigbarer RAM-Footprint (eine kleine
  Referenztabelle, ein Lookup bei Kader-Anlage). Keine Auswirkung auf Spiele,
  Dienste oder Trainings — diese bleiben über `team_id` unverändert.
- Kein Konflikt mit `qualifikations-kader` (dessen `kader.type`/`is_active` sind
  Spec-Drift, real nicht im Schema) — dieser Change führt **keine** solche Spalte
  ein.
