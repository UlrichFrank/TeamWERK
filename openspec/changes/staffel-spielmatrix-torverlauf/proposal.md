## Why

Der Reiter „Verlauf" auf `/staffeln` zeigt heute den Tabellenverlauf der **Staffel** —
zehn Mannschaften, eine Platzierungskurve je Spieltag. Was fehlt, ist der Blick auf die
**eigene Mannschaft über ihre Spiele**: wer hat in welchem Spiel getroffen, wer war wann
dabei, und wie ist ein Spiel eigentlich gelaufen.

Beides liegt vollständig in der Datenbank und kostet keinen zusätzlichen Abruf beim
Verband:

- `bwhv_player_games` trägt je ausgewertetem Bericht und Spieler Tore, Siebenmeter,
  Zeitstrafen und Karten. Heute wird daraus nur über die **ganze Saison** summiert
  (Ranglisten, `StaffelStats`) — die Zeile je Spiel ist da, wird aber nie gezeigt.
- `bwhv_events` trägt jedes Tor mit Spielzeit in Sekunden, Seite und Schütze. Heute
  zeigt `SpielberichtPanel` daraus eine Differenzkurve („wer führt") und eine
  Ereignisliste. Nicht sichtbar ist, was daneben in denselben Zeilen steht: **Läufe**
  (drei Tore hintereinander) und die **Spielsituation**, in der ein Tor fiel.

Vorbild ist wie schon bei `staffel-statistiken` die öffentliche Auswertung
`UlrichFrank/handballnet_crawler` (`components/handball/GameTable.tsx`,
`GameTimelineDialog.tsx`), dort clientseitig aus PDF-Zeilen gerechnet.

## What Changes

- **Spielmatrix der eigenen Mannschaft** im Reiter „Verlauf", unter dem bestehenden
  Tabellenverlauf: Zeile = Spieler, Spaltengruppe = Spiel, Zelle = seine Werte in diesem
  Spiel (Tore · 7m-Versuche · 7m-Tore · 2-Min · Gelb · Rot), rechts eine Saisonsumme je
  Spieler mit Spielzahl, unten eine Mannschaftssumme je Spiel. Spaltenkopf trägt Datum,
  Endstand und die Paarung.
- **Die Mannschaft wird abgeleitet, nicht gewählt**: dieselbe Regel, die schon die
  Zeilen-Hervorhebung trägt (`Store.Affiliation` über `bwhv_games.game_id` → `games` →
  `game_teams`). Ohne verknüpfte Begegnung gibt es keine Matrix, sondern einen Hinweis —
  eine über Namensähnlichkeit geratene Mannschaft wäre unsichtbar falsch.
- **Tor-Momentum je Spiel**: ein Knopf „Ablauf" im Spaltenkopf öffnet einen Dialog mit
  einer Zeitachse je Halbzeit. Jedes Tor ist ein Kreis auf der Achse der werfenden
  Mannschaft; die **Größe** zeigt den Lauf (wievieltes Tor in Folge ohne Gegentreffer),
  die **Farbe** die Spielsituation nach diesem Tor (in Führung · unentschieden · im
  Rückstand). Zeiger/Tastaturfokus nennt Minute, Schütze, Spielstand und Siebenmeter.
- **Die Spieldauer kommt aus den Einstellungen**, nicht aus einer Annahme: die Matrix
  liefert je Mannschaft die `half_duration_minutes` ihrer Altersklasse mit
  (`age_class_game_rules` über `teams.age_class` — derselbe Weg wie bei der
  Dienst-Dauer). Der Bericht behält das letzte Wort: widerspricht die gepflegte Zahl dem
  Verlauf, wird sie verworfen (§7). Welches Tor zu welcher Halbzeit gehört, hängt an ihr
  ohnehin nicht — das entscheidet der Halbzeitstand aus dem Berichtskopf (§6).
- **Keine neue Route für den Ablauf**: `GET /api/bwhv-games/{id}/report` liefert die
  Ereignisse bereits vollständig. Momentum, Situation und Halbzeitgrenze sind reine
  Ableitung im Browser, in `web/src/lib/torverlauf.ts` gekapselt und mit Vitest geprüft.
- **Eine neue Lese-Route** für die Matrix: `GET /api/staffeln/{id}/player-games`. Die
  Summen rechnet der Server (SQL-Aggregat in `internal/gamestats`), wie schon bei allen
  anderen Staffel-Statistiken — das Frontend lädt nicht alle Berichte einer Saison, um
  daraus selbst zu summieren.

Keine Breaking Changes: bestehende Routen, Antwortformate und Reiter bleiben. Die
Differenzkurve („Torkurve") im Spielbericht-Panel bleibt unverändert — sie beantwortet
eine andere Frage (§8).

## Capabilities

### New Capabilities
- keine

### Modified Capabilities
- `bwhv-staffel-statistiken`: um die Spielmatrix einer Mannschaft (Spieler × Spiel) und
  die daraus abgeleiteten Summen erweitert.
- `bwhv-staffel-tabellen`: Der Reiter „Verlauf" trägt künftig zwei Darstellungen
  (Tabellenverlauf der Staffel **und** Spielmatrix der eigenen Mannschaft samt
  Tor-Momentum je Spiel). Zugleich wird eine seit `bwhv-spielberichte` abgedriftete
  Zusage berichtigt: der Spielbericht wird sehr wohl in dieser Ansicht gezeigt
  (aufklappbar im Spielplan), nicht nur in der Detailansicht des Spiels.

## Impact

**Backend** (`internal/gamestats`): eine neue Lese-Route im Authenticated-Tier,
`GET /api/staffeln/{id}/player-games`, mit der Aggregat-Funktion `PlayerGameMatrix` in
`aggregates.go`. Keine Mutations-Route, also kein `Broadcast`; die Seite hängt am
bestehenden SSE-Ereignis `bwhv-updated`. Die Antwort ist **nutzerabhängig** (die
Zugehörigkeit entscheidet, welche Mannschaft) — damit die zweite nach `/affiliation`;
die übrigen Statistik-Routen bleiben nutzerunabhängig.

**Frontend**: `web/src/lib/torverlauf.ts` (Ableitung: Läufe, Situation, Halbzeitgrenze,
Achsenlänge samt Prüfung der konfigurierten Spieldauer — reine Funktionen, Vitest), `web/src/components/staffeln/Spielmatrix.tsx`
(Tabelle mit fixierter Namensspalte), `web/src/components/staffeln/TorMomentum.tsx`
(Inline-SVG, eine Achse je Halbzeit), Dialog dafür, sowie Typen und Abruf in
`web/src/lib/staffeln.ts`. Der Reiter „Verlauf" in `StaffelnPage.tsx` rendert beides
untereinander. Neue Abhängigkeiten: **keine** (kein Canvas, keine Diagramm-Bibliothek —
dieselbe Begründung wie bei `StandingsChart`).

**Datenbank**: **keine Migration**. Alle Werte sind Aggregate über vorhandene Zeilen.

**Nicht betroffen**: `games` (unverändert ohne Ergebnisspalten), der Abruf beim Verband,
Dienst-Regeneration, iCal-Feed, RSVP, Anwesenheit, `SpielberichtPanel`.

## Test-Anforderungen

Pflicht nach `docs/agent/07-testing.md`: je Route Happy-Path und Fehlerfall, plus die
Invariante, die der Test garantiert.

| Route | Test | Status | Garantierte Invariante |
|---|---|---|---|
| `GET /api/staffeln/{id}/player-games` | `TestGetPlayerGames_MatrixDerEigenenMannschaft` | 200 | Je Spieler der eigenen Mannschaft eine Zeile, je gespieltem Spiel eine Spalte, Zelle trägt die Werte dieses Berichts |
| | `TestGetPlayerGames_OhneZugehoerigkeitLeer` | 200 | Nutzer ohne verknüpfte Begegnung erhält `teams: []` — keine über Namensähnlichkeit geratene Mannschaft (design.md §2) |
| | `TestGetPlayerGames_SpielOhneBerichtIstSpalteOhneWerte` | 200 | Gespieltes Spiel ohne ausgewerteten Bericht ist eine Spalte mit `hasReport=false`; alle Zellen bleiben leer, der Endstand steht trotzdem im Kopf (design.md §3) |
| | `TestGetPlayerGames_NichtImKaderIstLeerNichtNull` | 200 | Ein Spieler ohne Zeile in der Mannschaftsliste eines Berichts hat dort **keine Zelle** (`null`), nicht eine Zelle mit `0` (design.md §4) |
| | `TestGetPlayerGames_DritteZeitstrafeZaehltAlsRot` | 200 | `twoMinCounted`/`redCounted` greifen auch hier: drei Zeitstrafen ⇒ `2× 2min + 1× Rot`, nicht `3× 2min` (Gotcha „Staffel-Statistiken" §6) |
| | `TestGetPlayerGames_SchreibweiseDesSpielplans` | 200 | Die Mannschaft der Zeile wird über `pg.side` an den Spielplan-Namen gebunden, nicht über `bwhv_players.team_name` (design.md §5) |
| | `TestGetPlayerGames_ParseFailedLiefertKeineWerte` | 200 | Ein Bericht im Zustand `parse_failed` steuert keine Zelle bei |
| | `TestGetPlayerGames_ZweiEigeneMannschaftenInEinerStaffel` | 200 | Zwei eigene Mannschaften derselben Staffel ergeben zwei Einträge in `teams`, jeder mit eigenen Spielen und Spielern (design.md §2) |
| | `TestGetPlayerGames_SpieldauerAusDenEinstellungen` | 200 | Die Antwort trägt je Mannschaft die `half_duration_minutes` ihrer Altersklasse aus `age_class_game_rules` |
| | `TestGetPlayerGames_OhneAltersklassenRegelKeineSpieldauer` | 200 | Ohne gepflegte Regel bleibt das Feld leer (`null`) statt auf einen erfundenen Standardwert zu fallen |
| | `TestGetPlayerGames_UnbekannteStaffel` | 404 | Unbekannte Staffel-ID erfindet keine Matrix |
| | `TestGetPlayerGames_OhneToken` | 401 | Authenticated-Tier greift |

**Reine Einheitentests ohne Route** (Vitest, `web/src/lib/torverlauf.test.ts`):

| Gegenstand | Test | Garantierte Invariante |
|---|---|---|
| Lauf (Momentum) | `zaehlt aufeinanderfolgende Tore derselben Mannschaft` | Drei Tore in Folge ⇒ 1, 2, 3; das Gegentor setzt auf 1 zurück |
| Situation | `bestimmt Fuehrung, Gleichstand und Rueckstand aus Sicht des Schuetzen` | Situation bezieht sich auf den Stand **nach** dem Tor und auf die werfende Mannschaft |
| Halbzeitgrenze | `trennt die Halbzeiten am Halbzeitstand des Kopfes` | Erste Halbzeit endet mit dem Tor, das den Halbzeitstand herstellt — nie über eine geschätzte Spielzeit (design.md §6) |
| | `ohne Halbzeitstand bleibt eine durchgehende Achse` | Fehlt `homeGoalsHt`, gibt es eine Achse statt zweier falscher |
| Achsenlänge | `nutzt die konfigurierte Spieldauer` | Eine zur Begegnung passende Altersklassen-Regel wird übernommen (design.md §7) |
| | `verwirft eine zu lange konfigurierte Spieldauer` | 30 Minuten bei einer 2×25-Begegnung (Fall der Fixture `905272`) würden das erste Tor der zweiten Halbzeit vor den Achsenbeginn legen — die Zahl wird verworfen |
| | `verwirft eine zu kurze konfigurierte Spieldauer` | Ein Tor der ersten Halbzeit jenseits des gepflegten Endes verwirft die Zahl ebenfalls |
| | `leitet die Spieldauer ohne Regel aus dem Verlauf ab` | Ohne `half_duration_minutes` trifft die Ableitung 2×30, 2×25 und 2×20 |
| | `kein Tor liegt ausserhalb seiner Achse` | Invariante über alle vier Fälle — die Achse darf zu lang, nie zu kurz sein (design.md §7) |
| | `die Halbzeitzuordnung haengt nicht an der Spieldauer` | Dieselbe Torreihe mit drei verschiedenen Spieldauern ergibt dieselbe Aufteilung auf die Halbzeiten (design.md §6) |
| Nur Tore | `Auszeiten und Strafen erzeugen keinen Kreis` | Nur `goal`/`seven_m_goal` zählen; ein verworfener Siebenmeter ist kein Tor |
| Siebenmeter | `kennzeichnet Tore aus Siebenmetern` | `seven_m_goal` ist als solcher erkennbar |

**Komponententests** (Vitest + jsdom):

| Gegenstand | Test | Garantierte Invariante |
|---|---|---|
| `Spielmatrix` | `unterscheidet nicht im Kader von null Toren` | `–` und `0` sind zwei verschiedene Anzeigen (design.md §4) |
| | `zeigt Ablauf nur bei ausgewertetem Bericht` | Kein Knopf ohne Tordaten |
| | `nennt Spiele mit und ohne Bericht getrennt` | Die Fußzeile weist beide Zahlen aus — eine fehlende Auswertung bleibt sichtbar (design.md §3) |
| `TorMomentum` | `jeder Kreis traegt eine Beschriftung` | Minute, Schütze und Spielstand stehen in `<title>`; Farbe ist nicht der einzige Träger der Aussage (design.md §9) |

**Gates, die mitlaufen müssen:** Objektrechte-Matrix (`internal/permissions/
object_matrix_test.go` — die neue `{id}`-Route braucht einen `openByDesign`-Eintrag,
sonst rot), Broadcast-Gate (Lese-Route, kein Allowlist-Eintrag nötig),
`buttonStyles.gate.test.ts` und die brand-Token-Prüfung für die neuen Komponenten.
