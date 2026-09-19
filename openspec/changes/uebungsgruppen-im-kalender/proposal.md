## Why

Übungsgruppen (`kader.kind='practice'`, ohne `teams`-Zwilling) sind seit der Capability
`uebungsgruppen` ein vollwertiger Terminträger: Trainingstermine, RSVP, Anwesenheit. An zwei
Stellen kommen ihre Termine bei den Betroffenen trotzdem nicht an:

1. **Der iCal-Feed kennt sie strukturell nicht.** `fetchTrainings` löst den Kader über
   `JOIN teams t ON t.id = ts.team_id` auf — bei einer Übungsgruppe ist `team_id` NULL, der
   JOIN matcht nicht, der Termin fehlt. Das war beim Bau der Übungsgruppen eine bewusste,
   dokumentierte Auslassung (`TestIcalFeed_OhneUebungsgruppe`); wer sein Fördertraining im
   Handy-Kalender haben will, hat heute keinen Weg dorthin.

2. **`/termine` schneidet Termine jenseits des Saisonendes ab.** Die Seite lädt bis
   `seasons.end_date` der aktiven Saison. Eine Trainingsserie darf aber über dieses Datum
   hinausreichen — und tut es in der Praxis genau bei den Übungsgruppen (Stand heute in der
   Prod-DB: die Serien „Förderkinder 2016/2017" laufen bis 03.07.2027, die aktive Saison
   endet am 30.06.2027). Diese Termine sind über die API sichtbar, werden vom Client aber
   nie angefragt. Betroffene wie Neea Manz sehen 15 von 16 Terminen ihrer Übungsgruppe —
   ohne Hinweis, dass etwas fehlt.

## What Changes

- **Neuer Feed-Toggle `include_practice_groups`** (Migration `067`, Default `1`) neben den
  fünf bestehenden. Er steuert ausschließlich Übungsgruppen-Termine; `include_training`
  behält seine bisherige Bedeutung „Mannschaftstrainings".
- **`fetchTrainings` ankert am Kader statt am Team**: `JOIN kader k ON k.id = ts.kader_id`
  (statt `team_id` + `season_id`) und `LEFT JOIN teams`. Für Mannschaftstermine ist das
  äquivalent — `ts.kader_id` trägt Team und Saison bereits —, für Übungsgruppen ist es der
  einzige Weg. Beschriftung fällt bei fehlendem Team auf `kader.name` zurück
  (`SUMMARY:Training: Förderkinder 2016`).
- **`/profil` → Kalender-Abo** bekommt den sechsten Schalter („Übungsgruppen"). Gilt über
  dieselbe Komponente auch für die Kind-Tokens auf der Kind-Profilseite.
- **`/termine` lädt bis mindestens 365 Tage in die Zukunft**, auch wenn die aktive Saison
  früher endet. Die Saisongrenze bleibt die *untere* Schranke des Fensters, nicht die obere.
- **Tests**: `TestIcalFeed_OhneUebungsgruppe` wird zu `TestIcalFeed_MitUebungsgruppe`
  umgedreht (der Test hat genau diesen Moment abgesichert), plus Toggle-aus-Fall,
  Beschriftungs-Fall und ein Vitest für die Fensterberechnung.

## Capabilities

### Modified Capabilities

- `ical-feed`: Übungsgruppen-Termine sind Feed-Inhalt, gesteuert über einen eigenen Toggle.
- `termine-unified-view`: das Ladefenster der Terminliste ist spezifiziert und deckt Termine
  jenseits des Saisonendes ab; das Zeilenlimit der Termin-Abfrage steigt von 200 auf 1000.
- `uebungsgruppen`: der iCal-Feed zählt nicht mehr zu den Flächen, die über die fehlende
  `teams`-Zeile ausgeschlossen bleiben.

## Impact

- **DB**: Migration `067_calendar_token_practice_groups` — eine Spalte auf `calendar_tokens`,
  additiv, Default `1`.
- **Code**: `internal/calendar/handler.go`, `web/src/components/profile/ProfileKalenderTab.tsx`,
  `web/src/pages/TerminePage.tsx`.
- **Verhalten für Bestandsabos**: Default `1` heißt, dass ein bestehendes Abo nach dem Deploy
  zusätzliche Termine zeigt. Das ist die gewünschte Richtung (die Termine fehlten bisher);
  wer sie nicht will, schaltet den Schalter aus.
- **Keine Berechtigungsänderung**: die Kader-Zugehörigkeit bleibt die einzige Auflösung, der
  Funktionsträger-Bypass bleibt draußen.
- **CHANGELOG**: entsteht aus den Commit-Betreffen (`make build` → `scripts/gen-changelog.py`).
