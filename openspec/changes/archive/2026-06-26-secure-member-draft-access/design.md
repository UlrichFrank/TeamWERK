## Context

Die Änderungsantrag-Routen sind Teil des Self-Service-Profils: ein Mitglied (oder dessen Elternteil) beantragt eine Änderung an Stammdaten, der Vorstand genehmigt sie. Lesen (`GET .../change-drafts`) und Schreiben (`POST .../change-request`) sitzen im Authenticated-Tier, weil sie prinzipiell jedem Eingeloggten offenstehen — die **Objekt-Ebene** (welches Mitglied) ist aber dynamisch und gehört in den Handler, nicht in den Router-Tier. Genau dieser handlerseitige Ownership-Check fehlt heute, während `accept`/`delete` der Drafts korrekt über `RequireClubFunction` im Router gegated sind.

Das Projekt hat dieses Muster bereits gelöst: `isOwn(userID, memberID)` und `isParentOf(userID, memberID)` (`internal/members/handler.go:2255`) werden in den Profil-/Kind-Handlern verwendet. Diese Änderung schließt die Lücke konsistent.

## Goals / Non-Goals

**Goals:**
- Kein Aufrufer kann fremde Mitglieds-PII über die Antragsrouten lesen.
- Kein Aufrufer kann fremde Anträge erzeugen, überschreiben oder verdrängen.
- Kein Aufrufer kann einen Bankdaten-Envelope unter fremdem Namen einreichen.
- Wiederverwendung der vorhandenen Helfer, keine neue Autorisierungsmechanik.

**Non-Goals:**
- Keine Änderung am Genehmigungsweg (`accept`/`reject`) — der ist bereits korrekt gegated.
- Keine Änderung am ZK-Krypto-Modell oder am `bankdaten`-Draft-Format (`bankdaten-draft`-Capability bleibt unangetastet).
- Keine Schemaänderung.

## Decisions

**D1 — Gate im Handler, nicht im Router.** Die Berechtigung hängt vom Verhältnis Aufrufer↔`{id}` ab (Eigentum/Elternschaft), das der Router-Tier nicht kennt. Daher prüft jeder Handler zu Beginn selbst und gibt 403 zurück, bevor er Daten liest/schreibt. Alternative (Router-Tier auf `vorstand`) verworfen: würde den legitimen Self-Service von Mitglied/Eltern brechen.

**D2 — Zwei Berechtigungsstufen.** Allgemeines Lesen/Schreiben: Eigentümer ∨ Eltern ∨ admin ∨ vorstand ∨ kassierer. Bankdaten-Schreiben (`field_name='bankdaten'`): nur Eigentümer ∨ Eltern. Begründung: Finance genehmigt Bankdaten, reicht sie aber nicht ein; eine Vorstands-„Einreichung" für ein fremdes Mitglied wäre genau der missbrauchbare Pfad aus B-3. Korrektur durch Kassierer läuft über die separate, feld-gewhitelistete Route `PUT /api/members/{id}/bank-details`.

**D3 — `old_value` durch das Gate abgedeckt.** Da das Gate vor jeder Datenrückgabe greift, wird der `old_value`-Snapshot nie an Unberechtigte ausgeliefert; eine separate Redaktion ist nicht nötig. Die bestehende `redactBankDrafts`-Logik (`drafts.go`) bleibt als zweite Schicht erhalten.

## Risks / Trade-offs

- **[Regression bei legitimen Eltern-Flows]** → Persona-Test `elternteil` auf das eigene Kind (2xx) plus Gegenprobe fremdes Kind (403).
- **[Doppelte Prüfung Lesen/Schreiben divergiert]** → gemeinsamer Helfer (z.B. `h.canAccessMember(claims, memberID)`), in beiden Handlern aufgerufen, ein einziger Wahrheitsort.
- **[`vorstand`/`kassierer` erwarten evtl. Bankdaten-Einreichung]** → bewusst ausgeschlossen (D2); Korrektur über `bank-details`-Route bleibt verfügbar und ist dokumentiert.

## Migration Plan

Reiner Verhaltens-Fix, keine Datenmigration. Deploy in einem Schritt; Rollback durch Zurücknehmen des Commits. Vor Deploy sicherstellen, dass kein legitimer Client auf das alte (ungegatete) Verhalten angewiesen ist — laut Frontend ruft nur die eigene Profil-/Kind-Ansicht diese Routen auf.
