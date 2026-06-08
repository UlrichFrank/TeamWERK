# kinder-proxy-account Specification

## Purpose
Kinder ohne eigene E-Mail-Adresse erhalten einen `users`-Eintrag mit `can_login = 0` (Proxy-Account). Dieser Account dient als Ankerpunkt für das Dienstsystem und RSVP-Logik, ohne dass das Kind sich einloggen kann. Ein Admin kann den Proxy-Account später zu einem vollständigen Login-Account aktivieren.

## Requirements

### Requirement: Admin legt Proxy-Account für ein Mitglied an
Das System SHALL einem Admin ermöglichen, für ein Mitglied einen Proxy-Account (`can_login = 0`) anzulegen. Ein Proxy-Account benötigt keine E-Mail-Adresse. Nach dem Anlegen wird `members.user_id` auf den neuen Account gesetzt.

#### Scenario: Proxy-Account ohne E-Mail anlegen
- **WHEN** ein Admin `POST /api/members/{id}/proxy-account` aufruft (kein E-Mail-Feld)
- **THEN** legt das System einen `users`-Datensatz mit `can_login = 0` und `name = members.first_name + ' ' + members.last_name` an
- **THEN** wird `members.user_id` auf die neue `users.id` gesetzt
- **THEN** antwortet der Server mit HTTP 201 und `{ user_id: <neue id> }`

#### Scenario: Proxy-Account mit optionaler E-Mail
- **WHEN** ein Admin `POST /api/members/{id}/proxy-account` mit `{ email: "elternemail@example.com" }` aufruft
- **THEN** wird die E-Mail im Proxy-Account gespeichert (kein Unique-Konflikt mit dem Eltern-Account, da partieller Index nur für `can_login = 1` gilt)

#### Scenario: Mitglied hat bereits einen Account
- **WHEN** ein Admin versucht, für ein Mitglied mit bestehendem `user_id` einen Proxy-Account anzulegen
- **THEN** antwortet der Server mit HTTP 409

#### Scenario: Proxy-Account erscheint in Admin-Nutzerliste
- **WHEN** ein Admin `GET /api/users` aufruft
- **THEN** sind Proxy-Accounts (`can_login = 0`) in der Antwort als solche markiert (z.B. `"proxy": true`) und optisch vom regulären Login unterscheidbar

### Requirement: Proxy-Account kann nicht einloggen
Das System SHALL den Login für alle `users`-Einträge mit `can_login = 0` verweigern.

#### Scenario: Login-Versuch mit Proxy-Account-E-Mail
- **WHEN** jemand `POST /api/auth/login` mit der E-Mail-Adresse eines Proxy-Accounts aufruft
- **THEN** antwortet der Server mit HTTP 401 (identische Fehlermeldung wie bei falschem Passwort — keine Enumeration)

### Requirement: Admin aktiviert Proxy-Account zu vollständigem Login-Account
Das System SHALL einem Admin ermöglichen, einen Proxy-Account zu aktivieren: `can_login` auf 1 setzen, E-Mail eintragen und optional eine Einladungsmail versenden.

#### Scenario: Admin aktiviert Proxy-Account
- **WHEN** ein Admin `PUT /api/users/{id}` mit `{ can_login: 1, email: "kind@example.com" }` für einen Proxy-Account aufruft
- **THEN** wird `can_login = 1` und die E-Mail gesetzt
- **THEN** kann das Konto ab sofort für Login und Passwort-Reset genutzt werden

#### Scenario: Aktivierung schlägt fehl, wenn E-Mail bereits von einem Login-Account genutzt wird
- **WHEN** ein Admin versucht, einen Proxy-Account mit einer E-Mail zu aktivieren, die bereits ein anderer Login-Account (`can_login = 1`) trägt
- **THEN** antwortet der Server mit HTTP 409
