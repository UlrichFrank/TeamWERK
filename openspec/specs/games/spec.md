# Games — Spec

## Requirement: Auto-Regen für Drei-Tage-Fenster bei Game-Mutationen

Das System SHALL bei jeder Mutation eines Game-Events (Create, Update, Delete) implizit die Dienst-Slots im Drei-Tage-Fenster (Event-Datum ± 1 Tag) regenerieren. Die Regeneration SHALL die Helper `loadSameDayContext`, `classifySlotPosition` und `applyBehavior` aus dem `games`-Package verwenden und in derselben Transaction wie die Mutation laufen.

### Scenario: CreateGame triggert Auto-Regen für ±1-Tag-Fenster

- **WHEN** `POST /api/admin/games` mit `event_type=heim` und `date=D` aufgerufen wird
- **THEN** wird das Game persistiert
- **AND** wird `runAutoRegen` für die Datum-Menge `{D-1, D, D+1}` aufgerufen
- **AND** alle `is_custom=0`-Slots der drei Tage werden gemäß Template + Adjacency neu erzeugt
- **AND** die Mutation-Response enthält `regen_summary` mit `created`, `reduced`, `skipped`, `notified_users`, `conflicts`

### Scenario: UpdateGame mit Datums-Move triggert Auto-Regen für altes + neues Fenster

- **WHEN** `PUT /api/admin/games/{id}` mit geändertem `date` von D_alt auf D_neu aufgerufen wird
- **THEN** wird `runAutoRegen` für die Set-Union `{D_alt-1, D_alt, D_alt+1, D_neu-1, D_neu, D_neu+1}` aufgerufen (Duplikate eliminiert)
- **AND** die Response enthält `regen_summary`

### Scenario: UpdateGame ohne datums-/zeit-/typ-relevante Änderung triggert dennoch Auto-Regen

- **WHEN** `PUT /api/admin/games/{id}` nur `opponent` oder `venue_id` ändert
- **THEN** wird `runAutoRegen` für `{D-1, D, D+1}` aufgerufen (Konservatismus — billig genug)

### Scenario: DeleteGame triggert Auto-Regen für Nachbartage

- **WHEN** `DELETE /api/admin/games/{id}` ein Game am Datum D entfernt
- **THEN** wird nach dem Cascade-Delete `runAutoRegen` für `{D-1, D+1}` aufgerufen (D selbst hat keine Slots mehr)
- **AND** der `regen_summary` wird in die Response aufgenommen

## Requirement: CreateGame-Request für Heim/Auswärts ohne `slots[]`

Das System SHALL für `POST /api/admin/games` mit `event_type ∈ {heim, auswärts}` das Feld `slots[]` aus dem Request-Body ignorieren. Slots werden ausschließlich aus dem persistierten `games.template_id` generiert. Ist `template_id` `null`, werden keine Auto-Dienste erzeugt.

Für `event_type=generisch` werden `slots[]` mit `is_custom=1` persistiert. `template_id` ist optional — gesetzt löst Auto-Regen aus dem generisch-Template (Dauer aus `game_templates.duration_minutes`) aus; benutzerdefinierte und template-basierte Slots koexistieren (Konflikte werden im `regen_summary` als `conflicts` ausgewiesen).

### Scenario: Heimspiel mit `slots[]` im Request — `slots[]` wird ignoriert

- **WHEN** `POST /api/admin/games` mit `event_type=heim`, `template_id=7` und nicht-leerem `slots[]`-Array aufgerufen wird
- **THEN** ignoriert das Backend `slots[]` und erzeugt die Slots ausschließlich per Auto-Regen aus Template 7
- **AND** die Response liefert HTTP 201 mit `id` und `regen_summary`

### Scenario: Heimspiel ohne Vorlage erzeugt keine Auto-Slots

- **WHEN** `POST /api/admin/games` mit `event_type=heim` und `template_id=null` aufgerufen wird
- **THEN** wird das Game persistiert mit `template_id=NULL`
- **AND** der Auto-Regen erzeugt keine `is_custom=0`-Slots für dieses Event
- **AND** die Response liefert HTTP 201 mit `id` und `regen_summary`

### Scenario: Generisches Event mit `slots[]` persistiert `is_custom=1`

- **WHEN** `POST /api/admin/games` mit `event_type=generisch`, `template_id=null` und `slots[]` aufgerufen wird
- **THEN** werden alle Slots aus `slots[]` mit `is_custom=1` in `duty_slots` persistiert
- **AND** Auto-Regen für `{D-1, D, D+1}` läuft anschließend, betrifft aber nur eventuelle template-basierte Slots benachbarter Spiele

### Scenario: Generisches Event mit Template erzeugt Auto-Slots

- **GIVEN** ein generisch-Template mit `duration_minutes=240` und einem Item (Aufbau, anchor=start, offset=-30min)
- **WHEN** `POST /api/admin/games` mit `event_type=generisch`, `template_id=<tpl>` und `time=14:00` aufgerufen wird
- **THEN** wird das Game mit `template_id=<tpl>` persistiert
- **AND** Auto-Regen erzeugt einen `is_custom=0`-Slot um 13:30 (Anchor `start`, Offset -30min)
- **AND** Dauer für Adjacency-Berechnungen kommt aus `game_templates.duration_minutes` (240min)

## Requirement: Auflösung von `template_id` ohne Fallback

Das System SHALL beim Auto-Regen ausschließlich den persistierten Wert von `games.template_id` als Slot-Quelle verwenden. Ist der Wert `NULL`, werden keine `is_custom=0`-Slots für dieses Event erzeugt — unabhängig vom `event_type`. Der frühere ID-basierte Fallback („kleinste passende Template-ID") entfällt ersatzlos.

### Scenario: Auto-Regen für Event mit `template_id=NULL` erzeugt keine Slots

- **WHEN** `runAutoRegen` für ein Event mit `template_id IS NULL` läuft
- **THEN** werden alle vorhandenen `is_custom=0`-Slots des Events gelöscht und nicht ersetzt
- **AND** `is_custom=1`-Slots des Events bleiben unverändert

### Scenario: Kein impliziter Default bei NULL

- **GIVEN** mehrere `game_templates` mit `template_type='heim'` existieren
- **WHEN** ein Event mit `event_type='heim'` und `template_id IS NULL` regeneriert wird
- **THEN** wird KEINE Vorlage automatisch ausgewählt; das Event bleibt ohne Auto-Slots

## Requirement: `template_id` per `PUT /api/admin/games/{id}` änderbar

Das System SHALL im `PUT /api/admin/games/{id}`-Endpoint das Feld `template_id` mit Tri-State-Semantik akzeptieren:

| Body-Inhalt              | Verhalten                                  |
|--------------------------|--------------------------------------------|
| Feld nicht vorhanden     | `template_id` bleibt unverändert           |
| `"template_id": null`    | `template_id` wird auf NULL gesetzt        |
| `"template_id": <int>`   | `template_id` wird auf den Wert gesetzt    |

Nach der Persistierung läuft `runAutoRegen` für das Datum-Fenster (`oldDate ± 1` ∪ `newDate ± 1`) wie bisher; bei `NULL` werden bestehende `is_custom=0`-Slots des Events gelöscht und nicht ersetzt. Das Verhalten ist für alle `event_type` (`heim`/`auswärts`/`generisch`) gleich — `template_id` muss zum `template_type` passen (gleicher Wert), sonst sind die Slot-Quellen leer (kein Match).

### Scenario: Feld fehlt im Body — Wert bleibt unverändert

- **GIVEN** ein Game mit `template_id=5`
- **WHEN** `PUT /api/admin/games/{id}` ohne `template_id`-Feld im Body aufgerufen wird
- **THEN** bleibt `games.template_id=5` erhalten
- **AND** Auto-Regen verwendet weiterhin Template 5

### Scenario: Explizites `null` setzt auf NULL

- **GIVEN** ein Game mit `template_id=5` und mehreren `is_custom=0`-Slots
- **WHEN** `PUT /api/admin/games/{id}` mit `"template_id": null` aufgerufen wird
- **THEN** wird `games.template_id=NULL` gesetzt
- **AND** die `is_custom=0`-Slots des Events werden im Auto-Regen entfernt
- **AND** `is_custom=1`-Slots des Events bleiben unverändert
- **AND** die Response liefert HTTP 200 mit `regen_summary`

### Scenario: Wechsel der Vorlage regeneriert Slots

- **GIVEN** ein Game mit `template_id=5`
- **WHEN** `PUT /api/admin/games/{id}` mit `"template_id": 7` aufgerufen wird
- **THEN** wird `games.template_id=7` gesetzt
- **AND** die Slots werden aus Template 7 neu erzeugt
- **AND** die Response liefert HTTP 200 mit `regen_summary`

## Requirement: `regen_summary` in Mutation-Response

Das System SHALL die Antwort von `POST /api/admin/games`, `PUT /api/admin/games/{id}` und `DELETE /api/admin/games/{id}` um ein `regen_summary`-Objekt erweitern. Die UI auf `/kalender` und `/kalender/{id}` SHALL nach erfolgreicher Mutation die Änderungen als Banner oder Card anzeigen.

Das Schema:

```
regen_summary: {
  created: [{ date, duty_type, count }],
  reduced: [{ date, from, to, count }],
  skipped: [{ date, duty_type }],
  notified_users: [user_id, ...],
  conflicts: [{ date, duty_type, event_time, slot_ids }]
}
```

Listen werden pro Kategorie auf 20 Einträge gekappt; bei Überschreitung erscheint im Frontend „… und N weitere Änderungen".

### Scenario: Wizard zeigt Änderungsbericht nach Save

- **WHEN** ein Vorstand ein neues Heimspiel speichert und die Response `regen_summary.created` mit 3 Einträgen enthält
- **THEN** zeigt die UI eine Card „Folgendes hat sich geändert" mit den drei Slot-Anlagen, ggf. „N Helfer wurden benachrichtigt"

### Scenario: Leeres regen_summary

- **WHEN** das Auto-Regen-Fenster keine Slot-Änderungen ergibt (z.B. weil das Datum kein Template-relevantes Spiel hat)
- **THEN** ist `regen_summary` mit leeren Arrays gefüllt und das Frontend zeigt keine Card

## Removed: Manueller „Dienste generieren"-Knopf auf der Kalenderseite

**Reason:** Die Logik läuft jetzt implizit bei jeder Game-Mutation. Der Knopf ist überflüssig und verwirrend („muss ich das jetzt noch klicken?").

**Migration:** UI-Elemente werden aus `web/src/pages/KalenderPage.tsx` und `web/src/pages/SpieltagDetailPage.tsx` entfernt. Die Backend-HTTP-Endpunkte `POST /api/kalender/regenerate-day` und `POST /api/kalender/{id}/regenerate` bleiben als interne Wrapper um `runAutoRegen` erhalten, sind aber aus der Frontend-Nutzung entfernt. Ein späterer „Saison-Dienstplan optimieren"-Bulk-Workflow kann sie wieder aufgreifen (separater Change).
