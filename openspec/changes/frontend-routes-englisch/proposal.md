## Why

Mit `api-konsistenz-cleanup` (gemerged in `main`) sind alle Backend-Routen auf konsequentes Englisch konsolidiert (`/api/games`, `/api/members`, `/api/venues`, …). Die UI-Routen sind jedoch weiterhin deutsch (`/kalender`, `/mitglieder`, `/veranstaltungsorte`, …). Diese Asymmetrie war in `CLAUDE.md` als bewusste Entscheidung festgehalten („UI-Begriffe bleiben deutsch") — sie soll mit diesem Change aufgegeben werden:

- **Konsistenz bei Pfad-Sprache.** Bookmarks, Logs, Push-Deeplinks, Doku-Beispiele und API-Pfade in einer Sprache reduzieren kognitive Last. Heute wechseln Reviewer und neue Devs ständig zwischen `/mitglieder` (UI) und `/api/members` (API).
- **Stop drift.** Mehrere Specs nennen heute schon englische UI-Pfade aus Versehen (vermutlich generiert mit englischem Default); andere streng deutsche. Ein klarer Cut schließt diese Drift.
- **Sichtbarmachen einer überfälligen Entscheidung.** Solange die UI-Sprache deutsch bleibt (Labels, Inhalte), ist das ein reiner URL-/Code-Rename. Anwender sehen Pfade selten — Entwickler und Push-Notifications täglich.

User-facing UI-**Texte** (Labels, Buttons, Inhalte) bleiben deutsch. Das ist kein i18n-Schritt; nur die Pfad-Symbole wechseln.

## What Changes

### Route-Mapping (BREAKING — hart cut, keine Redirects)

**Public Routes (`web/src/App.tsx`)**

```
/passwort-vergessen   → /forgot-password
/login                bleibt
/register             bleibt
/join                 bleibt
/reset-password       bleibt
```

**Protected Routes (`web/src/App.tsx`)**

```
/mitglieder                       → /members
/mitglieder/:id                   → /members/:id
/profil                           → /profile
/profil/kind/:memberId            → /profile/child/:memberId
/dokumente                        → /documents
/dokumente/:folderId              → /documents/:folderId
/dienste                          → /duties
/mitfahrgelegenheiten             → /carpools
/anfragen                         → /requests
/einstellungen                    → /settings
/kader                            → /squad
/nutzer                           → /users
/diensttypen                      → /duty-types
/dienstplan-vorlagen              → /duty-templates
/dienstplan-vorlagen/:id          → /duty-templates/:id
/veranstaltungsorte               → /venues
/kalender                         → /schedule
/kalender/:gameId                 → /schedule/:gameId
/termine                          → /events
/termine/:type/:id                → /events/:id            ← Struktur-Wechsel, siehe unten
/mein-team                        → /my-team
/chat                             bleibt
/trainings (Redirect)             → entfällt (Redirect-Ziel war /termine)
/trainings/:id (Redirect)         → entfällt
```

### Struktur-Wechsel: `/termine/:type/:id` → `/events/:id`

Heute wird das Type-Segment (`spiel` | `training` | `ereignis`) im Pfad geführt, weil `TermineDetailPage` darüber unterscheidet, welche API zu laden ist (Game vs. Training-Session vs. Generic Event — die ID-Räume überlappen). Mit dem Wechsel auf `/events/:id` ohne Type-Segment muss diese Disambiguierung anders gelöst werden. **Diese Detail-Auflösung gehört in die Design-Phase**, mögliche Richtungen:

- **Prefix-IDs** im Pfad: `/events/g-:id`, `/events/t-:id`, `/events/e-:id`. Vorteil: ein Param, kein Query-Wirrwarr.
- **Query-Param**: `/events/:id?type=game`. Vorteil: minimaler Pfad-Eingriff.
- **Unifying-API**: `GET /api/events/:opaqueId` löst Game/Training/Event hinter den Kulissen. Vorteil: cleanste Außenseite, größter Eingriff.

Aufrufer, die heute deutlich machen: `internal/dashboard/handler.go` (SQL baut `'/termine/training/'||ts.id` und `'/termine/spiel/'||g.id`), `internal/scheduler/scheduler.go` (Push: `/termine?focus=game-%d`, `/termine?focus=training-%d`), `internal/games/handler.go`, `internal/trainings/handler.go`, sowie `TerminePage.tsx` (`navigate(\`/termine/training/${s.id}\`)`).

### Frontend-Aufräumarbeiten (kein Wahlrecht, hängt am Route-Rename)

- **`web/src/components/AppShell.tsx`** — 13 Nav-Einträge auf neue Pfade (Mein Profil, Kalender, Termine, Mein Team, Dokumente, Dienste, Mitfahrten, Nutzerverwaltung, Mitglieder, Kader, Diensttypen, Dienstplan-Vorlagen, Veranstaltungsorte, Einstellungen). **UI-Labels bleiben deutsch.**
- **`Link to=` / `Navigate to=` / `navigate(...)`-Stellen** in Pages und Komponenten: aktuell 25 Treffer (`DashboardPage`, `DocumentsPage`, `SpieltagDetailPage`, `LoginPage`, `MitfahrgelegenheitenPage`, `AdminDutyTemplateDetailPage`, `TerminePage`, …) — vollständige Liste während Implementierung.
- **Page-Datei-Umbenennungen** (Pflicht, damit `App.tsx`-Imports konsistent bleiben): `MitfahrgelegenheitenPage.tsx` → `CarpoolsPage.tsx`, `AdminKaderPage.tsx` → `SquadPage.tsx`, `KalenderPage.tsx` → `SchedulePage.tsx`, `SpieltagDetailPage.tsx` → `GameDetailPage.tsx`, `TerminePage.tsx` → `EventsPage.tsx`, `TermineDetailPage.tsx` → `EventDetailPage.tsx`. Andere Page-Komponenten bleiben unverändert (`MembersPage`, `ProfilePage`, `DocumentsPage`, … sind bereits englisch).
- **`?tab=`-Query-Werte** in `/einstellungen?tab=verein|saisons|altersklassen` bleiben deutsch (sind innen-Komponenten, kein Route-Schema-Bestandteil). Optional in eigenem Folge-Change.

### Backend-Deeplinks und Notifications (BREAKING für Code, transparent für Anwender)

Die Backend-Module senden Push/E-Mail mit hartcodierten UI-Pfaden. Diese müssen mit dem Frontend-Rename synchron umgestellt werden, sonst landen Nutzer auf 404:

| Stelle | heute | neu |
|---|---|---|
| `internal/duties/handler.go` (2×) | `/dienste` | `/duties` |
| `internal/trainings/handler.go` (3×) | `/termine`, `/termine?focus=training-%d` | `/events`, `/events?focus=training-%d` |
| `internal/scheduler/scheduler.go` (4×) | `/dienste`, `/termine?focus=…`, `/mitfahrgelegenheiten` | `/duties`, `/events?focus=…`, `/carpools` |
| `internal/games/handler.go` (3×) | `/termine?focus=game-%d`, `/dienste` | `/events?focus=game-%d`, `/duties` |
| `internal/auth/handler.go` (1×) | `/anfragen?id=%d` | `/requests?id=%d` |
| `internal/dashboard/handler.go` (2×) | `'/termine/training/'\|\|ts.id`, `'/termine/spiel/'\|\|g.id` | abhängig von `/events/:id`-Design (siehe oben) |

`?focus=game-X` / `?focus=training-X` nutzen bereits englische Typ-Bezeichner — die bleiben unverändert.

### Doku & Changelog

- `CLAUDE.md` — die Stance-Aussage „UI-Routen bleiben deutsch" wird umgekehrt; URL-Mapping-Tabelle aktualisieren; ggf. eine Notiz „UI-Pfade englisch, UI-Texte deutsch".
- `web/public/CHANGELOG.md` — Hinweis auf URL-Änderung (analog zur API-Konsistenz-Notiz).
- `docs/anleitung-*.md` — falls Beispielpfade enthalten, anziehen.

### Nicht im Scope

- **UI-Texte und Labels** bleiben deutsch (kein i18n).
- **`?tab=verein|saisons|altersklassen`** in `/settings` (deutsche Tab-Werte) — eigener kleiner Folge-Change, falls erwünscht.
- **DB-Tabellen-/Spalten-Renames** — gehören in den parallel laufenden Change `rename-mitfahrten`; dieser Change rührt keine SQL-Schemata an.
- **Go-Package-Namen** (`internal/duties`, `internal/games`, …) — bereits englisch; `internal/carpooling` ist es ebenfalls.
- **Test-Pfade in `cmd/teamwerk/web/dist/assets/*`** — Build-Output, wird durch nächsten `pnpm build` regeneriert.

### Wechselwirkung mit `rename-mitfahrten`

Der noch nicht gestartete Change `rename-mitfahrten` (0/39 Tasks) plante UI-Pfad `/mitfahrgelegenheiten` → `/mitfahrten`, DB-Tabelle, API-Pfade, SSE-Event-Namen und Page-Komponente. Mit *diesem* Change wird der UI-Pfad direkt zu `/carpools` (englisch) — `rename-mitfahrten` schrumpft auf das, was hier explizit *nicht* angefasst wird: DB-Tabelle, API-Pfad (`/api/mitfahrgelegenheiten` → `/api/carpools` oder `/api/mitfahrten`), SSE-Event, Page-Datei. Reihenfolge: `rename-mitfahrten` aktualisieren oder absorbieren, **bevor** dieser Change startet — sonst Merge-Konflikte in `AppShell.tsx` und `MitfahrgelegenheitenPage.tsx`.

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

Sammelliste — pro Spec werden UI-Pfad-Referenzen aktualisiert. Die meisten Specs nennen heute deutsche UI-Pfade, ein paar tragen schon (versehentlich) englische — beide Sorten werden vereinheitlicht.

- `documents-ui` — `/dokumente`, `/dokumente/:folderId` → `/documents/*`
- `duties` — `/dienste` (Notification-Targets)
- `game-rsvp`, `spiel-teilnahme`, `spiel-aufstellung` — `/kalender/:gameId` → `/schedule/:gameId`
- `kalender-modus-toggle`, `kalender-date-param`, `kalender-dienste-panel`, `kalender-agenda-view` — `/kalender` → `/schedule` (Capability-Name bleibt aus historischen Gründen, Pfade aktualisieren)
- `termine-unified-view` — `/termine`, `/termine/:type/:id` → `/events`, `/events/:id`
- `dashboard-migration` — Detail-URLs `/termine/training/`, `/termine/spiel/` an neue `/events/:id`-Struktur anpassen
- `carpooling-team-filter`, `carpooling-notifications`, `mitfahrgelegenheiten-board`, `mitfahrgelegenheiten-team-filter`, `mitfahrgelegenheiten-nav`, `mitfahrt-paarungen` — `/mitfahrgelegenheiten` → `/carpools` (Spec-Namen bleiben oder werden in `rename-mitfahrten` umbenannt)
- `members`, `name-aenderung`, `whatsapp-sichtbarkeit`, `email-aenderung`, `passwort-aenderung`, `notification-preferences`, `user-reminder-preference`, `maps-provider-preference`, `familie-im-profil`, `kind-profil`, `kind-profil-user-strang`, `member-absences` — `/mitglieder`, `/profil`, `/profil/kind/:memberId` aktualisieren
- `mein-team-back-button` — `/mein-team` → `/my-team`
- `qualifikations-kader`, `erweiterter-kader`, `sse-kader-sync`, `test-kader-gaps` — `/kader` → `/squad`
- `training-rsvp`, `push-trainings`, `push-games`, `push-duties` — Notification-Deeplinks aktualisieren
- `venue-csv-import` — `/veranstaltungsorte` → `/venues`
- `membership-request-deeplink` — Ziel `/anfragen?id=X` → `/requests?id=X`
- `test-auth-gaps`, `test-members-gaps` — UI-Pfade in Auth/Members-Testfällen
- `event-info-modal`, `sse-live-updates` — falls UI-Pfade referenziert
- `api-routes` — zentrale Pfad-Übersicht ergänzen um UI-Mapping

## Test-Anforderungen

Frontend-Routen haben kein Backend-Test-Pendant — Test-Strategie ist hier vor allem **Invarianten-Smoke** und **manuelle PWA-Verifikation**.

- **Invariante 1** (automatisierbar): `grep -rn -E "['\"]\/(mitglieder|profil|dokumente|dienste|mitfahrgelegenheiten|anfragen|einstellungen|kader|nutzer|diensttypen|dienstplan-vorlagen|veranstaltungsorte|kalender|termine|mein-team|passwort-vergessen)" web/src internal cmd` darf nach Cut nur in archivierten OpenSpec-Changes oder in `web/public/CHANGELOG.md` Treffer haben.
- **Invariante 2**: TypeScript-Build (`pnpm build`) muss grün sein — Imports aller umbenannten Pages müssen ziehen.
- **Invariante 3**: Vitest-Suite muss grün sein — Komponenten-Tests die UI-Pfade hartcoden mitziehen.
- **Manuell**: Push-Notification aus Spielanlage öffnet `/events?focus=game-X` → korrekte Page.
- **Manuell**: Anfrage-Notification aus Beitrittsantrag öffnet `/requests?id=X` → AdminUsersPage mit Highlight.
- **Manuell**: Deeplink in Dienst-Reminder öffnet `/duties` → DutyPage.
- **Manuell**: Dashboard-Karten-Klicks (`detail_url` aus SQL) führen auf das richtige `/events/:id` (abhängig von Design-Entscheidung).
- **Backend-Tests**: bestehende Tests in `internal/duties`, `internal/games`, `internal/trainings`, `internal/scheduler`, `internal/auth`, `internal/dashboard`, die hartcodierte Pfade in Erwartungswerten haben, anpassen.

## Migration / Deployment-Hinweise

- **Hart cut**, analog zu `api-konsistenz-cleanup` Phase 1: alte Bookmarks brechen, keine Redirect-Schicht. Hinweis in `CHANGELOG.md` und Vorstands-Chat-Ankündigung.
- **Kein Doppelmount**: Frontend + Backend deployen aus einem Binary (`//go:embed all:web/dist`) — kein Zeitfenster, in dem ein altes Frontend ein neues Backend (oder umgekehrt) anspricht.
- **Push-Subscriptions** sind nicht betroffen: Subscriptions enthalten keinen Pfad, der Pfad steckt im Push-Payload und wird beim Klick gegen den deployten Router resolved. Nach Deploy zeigen alle Notifications korrekt.
- **Reihenfolge im PR**:
  1. `rename-mitfahrten` koordinieren (entweder vorher abschließen oder UI-Anteil hierher absorbieren).
  2. Design-Entscheidung `/events/:id`-Disambiguierung treffen, **bevor** Frontend-Rename startet.
  3. Frontend-Routen umbenennen + Page-Dateien umbenennen + interne Links ziehen.
  4. Backend-Notification-URLs aktualisieren.
  5. Specs aktualisieren.
  6. Smoke-Greps + manuelle Push-Tests.
- **CLAUDE.md-Stance-Aussage explizit umkehren** als Teil dieses Changes — sonst entstehen wieder deutsche Pfade in zukünftigen Features.
