## MODIFIED Requirements

### Requirement: Eingeloggter Nutzer kann Vorname und Nachname ändern
Das System SHALL Vorname und Nachname im **Profil-Tab** (nicht im Konto-Tab) verwalten. Für Nutzer ohne Mitgliedsverknüpfung werden die Felder direkt in `users` gespeichert. Für verknüpfte Mitglieder erfolgt die Änderung ausschließlich über den Profil-Bundle-Change-Request (`field_name: "profil"`). Der Konto-Tab enthält keine Namensfelder mehr.

#### Scenario: Konto-Tab zeigt keinen Namen
- **WHEN** ein Nutzer den Konto-Tab der Profilseite öffnet
- **THEN** sind dort keine Vor-/Nachname-Felder vorhanden — nur E-Mail (read-only), Passwort ändern, E-Mail ändern

#### Scenario: Profil-Tab zeigt Namensfelder
- **WHEN** ein Nutzer den Profil-Tab öffnet
- **THEN** sind Vorname- und Nachname-Felder sichtbar und mit den aktuellen Werten vorbelegt

#### Scenario: Nicht-Mitglied speichert Namen direkt
- **WHEN** ein Nutzer ohne Mitgliedsverknüpfung Vorname/Nachname ändert und „Speichern" klickt
- **THEN** werden `users.first_name` und `users.last_name` aktualisiert und HTTP 204 zurückgegeben

#### Scenario: Mitglied beantragt Namensänderung via Bundle
- **WHEN** ein verknüpftes Mitglied Vorname/Nachname ändert und „Änderung anfordern" klickt
- **THEN** wird ein Profil-Bundle-Draft erstellt (siehe [[profil-change-request]]) — keine direkte Speicherung in `members`
