## Context

Siehe `proposal.md` — Why. Ausgangslage im Code:

- `bwhv_player_games` (Migration `068`) trägt je `(report_id, player_id)` `side`,
  `jersey_number`, `goals`, `seven_m_attempts`, `seven_m_goals`, `two_min`, `yellow`,
  `red`, `blue`. Gelesen wird sie heute nur summiert über die Saison
  (`statSelect` in `internal/gamestats/stats.go`) und je Bericht
  (`ReportForGame`) — nie als Matrix über die Spiele einer Mannschaft.
- `bwhv_events` trägt je Ereignis `seq`, `game_second`, `score_home`/`score_guest`
  (nur bei Toren gefüllt), `kind`, `side`, `player_id`. `GET /api/bwhv-games/{id}/report`
  gibt das vollständig als `ReportDetail.events` aus.
- Die Spielzeit ist **kumulativ über beide Halbzeiten**: `parseTimeline`
  (`internal/bwhv/parse_timeline.go:61`) rechnet `min*60+sec` aus der Spielzeitspalte,
  und `TestParseReport_SpielzeitInSekunden` belegt, dass sie im ganzen Bericht nie
  rückwärts läuft. Eine Rücksetzung zur Halbzeit gäbe es dort sofort als Fehlschlag.
- Die Halbzeitgrenze steht **nicht** im Verlauf, und sie liegt auch nicht beim letzten
  Tor der ersten Halbzeit. Beleg ist die eingecheckte Fixture
  `internal/bwhv/testdata/spielbericht_905272.pdf`: Endstand `29:25 (14:13)`, der
  Halbzeitstand `14:13` fällt bei Spielzeit **21:57**, das nächste Ereignis steht bei
  **25:27**, das letzte bei **49:06**. Die Halbzeiten dieser Begegnung dauern also
  25 Minuten, und zwischen 21:57 und 25:00 fiel schlicht kein Tor mehr. Nebenbei
  widerlegt dieselbe Zeile die naheliegende Annahme „B-Jugend spielt 2×30": die
  Begegnung ist eine `mB-RL-BW`.
- Die Spieldauer ist **konfiguriert**: `age_class_game_rules(age_class,
  half_duration_minutes, break_minutes)` wird unter Einstellungen gepflegt und über
  `teams.age_class` aufgelöst — derselbe Weg, den `internal/games/regen.go:1339` für die
  Dienst-Dauer geht. Die Tabelle deckt `A-`/`B-`/`C-`/`D-Jugend` ab und ist nicht
  vorbefüllt; `teams.age_class` ist dagegen freier Text, kann also auch eine Klasse
  ohne Regel tragen.
- Die **harte Kreuzprobe** (`internal/bwhv/crosscheck.go`) garantiert für jeden Bericht
  im Zustand `parsed`: Kopf-Endstand = Summe der Tor-Ereignisse. Ein abweichender
  Bericht wird `parse_failed` und trägt keine Zeile.
- `Store.Affiliation` (`aggregates.go:689`) löst die eigenen Mannschaftsnamen einer
  Staffel über `bwhv_games.game_id` → `games` → `game_teams` → `user_accessible_teams`
  auf — in der Schreibweise des **Spielplans**.
- `SpielberichtPanel` zeigt bereits eine `Torkurve` (Spielstand-Differenz über die Zeit)
  und eine Ereignisliste.
- `web/package.json` enthält **keine** Diagramm-Bibliothek; `StandingsChart` ist
  Inline-SVG.
- Höchste Migration ist `069`.

## Goals / Non-Goals

**Goals**

- Die Werte, die je Spieler und Spiel schon gespeichert sind, nebeneinander sichtbar
  machen — mit Saisonsumme je Spieler und Mannschaftssumme je Spiel.
- Den Ablauf eines Spiels so zeigen, dass Läufe und Spielsituation ablesbar sind.
- Ohne zusätzlichen Fremdabruf, ohne neue Tabelle, ohne neue Abhängigkeit.

**Non-Goals**

- Keine Matrix für fremde Mannschaften (§2).
- Kein Ersatz für die vorhandene Torkurve im Spielbericht (§8).
- Keine Auswertung über Saisons hinweg — die Staffel ist saisongebunden; die
  Saisonbilanz einer Person steht weiterhin unter `/api/members/{id}/saisonstatistik`.
- Keine Export-Funktion (das Vorbild erzeugt eine XLS; hier nicht gefragt).

## Decisions

### §1 Der Server rechnet, das Frontend stellt dar

Die Matrix ist ein SQL-Aggregat (`PlayerGameMatrix` in `internal/gamestats/
aggregates.go`), keine Client-Summierung. Das ist dieselbe Entscheidung wie bei allen
Statistiken dieser Ansicht, und sie hat hier einen zusätzlichen Grund: die Alternative
wäre, alle Berichte einer Saison einzeln über `/api/bwhv-games/{id}/report` zu laden —
bei 18 Spielen 18 Abrufe mit je zwei Mannschaftslisten und ~70 Ereignissen, für eine
Tabelle, die am Ende 20 Zeilen hat.

Der **Ablauf-Dialog** dagegen bekommt **keine** eigene Route: er braucht genau eine
vorhandene Antwort (`/api/bwhv-games/{id}/report`), und zwar erst beim Öffnen. Momentum
und Situation sind Ableitungen über eine bereits gelieferte Liste — dafür eine Route zu
bauen, hieße dieselben Ereignisse ein zweites Mal zu serialisieren.

### §2 Die Mannschaft wird abgeleitet — und es können mehrere sein

Die Matrix gilt für die **eigene** Mannschaft, aufgelöst über `Store.Affiliation`. Kein
Auswahlfeld über alle zehn Mannschaften der Staffel: die Frage lautet „wie läuft unsere
Saison", nicht „wie läuft die des Gegners", und jede zusätzliche Auswahl neben dem schon
vorhandenen Mannschafts-Umschalter im Seitenkopf wäre eine zweite, verwechselbare
Mannschaftsauswahl auf derselben Seite.

Ein Namensvergleich gegen `teams.name` kommt weiterhin nicht in Frage (die Begründung
steht ausführlich an `Affiliation`): ein Fehlschluss wäre **unsichtbar**, weil die
falsche Matrix genauso plausibel aussieht wie die richtige.

**Preis:** ohne eine einzige verknüpfte Begegnung — Saisonbeginn, bevor ein eigener
Termin mit `external_id` importiert ist — gibt es keine Matrix. Die Antwort ist dann
`teams: []`, und die Ansicht sagt warum. Das ist dieselbe bewusste Lücke wie bei der
Zeilen-Hervorhebung: eine fehlende Darstellung fällt auf, eine falsche nicht.

**Mehrzahl statt Sonderfall:** `Affiliation.TeamNames` ist eine Liste, und zwei eigene
Mannschaften in derselben Staffel sind möglich (zweite und dritte Mannschaft eines
Vereins in einer Bezirksklasse). Die Antwort ist deshalb `{"teams": [...]}` mit je
eigener Spiel- und Spielerliste, und die Ansicht rendert eine Matrix je Eintrag. Die
Alternative — „die erste gewinnt" — wäre ein stiller Verlust genau dort, wo er am
wenigsten auffällt.

### §3 Spalten sind alle gespielten Begegnungen, nicht nur die mit Bericht

Eine Spalte entsteht für jede Begegnung der Mannschaft mit erfasstem Ergebnis
(`home_goals IS NOT NULL`) — auch wenn dazu **kein** ausgewerteter Bericht vorliegt.
Diese Spalte trägt Datum, Paarung und Endstand im Kopf, aber keine Zellen, und ist als
„kein Bericht" gekennzeichnet.

Die naheliegende Alternative wäre, solche Spalten wegzulassen. Dann wäre aber die
Saisonsumme je Spieler über eine unbekannte Teilmenge gebildet, und die Lücke wäre
unsichtbar — genau die Fehlerklasse, wegen der `TeamStat` schon heute `Games` und
`ReportGames` getrennt ausweist und `FairPlay` bei fehlendem Bericht `nil` statt `0`
liefert. Die Fußzeile der Matrix nennt deshalb beide Zahlen („18 Spiele, davon 15 mit
Bericht").

Nicht gespielte Begegnungen (kein Ergebnis) erzeugen **keine** Spalte — eine
Vorschau-Spalte ohne jeden Wert wäre nur Breite.

### §4 „–" und „0" sind zwei verschiedene Aussagen

Eine Zelle ist `null`, wenn der Spieler in der Mannschaftsliste dieses Berichts **nicht
steht** (nicht im Kader, verletzt, gesperrt); sie trägt `0`, wenn er dabei war und nicht
getroffen hat. In der Anzeige `–` gegen `0`. Die Unterscheidung kostet nichts — sie ist
genau die Unterscheidung zwischen „keine Zeile in `bwhv_player_games`" und „Zeile mit
`goals = 0`" — und ohne sie wäre die Spalte „Spiele" je Spieler nicht erklärbar.

Die Spalte „Spiele" zählt entsprechend die Berichte, in deren Mannschaftsliste der
Spieler steht — dieselbe Definition wie `PlayerStat.Games` heute.

### §5 Die Mannschaft einer Spielerzeile spricht die Schreibweise des Spielplans

`bwhv_players.team_name` stammt aus der Mannschaftsliste des PDF,
`bwhv_games.home_team`/`guest_team` aus der JSON-Schnittstelle — dieselbe Mannschaft,
zwei mögliche Schreibweisen. `Affiliation` liefert die des Spielplans. Die Zuordnung
läuft deshalb über den bereits vorhandenen Ausdruck `teamNameExpr`
(`CASE pg.side WHEN 'home' THEN g.home_team ELSE g.guest_team END`), nicht über
`p.team_name`. Ein Vergleich gegen den Berichtsnamen liefe bei abweichender Schreibweise
auf eine **leere** Matrix hinaus — ohne Fehler, ohne Hinweis.

Ebenso gelten hier die beiden Umrechnungs-Ausdrücke `twoMinCounted`/`redCounted`: die
dritte Zeitstrafe ist die Rote Karte und zählt einmal, nicht zweimal. Neue Abfragen auf
`two_min`/`red` müssen sie nutzen — das steht schon als Regel in den Gotchas und gilt
für diese Abfrage genauso.

### §6 Die Halbzeitgrenze kommt aus dem Halbzeitstand, nie aus der Zeit

Der Bericht enthält **keine** Halbzeitmarke im Verlauf. Die Spieluhr steht in der Pause
still; sichtbar ist die Pause nur in der Uhrzeitspalte, und die ist als Trennsignal
unzuverlässig (eine lange Auszeit sieht genauso aus).

Verlässlich ist dagegen der **Halbzeitstand aus dem Kopf** (`bwhv_games.home_goals_ht`/
`guest_goals_ht`). Die Regel lautet deshalb: die erste Halbzeit endet mit dem Tor, mit
dem der mitgezählte Spielstand erstmals `(ht_home, ht_guest)` erreicht; alle Ereignisse
mit höherem `seq` gehören zur zweiten. Das ist eine Aussage über die **Reihenfolge**,
nicht über die Zeit, und damit unabhängig von jeder Spielzeit-Annahme.

Der Spielstand wird dabei **mitgezählt**, nicht aus `score_home`/`score_guest` gelesen:
das Feld ist nur bei Toren gefüllt, und die Kreuzprobe garantiert für jeden `parsed`
Bericht, dass die Summe der Tor-Ereignisse den Endstand ergibt. Das Zählen kann also
nicht auseinanderlaufen.

Umgekehrt gilt weiterhin die Regel aus `bwhv-spielberichte`: der Halbzeitstand wird
**nie aus dem Verlauf abgeleitet**. Hier wird er genutzt, nicht erzeugt.

**Fehlt der Halbzeitstand** (nullable), gibt es eine durchgehende Achse statt zweier
falscher. Das ist sichtbar anders und damit ehrlich.

### §7 Die Spieldauer kommt aus den Einstellungen — der Bericht hat das letzte Wort

Für die x-Position eines Tores der zweiten Halbzeit wird die Länge der ersten gebraucht,
denn die Spielzeit läuft kumulativ weiter (§6). Diese Länge ist **konfiguriert**:
`age_class_game_rules.half_duration_minutes`, aufgelöst über `teams.age_class` der
eigenen Mannschaft. Jede Spalte der Matrix ist ein Spiel **unserer** Mannschaft, deren
`teams.id` die Zugehörigkeit ohnehin schon liefert — die Regel steht damit ohne
Zusatzabfrage bereit und wird als `halfDurationMinutes` je Mannschaft mit der Matrix
ausgeliefert. Der Ablauf-Dialog wird nur aus der Matrix geöffnet und hat den Wert
deshalb immer in der Hand.

Der konfigurierte Wert wird aber **nicht blind** übernommen, denn er kann aus zwei
Gründen nicht zur Begegnung passen: die Altersklasse hat keine Regel (die Tabelle deckt
nur A- bis D-Jugend ab und ist nicht vorbefüllt, `teams.age_class` ist freier Text), oder
die gepflegte Zahl widerspricht dem Dokument. Der zweite Fall ist nicht theoretisch: die
Fixture zeigt eine `mB-RL-BW` mit **25-Minuten-Halbzeiten**, und „B-Jugend = 30" ist ein
naheliegender Eintrag. Mit 30 gezeichnet läge das erste Tor der zweiten Halbzeit (25:27)
bei Minute −4:33, also **vor** dem Beginn seiner eigenen Achse.

Der Bericht entscheidet deshalb, ob die Zahl passt:

```
abgeleitet = max( auf5Gerundet( ceil(maxSpielsekunde/60) / 2 ),
                  ceil(letzteSekundeHZ1 / 60) )

halbzeitMinuten =
    konfiguriert,   wenn konfiguriert vorhanden
                    und letzteSekundeHZ1  <= konfiguriert*60
                    und ersteSekundeHZ2   >= konfiguriert*60
    abgeleitet,     sonst
```

Die beiden Bedingungen sind genau die beiden Arten, auf die eine Achse unmöglich würde:
ein Tor der ersten Halbzeit jenseits ihres Endes, oder ein Tor der zweiten vor ihrem
Anfang. Was sie durchlassen, ist zeichenbar. Der abgeleitete Wert trifft die
gebräuchlichen Spielzeiten, weil die letzte Verlaufszeile eines Handballspiels in den
Schlussminuten liegt (Fixture: 49:06 bei 50 Minuten Spielzeit ⇒ 25), und sein zweiter
Term garantiert dieselbe Invariante auch ohne jede Konfiguration.

**Invariante, die in jedem Fall gilt: kein Tor liegt außerhalb seiner Achse.** Eine zu
lange Achse staucht die Kreise; ein abgeschnittenes Tor wäre ein verschwundener
Datenpunkt.

Und weiterhin gilt die Trennung aus §6: **welches Tor zu welcher Halbzeit gehört, hängt
an dieser Zahl nicht.** Ein falsch gepflegter Wert verschiebt höchstens Positionen auf
der Achse — er kann kein Tor in die falsche Halbzeit sortieren. Genau deshalb ist die
Zuordnung am Halbzeitstand festgemacht und nicht an einem Zeitvergleich, wie ihn das
Vorbild macht (`GameTimelineDialog.tsx:101`: `goal.time_in_minutes < halfDuration`) —
dort kippt bei falscher Spieldauer die Halbzeitzugehörigkeit mit.

### §8 Die neue Darstellung tritt neben die Torkurve, sie ersetzt sie nicht

`SpielberichtPanel` zeigt bereits eine Differenzkurve über die Spielzeit. Sie beantwortet
„wer lag wann vorn und wie deutlich" — eine Kurve über einen Wert. Das Momentum-Bild
beantwortet „wann fielen die Tore, in welchen Läufen und aus welcher Lage" — eine
Punktwolke über zwei Mannschaften. Dieselben Zeilen, zwei Fragen.

Die neue Darstellung wird deshalb **nicht** in `SpielberichtPanel` eingebaut: das Panel
ist mit zwei Mannschaftslisten, Kurve und Ereignisliste bereits lang, und eine zweite
Grafik unmittelbar über derselben Datenbasis lädt eher zum Vergleichen der Grafiken ein
als zum Lesen des Spiels. Sie lebt im Ablauf-Dialog der Matrix, dort, wo die Frage
entsteht.

### §9 Inline-SVG, und Farbe ist nicht der einzige Träger

Das Vorbild zeichnet auf ein `<canvas>` und baut Trefferprüfung für den Zeiger von Hand
(Abstandsrechnung je Kreis, `devicePixelRatio`-Skalierung, eigener Tooltip). Hier wird
es Inline-SVG, wie bei `StandingsChart`: ein Kreis ist ein `<circle>`, der Tooltip ein
`<title>`, und Tastaturfokus gibt es geschenkt. Kein Canvas, keine Bibliothek, kein
`ResizeObserver`.

Die Palette sind **brand-Tokens als Zahlenwert** (wie in `StandingsChart` begründet: die
Werte gehen an SVG-Attribute, eine Tailwind-Klasse ließe sich dafür nicht zur Bauzeit
erzeugen):

| Situation nach dem Tor | Token | Wert |
|---|---|---|
| in Führung | `brand-blue` | `#3E4A98` |
| unentschieden | `brand-text-subtle` | `#9CA3AF` |
| im Rückstand | `brand-warning` | `#F59E0B` |

Farbe ist dabei nicht der einzige Träger: jeder Kreis hat ein `<title>` mit Minute,
Schütze, Spielstand und Siebenmeter-Vermerk, und unter der Grafik steht eine Legende.
Die Größe trägt eine zweite, unabhängige Aussage (den Lauf), sodass die Grafik auch ohne
Farbunterscheidung etwas zeigt.

Radius: `4 + lauf * 1.5` px bei einer Achse von 640×120 — dieselbe Staffelung wie im
Vorbild, um einen Punkt kleiner, weil die Achse hier zweigeteilt und damit schmaler ist.

### §10 Keine Spalte „Blau"

`bwhv_player_games.blue` wird von `SaveReport` **konstant als `0`** geschrieben — der
Parser liest keine blauen Karten. Eine Spalte, die immer `–` zeigt, behauptet eine
Messung, die nicht stattfindet. Sie entfällt; das Vorbild führt sie mit, weil es dort
dieselbe leere Spalte aus derselben Quelle erbt.

Bleiben sechs Werte je Spiel: Tore · 7m-Versuche · 7m-Tore · 2-Min · Gelb · Rot.

### §11 Mobile: eine Zahl je Spiel

Sechs Spalten × 18 Spiele sind 108 Spalten. Auf dem Telefon zeigt die Matrix deshalb je
Spiel **nur die Tore**, ab `sm:` alle sechs Werte; die Namensspalte ist in beiden Fällen
fixiert (`sticky left-0`), die Tabelle scrollt waagerecht. Das Card-Layout aus den
Mobile-Konventionen greift hier bewusst nicht: eine Kreuztabelle in Karten aufzulösen
hieße, ihre einzige Aussage — den Vergleich über die Zeile — wegzuwerfen. Dieselbe
Ausnahme gilt schon für die Kreuztabelle der Staffel.

### §12 Der Reiter „Verlauf" trägt beides

`?tab=verlauf` zeigt künftig oben den Tabellenverlauf der Staffel, darunter die
Spielmatrix der eigenen Mannschaft. Beide beantworten „wie ist es gelaufen", einmal für
die Staffel, einmal für uns; ein eigener Reiter hätte die Leiste auf zehn getrieben, und
bestehende Verweise auf `?tab=verlauf` bleiben gültig.

## Risks / Trade-offs

- **Eine ungepflegte oder falsche Altersklassen-Regel** (§7) führt zu einer etwas zu
  langen Achse, nie zu einer falschen Halbzeitzuordnung und nie zu einem
  abgeschnittenen Tor. Die Fixture zeigt, dass der naheliegende Eintrag für die
  B-Jugend (30) für eine reale Begegnung dieser Klasse falsch wäre — die Prüfung gegen
  den Bericht ist deshalb kein Vorratsschutz, sondern der Regelfall.
- **`age_class_game_rules` deckt nur A- bis D-Jugend ab.** Mannschaften anderer Klassen
  (E-Jugend, Aktive) fallen auf die Ableitung aus dem Verlauf zurück. Die Tabelle zu
  erweitern ist eine eigene Frage der Einstellungen und nicht Teil dieses Changes.
- **Ohne Zuordnung keine Matrix** (§2). Trifft den Saisonbeginn und Mannschaften, deren
  Spiele nie über den H4A-Import kamen. Der Hinweistext nennt den Grund und den Weg
  (Spielimport bzw. `external_id` am eigenen Termin).
- **Breite Tabelle.** Eine volle Saison ist auf dem Desktop scrollbar, aber kein
  bequemer Überblick. Bewusst wie im Vorbild belassen: die Stärke der Darstellung ist
  der Vergleich über die Zeile.
- **Antwortgröße.** 20 Spieler × 18 Spiele × 6 Werte ≈ 2 000 Zahlen, als JSON gut unter
  100 kB. Ohne Paginierung — eine Staffel-Saison ist endlich und die Route wird selten
  gerufen.
- **Nutzerabhängige Antwort.** `player-games` ist nach `affiliation` die zweite Route
  dieser Ansicht, deren Inhalt am Token hängt. Das bleibt die Ausnahme; die vier
  Statistik-Routen bleiben für alle Nutzer gleich.

## Migration Plan

Keine Migration, keine Datenwanderung, kein Abruf beim Verband. Die Matrix ist ab dem
Deploy für jede Staffel gefüllt, zu der bereits ausgewertete Berichte vorliegen; ohne
Berichte zeigt sie die Spalten der gespielten Begegnungen ohne Werte (§3).

Rollback ist folgenlos: die Route verschwindet, der Reiter zeigt wieder nur den
Tabellenverlauf.

## Open Questions

- Soll das Momentum-Bild später die Torkurve in `SpielberichtPanel` ersetzen oder dort
  danebentreten? Erst beantworten, wenn beide eine Saison lang benutzt wurden (§8).
- Die Wertungsregel der Zwei-Punkte-Staffel (`pointsWin`/`pointsDraw`) bleibt unberührt —
  die Matrix rechnet keine Punkte.
