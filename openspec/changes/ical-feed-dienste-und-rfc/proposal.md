## Why

Der ICS-Feed baut Spiel-, Trainings- und Dienst-Events in drei getrennten Code-Pfaden, und die
Dienst-Variante ist dabei die ärmste geblieben: sie trägt keinen Ort, keine Beschreibung und —
gravierender — eine **erfundene Dauer von einer Stunde**, obwohl `duty_slots.hours_value` seit
Migration `052` die reale Dauer führt und Migration `054` sie über die Ablösekette sogar exakt auf
das Ende des Folgedienstes kappt. Ein dreistündiger Bewirtungsdienst blockt im Kalender eine
Stunde; wer die Zeit danach verplant, kollidiert mit einem Dienst, den er zugesagt hat.

Unabhängig davon ist der erzeugte Kalender an zwei Stellen nicht RFC-5545-konform: `DTSTAMP` fehlt
in jedem `VEVENT` (dort Pflicht), und `DTSTART;TZID=Europe/Berlin` verweist auf eine
Zeitzonendefinition, die im Kalender nirgends steht. Dazu kommt ein nachgewiesener Fehler: die
Zeilenfaltung schneidet bei 75 **Bytes** und zerteilt dabei UTF-8-Sequenzen — mit realistischen
Gegnernamen reproduzierbar, und ob es passiert, hängt allein an Namenslängen, die der Verein
jederzeit ändert.

## What Changes

**Dienst-Events auf das Niveau der übrigen Varianten heben**

- `LOCATION` am Dienst: aufgelöst über `duty_slots.game_id` → `games.venue_id` → `venues`, im
  selben Format wie bei Spielen und Trainings. Dienste ohne Spielbezug bleiben wie bisher ohne Ort.
- `DTEND` aus `duty_slots.hours_value` statt aus einer fixen Stunde. Nicht-positive Werte fallen
  auf eine Stunde zurück, damit eine kaputte Bestandszeile keinen Null-Längen-Termin erzeugt.
- `DESCRIPTION` am Dienst aus `duty_slots.role_desc`, sofern gepflegt.
- Dienste **ohne** `event_time` werden zum Ganztags-Event statt zu einem Termin um Mitternacht.
  Ohne diesen Punkt macht die echte Dauer das bestehende Mitternachts-Artefakt sichtbarer: ein
  dreistündiger Dienst ohne Uhrzeit blockte sonst 00:00–03:00 statt bisher 00:00–01:00.

**Mitgenommen, weil sonst eine dritte Kopie entstünde**

- Der Venue-String (`Name, Straße, PLZ Ort`) ist in `fetchGames` und `fetchTrainings` wortgleich
  dupliziert. Der Dienst-Ort wäre die dritte Kopie; stattdessen wird die Bildung einmal
  herausgezogen und von allen drei Stellen genutzt. War als eigener Refactor geplant — ihn nach
  dem Anlegen einer dritten Kopie nachzuholen wäre die schlechtere Reihenfolge.

**RFC-5545-Konformität**

- `DTSTAMP` in jedem `VEVENT`, abgeleitet aus dem `created_at` des jeweiligen Termins (stabil über
  Abrufe hinweg), Fallback auf die aktuelle Zeit.
- Ein `VTIMEZONE`-Block für `Europe/Berlin` im Kalender, auf den die `TZID`-Parameter verweisen.
- Zeilenfaltung an Rune- statt Byte-Grenzen: weiterhin maximal 75 Oktette pro Zeile, aber nie
  mitten in einer UTF-8-Sequenz.

**Nicht Teil dieses Changes** (aus derselben Analyse, bewusst zurückgestellt):

- Generische Termine verlieren den Kader-Zusatz (`gameTitle`-`default`-Zweig) und haben keinen
  Namens-Fallback, wenn `opponent` leer ist → eigener Change, ändert Titel-Zusagen.
- Der `Team (…)`-Wrapper nur bei Spielen und das fehlende Gattungswort bei generischen Terminen →
  bewusst so bzw. Geschmacksfrage.
- Der Kind-Feed spricht den lesenden Elternteil mit „Du" an → offene Produktentscheidung.

`SEQUENCE`/`LAST-MODIFIED` bleiben ebenfalls draußen: keine der drei Quelltabellen führt ein
`updated_at`, und ein konstantes `SEQUENCE:0` wäre eine Metadatenangabe, die nichts aussagt.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `ical-feed`: Die Anforderung „VEVENT-Struktur für einen Dienst" bekommt `LOCATION`,
  `DESCRIPTION` und ein aus `hours_value` abgeleitetes `DTEND`. Die Anforderung
  „Feed-Generierung" bekommt `DTSTAMP`, `VTIMEZONE` und die Zusage, dass die Faltung keine
  Mehrbyte-Zeichen zerteilt.

## Impact

- **Code:** `internal/calendar/handler.go` — `fetchDuties` (Query + Event-Bau), `fetchGames` und
  `fetchTrainings` (gemeinsamer Venue-Helfer), `renderICal` (Kalender-Rahmen), `writeLine`
  (Faltung), `calEvent` (Felder für `DTSTAMP` und Ganztags-Kennzeichen).
- **Tests:** `internal/calendar/handler_test.go` (Dienst-VEVENT, Rahmen-Pflichtfelder,
  Faltungs-Invariante über `utf8.Valid`).
- **DB:** keine Migration — alle benötigten Spalten (`hours_value`, `role_desc`, `game_id`,
  `created_at`) existieren.
- **API:** keine neue Route, kein geändertes Response-Schema der Token-Endpunkte.
- **Clients:** Bestehende Abonnements bleiben gültig; die UIDs ändern sich nicht. Kalender-Clients
  aktualisieren Dienst-Termine beim nächsten Abruf auf die neue Dauer.
