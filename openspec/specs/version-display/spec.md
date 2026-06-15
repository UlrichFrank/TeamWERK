## ADDED Requirements

### Requirement: Versions-Button öffnet Changelog-Modal

Die Versionsanzeige im Sidebar-Footer SHALL ein klickbarer Button sein. Klick darauf öffnet das `ChangelogModal`. Der Button SHALL **auch im Dev-Modus** sichtbar sein und in diesem Fall `v dev` anzeigen — damit ist erkennbar, dass die Anzeige funktioniert, ohne dass eine SSE-Verbindung läuft.

#### Scenario: Klick auf Versions-Button öffnet Modal
- **WHEN** ein eingeloggter Nutzer auf den Versions-Button (`v abc1234`) in der Sidebar klickt
- **THEN** öffnet sich das `ChangelogModal`

#### Scenario: Modal schließbar per ✕ und Escape
- **WHEN** das `ChangelogModal` offen ist
- **THEN** kann es per ✕-Button oder Escape-Taste geschlossen werden

#### Scenario: Dev-Modus zeigt „v dev"
- **WHEN** die App lokal mit `pnpm dev` läuft (`import.meta.env.DEV === true`)
- **THEN** zeigt der Versions-Button in der Sidebar `v dev`
- **THEN** öffnet ein Klick auf den Button das `ChangelogModal` (mit dem aktuell gebauten `CHANGELOG.md`)

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
