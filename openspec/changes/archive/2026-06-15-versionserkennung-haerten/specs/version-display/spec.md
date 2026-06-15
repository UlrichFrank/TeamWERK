## MODIFIED Requirements

### Requirement: Versions-Button öffnet Changelog-Modal

Die Versionsanzeige im Sidebar-Footer SHALL ein klickbarer Button sein. Klick darauf öffnet das `ChangelogModal`. Der Button SHALL **auch im Dev-Modus** sichtbar sein und in diesem Fall `v dev` anzeigen — damit ist erkennbar, dass die Anzeige funktioniert, ohne dass eine SSE-Verbindung läuft.

#### Scenario: Klick auf Versions-Button öffnet Modal (unverändert)

- **WHEN** ein eingeloggter Nutzer auf den Versions-Button (`v abc1234`) in der Sidebar klickt
- **THEN** öffnet sich das `ChangelogModal`

#### Scenario: Modal schließbar per ✕ und Escape (unverändert)

- **WHEN** das `ChangelogModal` offen ist
- **THEN** kann es per ✕-Button oder Escape-Taste geschlossen werden

#### Scenario: Dev-Modus zeigt „v dev"

- **WHEN** die App lokal mit `pnpm dev` läuft (`import.meta.env.DEV === true`)
- **THEN** zeigt der Versions-Button in der Sidebar `v dev`
- **THEN** öffnet ein Klick auf den Button das `ChangelogModal` (mit dem aktuell gebauten `CHANGELOG.md`)
