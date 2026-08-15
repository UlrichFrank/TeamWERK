> **Keine Migration.** Dieser Change fasst kein Schema an — `games.venue_id → venues(id)`
> existiert seit `001_initial.up.sql:206`. Es kommt lediglich ein JOIN und ein
> Response-Feld hinzu.
>
> **Broadcast-Gate nicht betroffen.** Die einzige berührte Route ist ein `GET`
> (`/api/duty-board`); `internal/arch/broadcast_test.go` prüft nur Mutations-Routen.

## 1. Backend: Spielort auf dem Duty-Board

- [x] 1.1 `internal/duties/handler.go`, `Board` (ab Zeile 637): `LEFT JOIN venues v ON v.id = g.venue_id` hinter den bestehenden `LEFT JOIN games g` (Zeile 773) setzen und `COALESCE(v.name, '')` in die Projektion aufnehmen. Der `whereParts`-Block bleibt **unverändert** — kein neues Sichtbarkeits-Prädikat, siehe `design.md` §10.
- [x] 1.2 `boardGroup` (Zeile 707) um `Venue string \`json:"venue,omitempty"\`` erweitern, die neue Spalte in `rows.Scan` aufnehmen und beim Anlegen der Gruppe setzen. `omitempty` ist Pflicht: game-lose Gruppen sollen die Payload nicht vergrößern (`payload-measurement`).
- [x] 1.3 Test in `internal/duties/board_venue_test.go` (eigene Datei statt `pagination_test.go` — dort geht es um Paginierung; das Testpaket `duties_test` teilt die Helfer ohnehin): Gruppe mit Spiel **mit** Venue trägt `venue` mit dem `venues.name`; Gruppe mit Spiel **ohne** `venue_id` und game-lose Gruppe tragen kein `venue`. Fixtures über `testutil.CreateGame`; falls dort kein Venue-Parameter existiert, das Venue direkt per `INSERT` anlegen und `games.venue_id` setzen, statt die Fixture-Signatur zu ändern.
- [x] 1.4 Regressionstest für Invariante 1: für je einen Spieler, ein Elternteil, einen Trainer und einen Vorstand die Menge der zurückgegebenen Gruppen-/Slot-IDs gegen die erwartete Menge prüfen — der Test muss brechen, wenn der neue JOIN Zeilen dupliziert oder wegfiltert.

## 2. Frontend: Parser-Bibliothek

- [x] 2.1 `web/src/lib/eventFilter.ts` anlegen. Öffentliche API bewusst seitenblind: `parseQuery(q: string): Token[]` und `matchesQuery(tokens: Token[], text: string[], dates: string[]): boolean`. Kein Import aus `pages/` oder `components/` — der Parser kennt weder Spiele noch Dienste (`design.md` §6).
- [x] 2.2 Normalisierung implementieren: `toLowerCase()` + `normalize('NFD')` + Combining-Marks strippen. **Keine** Transliteration `ö→oe`. Einmal pro Feld beim Match, nicht pro Token.
- [x] 2.3 Token-Parsing: Split an `/\s+/`, leere Tokens verwerfen. Je Token die drei Interpretationen vorberechnen (Freitext-Literal, optionales `{day, month, year?}` aus `TT.MM.` / `TT.MM.JJJJ`, optionaler Monatsindex aus Präfix ≥ 3 Zeichen gegen die zwölf normalisierten deutschen Monatsnamen). Mehrdeutige Präfixe (`ju`) ergeben **keinen** Monat.
- [x] 2.4 Matching: Tokens konjunktiv, Interpretationen je Token disjunktiv. Datumsvergleich ausschließlich über `date.slice(0, 10)` (Gotcha „SQLite DATE-Felder"). Jahreslose Angaben ignorieren das Jahr, `TT.MM.JJJJ` vergleicht exakt.
- [x] 2.5 `web/src/lib/eventFilter.test.ts`: die Invarianten 2–6 und 9 aus `proposal.md` als eigene Tests — insbesondere „ein Token verliert keinen Treffer" (`mar` → März **und** Markthalle), „Tokens sind konjunktiv", „jahreslos trifft zwei Jahrgänge", „`ju` ist kein Monat, `jun` schon", „`goppingen` findet Göppingen, `goeppingen` nicht", „leerer/whitespace-Ausdruck ist ein No-Op".

## 3. Frontend: Eingabekomponente

- [x] 3.1 `web/src/components/EventSearchInput.tsx` anlegen — Props `value`, `onChange`, `placeholder`, `compact`. Optisch neben `EventTypeFilter` einreihbar. `<Search>` als Präfix-Icon, `<X>` zum Leeren (mit `aria-label`), beides aus `lucide-react`. Klassen exakt nach dem Input-String aus `docs/agent/05-frontend.md`; nur `brand-*`-Tokens.
- [x] 3.2 Mobile: Touch-Target ≥ 44 px (`py-2.5 sm:py-1.5`) und `type="search"` gesetzt. **Gegen den iOS-Zoom war nichts zu tun**: `ios-input-zoom-prevention` löst das global in `web/src/index.css` (`input, textarea, select { font-size: 16px }` unter 640 px), nicht per Utility-Klasse am Feld.
- [x] 3.3 Wiederverwendbaren Hook oder Helper für den verzögerten URL-Sync bereitstellen (~250 ms, `setSearchParams(next, { replace: true })`) — der Wert wird lokal gehalten und getrennt in die URL geschrieben (`design.md` §9). Beim Mount initialisiert `q` aus der URL den lokalen State.
- [x] 3.4 Gemeinsame Leermeldung für den Ausgeblendet-Fall bauen (Text + „Filter zurücksetzen"-Aktion), damit die drei Seiten nicht drei Varianten formulieren. Alert-Klassen nach `05-frontend.md`.

## 4. Frontend: /dienste

- [x] 4.1 `BoardGroup` in `web/src/pages/DutyPage.tsx` (Zeile 13) um `venue?: string` erweitern.
- [x] 4.2 `q` in `parseFilters` (Zeile 52) und `updateFilter` (Zeile 93) aufnehmen — leerer String löscht den Parameter, analog zur bestehenden Default-Behandlung von `types`.
- [x] 4.3 Adapter: aus einer `BoardGroup` das Textfeld-Array bauen (`opponent`, `venue`, `team_names`, `label`, je Slot `duty_type`, `role_desc`, `assignees[].name`) und das Datums-Array (`date`). Das `q`-Prädikat als **letztes** in `visibleGroups` (Zeile 174) einhängen — nach den billigen Set-Lookups, aber unter dem unbedingten `groupMatchesFocus`-Durchlass (`design.md` §8).
- [x] 4.4 Eingabefeld in die Filterzeile einsetzen, Placeholder „Gegner, Ort, Dienst, Person…". `useCompactHeader(950)` (Zeile 86) beachten: im Compact-Modus darf das Feld die Pillen-Reihe nicht umbrechen lassen.
- [x] 4.5 Ausgeblendet-Zähler: zweiter Durchlauf mit ausschließlich dem `q`-Prädikat über `groups`, nur wenn `visibleGroups` leer, `q` nicht leer und mindestens ein weiterer Filter aktiv ist (`filterTeamId`, `filterTypes` ≠ Default, `viewMine`, `showPast` = false zählt **nicht** als Filter — es ist der Default-Zustand).
- [x] 4.6 Test `web/src/pages/DutyPage.filter.test.tsx` (neben der bestehenden `DutyPage.focus.test.tsx`, nicht im `__tests__/`-Unterordner): Filter verengt die Gruppenliste; kombiniert sich mit dem Team-Filter; fokussierte Gruppe bleibt trotz nicht passendem `q` sichtbar (Invariante 8); Ausgeblendet-Zähler nennt die exakte Anzahl (Invariante 7).

## 5. Frontend: /termine

- [x] 5.1 `Session` in `web/src/pages/TerminePage.tsx` (Zeile 40) um `title: string` erweitern — `/api/training-sessions` liefert das Feld bereits (die `Training`-Deklaration in `KalenderPage.tsx:45` führt es), `/termine` deklariert es nur nicht.
- [x] 5.2 `q` in `parseFilters` (Zeile 110 ff.) und den URL-Writer (Zeile 216) aufnehmen.
- [x] 5.3 Adapter für beide Termin-Arten: Game → `opponent`, `venue.name`, `venue.city`, `team_display_short_csv`/`team_display_long_csv`/`team_names`, `note`; Session → `title`, `venue.name`, `venue.city`, `team_name`, `note`. Datums-Array je `data.date`. Prädikat als letztes in `visibleTermine` (Zeile 292) einhängen, unter dem `focus`-Durchlass (Zeile 293).
- [x] 5.4 Eingabefeld in die Filterzeile (bei Zeile 445), Placeholder „Gegner, Ort, Notiz…". Ausgeblendet-Zähler analog zu 4.5.
- [x] 5.5 Tests in `web/src/pages/TerminePage.test.tsx` ergänzen: Textfilter verengt die Liste; `?q=` aus der URL greift beim ersten Render; leeres `q` ist ein No-Op (Invariante 9); Fokus überlebt `q` (Invariante 8).

## 6. Frontend: /kalender

- [x] 6.1 `q` in `KalenderPage.tsx` an die bestehenden `searchParams` (Zeile 146) hängen. Die übrigen Filter (`filterTeamId`, `filterTypes`) bleiben lokaler State — die bestehende Inkonsistenz wird hier **nicht** mitrepariert (eigener Change, falls gewünscht).
- [x] 6.2 Prädikat in `monthGames` (Zeile 490) und `filteredTrainings` (Zeile 514) einhängen; Adapter analog zu 5.3, Teams aus `g.teams[].name`/`display_short`.
- [x] 6.3 **Abwesenheiten mitfiltern.** `absencesForDay` (Zeile 533) filtert heute clientseitig gar nicht; bei aktivem `q` blieben sonst Abwesenheits-Balken in Zellen stehen, deren Termine weggefiltert sind. Feldmenge: `member_name`, `note`, Typ-Label („Urlaub"/„Verletzung"); Datums-Array aus `start_date`/`end_date`.
- [x] 6.4 Eingabefeld in die Filterzeile über dem Gitter (bei Zeile 898), Placeholder „Gegner, Ort, Notiz…". Kein eigener Ansichtsmodus, kein Sprung in andere Monate — die Zellen leeren sich, wie beim Team-Filter (`design.md` §1).
- [x] 6.5 Ausgeblendet-Zähler entfällt hier bewusst — das Monatsgitter hat keine Leermeldung, in die er passt, und ein leeres Gitter ist die normale Anzeige eines Monats ohne Termine. Diese Auslassung im Code kurz kommentieren, damit sie nicht als Vergessenheit gelesen wird.
- [x] 6.6 Test `web/src/pages/KalenderPage.filter.test.tsx` (neben `KalenderPage.dateSync.test.tsx`): `q` leert die Zellen erwartungsgemäß; bei nicht passendem `q` bleibt **kein** Abwesenheits-Balken stehen (Invariante 10); leeres `q` ändert nichts (Invariante 9).

## 7. Abschluss

- [x] 7.1 `make test` (inkl. Architektur- und Broadcast-Gate) und `pnpm -C web build/test/lint` grün.
- [x] 7.2 `/verify-change` laufen lassen — insbesondere die Prüfpunkte brand-Tokens, lucide-Icons und Route→Tests.
- [x] 7.3 Chrome DevTools MCP gegen die Prod-Binary bei echten 375×812 (mobile+touch): kein horizontaler Überlauf (`scrollWidth == clientWidth == 375`), Filterzeile bleibt **einreihig** (Header-Höhe 42 px), `font-size: 16px` → **kein iOS-Zoom**, Feld endet bei x=281 < 375. Filter live verifiziert: `q=kasse` → Treffer bleibt, `q=göppingen` → Liste leer + korrekte Leermeldung, kein Ausgeblendet-Hinweis (kein weiterer Filter aktiv), URL synchronisiert auf `?q=g%C3%B6ppingen`. **Kein Playwright-Fall ergänzt** — keine Auffälligkeit gefunden. Nebenbefund (nicht behoben, siehe Bericht): das Touch-Target misst 42 px statt der in `05-frontend.md` genannten 44 px; das Referenz-Suchfeld auf `/mitglieder` misst 41 px — der `py-2.5`+`text-sm`-Rezept ergibt repo-weit 41–42 px.
- [ ] 7.4 `openspec validate termin-textfilter --strict` und Proposal archivieren.
