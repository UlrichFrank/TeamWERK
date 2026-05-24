## Context

Die `users`-Tabelle wurde bereits mit Migration 007 auf `first_name` + `last_name` umgestellt. Backend und Frontend für Profil und Registrierung sind vollständig auf die neuen Felder migriert. Noch ausständig ist die `membership_requests`-Tabelle, die weiterhin ein einzelnes `name`-Feld besitzt. Das Beitrittsanfrage-Formular (`RequestMembershipPage`) sendet ebenfalls ein einzelnes `name`-Feld.

## Goals / Non-Goals

**Goals:**
- `membership_requests.name` in `first_name` + `last_name` aufteilen (Migration + Datenmigration)
- Backend-Handler für Beitrittsanfrage (`POST /api/auth/request-membership`, Genehmigung, Ablehnung) auf neue Felder umstellen
- Frontend-Formular `RequestMembershipPage` auf zwei separate Felder umstellen
- Admin-Ansicht `MembershipRequestsPage` zeigt `Vorname + Nachname`

**Non-Goals:**
- Keine Synchronisation von `users.first_name`/`last_name` nach `members.first_name`/`last_name` — beide Tabellen sind unabhängig. Der bestehende Workflow „Nutzer stellt Namensänderung an → Vorstand aktualisiert Mitgliedsdaten" bleibt vollständig erhalten.
- Keine weiteren Tabellen betroffen — `users` ist bereits fertig
- Keine Änderungen an JWT-Claims (enthalten kein `name`)
- Keine E-Mail-Template-Änderungen (E-Mail-Text kann `first_name || ' ' || last_name` konkatenieren)

## Decisions

### SQLite-Datenmigration via ALTER TABLE

SQLite unterstützt kein `ALTER COLUMN`. Der bestehende Ansatz aus Migration 007 (neue Spalten adden, UPDATE, DROP) wird identisch angewendet.

Bestandsdaten-Heuristik: erstes Wort → `first_name`, Rest → `last_name`. Falls kein Leerzeichen, gesamter Wert → `first_name`, `last_name` leer. (Identisch zu 007 für users.)

### Keine Team-ID mehr in Beitrittsanfrage

Der bestehende `POST /api/auth/request-membership`-Handler nimmt `team_id` entgegen, obwohl die aktuelle Implementierung es als optional / nullable behandelt. Dieses Design bleibt unverändert — es ist außerhalb des Scope dieser Änderung.

### Nächste freie Migrationsnummer: 009

Migration 008 (`games_end_time`) ist die letzte. Die neue Migration heißt `009_split_membership_request_name`.

## Risks / Trade-offs

- **Bestehende Anfragen mit schlechten Daten** (z.B. Name ohne Leerzeichen): `last_name` bleibt leer. Admins sehen dann nur den Vornamen. Akzeptabel — betrifft nur historische Einträge, und solche Anfragen sind ohnehin manuell zu prüfen.
- **Laufende Tokens / offene Requests:** Beim Deployen laufende pending-Requests werden durch die Migration automatisch aufgeteilt; keine Nutzerinteraktion nötig.

## Migration Plan

1. `009_split_membership_request_name.up.sql`: Spalten hinzufügen, UPDATE, DROP `name`
2. `009_split_membership_request_name.down.sql`: `name` zurückführen als Concat
3. Backend-Handler anpassen
4. Frontend anpassen
5. `make deploy` → Migrations laufen automatisch
