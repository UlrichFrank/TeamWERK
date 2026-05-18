### Requirement: Mitglieder-Navigation rollenbasiert einschränken
Der Mitglieder-Tab in der Sidebar SHALL nur für Nutzer mit Rolle `admin` oder `trainer` sichtbar sein. Nutzer mit Rolle `spieler` oder `elternteil` sehen diesen Eintrag nicht.

#### Scenario: spieler sieht keinen Mitglieder-Tab
- **WHEN** ein Nutzer mit Rolle `spieler` eingeloggt ist
- **THEN** erscheint kein „Mitglieder"-Eintrag in der Navigation

#### Scenario: elternteil sieht keinen Mitglieder-Tab
- **WHEN** ein Nutzer mit Rolle `elternteil` eingeloggt ist
- **THEN** erscheint kein „Mitglieder"-Eintrag in der Navigation

#### Scenario: admin sieht Mitglieder-Tab
- **WHEN** ein Nutzer mit Rolle `admin` oder `trainer` eingeloggt ist
- **THEN** ist der „Mitglieder"-Eintrag in der Navigation sichtbar

---

### Requirement: Profil zeigt verknüpfte Elternteile für Spieler
Das Profil eines `spieler` SHALL alle Nutzer anzeigen, die via `family_links` als Elternteil mit dem eigenen Mitgliedsprofil verknüpft sind. Die Anzeige ist read-only.

#### Scenario: Spieler mit verknüpften Elternteilen
- **WHEN** ein `spieler` das Profil aufruft und sein Mitgliedsprofil mindestens einen Eintrag in `family_links` hat
- **THEN** zeigt das Profil eine Sektion „Meine Familie" mit Name und E-Mail jedes verknüpften Elternteils

#### Scenario: Spieler ohne verknüpfte Elternteile
- **WHEN** ein `spieler` das Profil aufruft und kein Eintrag in `family_links` existiert
- **THEN** wird die Sektion „Meine Familie" nicht angezeigt

#### Scenario: Spieler ohne Mitgliedsprofil
- **WHEN** ein `spieler` das Profil aufruft und kein `members`-Eintrag mit seiner `user_id` existiert
- **THEN** wird weder eine Mitgliedskarte noch eine Familiensektion angezeigt

---

### Requirement: API liefert Elternteile im Profil-Endpoint
`GET /api/profile/me` SHALL für Nutzer mit Rolle `spieler` die verknüpften Elternteile zurückgeben.

#### Scenario: Response enthält parents-Feld für spieler
- **WHEN** ein `spieler` `GET /api/profile/me` aufruft
- **THEN** enthält die Response ein Feld `parents` mit einem Array von Objekten (`id`, `name`, `email`) aller verknüpften Elternteile

#### Scenario: parents ist leer wenn keine Verknüpfung
- **WHEN** ein `spieler` `GET /api/profile/me` aufruft und keine `family_links` existieren
- **THEN** ist `parents` ein leeres Array `[]`

#### Scenario: elternteil erhält kein parents-Feld
- **WHEN** ein `elternteil` `GET /api/profile/me` aufruft
- **THEN** enthält die Response kein `parents`-Feld (Verhalten unverändert)
