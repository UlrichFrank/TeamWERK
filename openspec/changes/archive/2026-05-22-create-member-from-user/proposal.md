## Why

Wenn sich jemand über den Einladungslink registriert, existiert danach zwar ein Login-Account, aber kein Mitgliedsdatensatz. Admins müssen das Mitglied bisher separat anlegen und manuell verknüpfen. Ein Button direkt in der Nutzerliste verkürzt diesen Prozess auf einen Klick.

## What Changes

- Neuer „Mitglied anlegen"-Button in jeder Zeile der Nutzerliste (`/admin/nutzer`)
- Button ist nur sichtbar, wenn der Nutzer noch **kein verknüpftes Mitglied** hat
- Beim Klick wird ein Mitglied mit den bereits bekannten Daten (Name aus Account) vorausgefüllt angelegt und sofort mit dem Account verknüpft
- Das neu erstellte Mitglied kann danach normal in der Mitgliederverwaltung bearbeitet werden

## Capabilities

### New Capabilities

- `create-member-from-user`: Mitglied direkt aus einem Nutzer-Account erstellen und verknüpfen

### Modified Capabilities

<!-- keine bestehenden Specs betroffen -->

## Impact

- **Backend:** Neuer Endpoint `POST /api/admin/users/{id}/create-member` im `members`-Package
- **Frontend:** `AdminUsersPage` — Button „Mitglied anlegen" in der Nutzerzeile, nur wenn `member_id` fehlt
- **DB:** Kein Schema-Change nötig — `members.user_id` FK existiert bereits
