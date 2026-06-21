# profile-datenschutz-tab Specification

## Purpose
TBD - created by archiving change profile-cross-team-visibility. Update Purpose after archive.
## Requirements
### Requirement: Datenschutz-Tab im Nutzerprofil

Das System SHALL in `/profil` einen Tab „Datenschutz" anbieten, einsortiert zwischen „Kalender" und „Sonstiges". Der Tab SHALL nur angezeigt werden, wenn der eingeloggte Nutzer ein eigenes Member hat (`ownMember !== null`).

#### Scenario: Tab nur bei eigenem Member sichtbar

- **WHEN** ein Nutzer ohne eigenes Member (z.B. reines Elternteil) `/profil` öffnet
- **THEN** sieht er den Datenschutz-Tab NICHT

#### Scenario: Tab-Reihenfolge

- **WHEN** ein Nutzer mit eigenem Member `/profil` öffnet
- **THEN** sind die Tabs in dieser Reihenfolge sichtbar: Account, Profil, Mitglied, Bank, Kalender, Datenschutz, Sonstiges

### Requirement: Toggle „Sichtbarkeit für Mitglieder"

Der Datenschutz-Tab SHALL einen Toggle „Sichtbarkeit für Mitglieder" anbieten, der den Wert von `members.cross_team_visible` für das eigene Member spiegelt. Eine kurze Beschreibung SHALL den Effekt erläutern (Mitglieder anderer Mannschaften sehen Name und Rückmeldung bei gemeinsamen Multi-Team-Terminen).

Das Speichern SHALL direkt über `PUT /api/members/{id}` erfolgen — ohne Draft-Workflow.

#### Scenario: Toggle spiegelt aktuellen Wert

- **WHEN** der Tab geladen wird
- **THEN** spiegelt der Toggle den aktuellen Wert von `cross_team_visible` für das eigene Member

#### Scenario: Direktes Speichern

- **WHEN** der Nutzer den Toggle umlegt und speichert
- **THEN** wird `PUT /api/members/{id}` mit `cross_team_visible` aufgerufen
- **AND** der neue Wert ist sofort wirksam (kein Approval-Step)

#### Scenario: Standardwert für neue Mitglieder

- **WHEN** ein Member neu angelegt wird (Migration oder Registrierung)
- **THEN** gilt `cross_team_visible = 0` (privat)

### Requirement: DSGVO-Anzeige (read-only) im Datenschutz-Tab

Der Datenschutz-Tab SHALL die DSGVO-Einwilligungen des eigenen Members anzeigen: „Datenverarbeitung eingewilligt" und „Datenweitergabe eingewilligt" je mit Status (Ja/Nein) und Datum (falls vorhanden). Das visuelle Control SHALL dem Stil von `MemberDatenschutzTab` im Admin entsprechen, ist aber **gesperrt** (read-only). Änderungen SHALL weiterhin nur über den bestehenden Draft-Workflow im „Profil"-Tab beantragt werden.

#### Scenario: DSGVO-Status wird angezeigt

- **WHEN** der Tab geladen wird
- **THEN** sieht der Nutzer den Status von `dsgvo_verarbeitung` und `dsgvo_weitergabe` mit zugehörigen `_date`-Werten
- **AND** beide Controls sind nicht bedienbar (kein Schreiben aus diesem Tab heraus)

#### Scenario: Hinweis auf Änderungsweg

- **WHEN** der Tab gerendert wird
- **THEN** zeigt er einen Hinweis, dass Änderungen über den „Profil"-Tab beantragt werden müssen (Draft-Workflow)

### Requirement: Admin- und Familien-Profil mit Sichtbarkeitstoggle

Der bestehende Member-Datenschutz-Tab (genutzt im Admin-Detail `/mitglieder/{id}` und im Familienzugang auf Kind-Profile) SHALL den Toggle „Sichtbarkeit für Mitglieder" zusätzlich anbieten, **bedienbar** (Schreibrecht), damit Eltern die Sichtbarkeit für ihre Kinder einstellen können.

#### Scenario: Elternteil setzt Sichtbarkeit des Kindes

- **WHEN** ein Elternteil auf das Member-Profil seines Kindes navigiert und den Datenschutz-Tab öffnet
- **THEN** sieht es den Toggle „Sichtbarkeit für Mitglieder" und kann ihn ändern
- **AND** das Speichern setzt `cross_team_visible` auf dem Kind-Member direkt

#### Scenario: Vorstand setzt Sichtbarkeit fremder Mitglieder

- **WHEN** ein Caller mit `vorstand`- oder `admin`-Funktion `/mitglieder/{id}` eines fremden Members im Datenschutz-Tab bearbeitet
- **THEN** kann er `cross_team_visible` direkt setzen

