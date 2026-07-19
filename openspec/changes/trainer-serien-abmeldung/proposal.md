## Why

Manche Spieler können an einer wiederkehrenden Trainingsserie dauerhaft (oder für einen längeren Zeitraum) nicht teilnehmen — z. B. weil sie fest in der A-Jugend mittrainieren, Berufsschule haben oder langfristig verletzt sind. Heute erscheinen diese Spieler in `/termine` als „keine Rückmeldung" und verzerren die Anwesenheitsstatistik ihres Teams, obwohl sie nie erwartet werden. Es gibt keinen Weg für den Trainer, das sauber zu erfassen — die bestehende `member-absences`-Capability wird vom Spieler/Elternteil selbst gepflegt und zählt als „entschuldigt" (bleibt also im Nenner).

## What Changes

- **Neue Trainer-gepflegte Serien-Abmeldung**: Ein Trainer des eigenen Teams (oder `sportliche_leitung`/Admin) kann einen Spieler für eine Trainingsserie als dauerhaft abgemeldet markieren — mit optionalem Enddatum (permanent, wenn keins) und optionalem Grund. Neue Tabelle `member_series_unavailabilities` (an `training_series` + `members` gehängt, `team_id` via Serie ableitbar).
- **Team-scoped, nicht member-global**: Die Abmeldung gilt nur für die betroffene Serie/das Team — nicht für andere Kader desselben Spielers. Nur Trainer des eigenen Teams dürfen eintragen; der Spieler selbst kann nichts tun.
- **RSVP + Anwesenheitserfassung gesperrt**: Für Sessions, die eine greifende Abmeldung haben, werden `POST .../response` (Spieler) und die Trainer-Anwesenheitserfassung serverseitig mit HTTP 403 abgelehnt (Prüfung live, keine vorab angelegten Response-Rows).
- **Statistik-Ausschluss (Lesart „nicht erwartet")**: Betroffene Session×Spieler fallen komplett aus dem Nenner — weder anwesend noch fehlt noch entschuldigt. In der Mitglieds-Detailliste erscheinen sie mit neuer Kategorie `unavailable`. Team-Aggregat: Ø über Pro-Spieler-Quoten (unterschiedliche Nenner) + Frontend-Fußnote.
- **Sichtbarkeit**: In `/termine` bleibt der Spieler in der Anwesenheitsliste sichtbar (Variante A), mit Badge „dauerhaft abgemeldet" + Grund; nur der Trainer sieht die Lösch-Aktion. Pflege der Abmeldungen über die Serien-Bearbeitung und aus dem Termin-Detail heraus.
- Neue Migration für `member_series_unavailabilities`; neue Routen unter dem Trainer-Tier; Broadcast `training-unavailability-changed` (global).

## Capabilities

### New Capabilities
- `serien-abmeldung`: Trainer-gepflegte, serien-gebundene Dauer-Abmeldung eines Spielers (Datenmodell, CRUD-Routen, Autorisierung auf das eigene Team, Broadcast). Definiert die Ableitung „greift eine Abmeldung für Session×Member?" als Single Source of Truth für die abhängigen Capabilities.

### Modified Capabilities
- `attendance-statistics`: Neue Kategorie `unavailable` in der Termin-Liste; Session×Member mit greifender Abmeldung fällt aus allen drei Säulen/dem Nenner; Team-Aggregat dokumentiert die Pro-Spieler-Nenner-Semantik.
- `training-attendance`: Trainer-Anwesenheitserfassung für einen abgemeldeten Spieler auf einer betroffenen Session wird mit HTTP 403 abgelehnt.
- `training-rsvp`: Spieler-/Eltern-RSVP auf einer betroffenen Session wird mit HTTP 403 abgelehnt.
- `training-sessions`: Das Session-Listing/Detail liefert pro Mitglied den Abmelde-Status (`unavailable_reason`, `permanent`) für die Anzeige in `/termine`.

## Impact

- **Backend**: neue Migration `internal/db/migrations/`; neues bzw. erweitertes Domain-Package (Handler + Routen in `internal/app/router.go` unter dem Trainer/sportliche_leitung-Tier); Anpassung der Attendance-Statistik-Query und der RSVP-/Attendance-Handler; Broadcast-Aufruf (Broadcast-Gate).
- **Frontend**: Serien-Bearbeitung (Abschnitt „Dauerhaft abgemeldete Spieler"), Termin-Detail (`/termine`, Badge + Trainer-Aktion), Anwesenheits-Sichten (neue Kategorie + Fußnote), neue API-Calls, `useLiveUpdates` auf `training-unavailability-changed`.
- **Keine** neuen externen Dienste, kein zusätzlicher RAM-relevanter Footprint (ein Index-Lookup pro Session×Member).
- Kein Konflikt mit `member-absences` — orthogonal, dürfen sich überlappen; bei Kollision gewinnt die Serien-Abmeldung für die Statistik (Nenner-Ausschluss > entschuldigt).
