## Context

Siehe `proposal.md` — Why. Ausgangslage im Code:

- `bwhv_games` trägt je Begegnung `date`, `home_goals`/`guest_goals` (nullable) und den
  Halbzeitstand. Die Tabelle ist **unabhängig vom PDF** gefüllt, weil Spielplan und
  Ergebnisse aus der JSON-Schnittstelle kommen, nicht aus dem Bericht.
- `bwhv_player_games` trägt je Bericht und Spieler `goals`, `seven_m_attempts`,
  `seven_m_goals`, `two_min`, `yellow`, `red`, `blue`. Nur für Berichte im Zustand
  `parsed` — `parse_failed` erzeugt bewusst keine Zeile (`bwhv-spielberichte`).
- `bwhv_reports.referees` ist heute **eine ungetrennte Textzeile**: `parseReferees`
  (`internal/bwhv/parse_header.go:96`) nimmt `textLine.Text()` und verliert damit die
  X-Positionen, die `textLine.Groups` trägt (`internal/bwhv/pdf.go:15`).
- `internal/gamestats` ist Domänen-Paket und importiert kein anderes Domänen-Paket.
- `web/package.json` enthält **keine** Chart-Bibliothek.
- Höchste Migration ist `068`.

Vorbild ist `UlrichFrank/handballnet_crawler` (`frontend/src/services/dataService.ts`,
`components/handball/CrossTable.tsx`, `StandingsChart.tsx`,
`components/statistics/*.tsx`). Es rechnet clientseitig und kennt nur Begegnungen mit
ausgewertetem PDF — genau der Unterschied, der unten Entscheidung 1 begründet.

## Goals / Non-Goals

**Goals:**

- Alle Statistiken der Vorbild-Seite auf `/staffeln`, aus vorhandenen Daten gerechnet.
- Kreuztabelle, Tabellenverlauf und die drei Tor-Statistiken funktionieren **ohne ein
  einziges ausgewertetes PDF** — sie brauchen nur Ergebnisse.
- Keine neue Laufzeit-Abhängigkeit im Frontend (1 GB RAM auf dem VPS, PWA-Cache).
- Die Rechenregeln liegen in Go und sind mit Tabellen-Tests belegbar.

**Non-Goals:**

- Kein zusätzlicher Abruf beim Verband. Der Poll-Zyklus bleibt unangetastet.
- Keine Ergebnisspalten an `games` — die Grenze aus `bwhv-spielberichte` §7 bleibt.
- Keine Statistik über mehrere Saisons oder Staffeln hinweg. Bezugsraum ist immer **eine**
  Staffel, wie bei Tabelle und Spielplan.
- Keine Export-Funktion (CSV/PDF) der Statistiken.
- Kein Mannschafts-Vergleich über Staffelgrenzen und keine Prognose-Rechnung.

## Decisions

### 1. Quelle der Tor-Statistiken sind die Ergebnisse, nicht die Spielerzeilen

Torverhältnis, Angriff, Verteidigung, Kreuztabelle und Tabellenverlauf werden aus
`bwhv_games.home_goals`/`guest_goals` gebildet.

Das Vorbild summiert stattdessen die `goals` der Spielerzeilen beider Mannschaften
(`dataService.getTeamRatioStats`). Das hat zwei Folgen, die wir nicht wollen: eine
Begegnung ohne freigegebenes PDF fehlt in jeder Statistik, und ein Bericht mit
abweichender Mannschaftsliste (in `bwhv-spielberichte` ein *gespeicherter* Fall mit
Warnung) liefert eine andere Summe als der amtliche Endstand.

Aus derselben Quelle folgt die Definition von „gespielt": `home_goals IS NOT NULL`. Das
Vorbild muss dafür `homeGoals === 0 && awayGoals === 0` als „nicht gespielt" behandeln
(`getStandingsProgression`) und verliert damit jedes echte torlose Spiel. Mit der
NULL-Prüfung braucht es diese Krücke nicht — eine der Anforderungen hält das fest.

Fair-Play, Torverteilung, Torschützen und Siebenmeter bleiben zwangsläufig an
`bwhv_player_games` und damit an den Berichten.

**Folge für die Darstellung:** In derselben Ansicht stehen Statistiken mit
unterschiedlicher Abdeckung nebeneinander (alle Begegnungen vs. nur berichtete). Eine
Mannschaft ohne Bericht darf in der Fair-Play-Wertung deshalb nicht als straffrei
erscheinen; ihre Wertung ist **leer**, nicht `0`. Jeder Reiter weist aus, auf wie vielen
Spielen er beruht.

### 2. Punktewertung fest 2:0 / 1:1

Der Tabellenverlauf setzt Sieg = 2, Unentschieden = 1:1, Niederlage = 0 an, wie das
Vorbild.

*Verworfene Alternative:* die Wertung aus `bwhv_staffeln.table_json` kalibrieren
(`PointsPlus + PointsMinus == 2 × Games` ⇒ Zwei-Punkte-Wertung) und den rekonstruierten
Endstand gegen die amtliche Tabelle prüfen. Das wäre ableitbar statt angenommen — auf
Wunsch bewusst nicht gebaut (Entscheidung des Auftraggebers, 21.09.2026).

**Konsequenz, die im Code sichtbar bleiben muss:** die Annahme steht als benannte
Konstante mit Kommentar an einer Stelle, nicht als Literal in der Query. In einer Staffel
mit Drei-Punkte-Wertung zeigt der Verlauf plausible, aber falsche Ränge; erkennbar wäre
das nur an einer Abweichung des letzten Spieltags von der amtlichen Tabelle. Die
Nachrüstung ist damit ein Einzeiler an einer Konstante plus die Vergleichsprüfung.

### 3. Spieltag ist ein Datum

Der Verlauf gruppiert nach `bwhv_games.date`: alle an einem Datum gespielten Begegnungen
bilden einen Punkt der Kurve. Eine Rundennummer liefert die Schnittstelle nicht, und der
Verband spielt Staffeln mit unterschiedlich vielen Begegnungen je Termin — eine gezählte
„Runde" wäre eine erfundene Ordnung. Das Vorbild nimmt dafür seine Dateinamen
(`yyyymmdd.json`), was auf dasselbe hinausläuft.

Das Datum wird auf `date[:10]` getrimmt, bevor es Gruppierungsschlüssel wird — die
DATE-Falle aus `docs/agent/06-gotchas.md` gilt hier genauso wie in `loadBulkRangeGames`.

### 4. Vier neue Lese-Routen, Spieler-Ranglisten bleiben eine

- `GET /api/staffeln/{id}/kreuztabelle`
- `GET /api/staffeln/{id}/tabellenverlauf`
- `GET /api/staffeln/{id}/teamstatistik`
- `GET /api/staffeln/{id}/schiedsrichter`

`teamstatistik` liefert **alle fünf** Mannschafts-Sichten in einer Antwort: sie entstehen
aus derselben Aggregation über dieselben Zeilen, fünf Routen wären fünfmal dieselbe Query
für fünf Reiter derselben Seite.

Die Siebenmeter-Rangliste bekommt **keine** eigene Route. `/ranglisten` liefert je
Spieler schon Versuche und Treffer und wird nur um `games` erweitert; „nach 7m-Treffern
sortiert, ohne Spieler ohne Versuch" ist Darstellung derselben Menge, nicht eine zweite
Aggregation. Der Fehlversuch (`attempts − goals`) wird serverseitig ausgewiesen, damit
die Subtraktion nicht an zwei Stellen lebt.

Alle vier sind Lese-Routen im **Authenticated-Tier**, wie Tabelle, Spielplan und
Ranglisten. Kein `Broadcast` — das Broadcast-Gate greift nur bei Mutationen, und die
Ansicht hängt am bestehenden SSE-Ereignis `bwhv-updated`.

Jede neue `{id}`-Route braucht einen Eintrag in `openByDesign` der Objektrechte-Matrix
(`internal/permissions/object_matrix_test.go`), sonst ist der Test rot. Begründung ist
dieselbe wie bei den drei bestehenden: öffentlich abrufbare Verbandsdaten, vereinsweit
sichtbar.

### 5. Aggregate in SQL, ein eigenes File

Die Aggregate kommen als `GROUP BY`-Queries in eine neue Datei `internal/gamestats/
aggregates.go`. `stats.go` trägt schon Spieler-Bilanz und Berichtsdetail; die
Mannschafts- und Verlaufsrechnung dort anzuhängen macht die Datei zum Sammelbecken.

Ausnahme ist der **Tabellenverlauf**: er ist eine Schleife über Spieltage mit
kumulierendem Zustand und Neu-Sortierung nach jedem Schritt. Als Query wäre das ein
Window-Function-Konstrukt, das niemand mehr liest; er läuft deshalb in Go über die nach
Datum sortierten Begegnungen (~810 Zeilen je Staffel, eine Query, keine N+1).

### 6. Gini über die Saisonsumme je Spieler, nicht je Spieler-Spiel

Das Maß der Ungleichverteilung wird über die **Saisonsummen** der Spieler einer
Mannschaft gebildet. Das Vorbild schiebt pro Spiel eine Zeile je Spieler in dieselbe
Liste (`getGoalDistributionStats`) und misst damit die Streuung über Spieler-*Spiele* —
ein Spieler mit vielen Einsätzen erscheint mehrfach, und wer einmal ausfällt, zieht den
Wert nach oben, ohne dass sich die Rollenverteilung geändert hat. Gemeint ist „hängt die
Mannschaft an einzelnen Werfern", und das ist eine Frage an die Saisonbilanz.

Formel bleibt die übliche: `G = 2·Σ(i·xᵢ)/(n²·μ) − (n+1)/n` über aufsteigend sortierte
Werte, `0` bei `μ = 0` (eine Mannschaft ohne Tor hat keine Verteilung). Median und
Durchschnitt daneben, weil der Gini allein schwer zu lesen ist.

### 7. Schiedsrichter: Spalten trennen, Namensregel nur als Rückfall

`parseReferees` liefert künftig `[]string` und arbeitet auf `textLine.Groups` statt auf
`.Text()`: zwei Läufe ⇒ zwei Namen, und die Trennung stammt aus dem Dokument. Das ist
derselbe Weg, den `bwhv-spielberichte` für die Mannschaftsliste vorschreibt
(„Spaltengrenzen aus der Kopfzeile, nicht als feste Koordinaten").

Nur wenn das Dokument **einen** Lauf liefert, greift eine Namensregel (Aufteilung an der
Wortmitte in zwei Vorname-Nachname-Paare). Die rät und kann bei Doppelnamen,
Adelspartikeln und Bindestrichnamen falsch schneiden — deshalb wird dieser Fall am
Bericht als unsicher vermerkt und in der Rangliste gekennzeichnet, statt als gesichertes
Ergebnis durchzulaufen.

Speicherung: Migration `069` ergänzt `bwhv_reports.referees_json` (die getrennten Namen)
und `referees_uncertain`. **`referees` bleibt** — die Rohzeile ist der Beleg und erlaubt
eine verbesserte Trennung ohne erneuten Fremdabruf, genau wie das behaltene PDF. Additiv,
also rollback-fähig nach der Deploy-Konvention.

Bestandsberichte behalten ihre Rohzeile und tragen zunächst kein `referees_json`; sie
erscheinen erst nach einem Reparse in der Rangliste. Ein Backfill, der die Rohzeilen
nachträglich durch die Namensregel schickt, ist bewusst **nicht** Teil dieses Changes: er
würde für den gesamten Bestand den unsicheren Pfad zum Hauptpfad machen.

### 8. Fieberkurve als Inline-SVG

Eine Komponente `StandingsChart` zeichnet `<polyline>` je Mannschaft in einem
`viewBox`-SVG, Y-Achse invertiert (Rang 1 oben), X-Achse die Spieltage. Farben aus den
`brand-*`-Tokens; bei mehr Mannschaften als Tokens wird über eine feste Palette
rotiert und zusätzlich die Strichführung variiert, damit die Zuordnung ohne Farbsehen
möglich bleibt.

*Verworfene Alternative:* `recharts` (~95 kB gzip), wie im Vorbild. Tooltip, Legende und
responsives Verhalten wären geschenkt, dafür eine Abhängigkeit, die sonst niemand im
Projekt nutzt, plus Bundle im PWA-Cache. Ein Rang-Verlauf ist eine Polyline — das
rechtfertigt die Bibliothek nicht.

Hover/Legende bauen wir selbst: Legende als Liste unter der Grafik, Hervorhebung der
Linie bei Hover über den Legendeneintrag. Auf Mobile (unter `sm:`) horizontal scrollbar
in einem `overflow-x-auto`-Container mit fester Mindestbreite — dasselbe Muster wie die
Kreuztabelle.

### 9. Kreuztabelle: gedrehte Spaltenköpfe, feste Breiten

Bei zehn Mannschaften sind die Spaltenköpfe lange Vereinsnamen. Das Vorbild dreht sie um
90° und setzt `table-layout: fixed` mit fester Kopfzeilen-Höhe (`CrossTable.tsx`); das
übernehmen wir, weil es das Problem ohne Abkürzungstabelle löst. Erste Spalte fixiert,
Rest in `overflow-x-auto`.

### 10. Eigene Zugehörigkeit wird abgeleitet, nicht über Namen geraten

Die Hervorhebung braucht zwei Antworten: welche **Mannschaften** der Staffel sind die
eigenen, und welche **Spielerzeilen** gehören dem Nutzer.

Die Spieler sind der einfache Teil: `bwhv_players.member_id` ist für eigene Spieler schon
gesetzt (`spieler-saisonstatistik`). Abgeglichen wird gegen die Mitglieder des Accounts
(`members.user_id`) plus die Kinder über `family_links` — dieselbe Menge, die
`dutyfairness` und `attendance.canSeeMemberStats` heranziehen.

Bei den Mannschaften ist der naheliegende Weg falsch: `bwhv_games.home_team` trägt die
**Schreibweise des Verbands** („Team Stuttgart 2"), `teams.name` die des Vereins. Ein
Namensvergleich müsste Vereinsnamen, Mannschaftsnummer und Suffixe aufeinander abbilden —
`internal/h4aimport/staffel.go` tut genau das für den Spielimport und braucht dafür
`TeamNumberFromAlias`/`TeamNumberFromName` plus die Regel „die 2 spielt immer in der
niedrigeren Staffel". Diese Maschinerie hier zu wiederholen, hieße dieselbe Ratearbeit an
einer zweiten Stelle zu pflegen, und ein Fehlschluss wäre unsichtbar: hervorgehoben wäre
die falsche Zeile, und nichts würde widersprechen.

Stattdessen liefert die Verknüpfung die Antwort umsonst. Eine Begegnung mit
`bwhv_games.game_id IS NOT NULL` **ist** ein eigenes Spiel; `games.is_home` sagt, auf
welcher Seite die eigene Mannschaft steht, `game_teams` welche es ist. Daraus fällt der
BWHV-Name der eigenen Mannschaft ohne jeden Namensvergleich ab:

```
game_teams.team_id ∈ user_accessible_teams(user)
  ∧ bwhv_games.game_id = games.id
  ⇒ eigener Name = is_home ? home_team : guest_team
```

`user_accessible_teams` ist dabei die richtige Quelle, weil sie Stammkader, erweiterten
Kader, Trainer und Eltern schon zusammenfasst — die Menge, die die Anforderung nennt.

**Preis dieser Wahl:** ohne eine einzige verknüpfte Begegnung gibt es keine
Hervorhebung. Das tritt auf, solange zu einer Staffel noch kein eigenes Spiel importiert
oder angelegt ist (Saisonbeginn). Der Spec-Satz „keine Zugehörigkeit, wenn sie nicht
belegbar ist" nennt das ausdrücklich — eine fehlende Hervorhebung ist ein sichtbarer
Mangel, eine falsche wäre eine stille Fehlinformation.

**Eigene Route, nicht in jede Antwort gemischt:** `GET /api/staffeln/{id}/affiliation`
liefert `{teamNames: [...], playerIds: [...]}`. Die vier Statistik-Routen bleiben damit
nutzerunabhängig — sonst hinge jede Antwort am Token, wäre pro Nutzer verschieden und
jeder Statistik-Test müsste die Zugehörigkeit mit auswerten. Das Frontend lädt die Route
einmal je Staffel neben den Statistiken und hat die Mengen für alle neun Reiter.

Der Routenname ist englisch nach der Konvention aus `docs/agent/04-api-db.md`, auch wenn
die Ansicht selbst unter einem deutschen Pfad liegt.

### 11. Hervorhebung: fett plus Zeilenmarkierung

Die Zeile wird `font-semibold` gesetzt **und** bekommt einen dezenten Hintergrund aus
`brand-table-select`. Fett allein genügt nicht: in Torschützen-, Angriffs- und
Fair-Play-Tabellen ist die Wertspalte schon fett (so auch im Vorbild), eine zusätzlich
fett gesetzte Zeile wäre dort kaum vom Normalfall zu unterscheiden. In der Kreuztabelle
werden Zeile **und** Spalte der eigenen Mannschaft markiert, im Verlauf bekommt die eigene
Linie die doppelte Strichstärke.

Weil eine Auszeichnung, die nur aus Schriftschnitt und Hintergrund besteht, an einem
Screenreader vorbeigeht, trägt die eigene Zeile zusätzlich `aria-current="true"`. Das ist
der Grund, warum die Anforderung „nicht ausschließlich visuell" lautet.

## Risks / Trade-offs

- **Zwei verschiedene Abdeckungen in einer Ansicht** (alle Begegnungen vs. nur
  berichtete) → jeder Reiter weist die Zahl der zugrunde liegenden Spiele aus; fehlende
  Berichte ergeben eine **leere**, nicht eine nullwertige Wertung (Entscheidung 1).
- **Feste Zwei-Punkte-Wertung** → in einer Drei-Punkte-Staffel zeigt der Verlauf falsche
  Ränge, ohne dass es auffällt. Mitigation: benannte Konstante mit Kommentar an einer
  Stelle; die Kalibrierung aus `table_json` bleibt als nachrüstbare Option beschrieben
  (Entscheidung 2).
- **Namensregel schneidet falsch** → greift nur im Einspalten-Fall, wird am Bericht
  vermerkt und in der Rangliste gekennzeichnet; die Rohzeile bleibt erhalten, ein
  Reparse nach verbesserter Regel ist ohne Fremdabruf möglich (Entscheidung 7).
- **Schiedsrichter-Zahlen werden als Bewertung gelesen** („der pfeift streng") → die
  Rangliste zählt die Strafen *des Spiels*, nicht der Person; die Ansicht sagt das
  ausdrücklich. Das lässt sich nicht technisch verhindern.
- **Gini ohne Erklärung ist unlesbar** → Median und Durchschnitt stehen daneben, die
  Spalte trägt eine kurze Erläuterung.
- **Neun Reiter sind viel für eine Seite**, auf Mobile besonders → die Reiterleiste
  scrollt horizontal (`overflow-x-auto`), der gewählte Reiter steht in der Adresse.
- **Keine Hervorhebung am Saisonanfang**, solange keine eigene Begegnung verknüpft ist
  (Entscheidung 10) → bewusst in Kauf genommen: eine fehlende Markierung fällt auf, eine
  falsche nicht. Sobald der erste eigene Termin mit `external_id` importiert ist, greift
  sie von selbst.
- **Zugehörigkeit wandert mit dem Kader** → verlässt ein Spieler die Mannschaft, endet die
  Hervorhebung, weil `user_accessible_teams` die aktive Saison abfragt. Das ist gewollt,
  unterscheidet sich aber von den eingefrorenen Empfängermengen des Event-Logs; wer den
  Unterschied nicht kennt, könnte es für einen Fehler halten.
- **Rechenlast**: vier zusätzliche Aggregate über ~810 Begegnungen und die Spielerzeilen
  je Staffelwechsel. Alles indexgestützt (`idx_bwhv_games_staffel_date`,
  `idx_bwhv_player_games_player`), kein Cache — bei dieser Größenordnung ist ein Cache
  die teurere Komplexität. Wird im Betrieb nachgemessen, nicht vorab optimiert.

## Migration Plan

1. Migration `069` (additiv: `referees_json`, `referees_uncertain` an `bwhv_reports`) —
   läuft mit `make deploy` bzw. `make migrate-remote-up`.
2. Deploy. Die neuen Reiter sind unmittelbar gefüllt, soweit sie an Ergebnissen hängen
   (Kreuztabelle, Verlauf, Torverhältnis, Angriff, Verteidigung) — dafür genügt der
   vorhandene `bwhv_games`-Bestand.
3. Fair-Play, Torverteilung und Spieler-Ranglisten sind aus den vorhandenen Berichten
   sofort gefüllt.
4. Die **Schiedsrichter-Rangliste bleibt zunächst leer**, weil Bestandsberichte kein
   `referees_json` tragen. Sie füllt sich mit den Berichten kommender Spieltage. Wer sie
   für den Bestand rückwirkend will, braucht einen Reparse — eigener Change, siehe
   Entscheidung 7.

**Rollback:** `make deploy-rollback`. Die Migration ist additiv, die Spalten stören das
Vorgänger-Binary nicht; kein `migrate down` nötig (Deploy-Konvention).

## Open Questions

- Ob die Fieberkurve bei zehn Mannschaften ohne Interaktion lesbar bleibt, zeigt erst die
  Darstellung mit echten Daten. Falls nicht, ist die Antwort ein Filter auf einzelne
  Mannschaften — additiv, ohne Wirkung auf Spec, Schnittstelle oder Aufgabenschnitt.
