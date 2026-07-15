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

### Requirement: Admin- und Familien-Profil mit Sichtbarkeitstoggle

Der bestehende Member-Datenschutz-Tab (genutzt im Admin-Detail `/mitglieder/{id}` und im Familienzugang auf Kind-Profile) SHALL den Toggle „Sichtbarkeit für Mitglieder" zusätzlich anbieten, **bedienbar** (Schreibrecht), damit Eltern die Sichtbarkeit für ihre Kinder einstellen können.

#### Scenario: Elternteil setzt Sichtbarkeit des Kindes

- **WHEN** ein Elternteil auf das Member-Profil seines Kindes navigiert und den Datenschutz-Tab öffnet
- **THEN** sieht es den Toggle „Sichtbarkeit für Mitglieder" und kann ihn ändern
- **AND** das Speichern setzt `cross_team_visible` auf dem Kind-Member direkt

#### Scenario: Vorstand setzt Sichtbarkeit fremder Mitglieder

- **WHEN** ein Caller mit `vorstand`- oder `admin`-Funktion `/mitglieder/{id}` eines fremden Members im Datenschutz-Tab bearbeitet
- **THEN** kann er `cross_team_visible` direkt setzen

### Requirement: DSGVO-Einwilligungen mit Change-Request im Datenschutz-Tab

Der Datenschutz-Tab SHALL die DSGVO-Einwilligungen des eigenen Members anzeigen:
„Datenverarbeitung eingewilligt", „Datenweitergabe eingewilligt" und
„Foto-Veröffentlichung eingewilligt" — je mit Status (Ja/Nein) und Datum (falls
vorhanden). Zu **jedem** der drei Schalter SHALL ein kurzer Erklärtext angezeigt
werden, der beschreibt, was mit der Einwilligung verbunden ist (Verarbeitung der
Mitgliedsdaten; Weitergabe an Dritte; Veröffentlichung von Fotos auf
öffentlichen Kanälen des Vereins).

Die Schalter SHALL **aktiv (bedienbar)** sein. Änderungen am Schalter-Zustand
SHALL nur lokal in den Draft-Kandidaten laufen und NIE direkt auf den Member
geschrieben werden. Das Speichern SHALL ausschließlich über den bestehenden
Change-Request-Draft-Workflow erfolgen (`POST /api/members/{id}/change-request`
mit `field_name='dsgvo'`, `new_value={verarbeitung, weitergabe, foto_veroeffentlichung}`).

Der „Änderung anfragen"-Button SHALL gesperrt sein, solange die lokalen Werte
mit den Server-Werten übereinstimmen (kein Draft ohne Diff). Ein ausstehender
Draft SHALL pro geänderter Einwilligung als „(angefragt: Ja|Nein)" hinter dem
Schalter-Label sichtbar sein. Ein „Anfrage zurückziehen"-Button SHALL den Draft
löschen (`DELETE /api/members/{id}/change-drafts/{id}`) und die lokalen Werte
auf den Server-Stand zurücksetzen.

#### Scenario: DSGVO-Status wird angezeigt

- **WHEN** der Tab geladen wird
- **THEN** sieht der Nutzer den Status von `dsgvo_verarbeitung`, `dsgvo_weitergabe`
  und `foto_veroeffentlichung` mit zugehörigen `_date`-Werten
- **AND** alle drei Schalter sind bedienbar (`disabled=false`)

#### Scenario: Erklärtext je Schalter

- **WHEN** der DSGVO-Block gerendert wird
- **THEN** steht unter jedem der drei Schalter ein erläuternder Text zu seiner Bedeutung

#### Scenario: Anfrage-Button ohne Diff gesperrt

- **WHEN** der Nutzer den Tab öffnet und keine Schalter umgestellt hat
- **THEN** ist der Button „Änderung anfragen" gesperrt
- **AND** kein `POST /change-request` wird ausgelöst

#### Scenario: Änderung wird als Draft angefragt

- **WHEN** der Nutzer einen Schalter umstellt und „Änderung anfragen" klickt
- **THEN** wird `POST /api/members/{id}/change-request` mit
  `field_name='dsgvo'` und `new_value` als Objekt der drei Boolwerte gesendet
- **AND** danach zeigt der Tab pro abweichender Einwilligung „(angefragt: …)"
  hinter dem Label
- **AND** der Server-Wert von `dsgvo_verarbeitung` etc. bleibt unverändert bis
  zur Approval durch Vorstand

#### Scenario: Draft zurückziehen

- **WHEN** der Nutzer bei ausstehendem Draft „Anfrage zurückziehen" klickt
- **THEN** wird `DELETE /api/members/{id}/change-drafts/{id}` gesendet
- **AND** die Schalter werden auf den Server-Stand (`ownMember.dsgvo_*`) zurückgesetzt
- **AND** der Anfrage-Button ist wieder gesperrt (kein Diff)

