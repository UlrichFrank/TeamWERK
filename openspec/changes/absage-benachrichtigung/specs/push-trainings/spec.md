## MODIFIED Requirements

### Requirement: Push bei Trainings-Ereignissen

Das System SHALL allen Mitgliedern des betroffenen Teams und deren Elternteilen eine Push
Notification senden, wenn eine Trainingseinheit abgesagt, verschoben oder gelöscht wird —
sofern Push für Kategorie `trainings` nicht deaktiviert ist und die Benachrichtigung nicht
per `silent`-Flag unterdrückt wurde.

Für **bestehende** Einheiten MUSS die Notification-`url` auf den konkreten Trainingstermin
in der Termine-Seite zeigen (`/termine?focus=training-<id>`), damit der Empfänger direkt zu-
oder absagen kann. **Das gilt ausdrücklich auch für abgesagte Einheiten** (`status =
'cancelled'`): sie existieren weiter und zeigen ihren Absagegrund an.

Für **gelöschte** Einheiten und **gelöschte** Serien SHALL die `url` der **leere String**
sein. Die frühere Regelung (`/termine`) SHALL NICHT mehr gelten.

Die Benachrichtigung über eine gelöschte Einheit oder Serie SHALL Titel bzw. Serienname,
Datum bzw. Zeitraum und den Namen des auslösenden Nutzers enthalten sowie — falls angegeben
— den Löschgrund.

Wechselt eine Einheit über `PUT /api/training-sessions/{id}` von `status='active'` auf
`status='cancelled'`, SHALL das System die Notification „Training abgesagt" mit dem
erfassten `cancel_reason` senden — nicht mehr „Training geändert". Diese Meldung SHALL
**nur beim Zustandswechsel** ausgelöst werden; bleibt der Status `cancelled` oder wechselt
er zurück auf `active`, SHALL weiterhin „Training geändert" gesendet werden.

#### Scenario: Einheit wird abgesagt (Statuswechsel)
- **WHEN** ein Trainer eine Einheit mit `status='active'` über `PUT /api/training-sessions/{id}` auf `status='cancelled'` mit `cancel_reason='Halle gesperrt'` setzt
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training abgesagt"
- **THEN** enthält der Body Titel, Datum, den Namen des auslösenden Nutzers und „Halle gesperrt"
- **THEN** zeigt der Klick-Link auf `/termine?focus=training-<id>`

#### Scenario: Nachbearbeitung einer bereits abgesagten Einheit
- **WHEN** eine Einheit mit `status='cancelled'` erneut per PUT mit `status='cancelled'` gespeichert wird
- **THEN** wird die Notification „Training geändert" gesendet
- **THEN** wird KEINE zweite „Training abgesagt"-Notification gesendet

#### Scenario: Absage wird zurückgenommen
- **WHEN** eine Einheit von `status='cancelled'` auf `status='active'` gesetzt wird
- **THEN** wird die Notification „Training geändert" gesendet

#### Scenario: Einzelne Session gelöscht
- **WHEN** ein Trainer eine einzelne Trainingseinheit über `DELETE /api/training-sessions/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training abgesagt"
- **THEN** enthält der Body Titel, Datum im Format `TT.MM.JJJJ` und den Namen des auslösenden Nutzers
- **THEN** ist die `url` der leere String

#### Scenario: Session verschoben
- **WHEN** ein Trainer eine Einheit über `PUT /api/training-sessions/{id}` aktualisiert (Zeit oder Ort geändert, Status unverändert `active`)
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Training geändert"
- **THEN** zeigt der Klick-Link auf `/termine?focus=training-<id>` der geänderten Einheit

#### Scenario: Ganze Serie gelöscht
- **WHEN** ein Trainer eine gesamte Trainingsserie über `DELETE /api/training-series/{id}` löscht
- **THEN** erhalten alle Mitglieder des Kaders + deren Elternteile eine Push Notification „Trainingsserie beendet"
- **THEN** enthält der Body den Seriennamen, den betroffenen Zeitraum und den Namen des auslösenden Nutzers
- **THEN** ist die `url` der leere String

#### Scenario: Vorstand löscht ohne Benachrichtigung
- **WHEN** ein Nutzer mit Capability `suppress_event_notification` eine Einheit oder Serie mit `{"silent":true}` löscht
- **THEN** erhält kein Nutzer eine `trainings`-Notification
- **THEN** wird das SSE-Live-Update trotzdem gesendet

#### Scenario: Nutzer mit deaktiviertem Push
- **WHEN** ein Trainings-Ereignis eintritt und der Nutzer hat `push_enabled=0` für `trainings`
- **THEN** erhält dieser Nutzer keine Push Notification
