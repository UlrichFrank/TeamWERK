## Why

Das Dashboard zeigt Eltern heute nur ihr eigenes, aggregiertes Dienstkonto (`soll`
geschätzt aus `games_per_season × avg_slots_per_game`, `ist` als reine Zahl ohne
Kontext) — nicht aufgeschlüsselt nach Kind, nicht vergleichbar mit anderen Familien
desselben Kaders. Wer wissen will, ob die eigene Familie im Verhältnis zu anderen
ungefähr gleich viel leistet, hat dafür keine Fläche; nur der Vorstand kann das
(mühsam, über CSV-Export) einschätzen.

Gleichzeitig ist die bestehende `soll`-Formel (`dienstkonto-dynamische-soll-formel`)
eine **Schätzung** aus `games_per_season` — einer am Kader gepflegten Zahl, die von
der tatsächlichen Terminlage abweichen kann. Die Slots der bereits bekannten Termine
der Saison (Spiele, Trainings-Dienste, Vereinsfeste) sind aber längst in der DB und
lassen sich direkt zählen, statt geschätzt zu werden.

Dieser Change macht die geleisteten Dienste pro Kind sichtbar (als Balken statt
Zahl), ersetzt die Schätz-Formel durch eine aus echten Slots hergeleitete
Gesamtsumme, und gibt Familien wie Vorstand eine Rangliste, um Schieflagen selbst zu
erkennen statt sie nur zu vermuten.

## What Changes

- **Neue Zähllogik pro Kind** (aktive Saison, live berechnet, keine neue Tabelle):
  - `Geleistet` = eingetragene `duty_assignment` mit `event_date < heute`
  - `Vorhersage` = eingetragene `duty_assignment` mit `event_date >= heute`
  - `Gesamtsumme(Kader)` = team-gebundene `duty_slots` (über `game_id`→`game_teams`
    oder direkte `team_id`) + anteiliger Anteil generischer `duty_slots` (weder
    `game_id` noch `team_id`, z.B. Vereinsfest), proportional zur Spieleranzahl je
    Kader verteilt
  - `Fair-Anteil(Kind)` = `Gesamtsumme(Kader) / Anzahl Spieler im Kader` — ohne
    Geschwister-Deduplizierung, jedes Kind zieht seinen eigenen Anteil
- **`dienstkonto-dynamische-soll-formel` wird abgelöst**: Die bisherige Schätzung aus
  `games_per_season × avg_slots_per_game / player_count / parent_count` weicht der
  echten `Gesamtsumme`. Das Dashboard zeigt danach eine Zeile **pro Kind** statt
  einen aggregierten Wert pro Elternteil.
- **Dashboard-Kachel wird grafisch**: pro Kind ein Segment-Balken (Geleistet /
  Vorhersage / Rest bis Fair-Anteil), verlinkt auf die neue Rangliste-Seite.
- **BREAKING**: `GET /api/dashboard` liefert `dutyAccount` künftig als Liste (eine
  Position pro Kind) statt eines einzelnen aggregierten Objekts.
- **Neue Seite „Rangliste"**: Balkendiagramm, eine Zeile pro Kind, absteigend nach
  Fortschritt sortiert, mit dem bestehenden `TeamFilter`-Baustein filterbar
  (mehrere Teams gleichzeitig → ein Ranglisten-Block pro Team). Sichtbarkeit
  rollenabhängig:
  - Standard-Nutzer (Eltern/Spieler): nur eigene Teams zur Auswahl (eigene Kinder
    via `family_links` oder eigenes Kader als Spieler); eigene Zeile mit echtem
    Namen, alle anderen Zeilen nur mit Platzierung (kein erfundenes Pseudonym).
  - Vorstand (Rolle `admin` oder Vereinsfunktion `vorstand`): alle Teams wählbar,
    alle Zeilen mit echtem Namen.

## Capabilities

### New Capabilities

- `dienste-familien-rangliste`: Zähllogik (Geleistet/Vorhersage/Gesamtsumme/
  Fair-Anteil je Kind), die Rangliste-Seite mit rollenabhängiger Sichtbarkeit und
  Team-Filter-Scope.

### Modified Capabilities

- `dienstkonto-dynamische-soll-formel`: Die `soll`-Berechnung wechselt von der
  spielanzahl-basierten Schätzformel auf die tatsächliche Gesamtsumme bekannter
  Dienst-Slots; das Dashboard zeigt eine Zeile pro Kind statt einen aggregierten
  Wert, als Segment-Balken statt reinem Text.

## Impact

- `internal/dashboard/handler.go` — `queryDutyAccount` liefert eine Liste pro Kind;
  `computeSollForElternteil`/`computeAvgSlotsPerGame` werden durch die
  Gesamtsumme-Berechnung ersetzt
- `internal/duties/` (oder `internal/dashboard/`) — neuer Handler/Route für die
  Rangliste (Gesamtsumme/Fair-Anteil je Kader, Sichtbarkeits-/Anonymisierungslogik)
- Team-Scope-Query für Standard-Nutzer („eigene Teams") — `GET /teams` liefert heute
  vermutlich ungefiltert alle Teams; braucht eine rollenabhängige Einschränkung
- `web/src/pages/DashboardPage.tsx` — Segment-Balken pro Kind statt Text-Kachel
- `web/src/pages/` — neue Seite für die Rangliste, `App.tsx`-Route,
  `AppShell.tsx`-Nav-Eintrag
- Wiederverwendung: `web/src/components/TeamFilter.tsx` + `web/src/lib/teamFilter.ts`
- `openspec/specs/dienstkonto-dynamische-soll-formel/spec.md` — Delta
- Keine Migration, keine neue Tabelle — alles live berechnet; keine Berührung von
  `duty_accounts` (dessen bekannter Bug bleibt Gegenstand des separaten, offenen
  Change `dienstkonto-ist-buchung`)
