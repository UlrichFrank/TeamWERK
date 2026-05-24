## ADDED Requirements

### Requirement: Profilfelder als atomarer Bundle-Change-Request
Das System SHALL einen neuen Change-Request-Typ `field_name: "profil"` unterstützen, der `first_name`, `last_name`, `street`, `zip`, `city` und `iban` eines Mitglieds in einem einzigen atomaren Request bündelt.

#### Scenario: Bundle-Request wird erstellt
- **WHEN** ein verknüpftes Mitglied `POST /members/{id}/change-request` mit `{ "field_name": "profil", "new_value": { "first_name": "…", "last_name": "…", "street": "…", "zip": "…", "city": "…", "iban": "…" } }` aufruft
- **THEN** legt das System einen Draft mit `field_name = "profil"` an (oder ersetzt einen bestehenden per UPSERT) und antwortet mit HTTP 201

#### Scenario: Nur ein offener Profil-Draft pro Mitglied
- **WHEN** ein Mitglied einen zweiten Profil-Bundle-Request stellt, während ein erster noch offen ist
- **THEN** wird der bestehende Draft überschrieben (UPSERT) — es gibt immer maximal einen offenen Profil-Draft

#### Scenario: Admin akzeptiert Profil-Draft
- **WHEN** ein Admin/Trainer `POST /members/{id}/change-drafts/{draftId}/accept` für einen `field_name: "profil"`-Draft aufruft
- **THEN** werden `members.first_name`, `last_name`, `street`, `zip`, `city`, `iban` in einem UPDATE geschrieben und der Draft gelöscht

#### Scenario: Ungültiger field_name wird abgelehnt
- **WHEN** `POST /members/{id}/change-request` mit einem unbekannten `field_name` aufgerufen wird
- **THEN** antwortet das System mit HTTP 400

### Requirement: Profil-Tab zeigt rollenabhängige Speicher-Aktion
Das Frontend SHALL im Profil-Tab unterscheiden, ob der Nutzer ein verknüpftes Mitglied hat:
- Kein Mitglied: Button „Speichern" → `PUT /profile/me` mit `first_name`, `last_name`, Adresse
- Mitglied verknüpft: Button „Änderung anfordern" → `POST /members/{id}/change-request` mit `field_name: "profil"`

#### Scenario: Nicht-Mitglied speichert direkt
- **WHEN** ein Nutzer ohne Mitgliedsverknüpfung den Profil-Tab ausfüllt und „Speichern" klickt
- **THEN** wird `PUT /profile/me` aufgerufen und die Daten sofort in `users` gespeichert

#### Scenario: Mitglied stellt Änderungsanfrage
- **WHEN** ein verknüpftes Mitglied Felder im Profil-Tab ändert und „Änderung anfordern" klickt
- **THEN** wird `POST /members/{id}/change-request` mit `field_name: "profil"` und allen Formularwerten aufgerufen

#### Scenario: Offener Draft sperrt Formular
- **WHEN** ein verknüpftes Mitglied einen offenen Profil-Draft hat
- **THEN** zeigt das Formular einen Hinweis „Änderungsanfrage ausstehend" und die Felder sind schreibgeschützt; ein „Zurückziehen"-Link ist sichtbar
