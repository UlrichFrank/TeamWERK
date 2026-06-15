## Context

Die `DutyPage` (`web/src/pages/DutyPage.tsx`) zeigt heute Dienst-Gruppen chronologisch. Eine Gruppe entspricht entweder einem Spiel (mit n Slot-Zeilen für Schiri, Kasse, Hallendienst usw.) oder einer game-losen Sammelgruppe für „Sonstige Dienste" (Vereinsfest-Auf­bau, Mitgliederversammlung etc.). Das Backend (`GET /api/duty-board`) liefert bereits die Gruppen-Struktur und ist nach Datum sortiert.

Drei Schwächen heute:

1. Alle Karten haben einen uniform gelben Border-Top — der Spielcharakter ist nicht visuell ablesbar.
2. „Alle Dienste / Meine Dienste" ist ein binärer Segment-Toggle, nur für Trainer/Admin sichtbar. Es gibt weder Team- noch Typ-Filter.
3. Vergangene werden via Text-Link ein/ausgeblendet.

`TerminePage` (Trainings + Spiele + Events) und die kürzlich umgebaute `MitfahrgelegenheitenPage` haben dasselbe visuelle Pattern: `<h1>` + Team-Select + Event-Typ-Pills + (optional) Meine-Pill + Vergangene-Pill, dazu URL-Persistierung und Compact-Header. Das gleiche Pattern hier zu übernehmen ist die natürliche Wahl.

**Wiederverwendbare Bausteine** (existieren bereits, von TerminePage und MitfahrgelegenheitenPage genutzt):
- `web/src/lib/eventColors.ts` — `getEventColors(type)` liefert pro Event-Typ Klassen-Strings für Card und Filter-Pill
- `web/src/lib/teamName.ts` — `buildTeamShortNames(teams)` bildet kompakte Team-Kürzel
- `web/src/hooks/useCompactHeader.ts` — Viewport-Listener mit konfigurierbarer Schwelle

**Backend-Daten** liefert `GET /api/duty-board` heute pro Gruppe:
- `game_id`, `date`, `event_time`, `opponent`, `event_type` (für game-Gruppen), `team_name`, `label`, `past`, `slots[]`
- **Fehlt:** `team_id` (für den Frontend-Team-Filter zwingend nötig) und für game-lose Gruppen ein normalisierter `event_type`.

## Goals / Non-Goals

**Goals:**
- Konsistente visuelle Sprache zwischen `TerminePage`, `MitfahrgelegenheitenPage` und `DutyPage`.
- Schnelles Eingrenzen über Team und Event-Typ statt Scrollen.
- „Meine"-Filter für alle Rollen verfügbar (nicht nur Trainer/Admin).
- Filter-State per URL teilbar und reload-stabil.
- Compact-Header für Mobile ohne Layout-Bruch.
- Vorstand sieht denselben App-weiten Sichtbereich wie für Mitfahrgelegenheiten.

**Non-Goals:**
- Auflösen der inneren Gruppen-Struktur. Eine Karte = ein Spiel/Event mit n Slot-Zeilen — das Bündeln bleibt der semantische Mehrwert der Seite und wird nicht aufgegeben.
- Änderungen am Claim/Unclaim-Workflow, am Proxy-Account-Selektor, an `DutySlotList` oder an der Audience-Filterlogik (die bleibt unangetastet).
- Migration der bestehenden SQL-Subquery für Team-Zugriff auf `user_accessible_teams`. Carpooling nutzt die View, Duty-Board nutzt eine inline-Subquery — eine Konsolidierung ist sinnvoll, aber separat behandeln.
- Mobile-Card-Umbau (`MobileCard`) — die heutige Gruppenkarte funktioniert auf Mobile bereits.

## Decisions

### 1. Karten-Struktur bleibt: ein Spiel = eine Gruppe mit n Slot-Zeilen

Die heutige Anatomie

```
┌─ Sa 21.06. · 16:00 · Heim: TV Plochingen ────────── M1 ─┐
│  Schiedsrichter   [ Frei: 1 ]    [Ich übernehme]        │
│  Kasse            [ Frei: 0 ]    Ulrich F.              │
│  Hallendienst     [ Frei: 2 ]    [Ich übernehme]        │
└──────────────────────────────────────────────────────────┘
```

bleibt erhalten. „Chronologisch" bezieht sich auf die Reihenfolge der Gruppen — die liefert das Backend bereits.

**Alternative verworfen**: pro Slot eine eigene Karte. → Würde Kontext zerreißen und die Liste massiv verlängern. Hat keinen UX-Nutzen.

### 2. Filter-Bar-Aufbau (analog `TerminePage`)

```tsx
<h1>Dienste</h1>
<select> Alle Teams | M1 | wA1 | …  ← aus /teams (gefiltert wie heute)
<Pill type="heim">      ← getEventColors('heim').filter
<Pill type="auswärts">  ← getEventColors('auswärts').filter
<Pill type="generisch"> ← getEventColors('generisch').filter
<Pill type="mine">      ← UserCheck-Icon
<Pill type="past">      ← History-Icon
```

Pills toggeln je einen `Set<string>`-Eintrag für die Event-Typen. „Meine" und „Vergangene" sind eigenständige Booleans, werden aber visuell wie Pills gerendert.

### 3. Team-ID in Response — minimaler Backend-Eingriff

Im SQL-Scan wird `teamID` bereits gelesen (siehe `internal/duties/handler.go:478`), aber im JSON-Response-Struct `boardGroup` nicht ausgegeben. Hinzufügen:

```go
type boardGroup struct {
    GameID    *int    `json:"game_id"`
    TeamID    *int    `json:"team_id,omitempty"`   // NEU
    Date      string  `json:"date,omitempty"`
    EventTime string  `json:"event_time,omitempty"`
    Opponent  string  `json:"opponent,omitempty"`
    EventType string  `json:"event_type,omitempty"`
    TeamName  string  `json:"team_name"`
    Label     string  `json:"label,omitempty"`
    Past      bool    `json:"past"`
    Slots     []boardSlot `json:"slots"`
}
```

Im Group-Init: `if teamID > 0 { id := teamID; g.TeamID = &id }`. Für Spiel-Gruppen ohne expliziten `ds.team_id` (kommt selten vor, aber möglich) bleibt `TeamID = nil` — der Team-Filter ignoriert solche Gruppen dann; Frontend zeigt sie nur bei „Alle Teams".

### 4. Normalisierung des `event_type` für game-lose Gruppen

Heute setzt das Backend für `game_id IS NULL`-Gruppen nur `Label` (entweder `event_name` oder fallback „Sonstige Dienste"), `EventType` bleibt leerer String. Das Frontend kann diese Gruppen so nicht unter den „Sonstiges"-Filter einsortieren.

Fix: im else-Zweig der Group-Init zusätzlich `g.EventType = "generisch"` setzen. Das ist semantisch korrekt (es sind generische Vereinsdienste ohne Spiel-Bezug) und macht die Filter-Logik im Frontend symmetrisch.

### 5. Vorstand-Audienz (Backend) — Pattern aus Carpooling übernehmen

Carpooling-Handler unterscheidet:

```go
restricted := role != "admin" && role != "vorstand"
```

Im Duty-Board-Handler wird heute nur `claims.Role == "admin"` als Bypass abgefragt. Erweitern auf:

```go
if claims.Role == "admin" || claims.HasFunction("vorstand") {
    whereParts = `WHERE ds.season_id = (SELECT id FROM seasons WHERE is_active = 1)`
} else {
    // bisheriges team-Filter-SQL
}
```

`claims.HasFunction` existiert bereits (vgl. CLAUDE.md / `auth/middleware.go`). Damit ist die Logik:

| Rolle                                     | Sichtbarkeit                                   |
|-------------------------------------------|------------------------------------------------|
| System-Rolle `admin`                      | Alle Dienste aller Teams                       |
| Vereinsfunktion `vorstand`                | Alle Dienste aller Teams (**neu**)             |
| Trainer / Sportliche Leitung / Spieler / Eltern | Nur eigene Teams (player_memberships + family_links — wie heute) |

Die Audience-Filterung (`eltern`/`trainer`/Vereinsfunktionen) auf Slot-Ebene bleibt unverändert — das ist ein orthogonaler Filter und schon korrekt.

**Alternative verworfen**: Vorstand-Beisitzer ebenfalls bypasen. → Bewusst eng halten (Vorstand only), `vorstand_beisitzer` hat heute nicht denselben Vollzugriff in anderen Endpoints. Kann später erweitert werden.

### 6. „Meine"-Pill für alle Rollen

Heute: `isAdminOrTrainer` gatet den Toggle, weil für Spieler die Default-Ansicht ohnehin „meine Team-Dienste" zeigt. Die Semantik von „Meine" ändert sich aber: es soll „Slots, in denen ICH eingetragen bin" bedeuten — ein durchaus interessanter Quick-View auch für Spieler („was hab ich übernommen?").

Backend kennt diesen Filter bereits: `?view=mine` ergänzt eine `EXISTS`-Klausel auf `duty_assignments`. Keine Änderung nötig, nur das Frontend-Gating entfernen.

Konsistent mit Mitfahrgelegenheiten, wo „Meine" für alle sichtbar ist.

### 7. Vergangene-Pill statt Text-Link

Aus

```tsx
<button className="text-xs text-brand-text-muted hover:text-brand-blue">
  Vergangene einblenden
</button>
```

wird die History-Pill aus `TerminePage`. State `past: boolean` wandert in die URL als `?past=1`. Filter wirkt clientseitig (`groups.filter(g => past || !g.past)`) wie heute.

### 8. Card-Farbcodierung mit Past-Override

```tsx
const colors = getEventColors(g.event_type ?? 'generisch')
const cardClass = g.past
  ? 'bg-brand-surface-card border-brand-border opacity-60'
  : `${colors.card.bg} ${colors.card.border}`
return (
  <div className={`rounded-xl shadow border-t-4 overflow-hidden ${cardClass}`}>
    …
  </div>
)
```

Past schlägt Farbe. Das vermeidet leuchtende gelbe Karten für letzte Woche und bleibt konsistent zur Past-Logik von Termine (cancelled).

### 9. URL-Search-Params

Vier Parameter analog `TerminePage` + `MitfahrgelegenheitenPage`:
- `?team=<id>` (numerisch, optional)
- `?types=<heim,auswärts,generisch>` (CSV, Default = alle drei aktiv = kein Parameter)
- `?mine=1` (Default = nicht aktiv)
- `?past=1` (Default = nicht aktiv)

`parseFilters(sp)` und `updateFilter(patch)` werden aus `TerminePage` übernommen (durch Copy + Anpassung, keine Generalisierung in dieser Iteration).

### 10. Compact-Header

`useCompactHeader(950)` — gleiche Schwelle wie auf den anderen Seiten. Pills zeigen bei `compact === true` nur das Icon, Padding reduziert auf `px-2`.

### 11. Team-Filter-Datenquelle

Frontend lädt `/teams` für die Dropdown-Optionen + Kürzel-Map. `/teams` ist heute schon rollenabhängig gefiltert (Vorstand/Admin → alle Teams, andere → ihre Teams). Damit ergibt sich automatisch das gewünschte Verhalten: Vorstand sieht alle Teams im Dropdown, Spieler sieht nur die eigenen.

`buildTeamShortNames(teams)` liefert die Kurzbezeichnungen. Sortierung der Gruppen passiert weiterhin im Backend nach Datum — der Frontend-Filter nutzt die Kürzel-Map nur als Anzeige im Dropdown.

## Risks / Trade-offs

- **[Vorstand-Bypass erweitert Sichtbarkeit]** → Heute sieht ein Vorstand-Mitglied nur Dienste der Teams seiner Kinder/Kader. Künftig sieht er alle. Das ist gewünscht und konsistent mit Mitfahrgelegenheiten, aber bewusst zu kommunizieren. **Mitigation**: Im Changelog explizit erwähnen.

- **[`event_type=generisch` für game-lose Gruppen ist semantisch neu]** → Bisheriger leerer String wird gefüllt. Falls ein Frontend-Client (Mobile-App, externer Konsument) auf den leeren String prüft, bricht das. **Mitigation**: TeamWERK hat nur einen Web-Client; keine externen API-Konsumenten. Risiko sehr gering.

- **[`team_id` in Response für game-lose Gruppen]** → Bei game-losen Gruppen wird im Group-Key `team_id` mitverwendet (`other-{teamID}-{eventDate}`). Wenn ein game-loser Slot `team_id IS NULL` hat (möglich, z. B. vereinsweite Aktion), bleibt `TeamID = nil` und der Team-Filter im Frontend ignoriert die Gruppe (sie ist nur bei „Alle Teams" sichtbar). **Mitigation**: Das ist das richtige Verhalten — ein vereinsweiter Dienst gehört zu keinem Team.

- **[Visuelle Veränderung deutlich]** → Heutiges uniform-gelbes Layout weicht farbcodierten Karten. Konsistenz mit TerminePage/MitfahrgelegenheitenPage rechtfertigt das. **Mitigation**: Eintrag im CHANGELOG.

- **[5 Pills nebeneinander auf Mobile]** → Heim, Auswärts, Sonstiges, Meine, Vergangene + Team-Select + `<h1>` ist viel für 360 px. **Mitigation**: `useCompactHeader(950)` zeigt nur Icons; bei sehr schmalen Viewports kann der Header umbrechen (`flex-wrap`).

## Migration Plan

1. Backend: `Board`-Handler in `internal/duties/handler.go` anpassen:
   - Bypass-Check um `claims.HasFunction("vorstand")` erweitern.
   - `boardGroup.TeamID *int` ergänzen, beim Init befüllen.
   - Für game-lose Gruppen `EventType = "generisch"` setzen.
2. Backend-Tests ergänzen (`TestBoard_VorstandSeesAllTeams`, `TestBoard_GameIDNullGroupHasGenericEventType`, `TestBoard_GroupContainsTeamID`).
3. Frontend: `DutyPage.tsx` umbauen — `BoardGroup`-Interface ergänzen, Filter-Bar nach `TerminePage`-Vorbild bauen, Card-Farben via `getEventColors`, URL-Persistierung verdrahten.
4. Toten Code löschen: `isAdminOrTrainer`-Gating für den Meine-Toggle, alter Vergangene-Link.
5. Manuell testen (Vorstand, Spieler, Eltern, Trainer) und auf Mobile/Desktop verifizieren.

**Rollback**: Reines Frontend-Revert + Backend-Revert. Die zusätzlichen Response-Felder (`team_id`, `event_type=generisch`) sind additiv und brechen keine alten Clients.

## Open Questions

- **`vorstand_beisitzer` ebenfalls bypasen?** Vorerst nein — bleibt konsistent zu Carpooling. Falls Beisitzer im operativen Alltag denselben Bedarf haben, separater Change.
- **`user_accessible_teams`-View statt Inline-Subquery?** Wäre eine Konsolidierung. Im Scope dieses Changes nicht zwingend. Notiz im Repo, falls später angefasst.
