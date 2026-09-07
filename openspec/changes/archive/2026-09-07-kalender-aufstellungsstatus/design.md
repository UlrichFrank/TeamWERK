## Context

Siehe `proposal.md — Why` für die Motivation. Technisch relevant ist der Ist-Zustand von
`internal/calendar/handler.go`:

- `fetchGames` zieht die Termine über eine Union-Ableitung `kaderMembership`
  (`kader_members` / `kader_trainers` / `kader_extended_members`) und liefert pro Zeile ein
  Flag `is_extended`. Ein Nutzer kann über mehrere Kader an demselben Spiel hängen; die
  Query sortiert deshalb `ORDER BY … mem.is_extended` und die Schleife nimmt über
  `seen[id]` die erste Zeile — reguläre Zugehörigkeit schlägt erweiterte.
- Die Beschriftung entsteht in drei kleinen Helfern: `kaderLabel(teamName, isExtended)`,
  `ownTeamLabel(label)` und `gameTitle(eventType, isHome, opponent, teamLabel)`.
  `kaderLabel` wird **auch** von `fetchTrainings` benutzt.
- `calEvent.Description` trägt heute ausschließlich `games.note` und wird in `renderICal`
  nur ausgegeben, wenn es nicht leer ist.
- Die Aufstellung liegt in `game_lineup (game_id, member_id, added_by, added_at)`, PK
  `(game_id, member_id)`. `SaveLineup` löscht alle Zeilen des Spiels und schreibt die neue
  Menge — eine leere Aufstellung ist von „nie gepflegt" **nicht** unterscheidbar.

Der Feed ist ein reiner Lesepfad ohne Authentifizierung (Token in der URL) und ohne
SSE-Broadcast. Es gibt keine Migration und kein neues Schema.

## Goals / Non-Goals

**Goals:**

- Der Aufstellungsstatus steht im Kalendereintrag selbst — ohne dass der Empfänger die
  Anwendung öffnet.
- Die drei Zustände bleiben unterscheidbar; „keine Aufstellung gepflegt" wird nie zu einer
  Absage umgedeutet.
- Genau eine Stelle entscheidet über die Beschriftung, damit Spiel und Training denselben
  Kader-Zusatz tragen.

**Non-Goals:**

- Keine Benachrichtigung bei Änderung der Aufstellung. Der Feed wird gepollt; eine Push
  „Du bist jetzt aufgestellt" wäre ein eigener Change mit eigenem Empfängerkreis.
- Kein Status für den Stammkader und keiner für `generisch`-Events (siehe `proposal.md`).
- Keine Erweiterung der Push-/E-Mail-Meldungen auf den erweiterten Kader. Dass
  `teamMembersAndParents` ihn nicht erreicht, ist der Anlass dieses Changes, aber nicht
  sein Gegenstand — das wäre eine Änderung an der Empfängermenge dreier Meldungen
  (Anlage, Änderung, Absage) und gehört getrennt entschieden.

## Decisions

### 1. Status kommt aus zwei EXISTS-Spalten, nicht aus einem zweiten Query

`fetchGames` bekommt zwei zusätzliche Spalten:

```sql
EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id = g.id)                             AS lineup_exists,
EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id = g.id AND gl.member_id = mem.member_id) AS in_lineup
```

Beide hängen an der bereits vorhandenen Zeile (`mem.member_id` ist im Join da), kosten also
keinen weiteren Roundtrip und keine Map im Speicher. Der PK `(game_id, member_id)` deckt
beide Prüfungen ab.

*Alternative:* ein separater `SELECT game_id, member_id FROM game_lineup WHERE game_id IN (…)`
nach der Hauptquery und Auflösung in Go. Verworfen: zweiter Query, zweite Datenstruktur,
und der Feed hat keinen Hot-Path-Druck, der das rechtfertigt.

*Alternative:* `LEFT JOIN game_lineup`. Verworfen, weil der Join die Zeilenzahl verändert
und mit der bestehenden `seen[id]`/`ORDER BY`-Dedup kollidiert.

### 2. Drei Zustände als eigener Typ, nicht als zwei Booleans an der Beschriftung

Die Schleife leitet aus `lineup_exists`/`in_lineup` **einmal** einen Zustand ab
(`aufgestellt` / `nicht aufgestellt` / `offen`) und reicht ihn an Titel- und
Beschreibungs-Aufbau weiter. Zwei rohe Booleans an zwei Stellen auszuwerten hieße, die
Regel „leer ≠ nicht nominiert" zweimal zu schreiben — genau dort entsteht die
Falschaussage, die dieser Change verhindern soll.

Der Zustand gilt nur, wenn `is_extended` **und** `event_type IN ('heim','auswärts')`; sonst
bleibt er leer und weder Titel noch Beschreibung ändern sich.

### 3. `kaderLabel` wird gekürzt — für Spiel und Training gemeinsam

`<team> - erweiterter Kader` wird zu `<team> · erw. Kader`. Der Grund ist der Titel: mit dem
Status daneben wäre die Klammer sonst
`(mB1 - erweiterter Kader - nicht aufgestellt)` und damit auch in der Tagesansicht abgeschnitten.

Der Helfer ist geteilt, also ändert sich der Zusatz **auch beim Training**. Das ist
beabsichtigt: derselbe Zusatz in zwei Schreibweisen im selben Kalender wäre schlechter als
eine einmalige Umbenennung. Der Trenner ist der Mittelpunkt `·` — er grenzt die drei
Bestandteile optisch stärker ab als das bisherige `-`, das im Titel schon zweimal vorkommt
(`Heim: … – Gegner`).

*Alternative:* den Status vor den Titel ziehen (`Aufgestellt · Heim: …`). Fachlich besser
sichtbar, vom Auftraggeber aber ausdrücklich zugunsten der kompakten Klammer verworfen.
Der Preis ist in „Risks" festgehalten.

### 4. Beschreibung: Notiz und Satz, getrennt durch eine Leerzeile

`calEvent.Description` wird aus zwei Teilen zusammengesetzt (`note`, dann `\n\n`, dann der
Satz), wobei leere Teile entfallen. `escapeText` wandelt die Zeilenumbrüche in `\n` — die
bestehende Faltung auf 75 Oktetten trägt die längere Zeile ohne Änderung.

Der Satz steht **hinter** der Notiz: die Notiz ist die Aussage des Trainers zum Termin, der
Statussatz eine generierte Ergänzung. Umgekehrt schöbe sich generierter Text vor den
redaktionellen.

### 5. Keine Änderung an `SaveLineup`

Die Kappe „leer = offen" liegt vollständig im Lesepfad. `SaveLineup` bleibt unangetastet;
insbesondere wird **nicht** versucht, „bewusst leere Aufstellung" von „nie gepflegt" zu
unterscheiden — das bräuchte eine eigene Spalte am Spiel und eine Pflege-Entscheidung des
Trainers, die heute niemand trifft.

## Risks / Trade-offs

- **Der Status ist auf kleinen Geräten oft nicht sichtbar.** Die Klammer steht hinter
  `Heim: Team (`, Kalender-Apps schneiden in der Wochenansicht nach ~20 Zeichen ab.
  → Mitigation: der vollständige Satz steht im `DESCRIPTION`, das jede App beim Öffnen des
  Termins zeigt. Bewusst akzeptierter Trade-off zugunsten des kompakten Titels; ein
  vorangestellter Status wäre die Gegenmaßnahme, falls sich das in der Praxis als zu
  versteckt erweist.
- **Verzögerung durch Client-Polling.** Ändert der Trainer die Aufstellung, sieht der
  Spieler das erst beim nächsten Feed-Abruf (Apple/Google: Stundenraster, teils länger).
  → Mitigation: keine im Rahmen dieses Changes; der Kalender ist die Zweitanzeige, die
  Anwendung bleibt die Erstquelle. Wichtig für die Erwartungshaltung in der Ankündigung.
- **Ein Spiel ohne gepflegte Aufstellung trägt dauerhaft „Aufstellung offen".** Bei Teams,
  die die Aufstellung nie pflegen, steht der Zusatz an jedem Spiel des erweiterten Kaders.
  → Mitigation: bewusst so entschieden (Auftraggeber). Der Zusatz ist ehrlich — er sagt
  „unbekannt", nicht „abgesagt" — und erzeugt beim Trainer den richtigen Druck.
- **Umbenennung des Kader-Zusatzes trifft Bestandseinträge.** Bereits synchronisierte
  Termine ändern beim nächsten Abruf ihren Titel (UID bleibt, kein Duplikat).
  → Mitigation: keine nötig; der Feed ist als veränderlich spezifiziert. In der
  Ankündigung erwähnen, damit die Änderung nicht als Fehler gemeldet wird.
- **`game_lineup` ist nicht team-gebunden.** An einem Termin mit mehreren Mannschaften ist
  die Aufstellung eine gemeinsame Liste. → Für den Status irrelevant: gefragt wird immer
  nur „steht *mein* Mitglied drin", nie „wie viele stehen drin".
