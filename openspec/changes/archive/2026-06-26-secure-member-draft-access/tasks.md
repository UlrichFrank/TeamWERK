## 1. Gemeinsamer Ownership-Helfer

- [x] 1.1 In `internal/members` einen Helfer `canAccessMember(claims, memberID) (bool, error)` ergänzen, der Eigentümer (`isOwn`) ∨ Elternteil (`isParentOf`) ∨ `admin` ∨ `HasFunction("vorstand")` ∨ `HasFunction("kassierer")` prüft (vorhandene Helfer aus `handler.go` wiederverwenden)
- [x] 1.2 Einen Helfer `canSubmitBankDraft(claims, memberID) (bool, error)` ergänzen, der nur Eigentümer ∨ Elternteil zulässt

## 2. Gate in den Handlern

- [x] 2.1 `GetChangeRequestsHandler` (`drafts_handlers.go`): am Anfang `canAccessMember` prüfen → bei false `403` zurückgeben, bevor Drafts/`old_value` geladen werden
- [x] 2.2 `CreateChangeRequestHandler`: am Anfang `canAccessMember` prüfen → bei false `403`, bevor der UPSERT erfolgt
- [x] 2.3 In `CreateChangeRequestHandler` zusätzlich für `field_name == "bankdaten"` `canSubmitBankDraft` erzwingen → bei false `403`
- [x] 2.4 Sicherstellen, dass das `Broadcast`-Verhalten des Schreibpfads unverändert nur bei erfolgreichem Schreiben feuert

## 3. Tests (Happy-Path + Fehlerfall)

- [x] 3.1 `GET .../change-drafts`: Eigentümer → 200; fremder `spieler` → 403; `elternteil` auf eigenes Kind → 2xx; `vorstand` auf fremdes Mitglied → 200
- [x] 3.2 `POST .../change-request` (Nicht-Bankfeld): Eigentümer → 2xx; fremder `spieler` → 403, kein Draft in DB
- [x] 3.3 `POST .../change-request` mit `field_name='bankdaten'`: Eigentümer → 2xx; fremder `spieler` → 403; `kassierer` für fremdes Mitglied → 403; verifizieren, dass kein `bankdaten`-Draft angelegt/überschrieben wurde
- [x] 3.4 Negativtest: ein bestehender legitimer pending-Antrag wird durch einen fremden POST NICHT verdrängt (403)

## 4. Verifikation

- [x] 4.1 `/verify-change` ausführen (Build/Test/Lint + Invarianten: Route→Tests, Broadcast/useLiveUpdates, brand-Tokens)
- [x] 4.2 `openspec validate secure-member-draft-access --strict`
