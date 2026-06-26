## Why

Die Routen `GET /api/members/{id}/change-drafts` und `POST /api/members/{id}/change-request` (`GetChangeRequestsHandler`/`CreateChangeRequestHandler` in `internal/members/drafts_handlers.go`) verarbeiten die `{id}` direkt aus dem Pfad **ohne jeden Eigentums- oder Rollencheck** und liegen im breiten Authenticated-Tier (`internal/app/router.go:147-148`). Jeder eingeloggte Nutzer — auch ein `spieler` ohne Vereinsfunktion — kann dadurch per ID-Enumeration (1) fremde Änderungsanträge inklusive Klartext-PII (Name, Adresse, Telefon, E-Mail und den `old_value`-Snapshot der aktuellen Mitgliedsdaten) lesen, (2) fremde Anträge anlegen, überschreiben oder einen legitimen pending-Antrag verdrängen und (3) einen mit dem öffentlichen Gruppenschlüssel auf das **eigene** Konto verschlüsselten Bankdaten-Envelope als Antrag eines fremden Mitglieds hinterlegen, der bei Vorstands-Freigabe das SEPA-Konto des Opfers ersetzt.

Dies ist die einzige Klartext-PII-Leak der Anwendung (Sicherheitsaudit 2026-06-26, Befunde **B-1 High** + **B-3 Medium**) — DSGVO-relevant und potenziell Minderjährige betreffend. Die Helfer `isOwn`/`isParentOf` (`internal/members/handler.go:2255`) existieren bereits und werden in ~10 anderen Profil-/Kind-Handlern korrekt verwendet; hier fehlen sie schlicht.

## What Changes

- **Ownership-Gate für beide Handler:** `GetChangeRequestsHandler` und `CreateChangeRequestHandler` prüfen am Anfang, ob der Aufrufer eine Beziehung zum Ziel-Mitglied hat (Eigentümer, Elternteil, `admin`, `vorstand` oder `kassierer`), bevor irgendwelche Mitgliedsdaten gelesen oder geschrieben werden — sonst HTTP 403.
- **Bankdaten-Anträge nur durch Eigentümer/Eltern:** Für `field_name='bankdaten'` akzeptiert `POST .../change-request` ausschließlich Eigentümer oder Elternteil des Mitglieds (Selbstbedienungsmodell). Damit kann niemand einen fremden Bankdaten-Envelope unterschieben; die Integrität des ZK-Schreibpfads ist wiederhergestellt. Die Vertraulichkeit der Bankdaten war nie verletzt (Envelope bleibt opak).
- **`old_value` nicht an Fremde:** Der `old_value`-Snapshot wird nur an berechtigte Aufrufer zurückgegeben (durch das Gate ohnehin abgedeckt — explizit verankert).
- **Tests:** Pro Route Happy-Path (Eigentümer/Eltern 2xx) **und** 403-Fall (fremde Member-ID), plus 403 für fremden Bankdaten-Antrag.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `permissions`: Neue Anforderungen für die Self-Service-Änderungsantrag-Routen — Lese- und Schreibzugriff erzwingen Mitglieds-Ownership (Eigentümer/Eltern/Finance), Bankdaten-Anträge sind auf Eigentümer/Eltern beschränkt. Die bestehende Persona-Matrix wird um diese Routen ergänzt.

## Impact

- **Code:** `internal/members/drafts_handlers.go` (`GetChangeRequestsHandler`, `CreateChangeRequestHandler`), ggf. kleiner Helfer in `internal/members/handler.go` (Wiederverwendung `isOwn`/`isParentOf`/Finance-Check).
- **API-Verhalten:** Bisher fälschlich erfolgreiche Cross-Member-Zugriffe antworten künftig mit 403. **BREAKING** nur für nicht-vorgesehene (missbräuchliche) Aufrufmuster; legitime Self-Service- und Vorstands-Flows bleiben unverändert.
- **Tests:** Neue Happy-Path- und 403-Tests in `internal/members/*_test.go` (Persona-Fixtures aus `internal/testutil`).
- **Daten/Migration:** keine Schemaänderung.
- **SSE:** Schreibpfad behält bestehendes `Broadcast`-Verhalten.
