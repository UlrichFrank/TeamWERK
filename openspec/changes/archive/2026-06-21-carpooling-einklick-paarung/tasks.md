## 1. Backend — einseitiger Paarungs-Request

- [x] 1.1 `RequestPairing` (`internal/carpooling/paarungen_handler.go`): Body-Parsing auf optionale `bieteId`/`sucheId` + neue Felder `forUserId *int`, `plaetze *int` umstellen; Validierung: genau eine ID (einseitig) ODER beide (Altpfad), sonst 400.
- [x] 1.2 Helfer `getOrCreateSuche`/`getOrCreateBiete(tx, gameID, userID, plaetze)`: vorhandenen Eintrag der Spiegel-Seite für `(game_id, user_id)` finden (Biete via Unique-Index; Suche: ohne aktive Paarung) und sonst neu anlegen; gibt Eintrag-ID zurück.
- [x] 1.3 Einseitiger Suche-Pfad (`bieteId` gesetzt): `forUserId` auflösen (Default eingeloggter Nutzer), Berechtigung (`isChildOf`/self) **vor** Insert prüfen → 403 bei Fremdbezug; Suche-Spiegel get-or-create; `initiiert_von='suche'`.
- [x] 1.4 Einseitiger Biete-Pfad (`sucheId` gesetzt): Biete-Spiegel get-or-create immer für eingeloggten Nutzer; `initiiert_von='biete'`.
- [x] 1.5 Gesamten Ablauf (Spiegel-Insert + Kapazitäts-/Bestehensprüfung + Paarungs-Insert) in eine `tx` klammern; bei 403/409 `Rollback`, kein Eintrag persistiert. Bestehende Kapazitäts- und „bereits aktive Paarung"-Checks wiederverwenden.
- [x] 1.6 `h.hub.Broadcast("mitfahrgelegenheiten")` + Push (`pairing_requested`) wie bisher beibehalten; Event/Push erst nach erfolgreichem Commit.

## 2. Backend — Tests

- [x] 2.1 Test: Sucher ohne eigenen Eintrag, `{bieteId}` → 204, Suche-Eintrag angelegt, Paarung `pending`/`initiiert_von='suche'`.
- [x] 2.2 Test: Elternteil `{bieteId, forUserId=Kind}` ohne Kind-Eintrag → 204, Eintrag für Kind, Paarung angelegt.
- [x] 2.3 Test: Bieter ohne eigenen Eintrag, `{sucheId}` → 204, Biete-Eintrag angelegt, `initiiert_von='biete'`.
- [x] 2.4 Test: vorhandener Suche-Eintrag ohne aktive Paarung wird wiederverwendet (kein zweiter Eintrag).
- [x] 2.5 Test: `{bieteId, forUserId}` mit fremdem `forUserId` → 403, **kein** Eintrag angelegt.
- [x] 2.6 Test: einseitiger Request bei voller Kapazität → 409, **kein** Spiegel-Eintrag persistiert (Atomarität).
- [x] 2.7 Test: Altpfad `{bieteId, sucheId}` bleibt unverändert (Regression, Happy-Path + 403 Fremdbezug).
- [x] 2.8 Test: leerer Body / beide IDs fehlen → 400.

## 3. Frontend — Buttons & Mini-Dialog

- [x] 3.1 `MitfahrgelegenheitenPage.tsx`: `canRequestAsBiete`/`canInviteAsSuche` vom Vorhandensein eigener Einträge entkoppeln (nur: fremder Eintrag, freie Kapazität bei Biete, keine bestehende aktive Paarung mit mir).
- [x] 3.2 Mini-Dialog-Komponente (abgespeckte FormModal): Pflichtfeld Plätze; auf der Mitfahren/Suche-Seite zusätzlich Für-wen-Auswahl (ich / Kind …) wenn `children`-Liste vorhanden. Brand-Tokens, lucide-Icons, Button-Klassen-Strings gemäß CLAUDE.md.
- [x] 3.3 `onRequest` ruft den einseitigen Endpoint: Mitfahren → `POST /mitfahrt-paarungen { bieteId, forUserId?, plaetze }`; Platz anbieten → `{ sucheId, plaetze }`. Bei vorhandenem eigenem Eintrag weiterhin Direktpfad möglich.
- [x] 3.4 Fehlerbehandlung im Dialog (409 Kapazität / 403) als Inline-Alert (`Alert Fehler`-Klassen); `useLiveUpdates`-Reload bleibt unverändert.

## 4. Abschluss

- [x] 4.1 `/verify-change` ausführen (Build/Test/Lint + Invarianten: Route→Tests, Mutation→Broadcast, brand-Tokens, lucide-Icons, `openspec validate`).
- [ ] 4.2 `openspec validate carpooling-einklick-paarung --strict` grün (✓); abschließender Commit, ggf. Archivierung anstoßen. *(Commit dem Nutzer überlassen.)*

## Test-Anforderungen

Garantierte Invariante: Ein einseitiger Paarungs-Request legt **atomar** Spiegel-Eintrag + `pending`-Paarung an; schlägt Berechtigung oder Kapazität fehl, wird **nichts** persistiert. Der bestehende zweiseitige Pfad bleibt unverändert.

| Route | Testname | Erwarteter Status |
|---|---|---|
| `POST /api/mitfahrt-paarungen` `{bieteId}` (kein eigener Eintrag) | `TestRequestPairing_EinseitigSuche_LegtEintragAn` | 204 + Eintrag + `pending` |
| `POST /api/mitfahrt-paarungen` `{bieteId, forUserId=Kind}` | `TestRequestPairing_EinseitigFuerKind` | 204 + Eintrag für Kind |
| `POST /api/mitfahrt-paarungen` `{sucheId}` (kein eigener Eintrag) | `TestRequestPairing_EinseitigBiete_LegtEintragAn` | 204 + `initiiert_von='biete'` |
| `POST /api/mitfahrt-paarungen` `{bieteId}` (vorhandenes Gesuch) | `TestRequestPairing_WiederverwendetSuche` | 204, kein 2. Eintrag |
| `POST /api/mitfahrt-paarungen` `{bieteId, forUserId=fremd}` | `TestRequestPairing_FremderForUserId` | 403, kein Eintrag |
| `POST /api/mitfahrt-paarungen` `{bieteId}` (Kapazität voll) | `TestRequestPairing_EinseitigKapazitaetVoll` | 409, kein Eintrag |
| `POST /api/mitfahrt-paarungen` `{bieteId, sucheId}` | `TestRequestPairing_Altpfad_Regression` | 204 (Happy) / 403 (fremd) |
| `POST /api/mitfahrt-paarungen` `{}` | `TestRequestPairing_LeererBody` | 400 |
