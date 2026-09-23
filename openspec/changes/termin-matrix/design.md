## Kontext

Vorbild ist die SpielerPlus-Statistik: Zeile = Spieler, Spalte = Termin, Zelle = Symbol
für den Rückmeldestatus, davor eine Teilnahme-Quote. In TeamWERK gibt es schon zwei
Nachbarn, die jeweils die Hälfte können:

- Die Detailseite (`GET /api/games/{id}/participants`, `GET /api/training-sessions/{id}/attendances`)
  kennt den Status aller Spieler, aber für **einen** Termin.
- Die Anwesenheits-Statistik (`GET /api/teams/{id}/attendance-stats`) kennt **alle**
  Termine, aber nur als Summe und nur für die Vergangenheit.

## Entscheidungen

### 1. Eine Route, ein Package: `internal/attendance`

Die Matrix liest Trainings **und** Spiele. `games` und `trainings` sind Domänen, die
sich nicht gegenseitig importieren dürfen (Arch-Test); `internal/attendance` liest
schon heute beide Tabellenfamilien direkt und ist deshalb der natürliche Ort. Kein
Client-seitiges Zusammensetzen aus N Detail-Requests — bei 30 Terminen wären das 30
Anfragen pro Laden und pro Live-Update.

Die Route liegt im Authenticated-Tier; die Objektprüfung sitzt im Handler (wie
`attendance-stats`).

### 2. Sichtbarkeit: wer den Termin sehen darf, sieht die Zeilen

Den Status aller Spieler zeigt die Detailseite heute schon jedem, der den Termin
sehen darf (Kader, erweiterter Kader, Trainer, Eltern). Die Matrix erweitert das nicht,
sie bündelt es. Berechtigt sind deshalb:

- `admin`, `vorstand`, `sportliche_leitung`, Vereinsfunktion `trainer` (identisch zum
  Bypass in `GetParticipants`),
- sonst jeder, für den `user_accessible_teams` die Mannschaft in der aktiven Saison
  führt (Kader, erweiterter Kader, Trainer, Eltern beider Kader).

Sonst **403**; unbekannte Mannschaft **404** (vor der Rechteprüfung — die Existenz einer
Mannschaft ist über `/api/teams/names` ohnehin öffentlich).

**Enger** als die Detailseite: **keine fremden Gründe**. Ein Grund steht nur in Zeilen,
für die der Aufrufer antworten darf (eigene Zeile, eigene Kinder) — das ist die Teilmenge
der Regel „Gründe nur für Trainer, sich selbst und die eigenen Kinder", die ohne
Fallunterscheidung je Zelle auskommt; Trainer lesen fremde Gründe weiter auf der
Detailseite. **Anwesenheit (`present`)**
nur für die, die die Anwesenheits-Statistik sehen dürfen (`canSeeTeamStats`: Admin,
sportliche Leitung, Trainer dieser Mannschaft) — gleiche Grenze wie bei den Summen.

### 3. Zeilen = Kader der aktiven Saison, Spalten = Zeitraum

Die Zeilenmenge ist der Kader der Mannschaft in der **aktiven** Saison: Stammkader, dann
erweiterter Kader (ohne Doppelte), jeweils nach Name sortiert. Trainer sind keine Zeile.
Ohne aktive Saison ist die Zeilenmenge leer (200, leere Listen) — kein Fehler.

Spalten sind alle Trainings (`training_sessions.team_id = {id}`, auch abgesagte — sie
werden als abgesagt markiert statt still zu fehlen) und alle Spiele/Events
(`game_teams.team_id = {id}`) mit `date ∈ [from, to]`, chronologisch nach Datum + Uhrzeit.
`from`/`to` sind Pflicht; der Zeitraum ist auf **400 Tage** gedeckelt (Saison + Puffer,
analog zum Ladefenster der Liste in `terminWindow.ts`), sonst 400.

### 4. Zellwert = dieselbe Ableitung wie auf der Detailseite

```
Antwortzeile vorhanden      → status = response.status, is_default = false
sonst Voreinstellung der Rolle ∈ {confirmed, declined}
                            → status = default,         is_default = true
sonst                       → status = null (keine Rückmeldung)
Training + Serien-Abmeldung → unavailable = true (zusätzlich)
Trainer-Sicht + attendance  → present = true|false (zusätzlich)
```

Rolle heißt: Stammkader → `rsvp_default_players`, erweiterter Kader →
`rsvp_default_extended` des jeweiligen Termins. Das ist exakt die Regel aus
`GetParticipants`/`GetAttendances`; die Invariante „Matrix = Detailseite" ist der
Kern der Test-Anforderungen.

Die Zellen stehen als Array parallel zu `events` (gleicher Index), nicht als Map — die
Antwort bleibt bei 25 Spielern × 60 Terminen klein und der Client braucht keinen
Schlüssel-Lookup.

### 5. Frontend: zweite Ansicht auf `/termine`, nicht eigene Seite

`view=tabelle` schaltet um; Default bleibt die Liste. Typ-Filter und „Vergangene"
wirken auf die Spalten, das Ladefenster ist dasselbe (`terminLoadWindow`). Der
Mehrfach-Mannschaftsfilter der Liste passt nicht zur Matrix (die Zeilenmenge ist
**ein** Kader), deshalb ersetzt die Tabellenansicht ihn durch eine Einfachauswahl über
die Mannschaften des Nutzers. `team` bleibt derselbe URL-Parameter: beim Umschalten in
die Tabelle wird die erste positive ID übernommen, fehlt sie, die erste Mannschaft der
Auswahl. Das Textsuchfeld entfällt in der Tabelle.

**Teilnahme-Spalte** wird clientseitig über die **sichtbaren** Spalten gerechnet, damit
sie dem Typ-Filter folgt („Trainingsbeteiligung" = Filter auf Training). Zähler: erfasste
Anwesenheit, wenn vorhanden, sonst Zusage (inkl. Voreinstellung). Nenner: sichtbare,
nicht abgesagte Termine, bei denen der Spieler nicht per Serie abgemeldet ist.

**Layout:** Tabelle in einer Card mit eigenem horizontalem Scroll; erste Spalte (Name)
`sticky left-0`, damit der Name beim Scrollen stehen bleibt. Auch auf Mobile eine
echte Tabelle (Ausnahme von „Card-Layout statt `<table>`": eine Matrix hat keine
sinnvolle Card-Form — genau das ist ihr Zweck).

**Symbole** (lucide): Zusage `ThumbsUp` grün, Absage `ThumbsDown` danger, Vielleicht
`HelpCircle` warning, keine Rückmeldung `Circle` subtle, Serien-Abmeldung `Minus`
subtle; Voreinstellung = dasselbe Symbol mit reduzierter Deckkraft. Anwesenheit (nur
Trainer-Sicht) ersetzt das RSVP-Symbol durch `UserCheck`/`UserX`. Eine Legende unter der
Tabelle erklärt die Symbole.

### 6. Zu-/Absagen aus der Tabelle: dieselben Funktionen wie die Liste

Die Tabelle bekommt keinen eigenen RSVP-Pfad. Ein Tipp auf eine eigene Zelle öffnet einen
Dialog, dessen Schaltflächen **dieselben** Seitenfunktionen aufrufen wie die Karten der
Liste (`respondTraining`/`respondGame`/`openReasonModal`) — inklusive Begründungs-Dialog,
Fehlertext bei `rsvp_locked` und der Umschalt-Eigenheit von „Zusagen" in der eigenen Zeile.
Nach einer erfolgreichen Antwort lädt die Matrix still nach (der SSE-Echo tut das
ohnehin; das explizite Nachladen macht die Zelle ohne Verzögerung aktuell).

Welche Zeilen antippbar sind, entscheidet der Server (`can_respond`), nicht der Client:
eigenes Mitglied (`members.user_id`) oder Kind über `family_links` — genau die Zeilen, die
die Liste als „Ich" bzw. Kindername anbietet. Die Sperrinformation (Frist, Abwesenheit,
Begründungspflicht) liefert die Matrix mit, damit der Dialog ohne Zusatz-Request
entscheidet. Die Fristen standen bisher als Konstanten in `games` und `trainings`;
`attendance` darf keine der beiden Domänen importieren, deshalb liegen sie jetzt in
`internal/policy` und die beiden Domänen verweisen darauf.

### 7. „Bisher" und „Geplant"

Eine einzige Quote über Vergangenheit und Zukunft vermischt zwei Fragen: „wie zuverlässig
war der Spieler?" und „wer kommt?". Die Tabelle trennt sie am heutigen Datum (heutige
Termine zählen zu „Geplant", sie haben noch nicht stattgefunden). „Bisher" zählt die
erfasste Anwesenheit, wo es eine gibt, sonst die Zusage — für Spieler und Eltern, die die
Anwesenheit nicht sehen (§2), ist das die bestmögliche Näherung, und in der Trainer-Sicht
ist eine nicht erfasste Einheit ebenso auf die Zusage angewiesen.

## Risiken

- **Abfragegröße:** eine Mannschaft hat pro Saison ~80 Trainings + ~25 Spiele. Zwei
  Abfragen für die Spalten, zwei für die Antworten (je `IN`-freies `JOIN` über das
  Zeitfenster), eine für Anwesenheit, eine für Serien-Abmeldungen — unabhängig von der
  Spaltenzahl. Kein N+1.
