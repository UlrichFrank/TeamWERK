# Design

## 1. Eigener Toggle statt Erweiterung von `include_training`

`include_training` einfach auf Übungsgruppen mitwirken zu lassen wäre die kleinere Änderung
(kein Schema, kein UI). Dagegen spricht die Asymmetrie der beiden Terminarten: das
Mannschaftstraining ist der Pflichttermin der eigenen Mannschaft, die Übungsgruppe ein
Zusatzangebot, an dem oft nur ein Kind einer Familie teilnimmt und dessen Termine in einem
ganz anderen Rhythmus liegen (14-tägig statt zweimal pro Woche). Ein Abonnent, der den
Mannschaftskalender will, aber das samstägliche Fördertraining nicht, hätte ohne eigenen
Schalter keinen Weg — und umgekehrt.

Der Preis ist eine sechste Spalte und ein sechster Schalter. Beide sind mechanisch
gleichartig zu den fünf vorhandenen; es entsteht keine neue Form von Zustand.

**Default `1`.** Alle fünf Bestandsspalten tragen denselben Default, und die Änderung
existiert, weil die Termine fehlen — ein Default `0` lieferte nach dem Deploy denselben
leeren Kalender wie vorher und verlagerte die Behebung auf eine Aktion, von der der
Betroffene nichts weiß. Konsequenz: bestehende Abos wachsen beim ersten Refresh um die
Übungsgruppen-Termine. Das ist sichtbar und rückgängig zu machen (Schalter aus) — die
umgekehrte Reihenfolge wäre es nicht.

## 2. `ts.kader_id` ist der Anker, nicht `ts.team_id`

`fetchTrainings` löst heute über `JOIN teams t ON t.id = ts.team_id` + `JOIN kader k ON
k.team_id = ts.team_id AND k.season_id = ts.season_id` auf. Die Doppelbedingung rekonstruiert
den Kader aus Team und Saison — dabei steht er in `ts.kader_id` direkt. Für Mannschaftstermine
sind beide Wege dasselbe Ergebnis (`kader.team_id`/`kader.season_id` sind die Quelle, aus der
`training_sessions.team_id`/`season_id` projiziert werden); für Übungsgruppen ist der zweite
der einzig mögliche, weil `team_id` dort NULL ist.

Damit folgt der Feed derselben Auflösung wie `trainings.ListSessions`, wo dieselbe Umstellung
schon stattgefunden hat (Kommentar dort: „Sichtbarkeit hängt am Kader des Termins
(ts.kader_id), nicht am Team"). Zwei Fundstellen, die dieselbe Frage verschieden beantworten,
sind die Fehlerklasse, die diese Änderung gerade behebt — sie soll nicht in kleinerer Form
stehenbleiben.

`teams` wird zum `LEFT JOIN`, weil die Beschriftung weiterhin den Mannschaftsnamen bevorzugt.

## 3. Beschriftung: `kader.name` als Rückfall

Der Feed-Titel eines Trainings lautet `Training: <Label>`, wobei `<Label>` bisher `teams.name`
war. Bei einer Übungsgruppe gibt es keinen Teamnamen; der Gruppenname steht in `kader.name`
(„Förderkinder 2016"). `COALESCE(t.name, k.name, '')` liefert damit für beide Fälle die
Bezeichnung, unter der die Gruppe auch in `/termine` und im Filter erscheint.

Bewusst **nicht** `training_sessions.title`: der Titel ist pro Termin frei und driftet (in der
Prod-DB heißt derselbe Serientermin mal „Fördertraining 2016", mal „Förderkinder 2016"). Ein
Kalender-Abo soll einen stabilen, wiedererkennbaren Kalendereintrag liefern; die Notiz des
Termins steht ohnehin in `DESCRIPTION`.

Der `erw. Kader`-Zusatz aus `kaderLabel` bleibt unverändert angeschlossen. Er kann bei
Übungsgruppen praktisch nicht auftreten (`PUT /api/kader/{id}` lehnt
`extended_members_add` für `kind='practice'` mit 409 ab), aber `kaderMembership` kennt den
Zweig — eine Sonderbehandlung hier hieße, dieselbe Regel an zwei Stellen zu pflegen.

## 4. Das Ladefenster von `/termine`

Die obere Fenstergrenze war ursprünglich ein rollierendes 180-Tage-Fenster und wurde auf
`seasons.end_date` umgestellt, weil lange Saisons hinten abgeschnitten wurden. Die Umstellung
hat die Annahme eingeführt, dass kein Termin der Saison nach dem Saisonende liegt. Diese
Annahme ist falsch: `training_sessions` trägt die Saison als Spalte, nicht als
Datumsbedingung — eine Serie darf über das formale Saisonende hinauslaufen, und genau das
tun die Übungsgruppen-Serien.

Die Grenze wird deshalb zum **Maximum aus Saisonende und heute + 365 Tagen**. Beide Gründe
bleiben damit erfüllt: eine lange Saison wird nicht abgeschnitten (Saisonende gewinnt), und
ein Termin jenseits des Saisonendes fällt nicht heraus (das rollierende Fenster gewinnt).

**Warum nicht nach `season_id` filtern statt nach Datum?** Das wäre die genauere Frage
(„alle Termine dieser Saison"), verlangt aber einen neuen Query-Parameter auf
`GET /api/training-sessions` **und** auf `GET /api/games/my`, die beide heute dasselbe
`from`/`to`-Paar bekommen. Der Nutzen wäre auf den Randfall begrenzt, der Eingriff nicht.
Die Datumsgrenze bleibt, sie wird nur großzügig statt scharf.

**Warum nicht die Terminanlage auf das Saisonende begrenzen?** Weil der Termin am 03.07.2027
ein echter Termin ist, an dem echte Kinder trainieren. Eine Validierung, die ihn verbietet,
löst das Anzeigeproblem, indem sie die Fachlichkeit beschneidet.

## 5. Das Zeilenlimit von `GET /api/training-sessions`

Beim Nachmessen am Prod-Datenstand fiel eine zweite Ursache derselben Beschwerde auf, die
mit den Übungsgruppen nichts zu tun hat: `httpx.Paging(r, 100, 200)` deckelt die Antwort bei
200 Zeilen, der Client fragt `limit=500` an und bekommt stillschweigend 200. Im aktuellen
Bestand sieht ein Trainer mit mehreren Kadern 285 Termine im Saisonfenster, admin/vorstand
(Bypass `1=1`) 386 — beide sind heute abgeschnitten, ohne Hinweis in der Oberfläche.

Der Deckel steigt deshalb auf `httpx.Paging(r, 100, 1000)`. Er bleibt ein Deckel — die
Fundstelle, aus der die Regel stammt (`GET /api/games?limit=100000`), bleibt geschlossen —,
liegt aber jetzt jenseits der fachlichen Obergrenze: mehr als 1000 Trainingstermine in einem
Jahresfenster hätte der Verein nur, wenn sich die Zahl der Mannschaften verdreifacht.

Echtes Nachladen (Paging in der Oberfläche) bleibt die saubere Antwort und ist als
`incremental-list-sync` bereits vorgeschlagen. Sie hier vorwegzunehmen hieße, einen sichtbaren
Datenfehler hinter einer Umbauarbeit zu parken.
