## Context

TeamWERK hat zwei Tabellentypen für personenbezogene Daten:

- **`users`-Tabelle**: Kontodaten des eingeloggten Nutzers (`first_name`, `last_name`, `street`, `zip`, `city`). Wird beim Login/JWT nicht verwendet, aber für Profilanzeige und Benachrichtigungen.
- **`members`-Tabelle**: Offizieller Vereinsdatensatz (`first_name`, `last_name`, `street`, `zip`, `city`, `iban`, `account_holder`, …). Änderungen erfordern Admin-Freigabe via `member_change_drafts`.

Der `ProfileProfilTab` (Kontakt-Tab) war bisher so implementiert:
- Nutzer **ohne** verlinktes Mitglied: Daten aus `users`, Save via `PUT /profile/me`
- Nutzer **mit** verlinktem Mitglied: Daten aus `members`/Draft, Save nur via Change-Request

Das führte zu Datenverlust auf zwei Wegen:
1. `UpdateProfile` verwendet `nullableString()` auf NOT-NULL-Spalten (`first_name`, `last_name`). Leere Strings werden zu `nil`, der SQLite-NOT-NULL-Constraint schlägt fehl, der Fehler wird ignoriert → ganzer UPDATE schlägt lautlos fehl.
2. Für verlinkte Mitglieder wurde `users.street/first_name/…` nie geschrieben. Nach Draft-Löschung (Accept oder Reject) zeigte das Formular leere Felder.

## Goals / Non-Goals

**Goals:**
- Name + Adresse werden für alle Nutzer sofort in `users`-Tabelle gespeichert
- Für verlinkte Mitglieder entsteht zusätzlich ein Change-Request für den Mitglieds-Datensatz
- Datenverlust nach Draft-Accept/Reject ist ausgeschlossen
- `PUT /profile/me` ist fehlertolerant und schlägt nicht lautlos fehl

**Non-Goals:**
- Bankdaten-Tab: bleibt unverändert (IBAN/Kontoinhaber nur im Mitglieds-Datensatz, nur via Change-Request)
- Keine Migration bestehender `members.street`-Werte nach `users.street`
- Kein neues Datenbank-Schema

## Decisions

### D1: `users`-Tabelle als Single Source of Truth für Name + Adresse

**Entscheidung:** `PUT /profile/me` wird für alle Nutzer aufgerufen — unabhängig davon, ob ein verlinktes Mitglied existiert. Die `users`-Tabelle enthält immer den aktuell gespeicherten Stand.

**Rationale:** Die `users`-Tabelle ist dem Nutzer "eigen" — kein Admin-Approval-Workflow nötig. Kontaktdaten (Telefonnummer, Adresse für Benachrichtigungen) sind unabhängig vom Vereinsdatensatz. Der Mitglieds-Datensatz ist der offizielle Vereins-Record und darf weiterhin Approval-pflichtig bleiben.

**Alternativen:** Nur `members`-Tabelle verwenden → Datenverlust bei Draft-Reject bleibt, Adresse für Nicht-Mitglieder hätte keinen Speicherort. Beide Tabellen synchron halten → unnötige Komplexität, keine klare Ownership.

### D2: Zwei Writes beim Speichern für verlinkte Mitglieder

**Entscheidung:** Für Nutzer mit verlinktem Mitglied führt Save zwei Operationen durch:
1. `PUT /profile/me` → `users`-Tabelle (sofort, kein Approval)
2. `POST change-request (field_name='profil')` → Draft für Mitglieds-Datensatz (wartet auf Admin)

**Rationale:** Trennung von Nutzer-Kontaktdaten (immer aktuell) und Vereinsdatensatz (approval-pflichtig). Wenn der Admin den Draft ablehnt oder der Nutzer ihn zurückzieht, sind die Nutzer-Daten unberührt. Der Draft ist kein Sicherungsmechanismus mehr, sondern nur noch ein Änderungsauftrag an den Admin.

### D3: Formular lädt immer aus `users`-Tabelle

**Entscheidung:** Das Formular zeigt immer den Stand aus der `users`-Tabelle — nicht aus `ownMember` und nicht aus dem Draft. Der Draft-Banner zeigt nur noch den Pending-Status, befüllt das Formular aber nicht mehr.

**Rationale:** Konsistentes Verhalten: was der Nutzer zuletzt gespeichert hat, sieht er auch. Der Draft-Wert ist der Stand, den der Admin noch nicht freigegeben hat — das ist eine separate Information, kein aktueller Profilstand des Nutzers.

**Konsequenz:** Das zweite `useEffect` in `ProfileProfilTab` verliert seinen `ownMember`-Zweig. Die Dependency `[ownMember?.id]` entfällt — der Effekt läuft einmalig on mount.

### D4: `nullableString` nicht für NOT-NULL-Spalten

**Entscheidung:** `UpdateProfile` übergibt `first_name` und `last_name` direkt (nicht via `nullableString`). Für nullable Felder (`street`, `zip`, `city`) bleibt `nullableString` erhalten. Fehler von `ExecContext` werden nicht mehr ignoriert.

**Rationale:** `nullableString("")` gibt `nil` zurück, was NOT-NULL verletzt. Leerer String `""` ist ein valider Wert für `first_name NOT NULL DEFAULT ''`.

## Risks / Trade-offs

- **Divergenz `users` vs. `members`**: Nach dem Fix können `users.first_name` und `members.first_name` temporär unterschiedlich sein (bis Draft accepted). Das ist gewollt — sie repräsentieren verschiedene Dinge (Login-Name vs. Vereinsdatensatz).
- **Kein Backfill**: Bestehende Nutzer mit verlinktem Mitglied, die ihre Adresse bisher nur im Mitglieds-Datensatz haben, sehen nach dem Fix zunächst leere Adressfelder in `users`. Sie müssen einmalig speichern. Alternativ könnte ein einmaliger DB-Backfill `users.street = members.street WHERE users.id = members.user_id` durchgeführt werden — aber das ist außerhalb des Scope.

## Migration Plan

Keine DB-Migration nötig. Deployment: normales `make deploy` genügt.
