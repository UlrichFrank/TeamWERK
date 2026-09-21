## Why

Die Ansicht `/staffeln` zeigt heute Tabelle, Spielplan und eine Spieler-Rangliste. Die
Daten, die dort schon in der Datenbank liegen, geben deutlich mehr her: `bwhv_games`
trägt Endstand und Halbzeitstand **aller** Begegnungen einer Staffel (~810 je Saison,
auch ohne Spielbericht-PDF), `bwhv_player_games` die Einzelwerte jedes Spielers und
`bwhv_reports` die Schiedsrichter. Kreuztabelle, Tabellenverlauf und die üblichen
Mannschafts-Ranglisten sind daraus reine Rechnung — kein zusätzlicher Fremdabruf, kein
neues Datum beim Verband.

Vorbild ist die öffentliche Auswertung unter `ulrichfrank.github.io/handballnet_crawler`
(Repo `UlrichFrank/handballnet_crawler`), die dieselben Statistiken aus derselben Quelle
bildet — dort aber clientseitig aus den PDF-Spielerzeilen, weshalb sie Begegnungen ohne
Bericht nicht kennt.

## What Changes

- **Kreuztabelle** je Staffel: Heimmannschaft (Zeile) × Gastmannschaft (Spalte), Zelle
  trägt den Endstand oder — bei noch nicht gespielten Begegnungen — das Datum.
- **Tabellenverlauf** („Fieberkurve"): Platzierung jeder Mannschaft nach jedem Spieltag,
  rückgerechnet aus den Ergebnissen. Darstellung als Inline-SVG, **keine neue
  Chart-Bibliothek** (RAM- und Bundle-Constraint).
- **Fünf Mannschafts-Statistiken**: Torverhältnis (geworfen/bekommen/Differenz), bester
  Angriff (Tore, Ø je Spiel), beste Verteidigung (Gegentore, Ø je Spiel), Fair-Play
  (Zeitstrafen und Karten mit Punktwertung), Torverteilung (Ø und Median der Tore je
  Spieler plus Gini-Koeffizient als Maß der Abhängigkeit von einzelnen Werfern).
- **Schiedsrichter-Statistik**: Spiele je Schiedsrichter und die in diesen Spielen
  verhängten Strafen.
- **Spieler-Ranglisten erweitert**: Torschützen bekommen die Spalte „Spiele", die
  Siebenmeter-Schützen werden eine eigene Rangliste (nach 7m-Treffern sortiert, mit
  Fehlversuchen), Fair-Play bleibt.
- **Schiedsrichter-Namen werden einzeln erfasst.** Der Parser legt sie heute als eine
  ungetrennte Zeile ab (`bwhv_reports.referees`); künftig trennt er sie an den
  Spaltenpositionen der PDF-Textebene — dieselbe Quelle, aus der schon die
  Mannschaftsliste ihre Spaltengrenzen zieht. Eine Namensheuristik greift nur, wenn das
  Layout eine einzige Spalte liefert, und der unsichere Schnitt wird am Bericht vermerkt
  statt stillschweigend übernommen.
- **Eigene Mannschaft und eigene Person hervorgehoben**: In jeder Tabelle und jeder
  Statistik der Ansicht — Tabellenstand und Spielplan eingeschlossen — ist die Mannschaft
  des Nutzers hervorgehoben, in den Spieler-Ranglisten zusätzlich seine eigene Zeile
  (bei Eltern die ihrer Kinder). Die Zuordnung wird aus der Verknüpfung zwischen
  BWHV-Begegnung und eigenem Spieltermin abgeleitet, nicht über einen Namensvergleich
  geraten.
- Alle Auswertungen rechnet der **Server** (SQL-Aggregate in `internal/gamestats`); das
  Frontend stellt nur dar und lädt nicht den vollen Spielplan, um daraus Summen zu bilden.

Keine Breaking Changes: alle bestehenden Routen und Antwortformate bleiben, die
Spieler-Rangliste wird additiv erweitert.

## Capabilities

### New Capabilities
- `bwhv-staffel-statistiken`: Kreuztabelle, Tabellenverlauf und die aus Ergebnissen und
  Spielberichten gebildeten Mannschafts- und Schiedsrichter-Auswertungen einer Staffel,
  samt der Regeln, wann eine Begegnung als gespielt gilt und wie die Platzierung je
  Spieltag zustande kommt.

### Modified Capabilities
- `bwhv-staffel-tabellen`: Die Ansicht „Staffeln" trägt bisher drei Reiter (Tabelle,
  Spielplan, Ranglisten); die Anforderung wird auf die erweiterte Reiter-Struktur
  angehoben.
- `spieler-saisonstatistik`: Die zugesagten Ranglisten je Staffel werden um die Zahl der
  Spiele und eine eigene Siebenmeter-Rangliste erweitert.
- `bwhv-spielberichte`: Die Schiedsrichter eines Berichts werden als einzelne Personen
  erfasst statt als eine Textzeile; Quelle der Trennung sind die Spaltenpositionen der
  Textebene.

## Impact

**Backend** (`internal/gamestats`): neue Lese-Routen im Authenticated-Tier —
`GET /api/staffeln/{id}/kreuztabelle`, `/tabellenverlauf`, `/teamstatistik`,
`/schiedsrichter`, `/affiliation`; `/ranglisten` erweitert. Neue Datei für die Aggregate,
damit `stats.go` nicht weiter wächst. Keine Mutations-Route, also kein `Broadcast` — die
Seite hängt am bestehenden `bwhv-updated`.

`/affiliation` ist die einzige nutzerabhängige der neuen Routen; die vier
Statistik-Routen bleiben für alle Nutzer gleich. Sie liest `user_accessible_teams` und
`bwhv_players.member_id` — keine neue Zugehörigkeitslogik, nur die Verbindung zur
BWHV-Mannschaftsbezeichnung über `bwhv_games.game_id`.

**Parser** (`internal/bwhv`): `parseReferees` liefert statt eines Strings eine Liste;
`bwhv_reports.referees` bekommt eine zweite Spalte für die getrennten Namen (Migration,
additiv). Bestandsberichte behalten ihre Zeile und werden beim nächsten Reparse
nachgezogen.

**Frontend** (`web/src/pages/StaffelnPage.tsx`, `web/src/lib/staffeln.ts`): fünf neue
Reiter, eine SVG-Verlaufskomponente, eine Kreuztabellen-Komponente mit gedrehten
Spaltenköpfen. Neue Dependencies: keine.

**Datenbank**: eine additive Migration (Spalte für die getrennten Schiedsrichter-Namen).
Keine neue Tabelle — alle Statistiken sind Aggregate über vorhandene Zeilen.

**Nicht betroffen**: `games` (keine Ergebnisspalten, unverändert), Dienst-Regeneration,
iCal-Feed, RSVP, Anwesenheit. Der Abruf beim Verband ändert sich nicht.

## Test-Anforderungen

Pflicht nach `docs/agent/07-testing.md`: je Route Happy-Path und Fehlerfall, plus die
Invariante, die der Test garantiert.

| Route | Test | Status | Garantierte Invariante |
|---|---|---|---|
| `GET /api/staffeln/{id}/kreuztabelle` | `TestGetCrossTable_LiefertErgebnisUndDatum` | 200 | Gespielte Paarung trägt den Endstand, künftige das Datum, Diagonale ist leer |
| | `TestGetCrossTable_TorlosesSpielGiltAlsGespielt` | 200 | `0:0` erscheint als Ergebnis, nicht als Datum — die Krücke des Vorbilds ist nicht übernommen |
| | `TestGetCrossTable_OhneBerichtVollstaendig` | 200 | Vollständig ohne jeden ausgewerteten Spielbericht |
| | `TestGetCrossTable_UnbekannteStaffel` | 404 | Unbekannte Staffel-ID erfindet keine Tabelle |
| | `TestGetCrossTable_OhneToken` | 401 | Authenticated-Tier greift |
| `GET /api/staffeln/{id}/tabellenverlauf` | `TestGetProgression_RangJeSpieltag` | 200 | Ein Punkt je Spieldatum mit gespielter Begegnung, Rang aus kumulierten Ergebnissen |
| | `TestGetProgression_DifferenzBrichtGleichstand` | 200 | Sortierung Punkte → Tordifferenz → geworfene Tore |
| | `TestGetProgression_IsoTimestampWirdGetrimmt` | 200 | `"…T00:00:00Z"` gruppiert wie `"2026-09-21"` (DATE-Gotcha) |
| | `TestGetProgression_OhneErgebnisLeer` | 200 | Staffel ohne gespielte Begegnung liefert leeren Verlauf, keinen Fehler |
| | `TestGetProgression_OhneToken` | 401 | Authenticated-Tier greift |
| `GET /api/staffeln/{id}/teamstatistik` | `TestGetTeamStats_ToreOhneBericht` | 200 | Torverhältnis, Angriff, Verteidigung sind ohne PDF gefüllt |
| | `TestGetTeamStats_FairPlayLeerOhneBericht` | 200 | Mannschaft ohne Bericht hat **leere**, nicht nullwertige Fair-Play-Wertung |
| | `TestGetTeamStats_ParseFailedZaehltNicht` | 200 | `parse_failed` fließt in keine Wertung ein |
| | `TestGetTeamStats_FremdeMannschaftenEnthalten` | 200 | Mannschaften anderer Vereine sind enthalten |
| | `TestGetTeamStats_OhneToken` | 401 | Authenticated-Tier greift |
| `GET /api/staffeln/{id}/schiedsrichter` | `TestGetRefereeStats_SpieleUndStrafen` | 200 | Spiele je Schiedsrichter, Strafen **beider** Mannschaften summiert |
| | `TestGetRefereeStats_OhneNamenKeineZeile` | 200 | Bericht ohne Schiedsrichter erzeugt keine Zeile |
| | `TestGetRefereeStats_UnsichereTrennungGekennzeichnet` | 200 | Nach Namensregel getrennte Namen sind als unsicher markiert |
| | `TestGetRefereeStats_OhneToken` | 401 | Authenticated-Tier greift |
| `GET /api/staffeln/{id}/ranglisten` (erweitert) | `TestGetRanglisten_SpieleWerdenAusgewiesen` | 200 | `games` = Zahl der Berichte, in deren Mannschaftsliste der Spieler steht |
| | `TestGetRanglisten_SiebenmeterFehlversuche` | 200 | `missed = attempts − goals`, serverseitig gebildet |
| `GET /api/staffeln/{id}/affiliation` | `TestGetAffiliation_SpielerSiehtMannschaftUndSichSelbst` | 200 | Eigene Mannschaft über `game_id`→`game_teams` aufgelöst, eigene `playerIds` über `member_id` |
| | `TestGetAffiliation_ElternteilSiehtKind` | 200 | Eltern bekommen Mannschaft **und** Spielerzeilen des Kindes über `family_links` |
| | `TestGetAffiliation_OhneZugehoerigkeitLeer` | 200 | Nutzer ohne Kaderzugehörigkeit erhält leere Mengen, keinen Fehler |
| | `TestGetAffiliation_KeinNamensvergleich` | 200 | Ohne verknüpfte Begegnung bleibt die Mannschaftsmenge leer, auch bei ähnlichem Namen (design.md §10) |
| | `TestGetAffiliation_TrainerUndErweiterterKader` | 200 | `user_accessible_teams` deckt Trainer und erweiterten Kader mit ab |
| | `TestGetAffiliation_OhneToken` | 401 | Authenticated-Tier greift |

**Reine Einheitentests ohne Route:**

| Gegenstand | Test | Garantierte Invariante |
|---|---|---|
| Gini-Koeffizient | `TestGini_GleichverteilungMedianAlleinwerfer` | ≈0 bei Gleichverteilung, nahe 1 beim Alleinwerfer, genau 0 bei torloser Mannschaft |
| Gini-Bezugsgröße | `TestGini_UeberSaisonsummeNichtProSpiel` | Gerechnet wird über die Saisonsumme je Spieler, nicht je Spieler-Spiel (design.md §6) |
| Schiedsrichter-Trennung | `TestParseReferees_ZweiSpaltenZweiNamen` | Zwei Textläufe ⇒ zwei Namen, `uncertain=false` |
| | `TestParseReferees_EineSpalteIstUnsicher` | Ein Textlauf ⇒ Namensregel greift und setzt `uncertain=true` |
| | `TestParseReferees_PlatzhalterErzeugtNichts` | `N.N. N.N.` ergibt eine leere Liste |
| | `TestParseReport_UnbrauchbareSchiedsrichterzeileKipptNicht` | Bericht bleibt `parsed`, die Namen sind entbehrlich |

**Gates, die mitlaufen müssen:** Objektrechte-Matrix (`internal/permissions/
object_matrix_test.go` — vier neue `{id}`-Routen brauchen einen `openByDesign`-Eintrag,
sonst rot), Broadcast-Gate (Lese-Routen, kein Allowlist-Eintrag nötig),
`buttonStyles.gate.test.ts` und die brand-Token-Prüfung für die neuen Komponenten.
