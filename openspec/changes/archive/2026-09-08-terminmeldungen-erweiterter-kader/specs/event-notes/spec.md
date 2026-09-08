## MODIFIED Requirements

### Requirement: Debounced Push-Notification bei Hinweis-Änderung

Das System SHALL **5 Minuten nach der letzten Änderung** eines Hinweistexts
eine Push-Notification an alle Mitglieder und Eltern der betroffenen Teams
versenden. Die Empfängermenge SHALL der Regel der Capability
`terminmeldung-empfaenger` folgen und damit **Stammkader und erweiterten
Kader samt deren Elternteilen sowie die Trainer des Kaders** umfassen. Jeder neue `PUT`-Aufruf SHALL den
5-Minuten-Timer für diesen Termin **zurücksetzen**, sodass mehrfache
Korrekturen nur **einen** Push auslösen. Ein Push SHALL **nie** für ein
Event in der Vergangenheit versendet werden (`event_date < today`). Ein Push
SHALL **nie** ohne Hinweistext versendet werden.

Die Debounce-Queue SHALL in der Tabelle `pending_event_notes_push (ref_type,
ref_id, note_text, notify_after, updated_by)` persistiert werden, mit
Primary Key `(ref_type, ref_id)`. Der Scheduler-Job SHALL minütlich laufen
und fällige Rows (`notify_after <= now`) abarbeiten. Die Row SHALL nach der
Verarbeitung **immer** gelöscht werden, unabhängig davon, ob ein Push
abgesetzt wurde.

#### Scenario: Erster Hinweistext erzeugt pending-Row mit notify_after = now+5min

- **WHEN** ein berechtigter Nutzer `PUT /api/{trainings|games}/{id}/note`
  mit nicht-leerem `note` aufruft
- **THEN** existiert in `pending_event_notes_push` eine Row mit
  `(ref_type, ref_id) = ('training'|'game', id)`, `note_text = note`,
  `notify_after ≈ now + 5 Minuten`

#### Scenario: Zweiter Edit innerhalb von 5 Minuten setzt Timer zurück

- **GIVEN** eine pending-Row mit `notify_after = t_0 + 5min`
- **WHEN** zur Zeit `t_1 < t_0 + 5min` ein weiterer `PUT …/note`-Aufruf
  erfolgt
- **THEN** wird `notify_after` auf `t_1 + 5min` aktualisiert
- **AND** `note_text` auf den neuen Text aktualisiert

#### Scenario: Leerer Hinweistext entfernt pending-Row ohne Push

- **GIVEN** eine pending-Row für ein Event
- **WHEN** ein berechtigter Nutzer `PUT …/note` mit Body `{"note": ""}`
  aufruft
- **THEN** wird die pending-Row gelöscht
- **AND** es wird kein Push versendet

#### Scenario: Scheduler versendet Push für zukünftiges Event und löscht Row

- **GIVEN** eine pending-Row mit `notify_after <= now` für ein Event mit
  `event_date >= today`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird die Benachrichtigung an die Empfängermenge der betroffenen
  Teams mit `category` `'trainings'` bzw. `'games'`, dem Hinweistext als
  Body und der Detail-URL als `url`-Argument versendet
- **AND** die pending-Row wird gelöscht

#### Scenario: Erweiterter Kader erhält den Termin-Hinweis

- **GIVEN** eine fällige pending-Row für ein zukünftiges Event einer
  Mannschaft, in deren erweitertem Kader ein Mitglied steht
- **WHEN** der Scheduler-Tick läuft
- **THEN** erhalten dieses Mitglied und seine über `family_links`
  verknüpften Elternteile denselben Hinweis-Push wie der Stammkader

#### Scenario: Scheduler unterdrückt Push für vergangenes Event

- **GIVEN** eine pending-Row mit `notify_after <= now` für ein Event mit
  `event_date < today`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die pending-Row wird trotzdem gelöscht (Aufräumen)

#### Scenario: Scheduler ignoriert noch-nicht-fällige Rows

- **GIVEN** eine pending-Row mit `notify_after > now`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die Row bleibt unverändert in der Tabelle

#### Scenario: Scheduler verarbeitet pending-Row eines bereits gelöschten Events sauber

- **GIVEN** eine pending-Row, deren referenziertes Event in der Zwischenzeit
  gelöscht wurde (z. B. weil der DELETE-Handler die Cleanup-Logik nicht
  ausgeführt hat oder ein Race lief)
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die pending-Row wird gelöscht
