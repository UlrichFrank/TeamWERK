## Capability: kalender-agenda-view

Mobile Agenda-Darstellung des Monatskalenders auf Viewports < 640px.

### Requirement: Agenda-Liste auf Mobile

Auf Viewports < 640px (`sm:hidden`) zeigt `KalenderPage` anstelle des 7-Spalten-Grids eine scrollbare Liste aller Events des aktuell gewählten Monats.

- Tage ohne Events werden nicht angezeigt
- Tage mit Events erscheinen als Datums-Trenner gefolgt von Event-Karten
- Innerhalb eines Tages: Spiele vor Trainings, jeweils sortiert nach Uhrzeit
- Vergangene Events (vor heute) sind optisch gedimmt (analog zum Desktop-Grid: `opacity-70`)

### Requirement: Event-Karte Spiel

Jede Spiel-Karte zeigt:
- Teamname(n), Gegner, Heimspiel/Auswärtsspiel-Indikator
- Uhrzeit
- Dienst-Ampel (farbiger Dot: rot/gelb/grün nach Slot-Belegung) — identische Logik wie Desktop

Tap navigiert zu `/kalender/<game.id>`.

### Requirement: Event-Karte Training

Jede Trainings-Karte zeigt:
- Trainingsbezeichnung (Name/Beschreibung)
- Uhrzeit und Ort (falls vorhanden)

Tap navigiert zu `/trainings/<training.id>`.

### Requirement: Monatswechsel-Navigation

Die ◀ / ▶ Navigation und die Monats-/Jahresanzeige sind auf Mobile identisch zur Desktop-Navigation. Monatswechsel lädt neue Daten (bestehende Logik).

### Requirement: FAB für Admins und Trainer

Nutzer mit Rolle `admin`, `vorstand` oder `trainer` sehen auf Mobile einen Floating Action Button (FAB) unten rechts (`fixed bottom-6 right-6`).

- Mindestgröße: 48×48px (Touch-Target ≥ 44px)
- Öffnet den bestehenden Event-Wizard ohne vorausgewähltes Datum (User wählt Datum im Wizard)
- Nur sichtbar auf Mobile (`sm:hidden`)

### Requirement: Leerzustand

Wenn der gewählte Monat keine Events enthält, zeigt die Agenda-Liste einen zentrierten Hinweistext: „Keine Events in diesem Monat."

---

### Requirement: Abwesenheiten im Kalender farblich nach Herkunft unterschieden
Der Kalender SHALL eigene Abwesenheiten (`is_own: true`) in `brand-yellow` und Team-Abwesenheiten (`is_own: false`) in `brand-blue` darstellen.

#### Scenario: Eigene Abwesenheit bleibt gelb
- **WHEN** eine Abwesenheit `is_own = true` hat
- **THEN** wird sie mit `bg-brand-yellow/20 border-brand-yellow/60` dargestellt (unverändert)

#### Scenario: Team-Abwesenheit erscheint blau
- **WHEN** eine Abwesenheit `is_own = false` hat
- **THEN** wird sie mit `bg-brand-blue/20 border-brand-blue/60` dargestellt

---

### Requirement: Personendetails für Team-Abwesenheiten nur per Tooltip und Click
Team-Abwesenheiten (`is_own: false`) SHALL den Namen des Mitglieds und den Typ im `title`-Attribut (Tooltip) anzeigen. Per Click öffnet sich die vorhandene Detailansicht (InfoModal).

#### Scenario: Tooltip zeigt Name und Typ
- **WHEN** ein Nutzer über eine Team-Abwesenheit hovert
- **THEN** zeigt der Browser-Tooltip: `{member_name}: {type} {start_date}–{end_date}`

#### Scenario: Click öffnet Detailansicht
- **WHEN** ein Nutzer auf eine Team-Abwesenheit klickt
- **THEN** öffnet sich das Info-Modal mit den Details der Abwesenheit (Typ, Zeitraum, ggf. Notiz)
