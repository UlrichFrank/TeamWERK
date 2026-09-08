## Context

Siehe `proposal.md` — Why. Für die Umsetzung zählen vier Eigenschaften des Bestands:

- `internal/calendar/handler.go` baut die drei Event-Arten in `fetchGames`, `fetchTrainings` und
  `fetchDuties` getrennt auf und rendert sie über einen gemeinsamen `renderICal`/`writeLine`-Pfad.
  Alles, was dieser Change ändert, liegt in dieser einen Datei.
- Alle benötigten Spalten existieren: `duty_slots.hours_value` (Migration `052`),
  `duty_slots.role_desc`, `duty_slots.game_id`, `games.venue_id` sowie `created_at` auf `games`,
  `training_sessions` und `duty_slots`. Keine Migration nötig.
- `hours_value` trägt **bereits den gekappten Wert** aus der Dienst-Ablösung (Migration `054`,
  `applyChainCaps`). Der Feed muss die Kette also nicht kennen — er liest den Wert, den auch das
  Dienstkonto verrechnet. Das ist der Grund, warum die richtige Dauer eine kleine Änderung ist.
- `created_at` ist ein SQLite-`DATETIME` mit `CURRENT_TIMESTAMP`-Default: der String kommt als
  `"2026-08-16 15:44:12"` an — UTC, aber **ohne Zonenkennung** (Gotcha „SQLite DATETIME-Felder",
  `docs/agent/06-gotchas.md`).

## Goals / Non-Goals

**Goals:**

- Der Dienst-Termin im Kalender bildet Ort, Dauer und Rolle so ab, wie sie in der Anwendung stehen.
- Der erzeugte Kalender erfüllt die RFC-5545-Pflichten, die ohne Schemaänderung erfüllbar sind.
- Die Faltungs-Invariante wird mechanisch geprüft, nicht nur behauptet.

**Non-Goals:**

- Kein `SEQUENCE`, kein `LAST-MODIFIED`. Beides braucht ein `updated_at`, das keine der drei
  Tabellen führt (siehe Entscheidung 2).
- Keine Änderung an Titeln (`SUMMARY`) irgendeiner Variante — das betrifft die zurückgestellten
  Fälle rund um generische Termine.
- Kein `ETag`/`If-None-Match` auf dem Feed-Endpunkt. Entscheidung 1 hält die Tür dafür offen,
  geht aber nicht durch sie.
- Kein Verweis auf die Dienst-Anleitung (`duty_types.instruction_md`) in der `DESCRIPTION`
  (siehe Entscheidung 6).

## Decisions

### 1. `DTSTAMP` kommt aus `created_at` des Termins, nicht aus `time.Now()`

RFC 5545 definiert `DTSTAMP` bei `METHOD:PUBLISH` als Erzeugungszeitpunkt des Kalenderobjekts —
`time.Now()` wäre also die buchstabengetreue Lesart. Sie hat aber eine unangenehme Folge: **jeder
Abruf liefert einen anderen Body**, auch wenn sich kein Termin geändert hat. Das macht den Feed
unvergleichbar (kein Diff zwischen zwei Abrufen), verhindert jedes spätere `ETag` und erzeugt bei
Clients, die auf Byte-Gleichheit prüfen, unnötige Arbeit.

`created_at` des jeweiligen Termins ist stabil, liegt bereits in allen drei Tabellen und ist
inhaltlich näher an dem, was ein Leser erwartet („seit wann gibt es diesen Eintrag"). Gewählt:
`created_at`, geparst **explizit als UTC** (die Spalte trägt keine Zonenkennung, siehe Context —
naives Parsen verschöbe den Wert um den lokalen Offset). Ist der Wert leer oder nicht parsebar,
fällt das Feld auf `time.Now().UTC()` zurück; ein `VEVENT` ohne `DTSTAMP` darf nicht entstehen.

*Verworfen:* `time.Now()` pro Rendering (siehe oben). *Verworfen:* eine feste Konstante — sie wäre
stabil, aber schlicht falsch.

### 2. Kein `SEQUENCE`, obwohl der Fall aus derselben Analyse stammt

`SEQUENCE` ist die Revisionsnummer eines Termins; sie ist nur nützlich, wenn sie bei jeder
Änderung steigt. Keine der drei Quelltabellen führt ein `updated_at`, und ein konstantes
`SEQUENCE:0` sagt einem Client exakt das, was er ohne das Feld ohnehin annimmt. Die ehrliche
Variante wäre `updated_at` auf `games`, `training_sessions` und `duty_slots` — mit Nachziehen an
jedem Schreibpfad, inklusive der Regen-Engine, die Slots massenhaft löscht und neu anlegt. Das ist
ein eigener Change mit Migration, kein Nebenprodukt hier.

### 3. Statische `VTIMEZONE`-Komponente statt Umstellung auf UTC

Zwei Wege führen zu einem konformen Kalender: eine `VTIMEZONE`-Komponente ergänzen, oder alle
Zeitstempel als UTC (`…Z`) ausgeben und `TZID` streichen. Beide liefern denselben Zeitpunkt.

Gewählt: `VTIMEZONE`. Die bestehende Spec, die bestehenden Tests und die Beispiele in der
Dokumentation nennen alle die `DTSTART;TZID=Europe/Berlin`-Form; UTC würde diese Zusage brechen
und die Rohdatei für Fehlersuche schlechter lesbar machen („20260919T120000Z" statt „14:00").
Der Block wird einmal pro Kalender vor dem ersten `VEVENT` ausgegeben, mit den EU-Regeln
(`CET`/`CEST`, letzter Sonntag im März bzw. Oktober).

*Trade-off:* Diese Regeln stehen dann als Literal im Code, während die tatsächlichen Zeitstempel
über `time/tzdata` gerechnet werden. Schafft die EU die Umstellung ab, wandert tzdata mit und der
Block bleibt stehen — die beiden Quellen liefen auseinander. Das ist ein angekündigtes, seit Jahren
verschobenes Ereignis mit langer Vorlaufzeit; ein Kommentar am Block hält die Abhängigkeit fest.

### 4. Faltung an Rune-Grenzen, Limit bleibt in Oktetten

RFC 5545 zählt Oktette, nicht Zeichen — das Limit von 75 bleibt also, was es ist. Geändert wird nur
die Wahl der Schnittstelle: statt hart bei Index 75 wird die größte Rune-Grenze **≤ 75 Oktette**
gesucht (Fortsetzungszeilen entsprechend ≤ 74, weil das führende Leerzeichen mitzählt). Ein
einzelner Codepoint ist höchstens 4 Byte lang, ein Fortschritt ist damit immer garantiert; eine
Endlosschleife bei „Rune passt nicht" kann es nicht geben.

*Verworfen:* Falten nach Zeichenzahl (75 Runen) — das erzeugt Zeilen bis 300 Oktette und verletzt
den RFC in die andere Richtung.

Die Invariante wird als Test formuliert, nicht als Kommentar: über einen gerenderten Feed mit
absichtlich langen Namen wird jede Zeile auf `utf8.Valid` **und** `len ≤ 75` geprüft. Das ist der
Regressionsschutz, den der heutige Code nicht hat — der Fehler war ja nicht sichtbar, solange die
Testdaten kurz genug waren.

### 5. Dienst-Ort über den Spielbezug, gemeinsamer Venue-Helfer

`fetchDuties` bekommt einen `LEFT JOIN` über `duty_slots.game_id` → `games.venue_id` → `venues`.
Beide Joins sind `LEFT`: ein Dienst ohne Spielbezug (Slots mit `team_id` statt `game_id`) und ein
Spiel ohne Venue bleiben ohne Ort — wie heute.

Die Adresszeile selbst entsteht ab jetzt an einer Stelle. Das war als eigener Refactor geplant
(Fall 06 der Analyse), wird aber hier mitgenommen: die Alternative wäre, den vorhandenen
Zwölfzeiler ein drittes Mal zu kopieren und die Zusammenführung danach nachzuholen. Sichtbares
Verhalten ändert sich dadurch nicht.

### 6. `DESCRIPTION` nur aus `role_desc`, kein Link auf die Anleitung

`duty_types.instruction_md` wäre der naheliegende zweite Bestandteil, ist aber Markdown mit
Bildreferenzen auf `/dokumente/datei/{id}`. In eine Kalenderbeschreibung gehört weder Markup noch
ein relativer Pfad; ein klickbarer Link bräuchte eine absolute Basis-URL, die der Calendar-Handler
heute nicht kennt (kein `cfg` im Struct). Zurückgestellt, bis es einen Grund gibt, dem Handler die
Konfiguration zu geben.

### 7. Dienst ohne Uhrzeit wird ein Ganztags-Event

`duty_slots.event_time` ist nullable; heute wird daraus `00:00` und ein einstündiger Termin um
Mitternacht. Mit der echten Dauer würde daraus ein dreistündiger Termin um Mitternacht — dasselbe
Artefakt, nur auffälliger. Statt es größer zu machen, wird der Fall richtig abgebildet:
`DTSTART;VALUE=DATE:<Tag>` und `DTEND;VALUE=DATE:<Folgetag>`, also ein Ganztags-Eintrag ohne
erfundene Uhrzeit.

Dafür bekommt `calEvent` ein Kennzeichen `AllDay`; `renderICal` wählt daraufhin die
`VALUE=DATE`-Form statt `TZID`. Kein weiterer Aufrufer setzt es.

## Risks / Trade-offs

- **Die neue Dienstdauer erreicht Abonnenten verzögert** → Google und Apple holen abonnierte
  Kalender im Stunden- bis Tagesrhythmus. Kein Handlungsbedarf, aber die Erwartung: der Effekt ist
  nicht sofort sichtbar, und ein Test „am eigenen Handy" braucht ggf. einen manuellen Refresh.
- **`DTSTAMP` aus `created_at` ist kein Änderungszeitstempel** → Ein bearbeiteter Termin behält
  seinen Wert. Für `METHOD:PUBLISH`-Feeds werten Clients ohnehin UID plus Inhalt aus; die saubere
  Lösung ist Entscheidung 2 und damit vertagt.
- **Bestands-Slots tragen die Dauer, die beim letzten Regen-Lauf entstanden ist** → Die
  Ablöse-Kappung rechnet Bestandsslots nicht rückwirkend nach (bekannter Rest aus Migration `054`).
  Der Feed zeigt damit denselben Wert wie das Dienstkonto — konsistent, aber nicht zwingend die
  Zeit, die tatsächlich anfällt. Das ist eine Eigenschaft der Datenlage, keine des Feeds.
- **Ganztags-Dienste verlieren ihre Sortierung innerhalb des Tages** → Ohne Uhrzeit landet der
  Eintrag im Ganztags-Band des Clients. Das ist gewollt: „irgendwann an diesem Tag" ist die
  Aussage, die in den Daten steht.
- **Statische EU-Zeitzonenregeln** → siehe Entscheidung 3.

## Migration Plan

Keine DB-Migration, kein Datenumbau, keine Änderung an gespeicherten Tokens. Deploy über den
normalen Weg (`make deploy`). Rollback ist ein Rückbau des Binaries — die Feed-URLs und
Abonnements der Nutzer bleiben in beiden Richtungen gültig, weil sich weder Pfad noch UIDs ändern.
