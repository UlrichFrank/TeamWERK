## 1. Backend — Team-ID in Response ergänzen

- [x] 1.1 In `internal/mitfahrgelegenheiten/handler.go` (oder dem äquivalenten Handler) das Response-Game-Objekt um `team_id: int` ergänzen. Bei generischen Multi-Team-Events `team_ids: []int` mitliefern.
- [x] 1.2 SQL-Query um `team_id` erweitern (JOIN auf `games.team_id` bzw. relevante Junction-Tabelle für generische Events).
- [x] 1.3 Test ergänzen: `GET /api/mitfahrgelegenheiten` liefert für ein Heimspiel ein `team_id`-Feld; für ein generisches Multi-Team-Event ein `team_ids`-Array.

## 2. Frontend — Typdefinitionen und Daten-Hooks

- [x] 2.1 In `MitfahrgelegenheitenPage.tsx` das `GameCarpoolData.game`-Interface um `teamId: number` und (für generische Events) `teamIds?: number[]` erweitern.
- [x] 2.2 Im Mount-Hook zusätzlich `/teams` parallel zu `/teams/my` laden — Ergebnis als `teams: Team[]` für `buildTeamShortNames()` speichern.
- [x] 2.3 `teamShortNames`-Map via `useMemo(() => buildTeamShortNames(teams), [teams])` ableiten.

## 3. Frontend — Filter-State per URL-Search-Params

- [x] 3.1 `useSearchParams()` einführen und `parseFilters(sp)` analog zu `TerminePage.tsx` implementieren — drei Params: `team`, `types`, `mine`.
- [x] 3.2 Default-State (alle Pills aktiv, kein Team, kein Mine) erzeugt **keine** Params in der URL — `updateFilter()` löscht den jeweiligen Schlüssel.
- [x] 3.3 Bestehende lokale States (`activeTab`, `viewMine`, `filterTeamId`) durch URL-State ersetzen; Re-Render bei Param-Wechsel.

## 4. Frontend — Header-Leiste umbauen

- [x] 4.1 Bestehende Tab-Leiste (`<div className="flex gap-1 mb-6 border-b ...">`) **entfernen**.
- [x] 4.2 Header analog `TerminePage.tsx` aufbauen: `<h1>` + `<select>` (Teams) + drei Event-Typ-Pills (`Home`, `Plane`, `Calendar`-Icon) + eine "Meine"-Pill (`UserCheck`-Icon).
- [x] 4.3 Pills nutzen `getEventColors(type).filter` als Active-Klasse. Inactive-Stil bleibt `bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text`.
- [x] 4.4 "Meine"-Pill verwendet `bg-brand-yellow text-brand-black border-brand-yellow` im Active-State (gleicher Stil wie Typ-Pills im Active).
- [x] 4.5 `useCompactHeader(950)` einsetzen — bei `compact` Labels in `<span>` ausblenden und Padding auf `px-2` reduzieren.

## 5. Frontend — Sortierung und Filterung

- [x] 5.1 `sortKey(d)` implementieren: `date + 'T' + time + '|' + teamKey`, wobei `teamKey = teamShortNames.get(d.game.teamId) ?? d.game.team` (Fallback Long-Name).
- [x] 5.2 Bei generischen Multi-Team-Events: alphabetisch kleinsten Team-Kürzel aus `teamIds` als Sortierschlüssel verwenden.
- [x] 5.3 `visibleGames = response.games.filter(d => filterTypes.has(d.game.eventType) && teamFilter(d) && mineFilter(d)).sort((a,b) => sortKey(a).localeCompare(sortKey(b)))`.
- [x] 5.4 Tab-Filter-Logik (`tabGames`, `countForTab`, `TAB_LABELS`, `EventTab`-Typ) **entfernen**.

## 6. Frontend — GameCard farblich kodieren

- [x] 6.1 `GameCard`-Wrapper-Klassen umstellen: `bg-brand-surface-card ... border-brand-yellow` durch `getEventColors(data.game.eventType).card.bg` und `getEventColors(...).card.border` ersetzen.
- [x] 6.2 Sicherstellen, dass die Card weiterhin `rounded-xl shadow border-t-4 overflow-hidden` Layout-Klassen behält — nur Farbe wird dynamisch.
- [x] 6.3 Card-Innenleben (Header mit Datum/Titel, Biete/Suche-Spalten, In-Card-Tabs, Paarungen, Buttons) bleibt unverändert.

## 7. Manuelles Testen und Cleanup

- [ ] 7.1 Vite-Dev-Server starten (`cd web && pnpm dev`). Mitfahrgelegenheiten-Seite öffnen, Filter-Kombinationen testen: alle Pills aktiv, einzelne Pills, Team-Wechsel, Meine-Pill.
- [ ] 7.2 URL-Persistierung verifizieren: Reload nach Filter-Wechsel erhält den State; saubere URL bei Default-State; Deep-Link `?team=X&types=heim&mine=1` lädt korrekt.
- [ ] 7.3 Mobile-Layout prüfen (Viewport < 950 px): Pills zeigen nur Icons; bei < 640 px funktioniert In-Card-Tab-Layout unverändert.
- [ ] 7.4 Visuell kontrollieren: Heimspiel-Card gelb, Auswärtsspiel-Card grau, generisches Event blau.
- [ ] 7.5 Generisches Multi-Team-Event manuell anlegen und prüfen: erscheint genau einmal, sortiert nach kleinstem Team-Kürzel.
- [x] 7.6 Toten Code löschen: alte `EventTab`-Typdefinition, `TAB_LABELS`, `activeTab`/`setActiveTab`-States — wenn nicht mehr referenziert.
