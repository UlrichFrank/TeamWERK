## Why

Persönliche Daten (Name, Adresse) im Profil gehen nach dem Speichern verloren — entweder sofort (Andrea, kein verlinktes Mitglied) oder sobald eine ausstehende Änderungsanfrage angenommen oder abgelehnt wird (Ulrich, verlinktes Mitglied).

Ursachen:
1. **Silent failure im Backend:** `UpdateProfile` verwendet `nullableString()` für `first_name`/`last_name`. Leere Strings werden zu `nil`, was den SQLite-NOT-NULL-Constraint verletzt. Der Fehler wird ignoriert, der Handler antwortet trotzdem `204` — die Daten werden nicht gespeichert, die UI zeigt „Gespeichert".
2. **Falscher Save-Pfad für verlinkte Mitglieder:** Nutzer mit verlinktem Mitglied speichern ausschließlich über einen Change-Request-Draft. Die `users`-Tabelle wird nie aktualisiert. Wenn der Draft gelöscht wird (Accept/Reject), sind die Profildaten weg.
3. **Race condition beim Laden:** `ProfileProfilTab` macht einen eigenen `GET /profile/me`-Call und liest `users.street/zip/city` (für verlinkte Mitglieder leer). Dieser Call kann später ankommen als die useEffects, die korrekte Werte aus `ownMember` oder dem Draft gesetzt haben, und überschreibt sie mit leeren Strings.

## What Changes

**Zwei verschiedene Datentypen, zwei verschiedene Modelle:**

| Daten | Speicherort | Save-Modell |
|---|---|---|
| Name + Adresse (Kontakt-Tab) | `users`-Tabelle | Sofort gespeichert; für verlinkte Mitglieder zusätzlich Change-Request |
| Bankdaten (Bankdaten-Tab) | `members`-Tabelle | Nur via Change-Request (bleibt unverändert) |

**Änderungen am Kontakt-Tab (Name + Adresse):**

- `PUT /profile/me` wird jetzt für alle Nutzer aufgerufen (auch für verlinkte Mitglieder)
- `users.first_name`/`last_name` werden ohne `nullableString` geschrieben; Fehler werden nicht mehr ignoriert
- Das Frontend lädt Name und Adresse immer aus der `users`-Tabelle, nie aus `ownMember`
- Der Draft (für den Mitglieds-Datensatz) zeigt nur noch den Pending-Status an — er befüllt das Formular nicht mehr
- Button-Text ist immer „Speichern" (nicht mehr konditionell „Änderung anfordern")
- Ein abgelehnter oder zurückgezogener Draft löscht nur den Mitglieds-Update-Request; die Nutzerdaten in `users` bleiben intakt

**Bankdaten-Tab bleibt unverändert:** IBAN und Kontoinhaber liegen ausschließlich im Mitglieds-Datensatz. Speichern erfolgt weiterhin nur per Change-Request. Das Formular zeigt den Draft-Wert wenn vorhanden, sonst den Mitglieds-Wert — das ist korrekt so.

## Capabilities

### Modified Capabilities
- `profile-personal-data`: Nutzer können Name und Adresse in ihrem Profil speichern; die Daten landen sofort in der `users`-Tabelle und bleiben erhalten, unabhängig vom Status einer eventuellen Änderungsanfrage für den Mitglieds-Datensatz

## Impact

- `internal/members/handler.go`: `UpdateProfile` — `nullableString` für NOT-NULL-Spalten entfernen, Fehlerbehandlung ergänzen
- `web/src/components/profile/ProfileProfilTab.tsx`: Save-Flow, Lade-Logik, useEffect-Abhängigkeiten, Button-Text
- Keine neuen DB-Tabellen, keine neuen Routen, keine neuen Abhängigkeiten
