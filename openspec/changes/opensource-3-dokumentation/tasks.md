# Tasks — Open-Source-Dokumentation

## 1. Lizenz
- [ ] 1.1 `LICENSE` = AGPL-3.0-Volltext + „Copyright (C) 2026 Ulrich Frank"
- [ ] 1.2 SPDX-Kurzhinweis in README (`AGPL-3.0-or-later`)
- [x] 1.3 Entscheidung: KEINE Per-Datei-Header (nur LICENSE + README-Hinweis) — bestätigt 2026-06-21
- [ ] 1.4 AGPL §13 Source-Link im App-Footer als Folge-Task in ④ notieren

## 2. Screenshots (PII-frei)
- [ ] 2.1 App lokal mit synthetischem Seed starten
- [ ] 2.2 Je Modul ein Screenshot (Mitglieder, Kalender, Dienste, Carpooling, Chat, Dokumente, Beitragslauf, Dashboard, Admin)
- [ ] 2.3 Ablage in `screenshots/`; Prüfung: keine echten Personendaten sichtbar

## 3. FEATURES.md (as-built, mit Bildern)
- [ ] 3.1 Module aus `internal/`-Packages ableiten (nicht aus Konzept.md)
- [ ] 3.2 Pro Modul: Beschreibung + Rolle/Berechtigung + Screenshot
- [ ] 3.3 Abgrenzung „Vision vs. gebaut" (Verweis auf VISION.md)

## 4. ARCHITECTURE.md (Mermaid)
- [ ] 4.1 Systemüberblick-Diagramm (Browser/PWA → Go-Binary → SQLite; SMTP/Web-Push)
- [ ] 4.2 Request-Flow + SSE/Broadcast-Diagramm
- [ ] 4.3 Auth-Tiers-Diagramm (aus `router.go`)
- [ ] 4.4 Rollen × Vereinsfunktionen-Diagramm
- [ ] 4.5 Domänen-Packages/Layering-Diagramm (aus `arch_test.go`)
- [ ] 4.6 ER-Kernmodell-Diagramm (aus Migrations)
- [ ] 4.7 Deploy-Topologie-Diagramm
- [ ] 4.8 Fließtext: Stack, Konventionen, Verweis auf CLAUDE.md/AGENTS.md

## 5. Vision-Altlast & README
- [ ] 5.1 `TeamWERK-Konzept.md` → `docs/VISION.md`, oben als ursprüngliche Vision markieren
- [ ] 5.2 `README.md`: Pitch, Screenshot-Galerie, Quickstart, Badges (Lizenz/CI), Links
- [ ] 5.3 Querprüfung: keine Aussage in der Doku widerspricht dem Code-Stand
