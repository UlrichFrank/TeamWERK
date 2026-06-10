## MODIFIED Requirements

### Requirement: EventInfoModal zeigt schreibgeschützte Details eines Kalendereintrags

Das `EventInfoModal` SHALL Spieltag- oder Trainingsdaten schreibgeschützt anzeigen. Es enthält keine Bearbeiten-Buttons. Es verwendet das Standard-Modal-Design (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`).

#### Scenario: Modal für Heimspiel

- **WHEN** das `EventInfoModal` für ein Heimspiel geöffnet wird
- **THEN** zeigt es: Event-Typ-Icon, "Gegner"-Label mit Gegnername, Datum, Uhrzeit, Team(s) als Kurznamen, RSVP-Zahlen

#### Scenario: Modal für Auswärtsspiel

- **WHEN** das `EventInfoModal` für ein Auswärtsspiel geöffnet wird
- **THEN** zeigt es: Event-Typ-Icon, "Gegner"-Label mit Gegnername, Datum, Uhrzeit, Team(s) als Kurznamen, RSVP-Zahlen

#### Scenario: Modal für generisches Event ohne Enddatum

- **WHEN** das `EventInfoModal` für ein generisches Event ohne `end_date` geöffnet wird
- **THEN** zeigt es: Calendar-Icon, "Event-Name"-Label (nicht "Gegner") mit dem Event-Namen, Datum (nur Startdatum), Uhrzeit, Team(s) als Kurznamen, RSVP-Zahlen

#### Scenario: Modal für mehrtägiges generisches Event

- **WHEN** das `EventInfoModal` für ein generisches Event mit `end_date` geöffnet wird, das vom Startdatum abweicht
- **THEN** zeigt es unter "Datum" eine Datumsrange (z.B. "7. September – 10. September 2026") statt nur dem Startdatum
- **THEN** wird das Label "Event-Name" statt "Gegner" verwendet

#### Scenario: Modal für Training

- **WHEN** das `EventInfoModal` für ein Training geöffnet wird
- **THEN** zeigt es: Training-Icon (Dumbbell), Titel, Datum, Startzeit–Endzeit, Ort, Team-Kurzname, RSVP-Zahlen

#### Scenario: Keine Teams vorhanden

- **WHEN** das `EventInfoModal` geöffnet wird und keine Teams-Daten übergeben wurden (leeres Array oder undefined)
- **THEN** entfällt die Team-Zeile stillschweigend (kein leeres Feld)

## ADDED Requirements

### Requirement: EventInfoModal akzeptiert Team-Metadaten als Props

Das `EventInfoModal` SHALL folgende erweiterte Props akzeptieren:
- `Game.teams?: Array<{ id: number; name: string }>` — vorberechnete Kurznamen der beteiligten Teams
- `Game.end_date?: string | null` — Enddatum für mehrtägige generische Events
- `Training.team_name?: string` — Kurzname des zugehörigen Teams

#### Scenario: KalenderPage übergibt Teams mit Kurznamen

- **WHEN** `KalenderPage` das `EventInfoModal` für ein Spiel öffnet
- **THEN** werden `game.teams` mit Kurznamen (aus der `shortNames`-Map) als Props übergeben

#### Scenario: KalenderPage übergibt team_name für Training

- **WHEN** `KalenderPage` das `EventInfoModal` für ein Training öffnet
- **THEN** wird der vorberechnete Kurzname des Teams als `training.team_name` übergeben
