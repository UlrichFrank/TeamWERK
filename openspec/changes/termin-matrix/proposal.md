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
  - Teilnahme je Spieler in zwei Spalten, jeweils über die **gefilterten** Spalten:
    **„Bisher"** (vergangene Termine — erfasste Anwesenheit, wo vorhanden, sonst Zusage)
    und **„Geplant"** (heute und künftig — Zusagen), jeweils als `N (P %)`.
  - **Zu-/Absagen aus der Tabelle** mit derselben Semantik wie in der Liste: für die
    eigene Zeile und die Zeilen der eigenen Kinder, gesperrt ab Rückmeldefrist (außer
    Trainer/Vorstand/Admin), bei Abwesenheits-Sperre und Serien-Abmeldung; Begründungs-
    Dialog bei Vielleicht/Absage, wenn der Termin eine Begründung verlangt.
  - Spaltenkopf verlinkt auf die Termin-Detailseite.
- Keine Migration. Die Route selbst mutiert nicht; Zu-/Absagen laufen über die
  bestehenden `POST …/respond`-Routen (die broadcasten bereits). Die Ansicht abonniert
  `trainings`/`games` über `useLiveUpdates`.
- Die RSVP-Sperrfristen (Training 2 h, Spiel 18 h vor Beginn) wandern als Konstanten nach
  `internal/policy`, damit `attendance` sie ohne Import einer anderen Domäne kennt.

## Nicht-Ziele

- **Übungsgruppen** (Kader ohne `teams`-Zeile) bekommen keine Matrix — die Route hängt
  an `teams.id`, dieselbe Grenze wie die Anwesenheits-Statistik. In der Tabellenansicht
  stehen sie nicht zur Auswahl.
- **Keine fremden Absagegründe** in der Matrix. Nur Zeilen, für die der Aufrufer
  antworten darf (eigene, Kinder), tragen ihren Grund — dieselbe Teilmenge, die
  `rsvp-reason-visibility` Spielern und Eltern zeigt.
- **Kein Antworten für Dritte.** Trainer/Vorstand dürfen serverseitig zwar für jedes
  Kadermitglied antworten, die Liste bietet das aber nicht an — die Tabelle auch nicht.
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
| | Spieler: eigene Zeile `is_self`/`can_respond`, fremde Zeile nicht; eigener Grund sichtbar, fremder nicht | wie beschrieben |
| | Elternteil: Zeile des Kindes `can_respond` | `can_respond=true` |
| | Antwort mit `absence_id` | Zelle `locked=true` |
| | Spalte trägt `rsvp_locks_at` (Training −2 h, Spiel −18 h) und `rsvp_require_reason` | wie beschrieben |
| | Nutzer ohne Bezug zur Mannschaft | 403 |
| | unbekannte Mannschaft | 404 |
| | `from` ungültig / `from > to` / Zeitraum > 400 Tage | 400 |
| | unauthentifiziert | 401 |
| | Spiel einer anderen Mannschaft im Zeitraum | erscheint nicht als Spalte |

Invariante: Die Matrix zeigt für jede Zelle denselben Status, den die Detailseite des
Termins (`/participants` bzw. `/attendances`) für dieses Mitglied zeigt — inklusive der
virtuell angewandten Rollen-Voreinstellung.
