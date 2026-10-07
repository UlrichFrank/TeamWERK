# Design

## Context

Siehe proposal.md – Why. Relevanter Ist-Stand:

- `/kalender` kennt bereits `?date=YYYY-MM-DD` (Capability `kalender-date-param`) und initialisiert daraus Jahr/Monat. Der Termin-Dialog wird über den lokalen State `infoItem` geöffnet (`KalenderPage.tsx`), gespeist aus den geladenen Listen `games` und `trainings`.
- `/termine` und `/dienste` haben schon einen Fokus-Mechanismus für den Hinweg (`?focus=game-17`, Scroll + Ring-Hervorhebung, Fokus endet bei Filteränderung). Das Format `<kind>-<id>` wird für den Kalender übernommen.
- **Spiele lädt der Kalender nicht monatsweise**, sondern über `GET /api/games?limit=500`. Der Server deckelt per `httpx.Paging(r, 50, 200)` auf 200 und sortiert ab Saisonbeginn aufsteigend. In einer Saison mit mehr als 200 sichtbaren Terminen fehlen dem Kalender also die späten Spiele. Trainings werden monatsweise (`from`/`to`) geladen und sind davon nicht betroffen.

## Goals / Non-Goals

**Goals:**
- Ein Klick von einer Termin-Karte bzw. einem Dienst-Block landet im richtigen Monat mit offenem Termin-Dialog.
- Der Sprung funktioniert unabhängig von Kalender-Filtern und davon, ob der Termin in der geladenen Spieleliste steht.

**Non-Goals:**
- Die Matrix-/Tabellenansicht von `/termine` bekommt keine Aktion.
- Die Detailseiten (`/termine/spiel/{id}`, `/termine/training/{id}`) bekommen keine Aktion; das wäre ein naheliegender Folgeschritt.
- Die 200er-Kappung der Spieleliste im Kalender wird hier nicht behoben (siehe Risiken); der Fokus umgeht sie nur.

## Decisions

**1. Platzierung als Icon-Aktion, nicht als Text-Button oder Menüeintrag.**
- `/termine`: `CalendarDays`-Icon-Button rechts in der Kopfzeile der Karte, hinter den Zusage-Zählern. Die ganze Karte ist bereits klickbar (Detailseite); der Button stoppt die Propagation wie heute schon `MapsLink`.
- `/dienste`: derselbe Icon-Button im Block-Kopf rechts neben den Team-Namen, nur wenn `game_id` gesetzt ist.
- Alternativen: Eintrag in einem `ActionMenu` (`MoreVertical`) — auf den Karten gibt es kein Menü, eins nur dafür einzuführen kostet einen Klick mehr. Ein Text-Button „Im Kalender öffnen“ wäre auf Mobile zu breit für die ohnehin umbrechende Kopfzeile. Das Icon ist in der App bereits mit „Kalender“ konnotiert.
- Touch-Target: `p-2` um ein `w-4 h-4`-Icon, `aria-label` und `title` „Im Kalender öffnen“.

**2. Ziel-URL trägt `date` und `focus`.** `date` bestimmt den Monat ohne Zusatzabfrage und nutzt den bestehenden Parameter; `focus` benennt den Termin. Ein Helfer `calendarLink(kind, id, date)` in `web/src/lib/` baut die URL (Datum per `.slice(0, 10)`, SQLite-DATE-Gotcha) und wird von beiden Seiten genutzt.

**3. Termin auflösen: erst geladene Liste, dann Einzelabruf.** Nach dem Laden sucht der Kalender den Termin in `games` bzw. `trainings`. Fehlt er dort (200er-Kappung, Monatsgrenze), holt er ihn über `GET /api/games/{id}` bzw. `GET /api/training-sessions/{id}`. Diese Routen prüfen die Sichtbarkeit am Objekt; 403/404 → Hinweis „Dieser Termin ist nicht verfügbar“. Alternative „nur die geladene Liste“ wäre bei großen Saisons unzuverlässig und genau dann still kaputt, wenn der Termin spät in der Saison liegt.
- Die Antworten der Einzelrouten müssen auf die Form gebracht werden, die `EventInfoModal` erwartet (Teams, Zähler, Treffzeit, Venue). Wo Felder fehlen, wird der Dialog mit dem vorhandenen Stand geöffnet; das wird beim Implementieren gegen die echten Antworten geprüft (Task 3.3).

**4. Filter werden nicht angefasst.** Der Dialog ist datengetrieben (`infoItem`), nicht an eine sichtbare Kachel gebunden, und öffnet deshalb auch, wenn ein Filter die Kachel ausblendet. Die Hervorhebung entfällt dann einfach. Anders als bei `/termine` müssen keine Filter ausgesetzt werden, weil `focus` sofort wieder aus der URL verschwindet.

**5. `focus` wird nach dem Öffnen per `setSearchParams(..., { replace: true })` entfernt.** So bleibt die URL teilbar auf Monatsebene, und weder Neuladen noch Zurück öffnen den Dialog erneut. Ein Ref schützt davor, dass der Effekt bei späteren Reloads (SSE `games`) erneut greift.

**6. Hervorhebung:** Die Termin-Kachel im Gitter bekommt für ca. 2 s einen `ring-2 ring-brand-yellow` (gleiche Optik wie der Fokus auf `/termine`). Die Kacheln brauchen dafür eine stabile ID (`kalender-game-<id>`, `kalender-training-<id>`); scrollen ist im Monatsgitter nicht nötig.

## Risks / Trade-offs

- [Spieleliste im Kalender gekappt auf 200] → Der Fokus umgeht das über den Einzelabruf. Das eigentliche Problem (fehlende späte Spiele im Gitter) bleibt; als Folgebefund festhalten, nicht hier lösen.
- [Einzelabruf liefert andere Feldform als die Liste] → Mapping im Kalender, Test mit realer Antwortform (Task 3.3). Beim Implementieren festgestellt: `GET /api/games/{id}` trägt keine Dienst-Zähler (`slot_count` & Co.); ein so nachgeladenes Spiel zeigt „In Diensten öffnen“ gesperrt. Betrifft nur Spiele jenseits der 200er-Kappung und verschwindet mit deren Behebung.
- [Dialog öffnet über einem noch ladenden Monat] → Fokus erst auswerten, wenn Spiele und Trainings geladen sind.
- [Icon allein ist nicht selbsterklärend] → `title`-Tooltip auf Desktop, `aria-label` für Screenreader; dasselbe Muster wie andere Icon-Aktionen der App.

## Migration Plan

Reines Frontend-Feature, kein Datenmodell. Deploy mit dem nächsten Release; Rollback durch Zurücksetzen des Commits.
