# Design

## Context

`dutyfairness.Compute` baut je Request einen Snapshot der aktiven Saison. Heute hängen
`Geleistet`/`Vorhersage` am `Member` (`fairness.go`); `Team.Members` hält Zeiger auf
dieselben `Member`-Objekte. Ein Kind in zwei Stammkadern ist deshalb in beiden
`Team.Members` mit **einem** Zähler vertreten — der Kommentar am Typ sagt das auch
ausdrücklich („erscheint es in beiden Ranglisten mit denselben Zahlen"). `Ranked()`,
der Ranglisten-Handler und `dashboard.queryDutyAccount` lesen diesen Zähler.

Die Gesamtsumme je Team und das Soll (`Total / PlayerCount`) sind schon heute je Team
korrekt: ein Kind in zwei Kadern zählt in beiden `PlayerCount` voll. Falsch ist nur die
Ist-Seite. Die Aushilfe (erweiterter Kader) wird bereits je (Mitglied, Team) gezählt
(`AushilfePosition`) — dieselbe Form wird jetzt für den Stammkader gebraucht.

## Goals / Non-Goals

**Goals:**
- Ist-Zählung je (Mitglied, Stammteam); Rangliste und Dashboard lesen sie.
- Für Kinder mit genau einem Stammkader bleiben alle Zahlen bit-gleich.

**Non-Goals:**
- Kein anderes Soll (Gesamtsumme, `PlayerCount`, Generik-Verteilung bleiben).
- Keine Änderung der Stufen-Reihenfolge oder der Aushilfe-Logik.
- Keine Änderung an der Dienstbörse oder an `appdb.UserTeamsSQL`.
- Keine Response-Form-Änderung, kein Frontend-Code.

## Decisions

### 1. Zweistufige Teilung: erst zwischen Mitgliedern, dann zwischen deren Kadern

`countAssignments` bestimmt wie bisher die Ziel-Mitglieder (Stufen 1/2/5) und teilt die
Zuweisung gleichmäßig auf sie (`weight = 1/len(targets)`). **Neu** ist ein zweiter
Schritt: der Anteil eines Mitglieds wird gleichmäßig auf seine *passenden*
Stammteams verteilt:
- team-gebundener Slot: `memberTeams[m] ∩ slot.teams`
- generischer Slot oder Stufe 5: alle `memberTeams[m]`

**Warum so, und nicht gleich über alle (Mitglied, Team)-Paare teilen?** Die flache
Teilung gäbe einem Kind mit zwei Kadern bei einer Eltern-Zuweisung mehr Gewicht als
seinem Geschwister mit einem Kader (2/3 statt 1/2). Die zweistufige Teilung hält die
Invariante `Σ_team Position(m, team) = bisheriger Member-Zähler` exakt — damit bleiben
Familiensummen und alle bestehenden Tests für Ein-Kader-Kinder unverändert, und der
Change ist allein durch die neuen Mehrkader-Szenarien beschreibbar.

**Warum gleichmäßig und nicht proportional zum Soll der Teams?** Für generische Slots
ist das bereits konsistent: die Generik-Verteilung gibt jedem Spieler jedes Teams
denselben Anteil (`genericTotal / totalPlayers`), ein Kind in zwei Kadern bekommt also
in beiden Teams gleich viel Generik-Soll — gleichmäßige Ist-Teilung spiegelt das. Für
das gemeinsame Spiel und Stufe 5 gibt es kein fachliches Signal, welcher Mannschaft der
Dienst „gehört"; eine Soll-Gewichtung wäre eine erfundene Präzision und für Eltern
nicht nachvollziehbar.

### 2. Datenform: Zähler je Position statt am Member

`Member` verliert `Geleistet`/`Vorhersage`. Neu ist ein Zeilentyp je (Mitglied,
Stammteam) — z. B. `Standing{Member *Member; Team *Team; Geleistet, Vorhersage float64}`
—, gehalten in `Snapshot.standings map[[2]int]*Standing` (Schlüssel wie
`aushilfe`), und `Team.Members` wird zu `[]*Standing`. `loadMembers` legt die Positionen
beim Laden an, sodass jede Stammkader-Zeile genau eine Position hat (auch mit 0).

Alternative verworfen: Zähler als `map[teamID]float64` am `Member` behalten. Das hätte
`Ranked()` zwingen, jedes Mal den Team-Kontext mitzuschleppen, und `PositionsFor`
müsste Member + Team wieder zusammensetzen — die Position ist das eigentliche Objekt,
genau wie bei der Aushilfe.

`dutyfairness.Position` (Dashboard) und die neue Stammposition sind dann dieselbe
Sache; `PositionsFor` liefert direkt die Positionen, `queryDutyAccount` liest
`p.Geleistet` statt `p.Member.Geleistet`.

### 3. Sortierung unverändert, nur auf den Positionswert

`Team.Ranked()` sortiert die Positionen nach `Round2(Geleistet+Vorhersage)` absteigend,
Gleichstand `member_id` aufsteigend — wie bisher, nur auf den neuen Werten. Viertel-
Anteile (0,25 bei Geschwister + zwei Kadern) sind in `Round2` exakt darstellbar; ein
Drittel aus einem Drei-Kader-Kind wird wie bisher auf zwei Stellen gerundet.

### 4. Kein Feature-Flag, kein Übergang

Die Zahlen sind live gerechnet, es gibt keinen gespeicherten Stand, der migriert werden
müsste. Mit dem Deploy zeigen Rangliste und Kachel die neuen Werte.

## Risks / Trade-offs

- [Platzierungen springen für betroffene Familien sichtbar] → gewollt; Benutzerhandbuch
  erklärt die Teilung, betroffen sind nur Kinder mit mehreren Stammkadern. Vorab-Zählung
  der betroffenen Kinder (nicht ausführen, dem Nutzer übergeben):
  ```sql
  SELECT m.id, m.first_name || ' ' || m.last_name, COUNT(DISTINCT k.team_id)
  FROM kader_members km JOIN kader k ON k.id = km.kader_id
  JOIN members m ON m.id = km.member_id
  JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
  WHERE k.team_id IS NOT NULL
  GROUP BY m.id HAVING COUNT(DISTINCT k.team_id) > 1;
  ```
- [Ein Kind in zwei Kadern hat jetzt in Summe doppelt so viel Soll wie Ist-Kapazität
  aus einem Dienst] → das war schon vorher so (Soll je Team voll); bisher verdeckte die
  Doppelzählung der Ist-Seite das. Die ehrliche Anzeige ist das Ziel dieses Changes. Ob
  ein Mehrkader-Kind nur anteilig in `PlayerCount` eingehen sollte, ist eine separate
  Fairness-Frage und bewusst nicht Teil hiervon.
- [Die Paritäts-Aussage „Kachel = Rangliste"] bleibt gewahrt, weil beide weiter aus
  demselben Snapshot lesen → abgesichert durch einen Dashboard-Test mit Mehrkader-Kind.
- [Merge-Reihenfolge] Das Delta auf „Geleistet- und Vorhersage-Zählung je Kind" setzt
  den Text von `dienste-erweiterter-kader` voraus → jener Change wird zuerst archiviert
  (Task 1.1).
