# Design — Open-Source-Dokumentation

## Dokumenten-Landkarte

```
README.md            ← Tür: Pitch, Screenshots, Quickstart, Badges, Links
docs/FEATURES.md     ← Fachlich, as-built, pro Modul, mit Screenshots
docs/ARCHITECTURE.md ← Technisch, Mermaid-Diagramme, für Contributor
docs/VISION.md       ← ehem. TeamWERK-Konzept.md (Roadmap, klar markiert)
LICENSE              ← AGPL-3.0 Volltext
screenshots/         ← PNGs aus App mit Testdaten
```

## FEATURES.md — Gliederung (as-built, aus `internal/`)

Quelle der Wahrheit sind die echten Domänen-Packages, **nicht** die alte Konzept-Datei:

| Modul (fachlich) | Packages | Screenshot |
|---|---|---|
| Mitglieder & Mannschaften | `members`, `teams`, `kader`, `stammvereine` | Mitgliederliste, Detail |
| Termine & Kalender | `games`, `trainings`, `calendar`, `absences` | Kalender, Termin-Detail mit RSVP |
| Dienste / Dienstbörse | `duties` | Dienstbörse, Dienstkonto |
| Mitfahrgelegenheiten | `carpooling` | Fahrer-Zuordnung |
| Kommunikation | `chat`, `notifications`, `push`, `notify` | Chat |
| Dokumente & Videos | `files`, `upload` | Dokumentenablage |
| Beiträge & SEPA | `sepa`, `beitragslauf`, `beitragssaetze` | Beitragslauf, Beitragsmatrix |
| Dashboard & Verwaltung | `dashboard`, `auth`, `config`, `venues` | Dashboard, Admin-Settings |

Jeder Abschnitt: kurze fachliche Beschreibung + Rollen/Berechtigung + 1 Screenshot. Keine Versprechen über nicht gebaute Features (HALLS-Booking, EVENTS-Turnier aus der Vision bleiben in VISION.md).

## ARCHITECTURE.md — geplante Mermaid-Diagramme

1. **Systemüberblick** — Browser/PWA → Go-Binary (embed.FS) → SQLite; SMTP, Web-Push extern
2. **Request-Flow** — Chi-Router → Auth-Middleware (JWT) → Domain-Handler → `database/sql` → SQLite; Broadcast → SSE-Hub → Frontend `useLiveUpdates`
3. **Auth-Tiers** — Public / Authenticated / Trainer+sL / Vorstand(+) / Vorstand / Vorstand+Kassierer / Admin (aus `router.go`)
4. **Rollen × Vereinsfunktionen** — zwei orthogonale Dimensionen
5. **Domänen-Packages & Layering** — Foundation/Domain/Composition (aus `arch_test.go`)
6. **ER-Kernmodell** — members, family_links, teams, kader/kader_members, games, duty_*, beitrags_*
7. **Deploy-Topologie** — IONOS VPS, systemd, Nginx 443→8080, Certbot, Scheduler-Cron

Mermaid rendert nativ auf GitHub, ist versionierbar und PII-frei.

## Lizenz: AGPL-3.0

- `LICENSE` = unveränderter AGPL-3.0-Volltext
- `README` SPDX-Kurzhinweis: `SPDX-License-Identifier: AGPL-3.0-or-later`
- Copyright: „Copyright (C) 2026 Ulrich Frank"
- **Datei-Header**: **entschieden — NEIN.** AGPL ist auch ohne Per-Datei-Header gültig; `LICENSE` + README-Hinweis genügen. Spart 200+ Datei-Diffs.
- AGPL-Pflicht §13: gehosteter Dienst muss Quellcode-Zugang anbieten → ein „Source"-Link im App-Footer wird als Folge-Task notiert (Überschneidung mit ②/④)
- Dependency-Kompatibilität verifiziert: alle permissiv (MIT/BSD) → keine Inkompatibilität mit AGPL

## Screenshots ohne PII

App lokal mit **synthetischem Seed** starten, je Modul ein Screenshot, in `screenshots/` ablegen. Niemals echte Mitgliederdaten abbilden (hängt an ①).
