## Why

Spielergebnisse, Tabellenstände und Spielberichte des BWHV leben heute vollständig außerhalb
von TeamWERK. Der Spielstand eines Spiels existiert im System nur dort, wo ihn der
Medien-Mensch für `match_reports` von Hand abtippt — `games` hat keine Ergebnisspalten.
Wer wissen will, wo seine Mannschaft steht oder wer im letzten Spiel getroffen hat, verlässt
die App.

Ein Vorläufer außerhalb von TeamWERK (`handballnet_crawler`, Python + Selenium + Chrome)
hat genau das geleistet, aber gegen handball.net und mit erheblichem Betriebsaufwand.
Dieser Change holt dieselbe Funktion nach TeamWERK — und zwar auf einem deutlich
einfacheren Weg, der live verifiziert wurde (siehe `design.md` §1):

- **Es gibt eine öffentliche JSON-API.** `spo.handball4all.de/service/if_g_json.php`
  liefert je Staffel den kompletten Saison-Spielplan **inklusive Tabelle** in einem einzigen
  Aufruf (~57 KB) — kein HTML-Scraping, kein Selenium.
- **Die Spielberichte brauchen keine Zugangsdaten.** `sboPublicReports.php?sGID=<id>`
  antwortet mit HTTP 200 und einem PDF. Die gesamte Zugangsdaten-Disziplin aus
  `h4a-game-import` (nie loggen, nie persistieren, DB-Vollscan-Test) entfällt hier —
  es gibt kein Geheimnis zu schützen.
- **Das PDF hat eine saubere Textebene mit Koordinaten.** Verifiziert mit
  `github.com/ledongthuc/pdf` (MIT, reines Go, kein CGo): die Spaltenköpfe tragen ihre
  eigenen X-Positionen, die Spaltengrenzen lassen sich also aus dem Dokument ableiten
  statt hart zu kodieren.
- **Die Fremdschlüssel liegen bereits.** `games.external_id` ist die BWHV-Spielnummer
  (`gNo`) aus Migration `042`, `venues.hall_number` die Hallennummer (`gGymnasiumNo`).
  Die Kette `gNo 905272 → sGID 3504061 → PDF` wurde end-to-end nachgeprüft.
- **`sGID` ist selbst das Bereitschaftssignal.** Von 90 Spielen einer Staffel tragen exakt
  die gespielten ein `sGID`; künftige liefern `"sGID": 0`. Das System muss den
  Zeitpunkt der Berichtsfreigabe also nicht raten.

## What Changes

- **Neue DB-Objekte** (Migration `068`): `kader.staffel` sowie die Tabellen
  `bwhv_staffeln`, `bwhv_games`, `bwhv_reports`, `bwhv_players`, `bwhv_player_games`,
  `bwhv_events`.
- **Staffel-Zuordnung am Kader**: `kader` ist bereits saison-gebunden, die Staffel gehört
  dorthin. Pflege über `/admin/kader` mit Auswahl aus dem Live-Katalog **und**
  Freitext-Alternative; validiert gegen `gender`/`age_class` des Kaders über das
  vorhandene `h4aimport.ParseStaffel`. Übungsgruppen (`kind='practice'`) bleiben
  strukturell außen vor.
- **Scheduler-Job „BWHV-Poll"**: Eine Staffel wird poll-fähig ab dem frühesten Anwurf
  ihres Spieltags **+ 2 h** und alle 15 min abgefragt, bis jedes Spiel des Tages ein
  `sGID` trägt; Nachzügler am Folgetag. Ein Aufruf liefert Spielplan, Ergebnisse und
  Tabelle der ganzen Staffel.
- **Spielbericht-Abruf und -Parsing**: Für jede Begegnung mit `sGID` und ohne Bericht wird
  das PDF geladen, die Textebene extrahiert und in zwei Quellen zerlegt (Mannschaftsliste,
  Spielverlauf). Gegen den Kopf-Endstand wird kreuzgeprüft: Abweichung der Verlaufs-Summe
  vom Endstand ist `parse_failed` ohne Teilergebnis, kleinere Abweichungen zwischen Liste
  und Verlauf werden gespeichert und als Warnung sichtbar gemacht.
- **Umfang: die gesamte Staffel**, nicht nur eigene Spiele — inklusive der Berichte
  fremder Begegnungen (~810/Saison). Fremde Spiele werden **keine `games`-Zeilen**; sie
  leben ausschließlich in `bwhv_games` und verbinden sich nur dort mit `games.external_id`,
  wo es ein eigenes Spiel gibt.
- **PDF-Ablage**: Berichte bleiben dauerhaft unter `BWHV_REPORT_DIR` (~116 MB/Saison) und
  sind als Beleg herunterladbar. Ein Parser-Fix lässt sich damit rückwirkend anwenden,
  ohne H4A erneut zu belasten.
- **Spieler-Identität**: Name primär mit Levenshtein-Toleranz, Trikotnummer als Bestätigung
  und Gleichstandsbrecher — ein geliehenes Trikot zerreißt eine Person nicht, eine
  Schreibvariante erzeugt keine zweite. Eigene Spieler werden zusätzlich auf `members`
  aufgelöst (Name + Jahrgang + Kaderzugehörigkeit). Widersprüche werden gemeldet, nicht
  geraten.
- **Neuer Nav-Punkt „Staffeln"** mit Mannschafts-Umschalter und den Reitern Tabelle,
  Spielplan und Ranglisten. Der Bericht zu einem einzelnen Spiel erscheint im Spieldetail.
- **Spieler-Profil** bekommt eine Saisonbilanz (Tore, 7m-Quote, Strafen, Verlauf).
- **`match_reports`-Vorbefüllung**: Der Redaktions-Editor füllt `home_goals`, `away_goals`,
  `home_goals_ht`, `away_goals_ht` aus dem Bericht vor, wenn einer vorliegt — **im
  Frontend**, weil ein Backend-Import `matchreports → gamestats` Domain→Domain wäre und
  `internal/arch/arch_test.go` bricht (design.md §8).
- **Keine Benachrichtigung**: kein Push, kein Event-Log-Eintrag. Der SSE-`Broadcast` bleibt
  Pflicht, damit offene Sessions nachladen.

## Capabilities

### Added Capabilities

- **`bwhv-spielberichte`** — Poll des BWHV-Spielplans je Staffel, `sGID`-gesteuerter Abruf
  der öffentlichen Spielbericht-PDFs, Extraktion der Textebene, Parsing von
  Mannschaftsliste und Spielverlauf mit gestufter Kreuzprobe, dauerhafte PDF-Ablage.
- **`bwhv-staffel-tabellen`** — Tabellenstand und kompletter Staffel-Spielplan aller
  zugeordneten Staffeln, aus demselben Poll ohne PDF-Abruf; Ansicht „Staffeln" mit
  Mannschafts-Umschalter.
- **`spieler-saisonstatistik`** — Identitätsauflösung eigener und fremder Spieler
  (Name/Levenshtein primär, Trikotnummer bestätigend), Saisonbilanz im Spieler-Profil,
  Mannschafts- und Staffel-Ranglisten.
- **`kader-staffel-zuordnung`** — Staffelcode je Kader und Saison, Auswahl aus dem
  Live-Katalog mit Freitext-Alternative, Validierung gegen Geschlecht und Altersklasse.

### Modified Capabilities

- **`match-reports`** — Die Ergebnisfelder des redaktionellen Spielberichts werden aus dem
  offiziellen BWHV-Bericht vorbefüllt, sofern einer vorliegt; die Werte bleiben editierbar.

## Test-Anforderungen

| Route | Test | Erwartung / Invariante |
|---|---|---|
| `GET /api/staffeln` | `TestListStaffeln_HappyPath` | 200, je Kader der aktiven Saison mit gesetzter `staffel` eine Zeile |
| `GET /api/staffeln` | `TestListStaffeln_Unauthenticated` | 401 |
| `GET /api/staffeln/{id}/tabelle` | `TestStaffelTabelle_HappyPath` | 200, Tabellenzeilen aus dem letzten Snapshot |
| `GET /api/staffeln/{id}/tabelle` | `TestStaffelTabelle_UnbekannteStaffel` | 404 |
| `GET /api/staffeln/{id}/spielplan` | `TestStaffelSpielplan_EnthaeltFremdeBegegnungen` | 200, auch Spiele ohne `game_id` |
| `GET /api/staffeln/{id}/ranglisten` | `TestStaffelRanglisten_HappyPath` | 200, absteigend nach Toren |
| `GET /api/bwhv-games/{id}/report` | `TestReport_HappyPath` | 200, Mannschaftslisten + Spielverlauf |
| `GET /api/bwhv-games/{id}/report` | `TestReport_NochKeinBericht` | 404, solange `sGID` leer ist |
| `GET /api/bwhv-reports/{id}/pdf` | `TestReportPDF_HappyPath` | 200, `application/pdf`, `Content-Disposition: attachment` |
| `GET /api/bwhv-reports/{id}/pdf` | `TestReportPDF_VerworfenerBericht` | 404 bei `parse_failed` ohne Datei |
| `GET /api/members/{id}/saisonstatistik` | `TestSaisonstatistik_HappyPath` | 200, Tore/7m/Strafen der Saison |
| `GET /api/members/{id}/saisonstatistik` | `TestSaisonstatistik_UnbekanntesMitglied` | 404 |
| `GET /api/bwhv/staffel-katalog` | `TestStaffelKatalog_NurVorstand` | 403 für `standard` ohne `vorstand` |
| `POST /api/staffeln/{id}/poll` | `TestManuellerPoll_BroadcastetUndNurVorstand` | 200 + `Broadcast`, 403 ohne `vorstand` |
| `PUT /api/kader/{id}` | `TestKaderStaffel_UnpassenderCodeWirdAbgelehnt` | 400, `mB-RL-BW` an einem wC-Kader |
| `PUT /api/kader/{id}` | `TestKaderStaffel_UebungsgruppeAbgelehnt` | 409 bei `kind='practice'` |

**Garantierte Invarianten** (je mit eigenem Test im Domänen-Package):

- `TestPoll_ErzeugtKeineGamesZeilen` — ein Poll über eine Staffel mit 90 Begegnungen legt
  **keine** neue Zeile in `games` an; nur bestehende `external_id` werden verknüpft.
- `TestParse_EndstandAbweichungVerwirftBericht` — weicht die Verlaufs-Summe vom
  Kopf-Endstand ab, entsteht `state='parse_failed'` und **keine** Zeile in
  `bwhv_player_games`/`bwhv_events`.
- `TestParse_DetailabweichungWirdGespeichertUndGewarnt` — weicht nur die Mannschaftsliste
  vom Verlauf ab, wird gespeichert und `warnings_json` ist nicht leer.
- `TestMatching_SchreibvarianteIstDieselbePerson` — „Haßlöcher"/„Hasslocher" im selben
  Team ergeben **eine** `bwhv_players`-Zeile.
- `TestMatching_TrikotwechselZerreisstPersonNicht` — gleicher Name, andere Nummer ergibt
  **eine** Zeile; der Konflikt wird in `conflict` vermerkt.
- `TestMatching_ZweiGleicheNamenWerdenPerNummerGetrennt` — zwei Spieler gleichen Namens
  mit verschiedenen Nummern ergeben **zwei** Zeilen.
- `TestPollFenster_VorAnwurfPlusZweiStundenKeinAbruf` — eine Staffel, deren frühester
  Anwurf weniger als 2 h zurückliegt, wird nicht abgefragt.
- `TestPollFenster_AlleSGIDVorhandenBeendetDenTag` — sind alle Spiele des Tages versorgt,
  erfolgt kein weiterer Abruf.

## Impact

- **Migration** `internal/db/migrations/068_bwhv_spielberichte.up.sql`/`.down.sql`.
- **Neues Foundation-Package** `internal/bwhv/` — HTTP-Client (`cmd=po`, `cmd=ps`,
  PDF-Download), PDF-Textextraktion, Parser für Mannschaftsliste und Spielverlauf,
  Kreuzprobe. Kein DB-Zugriff. Tests gegen eingecheckte Fixtures in `testdata/`,
  **kein Live-Abruf im Test** (wie `h4aimport`).
- **Neues Domain-Package** `internal/gamestats/` — Persistenz, Identitätsauflösung,
  Ranglisten, Routen, `Broadcast`.
- `internal/scheduler/bwhv_poll.go` — Poll-Job, Fensterlogik, Abruf über
  `background.Go` (Goroutine-Gate).
- `internal/kader/handler.go` — `staffel` im `PUT`, Validierung, 409 für Übungsgruppen.
- `internal/app/router.go` — Lese-Routen im Authenticated-Tier, Katalog und manueller
  Poll im Vorstand-Tier.
- `internal/config` + `.env.example` + `deploy/setup-vps.sh` + `deploy/backup-cron.sh` +
  Backup-Targets + die idempotente Schleife im `deploy`-Target — neuer Storage-Pfad
  `BWHV_REPORT_DIR` (Checkliste aus `docs/agent/10-deployment.md`).
- **Neue direkte Abhängigkeiten**: `github.com/ledongthuc/pdf` (MIT, reines Go) und
  `github.com/agnivade/levenshtein` — letzteres liegt bereits als `// indirect` im Tree
  und wird nur direkt gemacht. Kein CGo, kein Zuwachs des Modulbaums über die eine
  PDF-Bibliothek hinaus.
- `web/src/pages/StaffelnPage.tsx` (neu), `web/src/pages/AdminKaderPage.tsx`
  (Staffel-Feld), Spieldetail (Reiter „Spielbericht"), Spieler-Profil (Saisonbilanz),
  `AppShell.tsx` (Nav-Eintrag), `MatchReportEditor` (Vorbefüllung).
- `internal/permissions/object_matrix_test.go` — die neuen `{id}`-Routen kommen mit
  Begründung nach `openByDesign` (vereinsweit sichtbare, öffentliche Verbandsdaten).
- `internal/arch/arch_test.go` — `bwhv` als Foundation, `gamestats` als Domain
  klassifizieren.
- **Doku**: Gotcha-Absatz in `docs/agent/06-gotchas.md` (Poll-Fenster, `og`/`o`-Falle,
  Kreuzprobe, Identitätsregel) und Betriebsvorbehalt in `docs/agent/10-deployment.md`.
