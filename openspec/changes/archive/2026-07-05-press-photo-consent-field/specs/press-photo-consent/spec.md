## ADDED Requirements

### Requirement: Datenfeld Foto-Veröffentlichungs-Einwilligung

Das System SHALL pro Mitglied die Einwilligung zur öffentlichen Foto-Veröffentlichung in zwei
Spalten der `members`-Tabelle führen: `foto_veroeffentlichung` (INTEGER NOT NULL DEFAULT 0, Werte
0/1) und `foto_veroeffentlichung_date` (DATE, nullable). Die Semantik ist ausdrücklich: „Fotos
dieser Person dürfen auf öffentlichen Kanälen des Vereins (Homepage `team-stuttgart.org`,
Spielberichte) veröffentlicht werden" — abgegrenzt von `photo_visible` (nur interne
Profilbild-Sichtbarkeit im Portal).

#### Scenario: Neuanlage ohne Einwilligung (opt-in)

- **WHEN** ein Mitglied neu angelegt wird, ohne dass `foto_veroeffentlichung` gesetzt wird
- **THEN** gilt `foto_veroeffentlichung = 0` (keine Einwilligung)
- **AND** `foto_veroeffentlichung_date` ist NULL

#### Scenario: Datum wird beim Aktivieren gesetzt

- **WHEN** `foto_veroeffentlichung` von 0 auf 1 wechselt
- **THEN** setzt das System `foto_veroeffentlichung_date` auf das aktuelle Datum
- **AND** beim Wechsel von 1 auf 0 wird das Datum nicht neu gesetzt

### Requirement: Bestandsmigration setzt Einwilligung auf „an"

Die Migration `022` SHALL die neuen Spalten anlegen und für **alle zum Migrationszeitpunkt
bestehenden** Mitglieder `foto_veroeffentlichung = 1` sowie `foto_veroeffentlichung_date` auf das
Migrationsdatum setzen. Der Spaltendefault für später angelegte Mitglieder SHALL 0 bleiben.

#### Scenario: Bestand bekommt Einwilligung

- **WHEN** die Migration `022` auf eine DB mit bestehenden Mitgliedern läuft
- **THEN** haben alle bestehenden Mitglieder `foto_veroeffentlichung = 1` mit gesetztem `_date`

#### Scenario: Rollback entfernt die Spalten

- **WHEN** die Down-Migration `022` läuft
- **THEN** sind die Spalten `foto_veroeffentlichung` und `foto_veroeffentlichung_date` entfernt

### Requirement: Feld in Member-API und Draft-Workflow

Die Member-API (`GET`/`POST`/`PUT /api/members`) SHALL das Feld `foto_veroeffentlichung`
(und `foto_veroeffentlichung_date` lesend) transportieren. Änderungen durch Vorstand SHALL
direkt geschrieben werden; Selbstauskunfts-Änderungen SHALL über den bestehenden
`field_name='dsgvo'`-Draft laufen, dessen `old_value`/`new_value`-Payload um den Schlüssel
`foto_veroeffentlichung` erweitert wird.

#### Scenario: Vorstand setzt Einwilligung direkt

- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` (oder Rolle `admin`) das Mitglied speichert und `foto_veroeffentlichung` ändert
- **THEN** wird der Wert direkt in `members` geschrieben, inkl. `_date`-Logik

#### Scenario: DSGVO-Draft trägt das Feld

- **WHEN** ein `dsgvo`-Draft angelegt oder angewendet wird
- **THEN** enthalten `old_value` und `new_value` neben `verarbeitung`/`weitergabe` auch `foto_veroeffentlichung`
- **AND** beim Annehmen wird `foto_veroeffentlichung` (mit `_date`-Logik) auf das Mitglied übernommen

### Requirement: Spielbericht-Publisher nutzt Foto-Einwilligung

Der Spielbericht SHALL den Warnhinweis „Mitglieder ohne Foto-Freigabe" anhand des Feldes
`foto_veroeffentlichung = 0` ermitteln und nicht mehr anhand von `photo_visible`. Betroffen ist die
Query `consentMissing` in `internal/matchreports/photo_consent.go`.

#### Scenario: Fehlende Foto-Einwilligung wird gelistet

- **WHEN** ein Team-Mitglied eines Spiels `foto_veroeffentlichung = 0` hat
- **THEN** erscheint es in der Consent-Warnliste des Spielberichts
- **AND** ein Mitglied mit `foto_veroeffentlichung = 1` erscheint dort nicht (unabhängig von `photo_visible`)

### Requirement: Erklärtexte zu jedem DSGVO-Schalter

Sowohl im Profil-Datenschutz-Tab als auch in der Mitglieder-Verwaltung SHALL zu jedem der drei
Einwilligungs-Schalter (`dsgvo_verarbeitung`, `dsgvo_weitergabe`, `foto_veroeffentlichung`) ein
kurzer Erklärtext angezeigt werden, der beschreibt, was mit der Einwilligung verbunden ist.

#### Scenario: Erklärtexte sind sichtbar

- **WHEN** der DSGVO-Block im Profil oder in der Mitglieder-Verwaltung gerendert wird
- **THEN** steht unter jedem der drei Schalter ein erläuternder Text zu seiner Bedeutung
