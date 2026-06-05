## ADDED Requirements

### Requirement: GameEditModal zeigt editierbare Felder eines Spieltags

Das `GameEditModal` SHALL die Felder `opponent`, `date`, `time` und `event_type` eines bestehenden Spieltags anzeigen und editierbar machen. Es verwendet das Standard-Modal-Design (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`).

#### Scenario: Modal öffnet sich mit vorausgefüllten Daten

- **WHEN** das `GameEditModal` für einen Spieltag geöffnet wird
- **THEN** sind alle Felder mit den aktuellen Werten des Spieltags vorausgefüllt

#### Scenario: Modal zeigt alle editierbaren Felder

- **WHEN** das `GameEditModal` offen ist
- **THEN** enthält es ein Eingabefeld für `opponent` (Gegner)
- **THEN** enthält es ein Datumfeld für `date`
- **THEN** enthält es ein Zeitfeld für `time`
- **THEN** enthält es eine Auswahl für `event_type` (heim / auswärts / generisch)

---

### Requirement: GameEditModal speichert Änderungen via PUT /api/admin/games/{id}

Das Speichern SHALL `PUT /api/admin/games/{id}` aufrufen. Bei Erfolg schließt sich das Modal und der Kalender aktualisiert sich.

#### Scenario: Erfolgreiches Speichern

- **WHEN** ein berechtigter User Änderungen vornimmt und auf „Speichern" klickt
- **THEN** wird `PUT /api/admin/games/{id}` mit den geänderten Feldern aufgerufen
- **THEN** schließt sich das Modal nach erfolgreicher Antwort
- **THEN** werden die Spieltag-Daten im Kalender aktualisiert (via Reload oder SSE)

#### Scenario: Fehler beim Speichern

- **WHEN** der Server mit einem Fehler antwortet
- **THEN** zeigt das Modal eine Fehlermeldung (Alert-Fehler-Klasse)
- **THEN** bleibt das Modal geöffnet

#### Scenario: Abbrechen ohne Speichern

- **WHEN** ein User auf „Abbrechen" klickt oder Escape drückt
- **THEN** schließt sich das Modal ohne API-Aufruf
- **THEN** bleiben die Spieltag-Daten im Kalender unverändert

---

### Requirement: Zugriff nur für berechtigte Rollen

Das `GameEditModal` SHALL nur für Nutzer mit Rolle admin, trainer, vorstand oder sportliche_leitung angezeigt werden. Es enthält selbst keine Rollenprüfung — der aufrufende Click-Handler steuert die Sichtbarkeit.

#### Scenario: Rollenprüfung im Click-Handler

- **WHEN** ein User mit Rolle spieler auf einen Spieltag im Termine-Modus klickt
- **THEN** wird `GameEditModal` NICHT geöffnet (stattdessen `EventInfoModal`)
