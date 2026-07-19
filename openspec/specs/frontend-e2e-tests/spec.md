# frontend-e2e-tests Specification

## Purpose

Diese Spezifikation beschreibt die Capability `frontend-e2e-tests`: eine Playwright-basierte End-to-End-Test-Suite, die echtes Browser-Verhalten (Scroll-Physik, Bild-Decode-Timing, Layout, Focus) absichert, das Vitest+jsdom prinzipiell nicht simulieren kann — insbesondere den Chat-Öffnungs-Scroll sowie Login- und Send-Flow.

## Requirements

### Requirement: Playwright-basierte E2E-Test-Suite

Das Projekt SHALL eine Playwright-Test-Suite in `web/e2e/` bereitstellen,
die gegen eine deterministisch geseedete Test-DB und einen von Playwright
orchestrierten Backend + Vite-Prozess läuft. Ziel-Browser ist Chromium
(headless im CI, headed lokal für Debug). Die Suite deckt Browser-
Verhalten ab, das Vitest+jsdom prinzipiell nicht simulieren kann (Scroll-
Physik, Bild-Decode-Timing, echtes Layout, aspect-ratio-Anwendung,
Focus-Management).

#### Scenario: Suite ist lokal ausführbar

- **WHEN** ein Entwickler `make test-e2e` im Repo-Root ausführt
- **THEN** startet Playwright automatisch das Backend gegen eine frisch
  geseedete SQLite-Test-DB (Port 18080), den Vite-Dev-Server (Port 15173)
  und läuft alle E2E-Tests in Chromium headless durch
- **AND** nach Abschluss werden beide Prozesse und die Test-DB-Datei
  aufgeräumt

#### Scenario: Suite ist CI-integriert

- **WHEN** ein Pull-Request gegen `main` läuft
- **THEN** startet der GitHub-Actions-Job `e2e` parallel zum bestehenden
  `gate`-Job und blockiert den Merge bei Rot
- **AND** die Chromium-Binary ist im CI zwischen Läufen gecached

#### Scenario: Pre-Push-Hook triggert E2E NICHT

- **WHEN** ein Entwickler `git push` ausführt
- **THEN** läuft der Pre-Push-Hook wie bisher (Vitest + Go-Tests + Lint +
  `openspec validate`) — Playwright wird NICHT lokal getriggert (zu
  langsam für jeden Push; Absicht)

### Requirement: Chat-Öffnungs-Scroll-Regressionsschutz

Die E2E-Suite SHALL das Öffnungs-Scroll-Verhalten des Chats testen, damit
Regressionen der `chat-open-at-unread`-Bugklasse (Scroll landet vor dem
Ende nach Bild-Loads) erkannt werden.

#### Scenario: Gelesene Konv mit Bildern landet am Ende

- **GIVEN** die Test-DB enthält eine Konv „E2E Chat mit Bildern" (3 Bild-
  Nachrichten + Text, alle als gelesen markiert)
- **WHEN** der eingeloggte Test-Admin die Konv anklickt und alle Bilder
  geladen sind (Playwright wartet auf `img` `load`-Event)
- **THEN** gilt `scrollHeight - scrollTop - clientHeight ≤ 5` (near-zero
  Sub-Pixel-Toleranz)

#### Scenario: Konv mit Ungelesenem landet am Divider

- **GIVEN** die Test-DB enthält eine Konv „E2E Chat unread" mit 3 nicht-
  gelesenen Nachrichten
- **WHEN** der Test-Admin die Konv anklickt und Bilder geladen sind
- **THEN** ist ein Element mit Text `"3 ungelesene Nachrichten"` sichtbar
  im Viewport (`isIntersectingViewport()`)
- **AND** der Divider-Abstand zum oberen Viewport-Rand ist zwischen 0 und
  clientHeight/2 (also im oberen Container-Bereich)

#### Scenario: Deep-Link ?openUser=<id> landet nicht am Anfang

- **WHEN** der Test-Admin `/chat?openUser=<test-userid>` besucht
- **THEN** wird die Direkt-Konv geöffnet und (falls unreadCount=0)
  `scrollTop > 0` — kein Landen am Konv-Anfang

### Requirement: Login- und Send-Flow abgedeckt

Die E2E-Suite SHALL den Login-Flow und einen Sende-Zyklus testen, weil
beide Basiskomponenten für alle anderen Interaktionen sind.

#### Scenario: Login führt nach /chat

- **WHEN** ein Nutzer im Login-Formular die Seed-Credentials
  `e2e@test.local` / `E2ETestPassword!` eingibt und Enter drückt
- **THEN** wird auf `/chat` weitergeleitet und die Konversationsliste ist
  sichtbar

#### Scenario: Nachricht senden erscheint in eigener Bubble

- **GIVEN** der Admin ist in einer geöffneten Test-Konv
- **WHEN** er `E2E-Test-<uuid>` ins Eingabefeld tippt und Enter drückt
- **THEN** erscheint innerhalb von 3 Sekunden eine Bubble mit exakt
  diesem Text in der Liste (bewusst mit UUID, damit Tests unabhängig
  voneinander lauffähig sind)
