## 1. Kalender-Rahmen und Faltung (RFC)

- [ ] 1.1 `writeLine` auf Rune-Grenzen umstellen: größte Grenze ≤ 75 Oktette für die erste Zeile,
      ≤ 74 für Fortsetzungszeilen. Test `TestRenderICal_FaltungZerteiltKeineRune` über einen Feed
      mit absichtlich langen Namen (`SG Köngen-Wendlingen-Unterensingen`, `mB1 · erw. Kader ·
      nicht aufgestellt`): jede Zeile der Ausgabe erfüllt `utf8.Valid` **und** `len ≤ 75`.
- [ ] 1.2 `VTIMEZONE`-Komponente für `Europe/Berlin` in `renderICal` vor dem ersten `VEVENT`
      ausgeben (CET/CEST, letzter Sonntag März/Oktober), mit Kommentar zur Abhängigkeit von den
      EU-Regeln. Test: genau ein `BEGIN:VTIMEZONE` mit `TZID:Europe/Berlin`, Position vor dem
      ersten `BEGIN:VEVENT`.
- [ ] 1.3 `calEvent` um `Stamp time.Time` erweitern; `created_at` in `fetchGames`,
      `fetchTrainings` und `fetchDuties` mitselektieren und über einen Helfer **explizit als UTC**
      parsen (Fallback `time.Now().UTC()`). `renderICal` schreibt `DTSTAMP:<…Z>` in jedes
      `VEVENT`. Tests: `DTSTAMP` in jedem Event vorhanden; zwei Renderings derselben Daten liefern
      denselben Wert.

## 2. Venue-Helfer zusammenführen

- [ ] 2.1 Die Adressbildung (`Name, Straße, PLZ Ort`) aus `fetchGames` und `fetchTrainings` in
      einen Helfer ziehen und von beiden Stellen aufrufen. Kein Verhaltenswechsel — die
      bestehenden Feed-Tests müssen unverändert grün bleiben.

## 3. Dienst-Events vervollständigen

- [ ] 3.1 `fetchDuties`: `LEFT JOIN games` über `duty_slots.game_id` und `LEFT JOIN venues` über
      `games.venue_id`, `LOCATION` über den Helfer aus 2.1 setzen. Tests: Dienst am Spiel trägt
      dieselbe `LOCATION` wie das Spiel; Dienst ohne Spielbezug und Dienst am Spiel ohne Venue
      tragen keine `LOCATION`-Zeile.
- [ ] 3.2 `DTEND` aus `duty_slots.hours_value` statt fixer Stunde; `hours_value <= 0` fällt auf
      eine Stunde zurück. Tests: `3.0` → drei Stunden, `1.5` → 90 Minuten, `0` → eine Stunde,
      und in keinem Fall `DTEND <= DTSTART`.
- [ ] 3.3 `DESCRIPTION` aus `duty_slots.role_desc`, wenn gepflegt. Tests: gepflegte Rolle steht im
      Event; leere Rolle erzeugt keine `DESCRIPTION`-Zeile.
- [ ] 3.4 Fehlende `event_time` als Ganztags-Event: `calEvent` um `AllDay bool` erweitern,
      `renderICal` schreibt dann `DTSTART;VALUE=DATE:<Tag>` und `DTEND;VALUE=DATE:<Folgetag>`.
      Tests: Dienst ohne Uhrzeit erzeugt die `VALUE=DATE`-Form; Dienst mit Uhrzeit bleibt bei
      `TZID`; kein Mitternachtstermin mit `hours_value`-Dauer.

## 4. Spec und Abschluss

- [ ] 4.1 Die zurückgestellten Fälle als Folge-Notiz festhalten (generische Termine ohne
      Kader-Zusatz und ohne Namens-Fallback, `SEQUENCE`/`updated_at`, Kind-Feed-Anrede), damit sie
      nicht mit diesem Change als erledigt gelten.
- [ ] 4.2 `/verify-change` ausführen (Build, `go test ./...`, `golangci-lint`, Frontend-Gate,
      `openspec validate`) und die Änderung gegen die Projekt-Invarianten prüfen.
- [ ] 4.3 Change archivieren (`openspec archive`), nachdem die Deltas in `openspec/specs/ical-feed`
      übernommen sind.
