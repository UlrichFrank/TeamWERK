## ADDED Requirements

### Requirement: Nutzer hat Telefonnummern
Das System SHALL eine Tabelle `user_phones(id, user_id, label, number, sort_order)` verwalten. Label ist Freitext (keine CHECK-Constraint). Frontend schlägt Privat/Mobil/Firma vor. Jeder Nutzer kann 0..n Nummern haben.

#### Scenario: Nutzer fügt Telefonnummer hinzu
- **WHEN** Nutzer `POST /api/profile/phones` mit `{label: "Mobil", number: "+49 711 ..."}` aufruft
- **THEN** Eintrag in `user_phones` angelegt, neue ID im Response

#### Scenario: Nutzer löscht Telefonnummer
- **WHEN** Nutzer `DELETE /api/profile/phones/{id}` aufruft und die Nummer gehört ihm
- **THEN** Eintrag gelöscht

#### Scenario: Nutzer kann fremde Nummern nicht löschen
- **WHEN** Nutzer `DELETE /api/profile/phones/{id}` mit einer Nummer eines anderen Nutzers
- **THEN** HTTP 403

#### Scenario: Admin sieht Telefonnummern eines Nutzers
- **WHEN** Admin `GET /api/admin/users/{id}` aufruft
- **THEN** `phones`-Array im Response mit allen Nummern des Nutzers

### Requirement: Nutzer hat optionale Adresse
Das System SHALL `street`, `zip`, `city` auf der `users`-Tabelle speichern (nullable). Nutzer können diese selbst setzen via `PUT /api/profile/me`.

#### Scenario: Nutzer setzt eigene Adresse
- **WHEN** Nutzer `PUT /api/profile/me` mit `street`, `zip`, `city`
- **THEN** Felder gespeichert, bei `GET /api/profile/me` zurückgegeben

### Requirement: Nutzer hat Profilbild
Upload via `POST /api/upload/user-photo` (selbe Regeln wie member-photo: ≤ 5 MB, jpeg/png/webp). Pfad in `users.photo_path`.

#### Scenario: Nutzer lädt Profilbild hoch
- **WHEN** Nutzer `POST /api/upload/user-photo` mit gültigem Bild
- **THEN** Datei gespeichert unter `storage/uploads/user-photos/`, `users.photo_path` gesetzt, `photo_url` im Response

### Requirement: Nutzer steuert Sichtbarkeit seiner Kontaktdaten
Das System SHALL eine Tabelle `user_visibility(user_id PK, phones_visible, address_visible, photo_visible)` verwalten. Default: alle Felder nicht sichtbar (Privacy by Default). Nutzer setzen Sichtbarkeit via `PUT /api/profile/visibility`.

#### Scenario: Nutzer schaltet Telefon frei
- **WHEN** Nutzer `PUT /api/profile/visibility` mit `{phones_visible: true}`
- **THEN** Zeile in `user_visibility` per INSERT OR REPLACE gesetzt

#### Scenario: Anderer Nutzer sieht freigegebene Telefonnummern
- **WHEN** Nutzer B `GET /api/members/{id}` aufruft, verknüpfter Nutzer A hat `phones_visible=true`
- **THEN** `phones`-Array im Response mit Nummern von Nutzer A

#### Scenario: Nicht freigegebene Daten bleiben verborgen
- **WHEN** Nutzer B ruft Mitglied ab, verknüpfter Nutzer A hat `phones_visible=false`
- **THEN** `phones` fehlt oder ist leer im Response

### Requirement: Nutzer sieht eigene Sichtbarkeitseinstellungen
`GET /api/profile/me` MUSS die aktuellen Visibility-Einstellungen zurückgeben.

#### Scenario: Nutzer liest eigenes Profil mit Visibility
- **WHEN** Nutzer `GET /api/profile/me`
- **THEN** Response enthält `visibility: {phones_visible, address_visible, photo_visible}`
