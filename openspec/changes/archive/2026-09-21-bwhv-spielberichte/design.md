# Design — BWHV-Spielberichte, Tabellen und Spielerstatistik

Alle in diesem Dokument genannten Aufrufe, Feldnamen und Zahlen wurden am 20.09.2026 live
gegen die Produktivsysteme des BWHV bzw. Handball4All verifiziert. Wo eine Angabe eine
Annahme ist und keine Messung, steht es dabei.

---

## 1. Datenquelle: öffentliche JSON-API statt Scraping

Die BWHV-Ergebnisseite (`bwhv.org/spielbetrieb/ergebnisse/tabellen#/schedule`) ist ein
AngularJS-Widget von `h4a.cloud.tricept.de`, das seine Daten über eine JSON-Schnittstelle
zieht. Diese Schnittstelle wird hier direkt genutzt — HTML-Parsing entfällt vollständig.

**Basis:** `https://spo.handball4all.de/service/if_g_json.php`

### 1.1 Staffel-Katalog

```
GET if_g_json.php?cmd=po&og=216&p=142           → 56 Klassen (Verbandsebene BWHV)
GET if_g_json.php?cmd=po&og=216&o=251&p=142     → 92 Klassen (Bezirk Stuttgart-Rems-Murr)
```

Antwort: `content.classes[]` mit `gClassID`, `gClassSname`, `gClassLname`.
**`gClassSname` ist exakt der Staffelcode**, den der Verein ohnehin verwendet
(`mB-RL-BW`, `gD-BOL-SRM`).

### 1.2 Die `og`/`o`-Falle

Das ist der einzige Punkt, an dem die API nicht tut, was man erwartet, und er kostet ohne
diese Notiz Stunden:

```
cmd=po&og=251          →  HTTP 401  {"status":-1,"statusText":"permission denied"}
cmd=po&og=241          →  HTTP 401
cmd=po&o=251&og=216    →  HTTP 200  ✓
```

`og` ist die Organisation der **Website**, nicht die gewünschte Auswahl. Bezirke werden über
`o` selektiert, während `og` auf `216` stehen bleibt. Ein direkter Zugriff auf eine
Bezirks-`og` wird serverseitig abgelehnt — unabhängig von `Referer`, `Origin` und
`User-Agent` (alle drei geprüft; der Fehler tritt auch aus dem Seitenkontext heraus auf und
erscheint dort wegen fehlender CORS-Header auf der 401-Antwort als „Failed to fetch").

Verifizierte Zuordnung für die neun Mannschaften des Vereins:

| Kader | Staffel | `og` | `o` | `gClassID` |
|---|---|---|---|---|
| mA1 | `mA-RL-BW` | 216 | — | 161276 |
| mA2 | `mA-BOL-SRM` | 216 | 251 | 167686 |
| mB1 | `mB-RL-BW` | 216 | — | 161291 |
| mB2 | `mB-OL-3-BW` | 216 | — | 161306 |
| mC1 | `mC-OL-3-BW` | 216 | — | 161331 |
| mC2 | `mC-BOL-SRM` | 216 | 251 | 167711 |
| gD | `gD-BOL-SRM` | 216 | 251 | 167736 |
| wB | `wB-BOL-1-SRM` | 216 | 251 | 167811 |
| wC | `wC-OL-2-BW` | 216 | — | 161391 |

**`gClassID` wird nicht gespeichert, sondern bei jedem Lauf aus dem Katalog aufgelöst.**
Sie ist saisonabhängig; der Staffelcode ist es nicht. Gespeichert wird also der Code, und
die ID ist ein Lookup — so übersteht die Zuordnung einen Saisonwechsel ohne Nachpflege.
Findet der Katalog den Code nicht, ist das eine sichtbare Meldung und kein stiller Leerlauf.

### 1.3 Staffel-Spielplan und Tabelle

```
GET if_g_json.php?cmd=ps&og=216[&o=<bezirk>]&p=142&cl=<gClassID>&ca=1
```

Ein Aufruf, ~57 KB, liefert **die komplette Saison**: `content.futureGames.games[]` (90
Begegnungen, gespielte wie künftige) und `content.score[]` (Tabelle, 10 Zeilen).
`ca=1` schaltet „alle Spiele" statt nur der kommenden frei.

Ein Spiel-Objekt, wörtlich aus der Antwort:

```json
{"gID":"9566466","sGID":"3504061","gNo":"905272","live":false,
 "gDate":"20.09.26","gWDay":"So","gTime":"16:00",
 "gGymnasiumNo":"21005","gGymnasiumName":"Erich-Bamberger Stadthalle",
 "gGymnasiumPostal":"76684","gGymnasiumTown":"Östringen",
 "gHomeTeam":"R-N Löwen 2","gGuestTeam":"Team Stuttgart",
 "gHomeGoals":"29","gGuestGoals":"25","gHomeGoals_1":"14","gGuestGoals_1":"13",
 "gReferee":"Einsmann,Zappe","robotextstate":"generated"}
```

Damit ist der Tabellenstand **ohne einen einzigen PDF-Abruf** aktuell — er fällt bei jedem
Poll nebenbei ab. Das ist der billigste Teil des ganzen Features.

### 1.4 `sGID` als Bereitschaftssignal

Die Report-URL baut das Widget als `head.repURL + sGID`, mit
`repURL = "https://spo.handball4all.de/misc/sboPublicReports.php?sGID="`.

Gemessen an der Staffel `mB-RL-BW`: von 90 Begegnungen tragen **3** ein `sGID` — genau die
gespielten. Die übrigen 87 liefern `"sGID": 0`.

Daraus folgt die zentrale Entwurfsentscheidung: **das System muss den Zeitpunkt der
Berichtsfreigabe nicht schätzen.** Ein zeitgesteuerter Abruf („Spielende + 1 h") müsste
Spieldauer je Altersklasse, Nachspielzeit-Puffer und eine Aufgabe-Heuristik nachbauen — und
läge bei jedem verlegten, abgebrochenen oder verspätet freigegebenen Spiel daneben. `sGID`
beantwortet dieselbe Frage autoritativ.

### 1.5 Der Spielbericht

```
GET https://spo.handball4all.de/misc/sboPublicReports.php?sGID=3504061
→ HTTP 200, application/pdf, 143.830 Bytes, PDF 1.4, 4 Seiten, keine Authentifizierung
```

**Kein Login.** Das ist der wesentliche Unterschied zu `h4a-game-import`, der als einzige
Stelle im System fremde Zugangsdaten entgegennimmt. Hier gibt es kein Geheimnis: keine
Credential-Regeln, kein `TestCredentialsWerdenNichtPersistiert`, kein Modal, das beim
Unmount Eingaben mitnehmen muss.

---

## 2. Abwägung: Daten fremder Spieler

**Entscheidung: fremde und eigene Spieler werden gleich behandelt** — erfasst, gespeichert,
über die Saison aggregiert und in Ranglisten geführt. Rund 800 Jugendliche anderer Vereine
sind damit mit Name, Jahrgang, Trikotnummer sowie Leistungs- und Disziplinardaten in der
Datenbank.

**Begründung (Projektleitung, 20.09.2026):** Die Quelle ist ein öffentlich und ohne
Zugangsbeschränkung abrufbares Dokument des Verbands. Es besteht kein Grund, die Daten
nicht zu erheben.

**Festgehaltener Vorbehalt.** Dieser Absatz steht hier aus demselben Grund wie der
H4A-ToS-Vorbehalt in `docs/agent/10-deployment.md` §„H4A-Spielimport": damit die Abwägung
eine bewusste bleibt und nicht als stillschweigende Annahme im Code verschwindet.
Anzumerken ist, dass „öffentlich abrufbar" und „darf zu eigenem Zweck gespeichert und
aggregiert werden" rechtlich nicht dasselbe sind, und dass das Projekt an anderer Stelle
eine engere Linie fährt (Zero-Knowledge bei Bankdaten; eigener Store für Trainingsnachweise
mit der ausdrücklichen Begründung „oft Minderjährige"). Die Entscheidung ist getroffen; der
Vorbehalt ist kein Einwand, sondern die Fundstelle, falls die Frage später noch einmal
gestellt wird. Kein Rechtsrat.

**Konsequenz für den Zuschnitt:** Es gibt keine Anonymisierungsschicht wie in
`dutyfairness`. `bwhv_players` trägt eigene und fremde Spieler in derselben Tabelle;
`member_id` ist gesetzt oder NULL, mehr unterscheidet sie nicht.

---

## 3. Betriebsvorbehalt (ohne Zugangsdaten, aber nicht folgenlos)

Der Abruf braucht kein Passwort — er pollt aber automatisiert einen fremden Dienst und lädt
pro Saison rund 116 MB PDFs. Dafür gelten dieselben Maßstäbe wie beim H4A-Import:

- **Eigener `User-Agent`** mit Projektname und Kontaktadresse, damit der Betreiber weiß,
  wer da klopft.
- **Kein Dauer-Polling.** Außerhalb der Fenster aus §4 findet kein Abruf statt; ein Tag
  ohne Spiel erzeugt keinen einzigen Request.
- **Serielle PDF-Abrufe mit Pause**, keine Parallelisierung über die Staffeln hinweg.
- **Ausgehend nur HTTPS**, Timeout je Request.
- Ein **manueller Anstoß** (`POST /api/staffeln/{id}/poll`) existiert für den Notfall und
  ist auf Vorstand/Admin beschränkt.

---

## 4. Poll-Fenster

Der Poll arbeitet **pro Staffel**, nicht pro Spiel — ein Aufruf liefert ohnehin alle
Begegnungen. Das Fenster wird aus dem gespeicherten Staffel-Spielplan abgeleitet, nicht aus
`games`: bei diesem Umfang zählen auch fremde Begegnungen, und die kennt `games` nicht.

```
Eine Staffel ist poll-fähig, wenn ALLE gelten:
  · sie hat heute mindestens eine Begegnung
  · jetzt ≥ frühester heutiger Anwurf + 2 h
  · mindestens eine heutige Begegnung hat noch kein sGID

Kadenz:   alle 15 min  (Scheduler läuft minütlich, Fenster wird je Lauf neu bewertet)
Ende:     sobald jede heutige Begegnung ein sGID trägt
Nachzug:  am Folgetag 08:00 ein letzter Lauf für offene Begegnungen der Vortage
```

**Kein Zustandsfeld für die Poll-Steuerung.** Der Zustand ist aus `bwhv_games`
(`date`, `time`, `sgid`) vollständig ableitbar; eine zusätzliche Spalte könnte von der
Wirklichkeit abweichen und wäre eine zweite Wahrheit.

**Aufwand, gerechnet:** 9 Staffeln × 57 KB ≈ 513 KB je Durchlauf. An einem Spieltag mit
gestaffelten Fenstern ergibt das grob 16 MB — unkritisch auf dem VPS.

**Henne-Ei:** Der erste Lauf einer Saison kennt noch keinen Spielplan und damit kein
Fenster. Deshalb läuft täglich um 06:00 ein **Katalog- und Spielplan-Lauf** für alle
zugeordneten Staffeln, unabhängig von Spieltagen. Er ist auch der Weg, auf dem Verlegungen
in den Spielplan kommen.

---

## 5. PDF-Verarbeitung

### 5.1 Textebene statt Rendering

Verifiziert mit `github.com/ledongthuc/pdf` (MIT, reines Go, kein CGo — passend zu
`modernc.org/sqlite`). Ausgabe für Seite 2 des Beispielberichts, `|x=NNN|` markiert eine
neue Textgruppe mit ihrer X-Position:

```
y= 692.5 Tore|x=322|7m/|x=384|Hinausstellungen|x=528|zus.
y= 687.0 Nr.|x=79|Name|x=207|Jahrgang|x=259|M|x=274|R|x=350|Verw.|x=466|Disq.|x=496|Ber.
y= 681.5 (ges)|x=321|Tore|x=387|1.|x=416|2.|x=444|3.|x=524|Strafe
y= 670.0  9|x=79|Gabriel Kliems  |x=296|2 |x=380|30:12|x=408|49:06
y= 635.0 20|x=79|Benedict Hofmann|x=296|3 |x=324|1/0
y= 564.5 42|x=79|Alexander Meier |x=294|10|x=350|22:50
```

**Die Spaltengrenzen werden aus der Kopfzeile abgeleitet, nicht hart kodiert.** Die
Kopfzeile trägt ihre eigenen X-Positionen; Werte werden dem Kopf zugeordnet, dessen
Bereich sie treffen. Ein fest verdrahtetes `x==296 → Tore` würde beim ersten
Layout-Update des Verbands still falsche Zahlen liefern — die Fehlerklasse, die
`dienst-zeitmodus-strikt` bewusst ausgeschlossen hat.

Beobachteter Versatz: Werte stehen links ihrer Kopfmitte (Kopf „1." bei x=387, Wert
`30:12` bei x=380). Die Zuordnung arbeitet deshalb über Bereiche zwischen benachbarten
Kopfpositionen, nicht über Gleichheit.

### 5.2 Zwei Quellen im selben Dokument

| | Mannschaftsliste (S. 2) | Spielverlauf (S. 3–4) |
|---|---|---|
| Parsing | Spaltenbinning über Kopfzeile | Regex auf Fließtext |
| Sprödigkeit | mittel (Layout) | gering |
| Liefert | vollständiger Kader, Jahrgang, Summen | Tore, 7m, Strafen, Karten, Auszeiten, exakte Zeitpunkte |
| Fehlt | Zeitpunkte | Spieler ohne Aktion, Jahrgang |

Beide werden gelesen. Die Mannschaftsliste liefert, wer überhaupt dabei war; der Verlauf
liefert, was passierte — und beide zusammen prüfen sich.

Zeilenformen im Verlauf (wörtlich):

```
Tor durch Samuel Birkle (87, Team Stuttgart)
7m-Tor durch Samuel Birkle (87, Team Stuttgart)
7m, KEIN Tor durch Benedict Hofmann (20, R-N Löwen 2)
Verwarnung für Samuel Birkle (87, Team Stuttgart)
2-min Strafe für Gabriel Kliems (9, R-N Löwen 2)
Auszeit Rhein-Neckar Löwen 2
```

Eine unbekannte Zeilenform wird als `kind='other'` mit `raw_text` gespeichert, nicht
verworfen — so geht keine Information verloren und ein neuer Ereignistyp fällt beim
Nachsehen auf.

### 5.3 Gestufte Kreuzprobe

```
  Kopf-Endstand  29:25 (14:13)
         ║
         ╠═══ Summe der Tor-Ereignisse im Verlauf
         ║        ungleich  →  state='parse_failed', failure_reason,
         ║                     KEINE Zeile in bwhv_player_games/bwhv_events
         ║                     PDF bleibt liegen → Reparse nach Fix
         ║        gleich    →  weiter
         ║
         ╚═══ Summenspalten der Mannschaftsliste
                  ungleich  →  speichern + Eintrag in warnings_json,
                               in der UI als Hinweis sichtbar
```

**Warum gestuft und nicht durchgehend streng:** Ein Widerspruch zwischen Kopf und Verlauf
heißt, dass der Parser ein Ereignis verloren hat — das Ergebnis wäre in jedem Fall falsch,
also darf nichts davon in die Datenbank. Ein Widerspruch zwischen Liste und Verlauf kann
dagegen im Quelldokument selbst liegen (der Zeitnehmer trägt eine Strafe in den Verlauf,
aber nicht in die Summenspalte). Diesen Fall als Fehlschlag zu werten hieße, ein fremdes
Dokumentationsproblem als eigenen Parser-Fehler auszugeben und den ganzen Bericht zu
verlieren. Die Warnung ist die ehrlichere Antwort.

**Bewusst keine dritte Liste** analog `InvalidSpan` vs. `Skipped`: hier gibt es nur zwei
Zustände, verwertbar und nicht verwertbar, plus eine Anmerkung am verwertbaren.

---

## 6. Identitätsauflösung

### 6.1 Die Regel

**Name primär mit Levenshtein-Toleranz, Trikotnummer bestätigend und
gleichstandsbrechend.** Identitätsraum ist `(staffel, team_name)` innerhalb einer Saison. Dass die Mannschaft
dazugehört, ist keine Vorsichtsmaßnahme: im verifizierten Beispielbericht tragen **beide**
Mannschaften eine 16 und eine 46. Trikotnummern kollidieren also schon innerhalb eines
einzelnen Spiels, nicht erst über die Staffel hinweg.

```
  "Haßlöcher" vs "Hasslocher"     Levenshtein 2   →  dieselbe Person
  Meier heute 42, nächstes Spiel 17 (Trikot vergessen)
                                  Name identisch  →  dieselbe Person
                                                     conflict vermerkt
  zwei "Bieler" im selben Team    Name identisch  →  Nummer entscheidet
                                                     zwei Zeilen
```

**Präzisierung aus der Umsetzung.** „Name primär" heißt nicht „Name allein". Die Regel
zerfällt in drei Fälle, und der dritte fiel erst beim Schreiben der Tests auf:

```
  exakter Name, andere Nummer     →  dieselbe Person   (Trikot vergessen)
  ähnlicher Name, gleiche Nummer  →  dieselbe Person   (Schreibvariante, bestätigt)
  ähnlicher Name, andere Nummer   →  ZWEI Personen     (nichts bestätigt die Identität)
```

Der dritte Fall wäre unter naivem „Name primär" eine Verschmelzung — und damit ein
geratener Treffer ohne ein einziges bestätigendes Merkmal. Er entsteht stattdessen als
neuer Spieler mit vermerktem Widerspruch, im Geist von `invalid_span`. Der vom
Projektinhaber benannte Fall (vergessenes Trikot) bleibt davon unberührt, weil er den
Namen *exakt* trägt. Unterschieden wird dabei „Nummer widerspricht" von „Nummer noch
unbekannt" — sonst erzeugte der erste Bericht jeder Person einen Widerspruch.

**Warum nicht die Nummer als Schlüssel**, obwohl sie tippfehlerfrei aus einer Zahlenspalte
kommt: ein vergessenes Trikot führt zu einer geliehenen Nummer, und die Nummer wäre dann
der zuverlässig falsche Schlüssel. Namen tragen Schreibvarianten, aber sie bezeichnen über
die Saison dieselbe Person. Schwelle: Levenshtein ≤ 2 bei Namen ab 5 Zeichen, exakt
darunter — der Wert gehört in eine benannte Konstante, nicht in den Aufruf.

`N.N. N.N.` (Platzhalter für nicht gemeldete Betreuer) ist von der Auflösung
ausgenommen und erzeugt keine `bwhv_players`-Zeile.

### 6.2 Eigene Spieler zusätzlich auf `members`

Nach der staffelinternen Auflösung werden Zeilen der eigenen Mannschaften gegen `members`
gematcht: Name (Levenshtein), `date_of_birth` gegen den Jahrgang aus der Mannschaftsliste,
`jersey_number` und Kaderzugehörigkeit der Saison. Die Trikotnummern des eigenen Vereins
sind laut Projektleitung gut gepflegt — sie sind hier also die stärkste Bestätigung, aber
aus dem Grund in §6.1 weiterhin nicht der Schlüssel.

Bleibt eine Zeile unaufgelöst, wird sie als solche angezeigt („Fabian Diekmann (86) —
nicht zugeordnet") und ist manuell zuordenbar; einmal zugeordnet, gilt sie für die Saison.
**Nicht geraten** — dieselbe Haltung wie `markPossibleDuplicate` im H4A-Import.

---

## 7. Datenmodell und die Grenze zu `games`

**Fremde Begegnungen werden niemals `games`-Zeilen.** 810 Spiele pro Saison in `games`
würden in den Spielplan, die Dienst-Regeneration, den iCal-Feed, die RSVP-Logik und die
Anwesenheit lecken — jede dieser Flächen müsste einen neuen Sonderfall lernen. Stattdessen
lebt der komplette Staffel-Spielplan in `bwhv_games`, und nur dort, wo eine `external_id`
auf ein bestehendes Spiel zeigt, wird `bwhv_games.game_id` gesetzt.

Das ist dieselbe Figur wie `training_sessions.team_id IS NULL` bei den Übungsgruppen: die
**Abwesenheit** der Verknüpfung ist das Gate, und das Modell fällt ohne Zutun geschlossen
aus. Ein „Reparatur"-Backfill, der `game_id` für fremde Spiele füllen wollte, risse
sämtliche Ausschlüsse auf einmal ein.

Aus demselben Grund bekommt `games` **keine Ergebnisspalten**: der Spielstand steht in
`bwhv_games` und gilt dort für eigene und fremde Begegnungen gleichermaßen.

```
 kader.staffel ──▶ bwhv_staffeln ──┬──▶ bwhv_games ──┬──▶ bwhv_reports
   (Saison)         (Snapshot,     │    (alle 90)    │     (nur mit sGID)
                     Tabelle)      │        │        │           │
                                   │        └── game_id ──▶ games (nur eigene)
                                   │                             │
                                   └──▶ bwhv_players             ├─▶ bwhv_player_games
                                          │                      └─▶ bwhv_events
                                          └── member_id ──▶ members (nur eigene)
```

---

## 8. Paketschnitt — und warum die `match_reports`-Vorbefüllung im Frontend sitzt

```
internal/bwhv/         FOUNDATION   HTTP-Client, PDF-Textebene, Parser, Kreuzprobe
                                    Kein DB-Zugriff. Fixtures in testdata/.
internal/gamestats/    DOMAIN       Persistenz, Identität, Ranglisten, Routen, Broadcast
```

Exakt die Teilung, die `h4aimport` (Foundation) und `games/h4aimport_handler.go` (Domain)
bereits vorleben. `internal/arch/arch_test.go` erzwingt sie.

**Zweite Folge, erst bei der Umsetzung aufgefallen:** `internal/scheduler` ist in
`arch_test.go` als **Foundation** klassifiziert und darf damit selbst keine Domäne
importieren. Der Poll-Job kann also nicht dort liegen. Statt `gamestats` zur Foundation
umzuwidmen — was die Begründung des nächsten Absatzes aushebeln würde, weil `matchreports`
es dann importieren dürfte — bekommt der Scheduler ein `AddJob(func())`, und die
Komposition (`main.go`) hängt `gamestats.SchedulerJob(db, cfg)` ein. Der Job selbst liegt
in `internal/gamestats/job.go`. Das ist ein kleiner Seam und erhält beide Regeln.

**Die nicht offensichtliche Folge:** Die Vorbefüllung der Ergebnisfelder des
redaktionellen Spielberichts kann **nicht** im Backend geschehen. `internal/matchreports`
ist ein Domain-Package; ein Import von `internal/gamestats` wäre Domain→Domain und bricht
den Architektur-Test. Ein gemeinsames Foundation-Package nur für diesen einen Lesezugriff
einzuziehen, wäre eine Verrenkung für eine Bequemlichkeit.

Deshalb: Der Redaktions-Editor ruft im Frontend zusätzlich den Bericht ab und befüllt die
vier Zahlenfelder vor. Sie bleiben editierbar; liegt kein Bericht vor, bleibt alles wie
bisher. Das Backend von `matchreports` ändert sich nicht.

---

## 9. Auth-Tier und Routen

**Sichtbarkeit: alle Eingeloggten.** Ein Tier, keine Objektrechte. Die Begründung ist die
aus §2: die Daten sind öffentlich abrufbar, eine Abstufung innerhalb des Vereins hätte
keinen Schutzzweck.

| Route | Tier |
|---|---|
| `GET /api/staffeln` | Authenticated |
| `GET /api/staffeln/{id}/tabelle` | Authenticated |
| `GET /api/staffeln/{id}/spielplan` | Authenticated |
| `GET /api/staffeln/{id}/ranglisten` | Authenticated |
| `GET /api/bwhv-games/{id}/report` | Authenticated |
| `GET /api/bwhv-reports/{id}/pdf` | Authenticated |
| `GET /api/members/{id}/saisonstatistik` | Authenticated |
| `GET /api/bwhv/staffel-katalog` | Vorstand (+ Admin) |
| `POST /api/staffeln/{id}/poll` | Vorstand (+ Admin) |
| `PUT /api/kader/{id}` (erweitert) | unverändert |

Die `{id}`-Lese-Routen kommen mit Begründung nach `openByDesign` in
`internal/permissions/object_matrix_test.go` — sie sind vereinsweit sichtbar, nicht
ungeschützt vergessen. `POST /api/staffeln/{id}/poll` ist eine Mutation und broadcastet
(`bwhv-updated`), sonst schlägt das Broadcast-Gate zu.

---

## 10. Benachrichtigung: bewusst keine

Kein Push, kein `notify.Send`, kein `user_events`-Eintrag. An einem Spieltag mit neun
Mannschaften entstünden bis zu neun Meldungen für ein Ergebnis, das die Beteiligten schon
kennen — sie standen in der Halle.

Der SSE-`Broadcast` bleibt trotzdem Pflicht (Hard Rule): eine offene Session, die eine
Staffel-Tabelle zeigt, muss nachladen. `bwhv-updated` wird also gesendet, aber niemand
bekommt eine Push.

Damit berührt dieser Change weder das Push-Fan-out-Gate noch die
`user_events`-Kategorien-CHECK — es braucht **keine neue Kategorie und keine Migration**
dafür.

---

## 11. Offene Punkte / Risiken

1. **Layout-Drift im PDF.** Die Kopfzeilen-Ableitung (§5.1) deckt Spaltenverschiebungen
   ab, nicht eine Umbenennung der Kopfzeilen selbst. Bei unbekanntem Kopf bricht der
   Parser mit klarer Meldung ab, statt zu raten. Die eingecheckte Fixture ist der
   Regressionsschutz; ein Live-Abruf im Test ist ausgeschlossen.
2. **`robotextstate` wird nicht ausgewertet.** Beobachtet wurde ausschließlich der Wert
   `"generated"`. Ob es andere gibt und ob ein `sGID` je ohne fertigen Bericht gesetzt
   wird, ist **nicht verifiziert**. Der Abruf behandelt einen fehlgeschlagenen PDF-Download
   deshalb als wiederholbar (`attempts`), nicht als endgültig.
3. **Periodenwechsel.** `p=142` ist die Hallenrunde 26/27; die Liste steht in
   `menu.period.list` und trägt `selectedID`. Der Katalog-Lauf liest `selectedID`, statt die
   Nummer zu verdrahten. Ob `selectedID` am Saisonübergang rechtzeitig umspringt, ist
   **nicht verifiziert** — hier ist im ersten Sommer ein Blick fällig.
4. **Namensgleichheit über Mannschaften hinweg** wird nicht aufgelöst: ein Spieler, der
   für zwei Mannschaften desselben Vereins aufläuft, ist zwei `bwhv_players`-Zeilen. Für
   eigene Spieler heilt das die `member_id`; für fremde bleibt es so. Bewusst akzeptiert.
5. **Kein Backfill vergangener Saisons.** Die API kennt 24/25 (`p=130`) und 25/26
   (`p=137`). Die Saison ist überall Parameter, ein Backfill bleibt damit als Folge-Change
   möglich — er bräuchte allerdings die Staffel-Zuordnung je Saison, die für frühere Jahre
   nicht gepflegt ist.
6. **Speicher.** ~116 MB PDFs pro Saison, kumulativ über Jahre. Eine Retention ist bewusst
   nicht Teil dieses Changes; bei Bedarf ist sie ein eigener, nach dem Muster von
   `trainingstagebuch-retention` gebauter Folge-Change.
