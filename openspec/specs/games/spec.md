## ADDED Requirements

### Requirement: Auto-Regen für Drei-Tage-Fenster bei Game-Mutationen

Das System SHALL bei jeder Mutation eines Game-Events (Create, Update, Delete) implizit die Dienst-Slots im Drei-Tage-Fenster (Event-Datum ± 1 Tag) regenerieren. Die Regeneration SHALL die Helper `loadSameDayContext`, `classifySlotPosition` und `applyBehavior` aus dem `games`-Package verwenden und in derselben Transaction wie die Mutation laufen.

#### Scenario: CreateGame triggert Auto-Regen für ±1-Tag-Fenster

- **WHEN** `POST /api/admin/games` mit `event_type=heim` und `date=D` aufgerufen wird
- **THEN** wird das Game persistiert
- **AND** wird `runAutoRegen` für die Datum-Menge `{D-1, D, D+1}` aufgerufen
- **AND** alle `is_custom=0`-Slots der drei Tage werden gemäß Template + Adjacency neu erzeugt
- **AND** die Mutation-Response enthält `regen_summary` mit `created`, `reduced`, `skipped`, `notified_users`, `conflicts`

#### Scenario: UpdateGame mit Datums-Move triggert Auto-Regen für altes + neues Fenster

- **WHEN** `PUT /api/admin/games/{id}` mit geändertem `date` von D_alt auf D_neu aufgerufen wird
- **THEN** wird `runAutoRegen` für die Set-Union `{D_alt-1, D_alt, D_alt+1, D_neu-1, D_neu, D_neu+1}` aufgerufen (Duplikate eliminiert)
- **AND** die Response enthält `regen_summary`

#### Scenario: UpdateGame ohne datums-/zeit-/typ-relevante Änderung triggert dennoch Auto-Regen

- **WHEN** `PUT /api/admin/games/{id}` nur `opponent` oder `venue_id` ändert
- **THEN** wird `runAutoRegen` für `{D-1, D, D+1}` aufgerufen (Konservatismus — billig genug)

#### Scenario: DeleteGame triggert Auto-Regen für Nachbartage

- **WHEN** `DELETE /api/admin/games/{id}` ein Game am Datum D entfernt
- **THEN** wird nach dem Cascade-Delete `runAutoRegen` für `{D-1, D+1}` aufgerufen (D selbst hat keine Slots mehr)
- **AND** der `regen_summary` wird in die Response aufgenommen

### Requirement: CreateGame-Request für Heim/Auswärts ohne `slots[]`

Das System SHALL für `POST /api/admin/games` mit `event_type ∈ {heim, auswärts}` das Feld `slots[]` aus dem Request-Body ignorieren. Slots werden ausschließlich aus dem aufgelösten Template (`template_id` oder via `findTemplateForGame` ermittelt) generiert.

Für `event_type=generisch` bleibt `slots[]` erhalten und wird mit `is_custom=1` persistiert.

#### Scenario: Heimspiel mit slots[] im Request — slots[] wird ignoriert

- **WHEN** `POST /api/admin/games` mit `event_type=heim` und nicht-leerem `slots[]`-Array aufgerufen wird
- **THEN** ignoriert das Backend `slots[]` und erzeugt die Slots ausschließlich per Auto-Regen aus dem aufgelösten Template
- **AND** die Response liefert HTTP 201 mit `id` und `regen_summary`

#### Scenario: Generisches Event persistiert slots[] mit `is_custom=1`

- **WHEN** `POST /api/admin/games` mit `event_type=generisch` und `slots[]` aufgerufen wird
- **THEN** werden alle Slots aus `slots[]` mit `is_custom=1` in `duty_slots` persistiert
- **AND** Auto-Regen für `{D-1, D, D+1}` läuft anschließend, betrifft aber nur eventuelle template-basierte Slots benachbarter Spiele

### Requirement: `regen_summary` in Mutation-Response

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

#### Scenario: Wizard zeigt Änderungsbericht nach Save

- **WHEN** ein Vorstand ein neues Heimspiel speichert und die Response `regen_summary.created` mit 3 Einträgen enthält
- **THEN** zeigt die UI eine Card „Folgendes hat sich geändert" mit den drei Slot-Anlagen, ggf. „N Helfer wurden benachrichtigt"

#### Scenario: Leeres regen_summary

- **WHEN** das Auto-Regen-Fenster keine Slot-Änderungen ergibt (z.B. weil das Datum kein Template-relevantes Spiel hat)
- **THEN** ist `regen_summary` mit leeren Arrays gefüllt und das Frontend zeigt keine Card

## REMOVED Requirements

### Requirement: Manueller „Dienste generieren"-Knopf auf der Kalenderseite

**Reason:** Die Logik läuft jetzt implizit bei jeder Game-Mutation. Der Knopf ist überflüssig und verwirrend („muss ich das jetzt noch klicken?").

**Migration:** UI-Elemente werden aus `web/src/pages/KalenderPage.tsx` und `web/src/pages/SpieltagDetailPage.tsx` entfernt. Die Backend-HTTP-Endpunkte `POST /api/kalender/regenerate-day` und `POST /api/kalender/{id}/regenerate` bleiben als interne Wrapper um `runAutoRegen` erhalten, sind aber aus der Frontend-Nutzung entfernt. Ein späterer „Saison-Dienstplan optimieren"-Bulk-Workflow kann sie wieder aufgreifen (separater Change).
