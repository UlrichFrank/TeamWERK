## 1. Backend — Audienz-Bypass für Vorstand

- [x] 1.1 In `internal/duties/handler.go` (`Board`-Funktion, ca. Z. 352) den `whereParts`-Branch erweitern: Bypass auch bei `claims.HasFunction("vorstand")`, nicht nur bei `claims.Role == "admin"`. Sicherstellen, dass die Bedingung idempotent ist und keine Doppel-WHERE-Klauseln erzeugt.
- [x] 1.2 Test ergänzen: `TestBoard_VorstandSeesAllTeams` — Nutzer mit Vereinsfunktion `vorstand` (System-Rolle `standard`) sieht Slots eines Teams, in dem er kein Mitglied ist.

## 2. Backend — Response-Felder ergänzen

- [x] 2.1 In `boardGroup`-Struct (`internal/duties/handler.go`, ca. Z. 462) Feld `TeamID *int \`json:"team_id,omitempty"\`` ergänzen.
- [x] 2.2 In der Gruppen-Initialisierungslogik (ca. Z. 493) bei vorhandener `teamID > 0` das Feld befüllen. Game-Gruppen mit `ds.team_id IS NULL` bleiben mit `TeamID = nil`.
- [x] 2.3 Im else-Zweig der Gruppen-Initialisierung (game-lose Gruppen, ca. Z. 502) zusätzlich `g.EventType = "generisch"` setzen.
- [x] 2.4 Test ergänzen: `TestBoard_GroupContainsTeamID` — Antwort enthält für team-spezifische Gruppen ein numerisches `team_id`-Feld.
- [x] 2.5 Test ergänzen: `TestBoard_GameIDNullGroupHasGenericEventType` — game-lose Gruppe (z. B. Vereinsfest) kommt mit `event_type: "generisch"` zurück.

## 3. Frontend — Typdefinitionen und Daten-Hooks

- [x] 3.1 In `DutyPage.tsx` das `BoardGroup`-Interface um `team_id?: number | null` erweitern.
- [x] 3.2 Im Mount-Hook `/teams` parallel laden — `setTeams(...)` für `buildTeamShortNames()` und das Dropdown.
- [x] 3.3 `teamShortNames`-Map via `useMemo(() => buildTeamShortNames(teams), [teams])` ableiten.

## 4. Frontend — Filter-State per URL-Search-Params

- [x] 4.1 `useSearchParams()` einführen und `parseFilters(sp)` analog zu `TerminePage.tsx` implementieren — vier Params: `team`, `types`, `mine`, `past`.
- [x] 4.2 `ALL_TYPES = new Set(['heim', 'auswärts', 'generisch'])` — **kein** `training` (es gibt keine Trainings-Dienste).
- [x] 4.3 Default-State (alle Pills aktiv, kein Team, kein Mine, kein Past) erzeugt keine Params in der URL.
- [x] 4.4 Bestehende lokale States (`viewMine`, `showPast`) durch URL-State ersetzen; Reload-Stabilität verifizieren.

## 5. Frontend — Header-Leiste umbauen

- [x] 5.1 Bestehende Header-Konstruktion (`<div className="flex items-center justify-between mb-4 ...">` mit Toggle und Text-Link) **entfernen**.
- [x] 5.2 Header analog `TerminePage.tsx` aufbauen: `<h1>` + `<select>` (Alle Teams + Team-Optionen) + drei Event-Typ-Pills (`Home`, `Plane`, `Calendar`) + Meine-Pill (`UserCheck`) + Vergangene-Pill (`History`).
- [x] 5.3 Pills nutzen `getEventColors(type).filter` als Active-Klasse. Inactive-Stil bleibt `bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text`.
- [x] 5.4 Meine- und Vergangene-Pill verwenden `bg-brand-yellow text-brand-black border-brand-yellow` im Active-State (gleich wie Termine-Page).
- [x] 5.5 `useCompactHeader(950)` einsetzen — bei `compact` Label ausblenden, Padding auf `px-2`.
- [x] 5.6 Gating `isAdminOrTrainer` für die Meine-Pill **entfernen** — Pill ist für alle sichtbar.

## 6. Frontend — Filterung im Client

- [x] 6.1 `visibleGroups = groups.filter(g => { if (!showPast && g.past) return false; if (!filterTypes.has(g.event_type ?? 'generisch')) return false; if (filterTeamId !== null && g.team_id !== filterTeamId) return false; return true })`.
- [x] 6.2 Wenn alle Event-Typ-Pills deaktiviert sind: leere Liste mit Hinweis.
- [x] 6.3 Wenn `mine=1` gesetzt ist, URL-Param an `/duty-board?view=mine` durchreichen (bestehendes Backend-Verhalten — kein neuer Endpoint).

## 7. Frontend — Karten farblich kodieren

- [x] 7.1 Card-Wrapper-Klassen anpassen: bei `g.past` weiterhin `bg-brand-surface-card border-brand-border opacity-60`; sonst `getEventColors(g.event_type ?? 'generisch').card.bg` + `.card.border`.
- [x] 7.2 Layout-Klassen `rounded-xl shadow border-t-4 overflow-hidden` bleiben fix; nur Farbe wird dynamisch.
- [x] 7.3 Card-Innenleben (Header mit Datum/Spiel/Team-Name, `DutySlotList`) bleibt unverändert.

## 8. Manuelles Testen und Cleanup

- [ ] 8.1 Backend lokal starten + Vite-Dev-Server. Mit drei Test-Usern verifizieren: (a) Admin, (b) Vorstand-Funktion, (c) Spieler ohne Funktion.
  - Admin und Vorstand sehen alle Teams im Dropdown und alle Dienste in der Liste.
  - Spieler sieht nur eigene Teams.
- [ ] 8.2 Filter-Kombinationen testen: einzelne Typ-Pills, Team-Wechsel, Meine-Pill, Vergangene-Pill, Kombinationen.
- [ ] 8.3 URL-Persistierung: Reload nach Filter-Wechsel erhält State; Default-State erzeugt saubere URL; Deep-Link `?team=3&types=heim&mine=1&past=1` lädt korrekt.
- [ ] 8.4 Mobile-Layout prüfen (< 950 px): Pills zeigen nur Icons; Karten weiterhin lesbar.
- [ ] 8.5 Visuell kontrollieren: Heimspiel-Card gelb, Auswärtsspiel-Card grau, generisches Event blau, vergangene Gruppen grau-opak.
- [ ] 8.6 Game-lose Gruppe (z. B. Vereinsfest) manuell anlegen und prüfen: erscheint blau gefärbt und wird vom „Sonstiges"-Filter eingeschlossen / vom Heim/Auswärts-Filter ausgeschlossen.
- [ ] 8.7 Toten Code löschen: `isAdminOrTrainer` (sofern nirgendwo sonst genutzt) bleibt für `canEdit` an `DutySlotList`; nur das Meine-Gating entfernen.
- [x] 8.8 CHANGELOG-Eintrag ergänzen in `web/public/CHANGELOG.md` — UI-Vereinheitlichung, neuer Team-Filter, Meine für alle, Vorstand-Vollzugriff.

## 9. Commit & Archivierung

- [ ] 9.1 Conventional Commits pro Task — Format `<type>(<scope>): <beschreibung>` (siehe CLAUDE.md).
- [ ] 9.2 Abschluss-Commit archiviert das Change-Proposal nach `openspec/changes/archive/`.
