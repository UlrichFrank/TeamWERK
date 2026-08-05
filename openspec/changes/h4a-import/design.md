# Design — H4A-Import

Alle hier dokumentierten Mechanik-Details wurden vorab **live gegen Handball4All und die
lokale DB verifiziert** (Chrome DevTools + curl + SQLite-Abgleich). Die Zahlen und der
xajax-Payload sind gemessen, nicht geraten.

## 1. Datenquelle: `edit.php` (xajax), nicht CSV-Export

Bewusste Entscheidung des Auftraggebers: Direkt-Import über `edit.php` statt des
CSV-Downloads (`mannschaftsspielplaene.php`). Begründung und Trade-offs:

| | `edit.php` (gewählt) | CSV-Export (verworfen) |
|---|---|---|
| Umfang | ganze Saison | nur aktive Periode |
| Filter „eigene Spiele" | `opOwnGames` serverseitig | `own=1` serverseitig |
| Spielnummer (Anker) | ✓ | ✓ |
| interne game-ID | ✓ | ✗ |
| Hallen-Adresse | nur Nummer | volle Adresse |
| Format | HTML-in-JSON (xajax) | CSV im ZIP |
| Parsing-Risiko | **hoch** (HTML kann brechen) | niedrig |

Die Hallen-Adresse fehlt bei `edit.php` — deshalb ist der Hallennummer→Venue-Backfill
Voraussetzung (Abschnitt 5).

### 1.1 Der verifizierte xajax-Aufruf

Login (Formular-POST, kein Basic Auth, kein CSRF-Token):

```
POST https://meinh4a.handball4all.de/index.php
login=<user>&pw=<pass>&hvwsubmit=submit&submit=Anmelden
→ Set-Cookie: PHPSESSID=…   (Session-Auth für Folge-Requests)
```

Perioden-/Staffel-Optionen aus `GET /games/edit.php` auslesen (`<select id="ge_periods">`
liefert die Saison-IDs, z. B. `142` = Hallenrunde 26/27).

Spielabruf (xajax 0.5 — **jeder Wert braucht das `S`-Typ-Präfix**, Sonderzeichen zusätzlich
CDATA; genau daran scheitern naive Nachbauten):

```
POST https://meinh4a.handball4all.de/games/edit.php
Content-Type: application/x-www-form-urlencoded
X-Requested-With: XMLHttpRequest

xjxfun=xajax_update
xjxr=<unix-ms>
xjxargs[]=<xjxobj>
  <e><k>ge_statsel</k><v>S0</v></e>
  <e><k>ge_dasel</k>  <v>S<![CDATA[all;all]]></v></e>
  <e><k>ge_gameno</k> <v>S</v></e>
  <e><k>sbdasel</k>   <v>SLos</v></e>
  <e><k>ge_periods</k><v>S<periodId></v></e>
</xjxobj>
xjxargs[]=<xjxobj>
  <e><k>dummy</k>     <v>S1</v></e>
  <e><k>opOwnGames</k><v>Son</v></e>
</xjxobj>
```

Antwort: `application/json` mit `{"xjxobj":[{"cmd":"as","id":"gametable_container","data":"<table>…"}]}`.
Die Spiele stehen als HTML-Tabelle im `data`-Feld. **Verifiziert:** ohne `opOwnGames` 644
Zeilen (alle Hallenspiele inkl. Fremdvereine, `v_109` ist Hallenverantwortlicher), mit
`opOwnGames=Son` genau **146** Team-Stuttgart-Spiele (Heim + Auswärts, ganze Saison).

### 1.2 Tabellen-Spalten (pro `<tr>` mit `id="game<internalId>"`)

```
Staffel | Nr. | TL | Runde | Halle(Nr) | Datum | Zeit | Heim | Gast | Kommentar
mA-BOL-SRM | 211004 | 281 | 1 | 3029 | Sa, 19.09.2026 | 12:30 | HSC Schm/Oeff | Team Stuttgart 2 | Än. HT
```

- **`Nr.`** = BWHV-Spielnummer → `games.external_id` (Idempotenz-Anker, in beiden Quellen
  stabil und eindeutig über alle 146).
- **`Halle`** = nur Hallennummer → Auflösung über `venues.hall_number` (Abschnitt 5).
- **`Kommentar`** (`Än.` geändert, `Vl.` verlegt) = Änderungssignal direkt aus der Quelle;
  kann das Diff-Modal anreichern (optional, kein Muss).
- Datum `Sa, 19.09.2026` → `date`; `Zeit` → `time` (leer möglich → 00:00 + Warnung).

## 2. Vertrauensklasse: fremde Zugangsdaten (Sicherheit)

TeamWERK nimmt bisher **nie** ein fremdes Passwort entgegen — der Rest des Systems bewegt
sich Richtung Zero-Knowledge (der Server soll *weniger* sehen). Der H4A-Import ist bewusst
das Gegenteil und muss deshalb explizit abgesichert und dokumentiert sein:

- Zugangsdaten werden **nur im `preview`-Request** angenommen (HTTPS erzwungen).
- **Niemals loggen** — nicht in Request-Logs, nicht in Fehler-Dumps, nicht in `regen`-Reports.
- **Niemals persistieren** — keine DB-Zeile, keine Datei, keine Env-Var, kein Cache.
- Ausgehende H4A-Requests **nur über `https://`** (TLS-Pflicht, kein Downgrade).
- Nach dem Request fallen die Credentials aus dem Speicher (keine Referenz in Rückgaben).
- Bei H4A-Login-Fehler: **generische** Meldung („Anmeldung bei Handball4All fehlgeschlagen"),
  Passwort nicht zurückspiegeln, kein Timing-Orakel nötig (kein Sicherheitsziel).
- Kein Cron/Unattended-Login (bewusst) — der Import ist immer admin-getriggert.

Diese Regeln kommen zusätzlich als Gotcha nach `docs/agent/06-gotchas.md`.

**ToS-Vorbehalt:** Automatisierter Login-Zugriff kann H4A-Nutzungsbedingungen berühren.
Admin-getriggert (kein Dauer-Cron) ist die mildeste Form; vor Produktivnahme kurz mit
BWHV/H4A abklären. Als Change-Artefakt festgehalten, keine Rechtsberatung.

## 3. Zwei-Phasen-Flow

```
 Admin /kalender → „Aus Handball4All importieren"
   │ tippt user/pass (+ Saison-Dropdown aus ge_periods)
   ▼
 POST /api/games/import/h4a/preview  { user, pw, period_id }   ← Credentials NUR hier
   │  Server: login → GET edit.php (Perioden) → POST xajax (S-kodiert, opOwnGames)
   │          → 146 Zeilen parsen → Staffel→team, Halle→venue
   │          → Diff gegen games (Anker external_id) → logout, Session verwerfen
   ▼  { plan: { new[], changed[], unchanged[] }, mappings, warnings, plan_token }
 Diff-Modal: Zeilen einzeln (de)selektierbar; Template-Batch + selektiv;
             Staffel→Mannschaft-Zuordnung bestätigen/überschreiben
   ▼
 POST /api/games/import/h4a/apply  { decisions[], template_choices, mappings }  ← KEIN H4A
   │  Server re-validiert (aktive Saison, teams/venues existieren, template gültig)
   │  → INSERT/UPDATE games + game_teams → EIN Broadcast, EIN runAutoRegen über die
   │    Vereinigungsmenge aller Datumsfenster → EINE Notification (aggregiert)
   ▼  { imported, updated, skipped, regen_summary }
```

**Warum Plan zurück statt Server-State:** `apply` braucht H4A nicht mehr — das Passwort
lebt nur im `preview`. Der Client schickt die bestätigten Entscheidungen zurück; der Server
**re-validiert jede Zeile** gegen die DB (vertraut dem Client-Plan nicht blind: Team-Scope,
Saison, FK-Existenz, Template-Gültigkeit werden erneut geprüft). Da nur Vorstand/Admin
importieren und den Diff gesehen haben, ist das Bestätigen des Plans akzeptabel; die
Re-Validierung schützt gegen manipulierte/veraltete Pläne. Kein Server-seitiger
Import-State, keine TTL-Verwaltung.

## 4. Staffel → Mannschaft (gelerntes Mapping)

Der einzige nicht-deterministische Schritt. `Staffel` (`mA-BOL-SRM`) + eigener Vereinsname
(`Team Stuttgart` vs. `Team Stuttgart 2`) → `teams.id`. Verbundschlüssel, weil beide
Mannschaften theoretisch in derselben Staffel stehen könnten.

- **Erster Import**: unbekannte Staffeln werden im Modal manuell einer Mannschaft
  zugeordnet; die Zuordnung wird gelernt (Tabelle `h4a_staffel_team_map` oder Spalte an
  `teams`, Entscheidung in Tasks).
- **Folge-Importe**: bekannte Staffeln sind vorbelegt; nur neue Staffeln brauchen Handarbeit.
- Vereinsnamen-Erkennung über gepflegte Alias-Liste (nicht `strings.Contains("Team")` —
  „HandballTeam Heckengäu" wäre ein False-Positive).
- Kein Mapping gefunden → Zeile bleibt im Diff, aber nicht importierbar bis zugeordnet.

## 5. Hallennummer → Venue (deterministisch, verifiziert)

`Halle`-Nummer aus `edit.php` → `venues.id` über `venues.hall_number`. Voraussetzung ist der
Backfill der Hallennummer an die bereits importierten Venues.

**Backfill-Strategie** (Teil des erweiterten Hallenlisten-Imports, **nicht** als SQL-Migration
— eine `.up.sql` kann die CSV nicht lesen; die Migration legt nur die Spalte an):

- Match Bestands-Venue ↔ Hallenlisten-Zeile über **`(Name, Ort, Straße)`**.
- Eindeutiger Match → `hall_number` setzen.
- **Mehrdeutig** (mehrere Nummern für gleiche Adresse) oder **kein Match** → `hall_number`
  bleibt `NULL`, Eintrag in den Import-Report (`ambiguous` / `unmatched`).

**Verifiziert gegen lokale `./teamwerk.db` (1025 Venues):**

```
  1017  → genau eine Hallennummer   ✓ sauber backfillbar
     1  → MEHRDEUTIG                 ⚠ Sandberghalle Flein → 1018 ODER 1066
                                        (BWHV-Datenfehler: gleiche Adresse, zwei Nummern,
                                         widersprüchliche Kennzeichnung) → bleibt NULL
     7  → kein Hallenlisten-Match    ✓ manuelle Nicht-BWHV-Venues (Vereinsgaststätte,
                                        Jugendraum, Inselbad-Parkplatz) → korrekt NULL
     0  → Hallenlisten-Eintrag ohne Venue (früherer Import war vollständig)
```

Alle spielrelevanten Hallennummern lösen **eindeutig** auf (15/15 Quali + Stichproben
Hauptrunde). Beispiel: `3029 (Fellbach-Oeffingen) → venue 968`. Kein Team-Stuttgart-Spiel
findet in Flein statt → die eine Ambiguität ist für den Import irrelevant.

**Prod-Vorbehalt:** Der Abgleich lief gegen die lokale DB; der Prod-Bestand kann minimal
abweichen. Der Import ist fail-safe (NULL + Report bei Unsicherheit), also unabhängig vom
exakten Bestand sicher — vor dem echten Lauf denselben Report einmal auf Prod ziehen.

**Schema:** `venues.hall_number INTEGER` nullable; Partial-Unique-Index
`WHERE hall_number IS NOT NULL` (die Nicht-BWHV-Venues und die eine Ambiguität bleiben NULL
und dürfen kollidieren).

## 6. Idempotenz & Diff

- Anker: `games.external_id = Nr.` (BWHV-Spielnummer). Vor dem Import fehlt sie
  Bestandsspielen — die wurden bislang manuell angelegt. Erster H4A-Import erzeugt daher
  potentiell Duplikate zu manuell angelegten Spielen; das Modal zeigt Kandidaten (gleiches
  Datum+Team+Gegner ohne `external_id`) als **mögliche Dubletten** zum manuellen Verknüpfen.
- **Neu**: kein `games`-Eintrag mit dieser `external_id`.
- **Geändert**: `external_id` existiert, aber `date`/`time`/`venue`/`opponent`/`is_home`
  weicht ab → Diff-Zeile mit Alt→Neu pro Feld.
- **Unverändert**: alle Felder gleich → im Modal ausblendbar.
- **Löschungen werden NICHT abgeleitet** — `edit.php` liefert nur Spiele, die H4A kennt;
  „fehlt im Abruf" heißt nicht „abgesagt" (Auslosung wächst über die Saison). Absagen
  bleiben manuell.
- Verschiebt sich ein Datum, verwaisen ggf. `is_custom=1`-Dienstslots am alten Datum → im
  Diff als Hinweis anzeigen (Auto-Regen schont `is_custom`-Slots, räumt sie aber nicht um).

## 7. Batch-Verarbeitung (kein CreateGame-in-Schleife)

`CreateGame` feuert pro Spiel `notify.Send` (an alle Team-Mitglieder + Eltern),
`broadcastGame` und `runAutoRegen` über ein ±1-Tag-Fenster. 146× hintereinander wäre ein
Push-/SSE-/Regen-Sturm. Der `apply`-Pfad baut daher einen **eigenen** Batch-Insert:

- **Ein** `runAutoRegen` über die **Vereinigungsmenge** aller betroffenen Datumsfenster
  (`runAutoRegen` nimmt bereits `dates []string`).
- **Ein** Hub-Broadcast (`games`) am Ende.
- Notifications **aggregiert oder unterdrückt** (Import ist Vorstand-Aktion; ein
  „X Spiele importiert" statt 146 Einzel-Pushes) — Default: **keine** Spieler-Pushes beim
  Import, nur der Regen-Summary an den Importeur.

## 8. Auth-Tier & Routen

Analog Games-CRUD (Vorstand-Tier, Admin-Bypass):

```
POST /api/games/import/h4a/preview   RequireClubFunction("vorstand")   (admin bypass)
POST /api/games/import/h4a/apply     RequireClubFunction("vorstand")
```

Der erweiterte Hallenlisten-Import bleibt auf seinem bestehenden Tier
(`POST /api/venues/import`).

**Broadcast-Gate:** `preview` mutiert nichts → Allowlist-Eintrag mit Begründung
(„read-only, ruft externe Quelle ab, kein DB-Write"). `apply` mutiert → broadcastet `games`.

## 9. Offene Punkte / Risiken

- **HTML-Parsing ist der Sollbruch.** `edit.php` liefert keine API, sondern eine `<table>`.
  Defensives Parsing + harte Fehlermeldung („H4A-Format geändert, Adapter anpassen") statt
  stiller Teilergebnisse. Parser-Test gegen eingecheckte HTML-Fixture (keine Live-Abhängigkeit
  im Test).
- **ToS** (Abschnitt 2) — vor Produktivnahme klären.
- **Prod-Venue-Bestand** (Abschnitt 5) — Report einmal auf Prod verifizieren.
- **Saison-Mapping** H4A-Periode ↔ TeamWERK-`seasons`: der Import braucht eine aktive
  TeamWERK-Saison für `games.season_id`; die H4A-`period_id` wählt der Admin. Kein
  automatischer Abgleich der Zeiträume (bewusst manuell).
- **Passwortwechsel**: der genutzte Vereins-Account (`v_109`) ist geteilt; die
  Credentials-Eingabe im Import ändert daran nichts, aber die Modellgrenze (Server sieht das
  Passwort transient) muss dem Vorstand bewusst sein.
