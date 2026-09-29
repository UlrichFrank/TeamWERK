## ADDED Requirements

### Requirement: Verbindliche Typografie je Überschriften-Rolle
Jede Überschrift in der Oberfläche SHALL genau einer der folgenden Rollen angehören. Jede Rolle hat genau eine Schriftgröße, einen Schriftschnitt und eine Farbe:

| Rolle | Verwendung | Typografie |
|---|---|---|
| Seitentitel | `<h1>` einer App-Seite, auch Detailseiten | `text-2xl font-bold`, `brand-text` |
| Einstiegstitel | Kartentitel auf Einstiegsseiten (Anmelden, Registrieren, Passwort zurücksetzen) | `text-xl font-bold`, `brand-text` |
| Modal-Titel | Kopf eines Modals oder Dialogs | `text-lg font-bold`, `brand-text` |
| Abschnitt | Abschnitt einer Seite, Kartenkopf | `text-lg font-semibold`, `brand-text` |
| Unterabschnitt | Gliederung innerhalb eines Abschnitts oder Modals | `text-base font-semibold`, `brand-text` |
| Zwischenzeile | kleine Versal-Überschrift über einer Gruppe | `text-xs font-semibold uppercase`, `brand-text-muted` |

Die Rollen SHALL eine absteigende Hierarchie bilden: Keine Rolle einer tieferen Ebene ist größer oder kräftiger als die übergeordnete. Ausgenommen sind gerenderte Markdown-Inhalte (eigene Dokument-Hierarchie) und der Dateiname in der Kopfleiste eines Vollbild-Viewers, der eine Werkzeugleiste ist und keine Seitenüberschrift.

#### Scenario: Modal-Titel sind überall gleich
- **WHEN** zwei beliebige Modals der Anwendung geöffnet werden
- **THEN** haben ihre Titel dieselbe Schriftgröße und denselben Schriftschnitt

#### Scenario: Abschnittsköpfe im Profil entsprechen denen anderer Seiten
- **WHEN** ein Nutzer im Profil den Abschnitt „Persönliche Daten“ und auf der Datenschutzseite den Abschnitt „Kontakt“ ansieht
- **THEN** haben beide Überschriften dieselbe Größe, denselben Schnitt und dieselbe Farbe

### Requirement: Verbindliche Typografie für Menüeinträge
Einträge in Dropdown- und Aktionsmenüs SHALL in `text-sm` gesetzt sein. Destruktive Einträge SHALL sich nur in der Farbe (`brand-danger`) unterscheiden, nicht in Größe oder Schnitt.

#### Scenario: Menüs verschiedener Seiten wirken gleich
- **WHEN** das „Weitere Aktionen“-Menü im Kalender und das Aktionsmenü der Nutzerverwaltung geöffnet werden
- **THEN** haben die Einträge beider Menüs dieselbe Schriftgröße

### Requirement: Typografie-Rollen stammen aus einer geteilten Konstante
Die Klassen-Strings der Überschriften-Rollen und der Menüeinträge SHALL an genau einer Stelle im Frontend definiert sein und von den Aufrufern importiert werden, analog zu den Button-Klassen-Strings. Eine Überschrift SHALL Schriftgröße und Schriftschnitt nicht als Literal setzen.

Ein automatischer Test SHALL diese Regel prüfen. Begründete Ausnahmen SHALL in einer Allowlist im Test stehen. Ein Allowlist-Eintrag, dessen Fundstelle nicht mehr existiert, SHALL den Test ebenfalls fehlschlagen lassen, und Poison-Tests SHALL belegen, dass die Regel greift.

#### Scenario: Handgesetzte Überschrift fällt auf
- **WHEN** eine Seite eine `<h2>` mit `className="text-lg font-semibold …"` enthält, statt die Rollen-Konstante zu importieren
- **THEN** schlägt der Gate-Test mit Datei und Fundstelle fehl

#### Scenario: Layout-Klassen bleiben erlaubt
- **WHEN** eine Überschrift die Rollen-Konstante nutzt und zusätzlich `mb-4 truncate` anhängt
- **THEN** besteht der Gate-Test

#### Scenario: Verwaiste Allowlist-Einträge fallen auf
- **WHEN** eine Ausnahme in der Allowlist steht, die zugehörige Fundstelle aber entfernt wurde
- **THEN** schlägt der Gate-Test fehl

### Requirement: Gefüllte Buttons bauen die Button-Konstanten nicht nach
Ein gefüllter Aktions-Button (Markenfarbe Gelb oder Danger als Hintergrund) SHALL einen der verbindlichen Button-Klassen-Strings verwenden (Header-Control, Primary, Small, Danger). Eine davon abweichende Kombination aus Höhe und Schriftgröße für dieselbe Rolle SHALL NOT vorkommen. Ausgenommen sind Elemente, die keine Aktions-Buttons sind (Toggle-Chips, Badges, Checkbox-Labels).

#### Scenario: Split-Button der Dienstvorlagen passt zur Kopfzeile
- **WHEN** auf der Seite der Dienstvorlagen der Anlegen-Button neben der Seitenüberschrift steht
- **THEN** hat er dieselbe Höhe und Schriftgröße wie die Header-Controls anderer Seiten
