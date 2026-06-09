## ADDED Requirements

### Requirement: Versions-Button öffnet Changelog-Modal

Die Versionsanzeige im Sidebar-Footer SHALL ein klickbarer Button sein. Klick darauf öffnet das `ChangelogModal`.

#### Scenario: Klick auf Versions-Button öffnet Modal
- **WHEN** ein eingeloggter Nutzer auf den Versions-Button (`v abc1234`) in der Sidebar klickt
- **THEN** öffnet sich das `ChangelogModal`

#### Scenario: Modal schließbar per ✕ und Escape
- **WHEN** das `ChangelogModal` offen ist
- **THEN** kann es per ✕-Button oder Escape-Taste geschlossen werden

### Requirement: ChangelogModal zeigt Versionshistorie

Das `ChangelogModal` SHALL `CHANGELOG.md` fetchen, parsen und als Datum-Gruppen mit Badges darstellen.

#### Scenario: Datum-Gruppen werden angezeigt
- **WHEN** das Modal geöffnet wird
- **THEN** zeigt es Einträge gruppiert nach Datum (neueste zuerst)
- **THEN** jeder Eintrag hat ein farbiges `[feat]`- oder `[fix]`-Badge, einen Scope-Text und eine Beschreibung

#### Scenario: Ladeindikator während Fetch
- **WHEN** das Modal geöffnet wird und `CHANGELOG.md` noch lädt
- **THEN** wird ein Lade-Indikator angezeigt

#### Scenario: Fehler beim Laden
- **WHEN** `CHANGELOG.md` nicht geladen werden kann
- **THEN** zeigt das Modal eine Fehlermeldung

### Requirement: Update-Banner öffnet Changelog-Modal

Der Update-Banner SHALL bei Klick auf „Details" das `ChangelogModal` öffnen statt Inline-Text anzuzeigen.

#### Scenario: Details-Button öffnet Modal
- **WHEN** der Update-Banner sichtbar ist und der Nutzer auf „Details" klickt
- **THEN** öffnet sich das `ChangelogModal`
