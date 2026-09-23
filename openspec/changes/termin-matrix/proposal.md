## Why

`/termine` beantwortet heute die Frage „welche Termine stehen an und wie habe **ich**
geantwortet?". Die Frage des Trainers (und der Mannschaft) ist eine andere: „wer kommt
zu den nächsten Terminen, wer hat noch nicht geantwortet, wer fehlt regelmäßig?". Dafür
muss man jeden Termin einzeln öffnen. SpielerPlus löst das mit einer Statistik-Tabelle:
eine Zeile je Spieler, eine Spalte je Termin, in jeder Zelle ein Symbol für den
Rückmeldestatus, vorne eine Teilnahme-Quote.

Die Daten liegen vollständig vor (`training_responses`, `game_responses`, die
Rollen-Voreinstellungen `rsvp_default_players`/`rsvp_default_extended`,
`member_series_unavailabilities`, `*_attendances`). Es fehlt nur eine Abfrage, die sie
für **eine** Mannschaft und **viele** Termine auf einmal liefert, und eine Darstellung.

## What Changes

- **Neue Route** `GET /api/teams/{id}/rsvp-matrix?from=YYYY-MM-DD&to=YYYY-MM-DD`
  (Authenticated-Tier, Objektprüfung im Handler) im Package `internal/attendance`:
  liefert die Termine (Trainings und Spiele/Events) der Mannschaft im Zeitraum als
  Spalten und die Spieler des Kaders der **aktiven** Saison (Stamm- und erweiterter
  Kader) als Zeilen, je Zelle Rückmeldestatus, Voreinstellungs-Kennzeichen,
  Serien-Abmeldung und — nur für Trainer der Mannschaft / sportliche Leitung / Admin —
  die erfasste Anwesenheit.
- **`/termine` bekommt eine zweite Ansicht** „Tabelle" (Umschalter in der Kopfzeile,
  URL-Parameter `view=tabelle`). Die Liste bleibt Default und unverändert.
  - Mannschaftsauswahl als Einfachauswahl (`team=<id>`), weil die Matrix genau einen
    Kader als Zeilenmenge braucht.
  - Typ-Filter (Heim / Auswärts / Sonstiges / Training) und „Vergangene" wirken wie in
    der Liste — auf die Spalten.
  - Spalte „Teilnahme" je Spieler: Zusagen (bzw. erfasste Anwesenheit) / sichtbare,
    nicht abgesagte Termine, als `N (P %)` — rechnet über die **gefilterten** Spalten.
  - Spaltenkopf verlinkt auf die Termin-Detailseite.
- Keine Migration, keine Mutation (→ kein Broadcast nötig); die Ansicht abonniert
  `trainings`/`games` über `useLiveUpdates`.

## Nicht-Ziele

- **Übungsgruppen** (Kader ohne `teams`-Zeile) bekommen keine Matrix — die Route hängt
  an `teams.id`, dieselbe Grenze wie die Anwesenheits-Statistik. In der Tabellenansicht
  stehen sie nicht zur Auswahl.
- **Keine Absagegründe** in der Matrix. Die Sichtbarkeitsregeln der Gründe
  (`rsvp-reason-visibility`) bleiben der Detailseite vorbehalten.
- **Kein Antworten aus der Tabelle heraus.** Die Zelle ist Anzeige; RSVP geht weiter
  über Liste und Detailseite.
- **Trainer** erscheinen nicht als Zeile (sie sind bei Spielen per Definition zugesagt
  und haben keine Anwesenheitserfassung).

## Capabilities

### New Capabilities

- `termin-matrix`: Tabellarische Terminübersicht einer Mannschaft mit Rückmeldestatus
  je Spieler und Termin.

## Impact

- Backend: `internal/attendance/matrix.go` (+ Tests), eine Zeile in
  `internal/app/router.go`, Eintrag in der Objektrechte-Matrix
  (`internal/permissions/object_matrix_test.go`) und ggf. der Tier-Matrix.
- Frontend: `web/src/pages/TerminePage.tsx` (Umschalter, URL-Parameter),
  neue Komponente `web/src/components/TerminMatrix.tsx` (+ Vitest).

## Test-Anforderungen

| Route | Fall | Erwartung |
|---|---|---|
| `GET /api/teams/{id}/rsvp-matrix` | Spieler des Kaders, 1 Training + 1 Spiel im Zeitraum | 200; 2 Spalten chronologisch, Zeile je Kaderspieler, eigener Status korrekt |
| | Stammspieler ohne Antwort, Termin mit `rsvp_default_players='confirmed'` | Zelle `status='confirmed'`, `is_default=true` |
| | erweiterter Kader mit `rsvp_default_extended='none'` | Zelle `status=null` |
| | Serien-Abmeldung auf das Training | Zelle `unavailable=true` |
| | Spieler (kein Trainer) bei erfasster Anwesenheit | `present` fehlt im JSON |
| | Trainer der Mannschaft bei erfasster Anwesenheit | `present` gesetzt |
| | Nutzer ohne Bezug zur Mannschaft | 403 |
| | unbekannte Mannschaft | 404 |
| | `from` ungültig / `from > to` / Zeitraum > 400 Tage | 400 |
| | unauthentifiziert | 401 |
| | Spiel einer anderen Mannschaft im Zeitraum | erscheint nicht als Spalte |

Invariante: Die Matrix zeigt für jede Zelle denselben Status, den die Detailseite des
Termins (`/participants` bzw. `/attendances`) für dieses Mitglied zeigt — inklusive der
virtuell angewandten Rollen-Voreinstellung.
