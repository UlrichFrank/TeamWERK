## Context

Siehe proposal.md — Why. Stand heute:

- `openspec/specs/permissions/spec.md` (631 Zeilen) beschreibt 11 Personas, fünf Gates, Frontend-Routen, Nav und §10. Vier Gate-Listen und §10 stimmen nicht mehr mit `internal/app/router.go` überein; der Testspiegel `internal/permissions/matrix_test.go` (363 Endpunkte × 11 Personas, 17 Expected-Maps) ist korrekt und wurde bei jedem Change mitgepflegt, die Spec nicht.
- Persona-Listen liegen doppelt vor: `internal/permissions/personas_test.go` und `web/src/test/personas.ts`, beide mit Kommentar „spiegelbildlich, beide aktualisieren“. Die Frontend-Persona-Tests (`renderAsPersona.tsx`, `AppShell.permissions.test.tsx`, `RoleRoute.permissions.test.tsx`, sieben Page-Tests) iterieren über `PERSONAS` und erwarten pro Persona ein Ergebnis.
- Elf Test-Dateien und die Drift-Fehlermeldung verweisen auf `openspec/changes/permissions-baseline-tests/…`, einen längst archivierten Pfad.
- `medien` ist in `member_club_functions` (Migration `024`) und im Router-Tier `RequireClubFunction("medien","vorstand")` vorhanden, in `docs/agent/03-go.md` und in den Personas nicht.

## Goals / Non-Goals

**Goals:**

- Die `permissions`-Spec beschreibt den Ist-Zustand der Gates so, dass die Tier-Matrix aus ihr ableitbar ist — Endpunktlisten je Gate stammen aus der Matrix, nicht aus dem Gedächtnis.
- `medien` ist eine echte Persona mit Positivnachweis im Freigeber-Tier und im Frontend.
- Nachbar-Specs (`me-capabilities`, `nav-visibility`) widersprechen dem Code nicht mehr an Stellen, die `permissions` neu beschreibt.
- Offene Rechte-Entscheidungen bleiben sichtbar (§10), statt durch das Nachziehen legitimiert zu werden.

**Non-Goals:**

- Keine Änderung an Router, Policy, Handlern oder Frontend-Gates. Alle in §10 genannten Punkte (B4, B9, B11–B13, Objektrechte-Lücken, B10) sind eigene Changes.
- Keine Persona für Spieler-Trainer, spielende Eltern oder Kinder ohne Konto (weiter akzeptierte Lücke).
- Kein Umbau der Frontend-Gates auf Capabilities (§10 Punkt 8).

## Decisions

**D1 — Endpunktlisten je Gate aus der Tier-Matrix übernehmen, nicht aus dem Router abschreiben.**
Die Matrix ist die einzige Stelle, die jede Route genau einem Erwartungs-Muster zuordnet und das mechanisch prüft. Die Spec listet je Gate die Routen der zugehörigen Expected-Map (`exTrainer` → Trainer-Gate, `exVorstandTrainer` → v+t+sL-Gate, `exVorstand` → Vorstand-Gate, `exVorstandKassierer` → neues Vorstand-Kassierer-Gate, `exMembersList`, `exSeasonsRead`, `exMatchReportPublisher` → drei neue Ein-Routen-Requirements). Alternative — die Spec nur auf „siehe Matrix“ zu reduzieren — verworfen: dann beschreibt niemand mehr, *warum* eine Route in einem Tier liegt.

**D2 — Persona `medien` mit `is_parent=false`, ohne Eltern-Variante.**
Die drei `_elternteil`-Varianten beweisen, dass Elternschaft kein Funktions-Tier verändert; das ist nach 17 identischen Map-Paaren belegt und braucht keine vierte Wiederholung. `medien` bekommt in 16 Maps denselben Wert wie `spieler` (403 bzw. allowed/AnyOK) und in `exMatchReportPublisher` `httpAllowed`. Die Reihenfolge in `Personas` bleibt stabil (medien am Ende), damit die Token-User-IDs `i+100` der bestehenden Personas unverändert bleiben.

**D3 — Frontend-Tests: `medien` bekommt in `renderAsPersona.tsx` `nav = Basis + /spielberichte/pruefen`, keine Capability.**
`personaCapabilities`/`personaNavRoutes` sind eine handgeschriebene Spiegelung von `policy.Capabilities`/`NavFor`. Sie wird um `medien` ergänzt und bei der Gelegenheit um die neun fehlenden Nav-Routen (`/anwesenheit`, `/trainingstagebuch`, `/profil/trainingstagebuch`, `/uebungsgruppen`, `/tresor`, `/wartung`, `/dienste/rangliste`, `/spielberichte`, `/spielberichte/pruefen`) vervollständigt — sonst prüft der Persona-Test einen Nav-Ausschnitt gegen die neue Nav-Tabelle. `NO_VERWALTUNG_IDS` erhält `medien`.

**D4 — MODIFIED-Blöcke behalten alle bestehenden Szenario-Namen.**
`openspec validate` verweigert das Streichen von Szenarien in MODIFIED-Requirements. Überholte Szenarien („admin sieht kein „Mein Profil““, „vorstand_beisitzer hat heute keinen Sondereffekt“) behalten ihren Namen und bekommen den Ist-Zustand als Inhalt.

**D5 — §10 neu fassen statt streichen.**
Von den fünf alten Punkten sind fünf erledigt. Der Abschnitt bleibt als Ort für offene Rechte-Entscheidungen und nimmt die acht Befunde des Audits auf, die nicht Doku-, sondern Entscheidungssache sind. Die Domänen-Specs (`trainingstagebuch-sichtbarkeit`, `video-management`, `venue-csv-import`, `csv-import`, `uebungsgruppen`, `sepa-mandat-upload`) werden in diesem Change nicht angefasst — sie sagen heute das, was entschieden werden muss.

**D6 — `me-capabilities` und `nav-visibility` mitziehen, aber nur die widersprechenden Requirements.**
Beide Specs enthalten Aussagen, die `permissions` nach dem Nachziehen direkt widersprechen würden (Kassierer-Nav, Trainer-Broadcast, Capability-Vokabular). Sie in einem Folge-Change zu lassen hieße, absichtlich zwei widersprüchliche Specs zu archivieren.

## Risks / Trade-offs

- [Spec-Deltas sind lang und handgeschrieben; ein Tippfehler in einer Endpunktliste wäre ein neuer Drift] → Task 5.2 prüft die Listen maschinell gegen `routes.json`-Extrakt der Matrix (Skript im Change-Verzeichnis, wird nicht eingecheckt).
- [`medien` in 17 Maps nachtragen ist mechanisch, aber fehleranfällig] → der Matrix-Test meldet jede fehlende Persona pro Endpoint („hat keinen Eintrag in expected-Map“); ein Lauf reicht als Vollständigkeitsnachweis.
- [Frontend-Persona-Tests könnten für `medien` unerwartete Ergebnisse liefern (Seiten, die `clubFunctions` direkt lesen)] → genau das ist der Zweck: MatchReportFormPage sollte für `medien` die Freigeber-Ansicht rendern; ein roter Test dort wäre ein echter Befund.
- [§10 wächst und wirkt wie eine Wunschliste] → jeder Punkt trägt Fundstellen und den Satz, welche Entscheidung offen ist; Punkte ohne Entscheidung gehören nicht hinein.

## Migration Plan

Kein Deploy nötig — Tests und Doku. Push nach grünem Pre-Push-Gate; Archivierung synchronisiert die drei Specs.
