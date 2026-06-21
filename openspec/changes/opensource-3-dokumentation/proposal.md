## Why

Ein öffentliches Projekt braucht Dokumentation, die (a) Anwender/Vereine fachlich abholt und (b) Contributor technisch befähigt. Die vorhandene `TeamWERK-Konzept.md` beschreibt einen **nie gebauten** Stack (Laravel 11 / Vue 3 / PostgreSQL / Redis) und ist als Architektur-Wahrheit irreführend. Die tatsächliche Implementierung ist **Go 1.26 + Chi + SQLite + React 18 + Vite**.

Dieser Change erstellt **as-built**-Dokumentation: eine fachliche Feature-Beschreibung mit Bildern (Screenshots + Diagramme), eine technische Architektur-Beschreibung und die rechtliche Grundlage (AGPL-3.0). Die alte Konzept-Datei wird zur klar gekennzeichneten Vision/Roadmap.

## What Changes

- **`README.md`** — Einstieg: Was ist TeamWERK, Screenshots-Galerie, Quickstart, Lizenz-Badge, Verweise
- **`docs/FEATURES.md`** — fachliche Feature-Beschreibung **as-built**, nach Modulen gegliedert, mit **echten Screenshots** (Testdaten) je Modul
- **`docs/ARCHITECTURE.md`** — technische Architektur mit **Mermaid-Diagrammen** (Systemüberblick, Request-Flow, Auth-Tiers, Domänen-Packages, ER-Kernmodell, Deploy-Topologie)
- **`LICENSE`** — AGPL-3.0 Volltext + Copyright-Vermerk Ulrich Frank
- **Copyright-/Lizenz-Hinweis** in `README` (SPDX-Kurzform), Strategie für Datei-Header dokumentiert
- **`screenshots/`** — Bilddateien, mit Testdaten erzeugt (kein PII)
- `TeamWERK-Konzept.md` → `docs/VISION.md`, oben als „ursprüngliche Vision, nicht der gebaute Stand" markiert

## Capabilities

### New Capabilities

- `project-documentation`: Öffentlich konsumierbare, korrekte (as-built) Projektdokumentation — fachlich (Features + Bilder), technisch (Architektur + Diagramme) und rechtlich (Lizenz).

### Modified Capabilities

*(keine Anwendungs-Capability — reine Dokumentation)*

## Impact

- Neue Dateien: `README.md`, `docs/FEATURES.md`, `docs/ARCHITECTURE.md`, `LICENSE`, `docs/VISION.md`, `screenshots/*`
- Kein Anwendungscode-Verhalten geändert
- Voraussetzung: ① (PII-frei, damit Screenshots/Repo sauber) und idealerweise ② (entbrandet, damit Bilder/Texte neutral sind)
- Screenshots erfordern lokalen App-Lauf mit synthetischen Seed-Daten
- AGPL-Wahl: kompatibel mit allen Dependencies (Chi/JWT/migrate/godotenv = MIT, x/crypto = BSD, modernc.org/sqlite = BSD-3)
