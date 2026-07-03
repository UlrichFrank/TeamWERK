## ADDED Requirements

### Requirement: Voreinstellungs-basierte Antworten werden dezent gerendert

Die Termin-Detail-Tabelle (`web/src/pages/TermineDetailPage.tsx`) SHALL Zeilen, deren `rsvp_status` aus der Session/Spiel-Voreinstellung stammt (`rsvp_is_default=true`), visuell von aktiven Antworten unterscheiden: Die Status-Anzeige (Icon + Text) wird mit `text-brand-text-subtle italic` gerendert. Aktive Antworten bleiben in `text-brand-text` (nicht kursiv).

Trainer-Zeilen behalten die bestehende Darstellung ihres virtuellen `confirmed`-Defaults (dieser Change ändert die Trainer-Rendering-Logik nicht).

#### Scenario: Stammkader-Spieler mit Default „standardmäßig zugesagt"
- **WHEN** eine Session `rsvp_default_players='confirmed'` hat und ein Spieler hat keine `training_responses`-Row
- **THEN** zeigt die Detail-Tabellenzeile die Statuszelle mit CSS-Klasse `italic` und Textfarbe `text-brand-text-subtle`

#### Scenario: Aktive Antwort wird nicht kursiv gerendert
- **WHEN** derselbe Spieler eine `training_responses`-Row mit `status='confirmed'` hat
- **THEN** wird die Statuszelle in `text-brand-text` (nicht kursiv) gerendert

#### Scenario: Erweiterter Kader mit Default „standardmäßig abgesagt"
- **WHEN** eine Session `rsvp_default_extended='declined'` hat und ein Erweiterte-Kader-Mitglied hat keine Response
- **THEN** wird die Statuszelle mit „Absage"-Icon in `text-brand-text-subtle italic` gerendert

---

### Requirement: RSVP-Voreinstellungs-Editor im Bearbeiten-Modal

Die Bearbeiten-Modals für Trainings-Session, Trainings-Serie und Spiel (`TrainingEditModal.tsx`, `GameEditModal.tsx`, Series-Bulk-Formular in `AdminTrainingsPage.tsx`) SHALL zwei separate Radio-Gruppen anbieten, überschrieben mit der Sektionsüberschrift „RSVP-Voreinstellung":

- **„Kader-Spieler"** mit den drei Optionen „Standardmäßig zugesagt", „Standardmäßig abgesagt", „Keine automatische Rückmeldung" (gebunden an `rsvp_default_players`).
- **„Erweiterter Kader"** mit denselben drei Optionen (gebunden an `rsvp_default_extended`).

Zusätzlich SHALL die Checkbox „Begründung bei Absage erforderlich" (`rsvp_require_reason`) mechanisch **nicht kombinierbar** sein mit einer der Radio-Optionen „Standardmäßig abgesagt": ist die Checkbox aktiv, sind die `declined`-Radios `disabled`; ist eine der `declined`-Radios gewählt, ist die Checkbox `disabled`. Der `disabled`-Zustand trägt einen `title`-Tooltip „Nicht mit ‚Standardmäßig abgesagt' kombinierbar".

Die alte Checkbox „Alle Spieler standardmäßig zugesagt (Opt-Out)" entfällt vollständig.

#### Scenario: Radio-Auswahl wird gespeichert
- **WHEN** ein Trainer im TrainingEditModal die Radio-Option „Standardmäßig abgesagt" unter „Erweiterter Kader" wählt und speichert
- **THEN** enthält der `PUT`-Payload `rsvp_default_extended: "declined"`

#### Scenario: `rsvp_require_reason` sperrt `declined`-Radios
- **WHEN** die Checkbox „Begründung bei Absage erforderlich" gesetzt ist
- **THEN** sind die Radios „Standardmäßig abgesagt" in beiden Rollen `disabled`

#### Scenario: Aktive `declined`-Auswahl sperrt Reason-Checkbox
- **WHEN** eine der beiden Voreinstellungen auf „Standardmäßig abgesagt" gesetzt ist
- **THEN** ist die Checkbox „Begründung bei Absage erforderlich" `disabled`
