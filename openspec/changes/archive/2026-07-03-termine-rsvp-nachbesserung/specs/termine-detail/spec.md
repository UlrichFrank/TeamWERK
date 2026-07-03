## MODIFIED Requirements

### Requirement: RSVP-Voreinstellungs-Editor im Bearbeiten-Modal

Die Bearbeiten-Modals für Trainings-Session, Trainings-Serie und Spiel (`TrainingEditModal.tsx`, `GameEditModal.tsx`, Series-Bulk-Formular in `AdminTrainingsPage.tsx`) SHALL zwei separate Radio-Gruppen anbieten, überschrieben mit der Sektionsüberschrift „RSVP-Voreinstellung":

- **„Kader-Spieler"** mit den drei Optionen „Standardmäßig zugesagt", „Standardmäßig abgesagt", „Keine automatische Rückmeldung" (gebunden an `rsvp_default_players`).
- **„Erweiterter Kader"** mit denselben drei Optionen (gebunden an `rsvp_default_extended`).

Die Checkbox „Begründung bei Absage erforderlich" (`rsvp_require_reason`) und die Radio-Option „Standardmäßig abgesagt" SHALL **frei kombinierbar** sein: es gibt keine gegenseitige `disabled`-Kopplung und keinen Sperr-Tooltip. Beide Kontrollen sind jederzeit bedienbar, da eine Default-Absage ohne Nutzerhandlung entsteht (kein Grund erhebbar) und `rsvp_require_reason` nur aktive Absagen betrifft — die Einstellungen wirken auf disjunkte Gruppen.

Die alte Checkbox „Alle Spieler standardmäßig zugesagt (Opt-Out)" entfällt vollständig.

#### Scenario: Radio-Auswahl wird gespeichert
- **WHEN** ein Trainer im TrainingEditModal die Radio-Option „Standardmäßig abgesagt" unter „Erweiterter Kader" wählt und speichert
- **THEN** enthält der `PUT`-Payload `rsvp_default_extended: "declined"`

#### Scenario: Reason-Checkbox lässt `declined`-Radios aktiv
- **WHEN** die Checkbox „Begründung bei Absage erforderlich" gesetzt ist
- **THEN** sind die Radios „Standardmäßig abgesagt" in beiden Rollen weiterhin `enabled` (frei wählbar)

#### Scenario: Aktive `declined`-Auswahl lässt Reason-Checkbox aktiv
- **WHEN** eine der beiden Voreinstellungen auf „Standardmäßig abgesagt" gesetzt ist
- **THEN** ist die Checkbox „Begründung bei Absage erforderlich" weiterhin `enabled` (frei setzbar)
