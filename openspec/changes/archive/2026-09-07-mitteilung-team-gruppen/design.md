## Context

Siehe `proposal.md — Why`. Zwei Bestandsteile prägen den Zuschnitt:

**Der Chat kennt die Gruppen bereits.** `internal/chat/team_groups.go` liefert pro Team der
aktiven Saison drei Standardgruppen (`trainer` / `spieler` / `eltern`) plus „Alle Trainer",
jeweils mit fertiger Auflösungs-Query (`teamGroupMemberQuery`, `allTrainersMemberQuery`) und
kanonischer Kurzform (`db.TeamDisplayShort`). Der Gruppen-Picker beim Anlegen eines
Gruppenchats zeigt genau diese Liste — sie ist die Vorlage für die Ziel-Auswahl.

**Die Mitteilungen kennen nur einen Wert.** `broadcasts.target_type` ist eine Spalte mit
CHECK über fünf Werte; `mitteilung-zielgruppen` `design.md §4` hat Mehrfachauswahl bewusst
verworfen und den Weg dorthin beschrieben: „Zeilentabelle daneben, `target_type` als
Ein-Zeilen-Fall migrieren". Dieser Change geht ihn.

## Goals / Non-Goals

**Goals:**
- Ein Ziel-Vokabular, das Team-Gruppen und vereinsweite Gruppen trägt, ohne eine zweite
  Definition von „Spieler eines Teams" entstehen zu lassen.
- Autorisierung pro Ziel, hart serverseitig, mit einer Liste, die der Composer abfragen kann
  statt sie aus Rolle und Vereinsfunktion zu rekonstruieren.
- Bestandsmitteilungen bleiben unverändert zustellbar.

**Non-Goals:**
- Einzelpersonen als Ziel. Dafür gibt es den Direktchat; eine Mitteilung an eine Person wäre
  eine Nachricht ohne Rückkanal, also strikt schlechter.
- Anzeige der Ziele an Empfänger. `ListBroadcasts` selektiert sie nicht und soll das auch
  nicht — wer die Mitteilung bekommt, sieht sie; warum, ist für ihn ohne Belang.
- Nachträgliches Ändern der Ziele einer gesendeten Mitteilung. `EditBroadcast` fasst nur
  `body` an, die Empfängermenge bleibt beim Sendezeitpunkt eingefroren.

## Decisions

### 1. Die Gruppen-Auflösung wird wiederverwendet, nicht kopiert

`mitteilung-zielgruppen` `design.md §1` hat eine Wiederverwendung abgelehnt — aber gegen
`policy.FolderAccess`, und mit einem Argument über die *Richtung*: die Ordner-ACL beantwortet
ein Prädikat („darf dieser Nutzer?"), Mitteilungen brauchen die Mengenfrage („wer sind
alle?"). Das Argument trifft `team_groups.go` nicht: dessen Queries **sind** die Mengenfrage,
sie liefern `(user_id, name)` für eine Gruppe.

```
   policy.FolderAccess          chat/team_groups.go
   ───────────────────          ───────────────────
   WHERE m.user_id = ?          WHERE k.team_id = ?
   → Prädikat, 1 Subjekt        → Menge, 1 Gruppe
   ✗ nicht nutzbar              ✓ genau was gebraucht wird
```

**Entscheidung:** `resolveAudience` bekommt für die `team_*`-Kinds keine eigenen Queries,
sondern ruft die vorhandenen. Damit gilt für Mitteilungen dieselbe Definition wie für die
Chat-Gruppe — insbesondere zählt der **erweiterte Kader** zu `team_spieler` (die Query
vereinigt `kader_members` und `kader_extended_members`), was für eine Trainingsabsage
richtig ist.

**Verworfen:** eigene, „schlankere" Team-Queries in `audiences.go`. Zwei Definitionen
desselben Begriffs, die nur ein aufmerksamer Leser synchron hält — genau das Risiko, das
`§1` des Vorgänger-Designs beschreibt, hier aber ohne den Grund, der es dort rechtfertigte.

### 2. Zeilentabelle `broadcast_targets`, Spalte `target_type` entfällt

```sql
CREATE TABLE broadcast_targets (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    kind         TEXT    NOT NULL CHECK(kind IN (
                     'users','members','spieler','eltern',
                     'team_spieler','team_eltern','team_trainer','alle_trainer',
                     'legacy')),
    team_id      INTEGER REFERENCES teams(id),
    PRIMARY KEY (broadcast_id, kind, team_id)
);
```

Die `team_id` trägt bewusst **kein** `ON DELETE CASCADE` und keine `NOT NULL`-Pflicht: sie ist
Teil einer historischen Aufzeichnung, kein aktiver Verweis. Ein gelöschtes Team soll die
Mitteilung nicht mitnehmen; `RESTRICT` (der SQLite-Default) ist hier richtig, weil Teams
ohnehin nicht gelöscht, sondern über `is_active` stillgelegt werden.

**Warum die Spalte weg muss statt danebenzustehen:** bei Mehrfachzielen gäbe es keinen
ehrlichen Wert für `target_type` — weder das erste Ziel noch `legacy` beschreibt die
Mitteilung. Und weil ihr CHECK die neuen Kinds nicht kennt, wäre ein Tabellen-Rebuild
sowieso fällig. Also einmal richtig.

**Migration nach dem Muster aus `049`** (nächste freie Nummer: `055`): Tabelle neu anlegen
ohne `target_type`, `INSERT … SELECT`, alte droppen, umbenennen; danach
`INSERT INTO broadcast_targets SELECT id, target_type, NULL FROM <alt>`. Entscheidend und in
`049` bereits dokumentiert: der Migrationslauf setzt `PRAGMA foreign_keys=OFF`, ein
`DROP TABLE broadcasts` löst also **keine** Cascade-Deletes auf `broadcast_reads` aus — an
der Tabelle hängt die gesamte Zustellung.

### 3. Ein unerlaubtes Ziel lehnt den ganzen Request ab

Alternative wäre, unerlaubte Ziele stillschweigend zu verwerfen und den Rest zuzustellen. Das
verbietet sich aus demselben Grund, aus dem `recipients` überhaupt existiert
(`mitteilung-zielgruppen` `design.md §5`): der Absender sieht eine erfolgreich gesendete
Mitteilung und erfährt nie, dass die Hälfte des Publikums fehlt. HTTP 403 für den ganzen
Request ist laut und behebbar.

### 4. Senderecht ist nicht Leserecht — `kader_trainers` statt `user_accessible_teams`

`GET /api/chat/team-groups` löst für Nicht-Vorstand über `user_accessible_teams` auf. Diese
View enthält auch Teams, in denen jemand als Spieler, erweiterter Kader oder **Elternteil**
hängt. Als Grundlage fürs Senden wäre das zu weit: ein Trainer, der nebenbei Vater in der mD2
ist, dürfte an „Spieler mD2" senden — nicht sein Wirkungsbereich.

**Entscheidung:** die Ziel-Allowlist eines reinen Trainers kommt aus `kader_trainers` der
aktiven Saison. Daraus folgt der eigene Endpoint `GET /api/chat/broadcast-targets`: der
Composer kann die Liste weder aus `team-groups` (zeigt mehr) noch aus den JWT-Claims
(kennen keine Kader) ableiten.

**Folge, die benannt sein will:** die Capability `broadcast_messages` wird weiterhin aus den
JWT-Claims berechnet und kann für einen Trainer **ohne** Kader-Eintrag `true` liefern,
während der Server jedes Ziel ablehnt. Der Composer verlässt sich deshalb nicht auf die
Capability allein: eine leere Ziel-Liste zeigt er als Hinweis („keine Gruppen, an die du
senden kannst"), nicht als leeres Dropdown mit aktivem Senden-Button.

### 5. `alle_trainer` steht jedem Absender mit Senderecht offen

Die Gruppe ist vereinsweit, aber sie ist kein Publikum — sie ist das Kollegium. Ein Trainer,
der im Chat längst eine Trainer-Gruppe anlegen darf, soll dieselben Leute auch als Ansage
erreichen. Das ist die einzige Ausnahme von „vereinsweit = nur Vorstand/sL/Admin", und sie
ist im Spec-Text ausdrücklich als solche notiert.

### 6. Der Composer wird eine Checkbox-Liste, kein Multi-Select

Die Vorlage aus dem Gruppen-Picker (Suchfeld + Klick fügt hinzu + Chips) ist dort nötig, weil
Einzelpersonen dazukommen und die Liste dreistellig wird. Für Mitteilungen ist die Liste
kurz: vier vereinsweite Ziele plus ~3 × Anzahl der Teams. Zwei Blöcke („Vereinsweit",
„Gruppen") mit Checkboxen und Empfängerzahl je Zeile zeigen alles auf einmal — und machen
den Fall „ich habe zwei Gruppen gewählt, die sich überschneiden" ablesbar, den ein
Chip-Feld verdeckt.

## Risks / Trade-offs

**[Trainer erreichen Eltern jetzt ohne Rückkanal]** → Eine Mitteilung an `team_eltern` lädt
nicht zur Antwort ein; Rückfragen landen auf privaten Wegen (WhatsApp, Anruf) statt im
Gruppenchat. Mitigation: keine technische, sondern die Ankündigung — der Gruppenchat bleibt
der Ort für Dialog, die Mitteilung ist für Ansagen. Steht in den Tasks.

**[Doppelzustellung gefühlter Art]** → Wer eine Ansage zusätzlich in den Gruppenchat
schreibt, erzeugt zwei Pushes für dieselbe Information. Mitigation: keine; das ist eine
Nutzungsfrage, und eine technische Sperre wäre übergriffig.

**[Die Migration fasst `broadcasts` an]** → Ein Rebuild der Tabelle, an der die gesamte
Mitteilungs-Historie hängt. Mitigation: exakt das Muster aus `049`, das bereits einmal
produktiv gelaufen ist, plus ein Test, der eine Bestandszeile über die Migration hinweg
verfolgt (`broadcast_reads` unverändert, Ziel als Zeile vorhanden). DB-Backup vor dem Deploy.

**[`recipients` kann bei überlappenden Zielen kleiner sein als die Summe der Gruppen]** →
Wer „Spieler mB1" (14) und „Eltern mB1" (11) wählt, bekommt nicht zwingend 25 — Spieler mit
eigenem Account, die zugleich Elternteil sind, zählen einmal. Das ist korrekt und gewollt;
die Zahl in der Ziel-Liste ist die Gruppengröße, die Zahl in der Antwort der tatsächliche
Fan-out. Mitigation: die Antwort formuliert „An N Empfänger gesendet", nicht „N von M".

## Migration Plan

1. Migration `055_broadcast_targets` (up/down). `down` stellt `target_type` wieder her und
   füllt es aus der ersten Zielzeile je Broadcast bzw. `legacy` bei mehreren — der Rückweg
   ist verlustbehaftet und im SQL kommentiert.
2. Backend deployen; die neue Route und das neue Request-Format gehen zusammen mit dem
   Frontend live (Single-Binary, eingebettetes SPA — kein Versionsversatz möglich).
3. Trainer informieren (Tasks): was der neue Kanal ist, und wofür weiterhin der Gruppenchat.

**Rollback:** `make migrate-down` auf `055` plus Deploy der Vorgänger-Binary. Mitteilungen,
die zwischenzeitlich an mehrere Ziele gingen, verlieren dabei alle Ziele bis auf das erste;
ihre Zustellung (`broadcast_reads`) bleibt vollständig, weil sie an der Spalte nie hing.
