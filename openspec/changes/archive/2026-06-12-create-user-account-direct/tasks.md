## 1. Backend: Handler CreateUser

- [x] 1.1 In `internal/auth/handler.go` den Handler `CreateUser` implementieren: JSON-Body mit `email`, `first_name`, `last_name`, `password` lesen, Passwort bcrypt-hashen, INSERT in `users` (Rolle fest `standard`), bei UNIQUE-Verletzung 409 zurückgeben, sonst 201 + `{ "id": <lastInsertId> }`
- [x] 1.2 In `cmd/teamwerk/main.go` die Route `r.Post("/api/users", authH.CreateUser)` in der Vorstand-Gruppe registrieren

## 2. Frontend: Dropdown-Eintrag und Modal

- [x] 2.1 In `AdminUsersPage.tsx` die Hilfsfunktion `generatePassword()` hinzufügen (16 Zeichen, `crypto.getRandomValues`, Zeichenvorrat: a–z A–Z 0–9 + Sonderzeichen)
- [x] 2.2 State für das neue Modal anlegen: `showCreateModal`, `createEmail`, `createFirstName`, `createLastName`, `createPassword`, `createLoading`, `createError`, `createCopied`
- [x] 2.3 Dropdown um Eintrag „Account anlegen" erweitern (öffnet Modal, generiert sofort ein Passwort)
- [x] 2.4 Modal „Account anlegen" implementieren: Felder E-Mail, Vorname, Nachname, Passwort (readonly) mit Copy-Button (`Copy`→`Check`-Icon für 2 s) und „Neu generieren"-Button
- [x] 2.5 Submit-Handler: `POST /api/users`, bei Erfolg Modal schließen + `refreshUsers()`, bei 409 Fehlermeldung „E-Mail bereits vergeben" anzeigen
- [x] 2.6 `useEscapeKey`-Aufruf um den neuen Modal-Close-Handler erweitern
