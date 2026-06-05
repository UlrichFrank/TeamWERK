## 1. Agenda-Liste: Datenaufbereitung

- [ ] 1.1 Hilfsfunktion `agendaDays` ableiten: Iteriert über alle Tage des aktuellen Monats, gibt nur Tage zurück, die mindestens ein Spiel oder Training haben (aus bestehenden `gamesByDate` / `trainingsByDate` Maps)
- [ ] 1.2 Sortierung sicherstellen: Innerhalb eines Tages Spiele vor Trainings, je nach `event_time` / `start_time`

## 2. Agenda-View Markup (sm:hidden)

- [ ] 2.1 Unterhalb der Monatsnavigation einen `sm:hidden`-Container ergänzen
- [ ] 2.2 Leerzustand rendern: wenn `agendaDays` leer, Hinweis „Keine Events in diesem Monat." anzeigen
- [ ] 2.3 Datums-Trenner pro Tag: Wochentag + Datum, optisch als Label (z.B. `text-xs font-semibold uppercase text-brand-text-muted`)
- [ ] 2.4 Spiel-Karte: Teamname(n), Gegner, Heimspiel/Auswärtsspiel-Icon (`<Home>` / `<MapPin>`), Uhrzeit, Dienst-Ampel-Dot — Tap navigiert zu `/kalender/<game.id>`
- [ ] 2.5 Trainings-Karte: Bezeichnung, Uhrzeit + Ort — Tap navigiert zu `/trainings/<training.id>`
- [ ] 2.6 Vergangene Tage (vor `todayStr`) mit `opacity-70` dimmen

## 3. Desktop-Grid kapseln (hidden sm:block)

- [ ] 3.1 Bestehenden Kalender-Grid-Block (`<div ref={calendarRef} ...>` inkl. Wochen-Header) in `<div className="hidden sm:block">` einwickeln — keine inhaltlichen Änderungen

## 4. FAB für Admins/Trainer

- [ ] 4.1 `<button className="sm:hidden fixed bottom-6 right-6 ...">` mit `<Plus>`-Icon ergänzen, nur wenn `canEdit` true
- [ ] 4.2 FAB öffnet bestehenden Wizard (`openWizardWithDate('')` oder neues leeres Datum) — kein neuer State nötig
- [ ] 4.3 `pb-20` am Agenda-Container ergänzen, damit FAB letzten Eintrag nicht überdeckt

## 5. Verifikation

- [ ] 5.1 Mobile (375px): Agenda-Liste zeigt Events, kein horizontaler Overflow, FAB sichtbar für Admin
- [ ] 5.2 Mobile (375px): Tap auf Spiel-Karte navigiert zu Spieltag-Detail
- [ ] 5.3 Mobile (375px): Tap auf Trainings-Karte navigiert zu Trainings-Detail
- [ ] 5.4 Mobile (375px): FAB öffnet Wizard
- [ ] 5.5 Desktop (> 640px): Grid-Kalender unverändert, FAB nicht sichtbar
- [ ] 5.6 Monatswechsel funktioniert auf Mobile und Desktop
