## Why

Die Spec `permissions` bezeichnet sich als „Quelle der Wahrheit für die mechanischen Drift-Tests“, ist aber an mehreren Stellen nachweislich veraltet: Beitrittsanfragen und Einladungen stehen im Trainer-Gate statt im Vorstand-Tier, `GET /api/venues` und `GET /api/seasons` im falschen Gate, der Kassierer wird als „ohne Wirkung“ geführt, obwohl er ein eigenes Router-Tier, zwei Capabilities und vier Nav-Einträge hat, und §10 nennt drei Designlöcher, die längst geschlossen sind. Der Testspiegel `internal/permissions/matrix_test.go` folgt dem Code, nicht der Spec — die Spec ist damit keine Wahrheit mehr, sondern eine irreführende Kopie. Zwei Nachbar-Specs (`me-capabilities`, `nav-visibility`) widersprechen dem Code an denselben Stellen. Befund aus dem Rechte-Audit vom 19.09.2026 (Punkte B1, B2, B3/B5, B6, B15, B16, §10.1–10.3, C1).

Zweitens fehlt die Vereinsfunktion `medien` (Migration `024`, Spielbericht-Freigabe) als Persona: die Tier-Matrix prüft das Freigeber-Tier `RequireClubFunction("medien","vorstand")` nur über den Admin-Bypass, und das Rollenmodell in `docs/agent/03-go.md` nennt die Funktion nicht.

## What Changes

- **`permissions`-Spec nachgezogen:** Gate-Listen werden aus der tatsächlichen Router-Zuordnung übernommen (Trainer-Gate, Vorstand-Trainer-sL-Gate, Vorstand-Gate), drei bisher unbeschriebene Tiers bekommen eigene Requirements (Vorstand-Kassierer-Gate, Mitgliederlisten-Gate, Saisons-Lese-Gate, Spielbericht-Freigeber-Gate). Frontend-RoleRoute- und Nav-Tabellen spiegeln `App.tsx` und `policy.NavFor`, inklusive der seit Baseline hinzugekommenen Routen. §10 wird neu gefasst: erledigte Punkte raus, tatsächlich offene Inkonsistenzen rein.
- **Persona `medien`** (12. Persona, Kurzcode `m`) in Spec, `personas_test.go`, `web/src/test/personas.ts`, allen Expected-Maps der Tier-Matrix und den Frontend-Persona-Tests. Das Freigeber-Tier erhält damit einen echten Positivnachweis.
- **`me-capabilities`:** Capability-Vokabular vollständig (19 statt 11 Keys), `broadcast_messages` für `trainer` (eingeschränkt auf eigene Kader-Gruppen, wie `chat-broadcasts` es festlegt), `manage_club`/`manage_fees` für `kassierer`.
- **`nav-visibility`:** „Mitglieder“ auch für `kassierer` (wie `kassierer-member-zugriff`).
- **Doku:** Rollenmodell in `docs/agent/03-go.md` um `medien` ergänzt; veraltete Pfadverweise auf `openspec/changes/permissions-baseline-tests/…` in Tests und Fehlermeldung des Drift-Checks auf `openspec/specs/permissions/spec.md` umgestellt.
- **Kein Verhalten ändert sich.** Weder Router noch Policy noch Frontend-Gates werden angefasst; jeder Test, der heute grün ist, bleibt grün. Neu ist nur die zusätzliche Persona-Spalte in den Tests.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `permissions`: Persona-Definition (12 Personas), Authenticated-Requirement (Zählwort), Trainer-Gate, Vorstand-Trainer-sL-Gate, Vorstand-Gate (Endpunktlisten korrigiert), neue Requirements für Vorstand-Kassierer-Gate, Mitgliederlisten-Gate, Saisons-Lese-Gate, Spielbericht-Freigeber-Gate; Frontend-RoleRoute-Sichtbarkeit, Sidebar-Navigations-Items, Inline-Gates und §10 an den Ist-Stand angepasst.
- `me-capabilities`: Requirement „Capability-Vokabular“ auf den vollständigen Satz und die tatsächliche Zuordnung (trainer, kassierer).
- `nav-visibility`: Requirement „Mitglieder Sichtbarkeit“ um `kassierer`.

## Impact

- `openspec/specs/permissions/spec.md`, `me-capabilities/spec.md`, `nav-visibility/spec.md` (über Archivierung).
- `internal/permissions/personas_test.go`, `matrix_test.go` (17 Expected-Maps + Drift-Fehlermeldung), `web/src/test/personas.ts`, `web/src/test/renderAsPersona.tsx`, `web/src/components/__tests__/AppShell.permissions.test.tsx`, `web/src/__tests__/RoleRoute.permissions.test.tsx` sowie die Kommentarzeilen in neun weiteren `*.permissions.test.tsx`.
- `docs/agent/03-go.md` (Rollen-Tabelle).
- **Bewusst außen vor** (jeweils eine Entscheidung, keine Doku-Korrektur): Vorstand liest Trainingstagebücher entgegen `permissions` und `trainingstagebuch-sichtbarkeit` (B4); Trainer sehen Videos aller Mannschaften entgegen `video-management` (B9); CSV-Importe für Hallen und Einladungen im Vorstand- bzw. v+t+sL-Tier entgegen admin-only-Specs (B11, B12); Übungsgruppen-Tier entgegen `uebungsgruppen` (B13); die 14 Objektrechte-Lücken. Diese Punkte werden in §10 als offene Entscheidungen geführt, nicht als Ist-Zustand legitimiert.
