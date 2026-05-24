## 1. Datenbankmigration

- [x] 1.1 `009_split_membership_request_name.up.sql` erstellen: `first_name` + `last_name` zu `membership_requests` hinzufügen, Bestandsdaten aufteilen, `name`-Spalte entfernen
- [x] 1.2 `009_split_membership_request_name.down.sql` erstellen: `name` als Concat zurückführen, neue Spalten entfernen
- [x] 1.3 Migration lokal testen: `make migrate-up` und `make migrate-down` erfolgreich

## 2. Backend — Auth-Handler

- [x] 2.1 Struct für `POST /api/auth/request-membership` auf `FirstName`/`LastName` umstellen (statt `Name`)
- [x] 2.2 INSERT-Query in `RequestMembership`-Handler auf `first_name`, `last_name` anpassen
- [x] 2.3 SELECT-Queries in `ApproveMembership` und `RejectMembership` auf `first_name`, `last_name` umstellen (für E-Mail-Text)
- [x] 2.4 Response-Struct für `GET /api/admin/membership-requests` auf `first_name`, `last_name` umstellen

## 3. Frontend — Beitrittsanfrage-Formular

- [x] 3.1 `RequestMembershipPage.tsx`: State-Variable `name` durch `firstName` + `lastName` ersetzen
- [x] 3.2 Zwei separate Eingabefelder (Vorname, Nachname) mit korrekten Labels einfügen
- [x] 3.3 POST-Body anpassen: `{ first_name, last_name, email, comment }`

## 4. Frontend — Admin-Ansicht Beitrittsanfragen

- [x] 4.1 `MembershipRequestsPage.tsx`: Typ-Definition `name: string` durch `first_name: string; last_name: string` ersetzen
- [x] 4.2 Anzeige von `r.name` durch `r.first_name + ' ' + r.last_name` ersetzen
