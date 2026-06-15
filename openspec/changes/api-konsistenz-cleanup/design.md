# Design

## Ausgangslage

Audit der echten Routen in `cmd/teamwerk/main.go` (Stand 2026-06-15):

- **Keine `/api/admin/*`-Group existiert.** Alle Routen liegen flach unter `/api/<resource>`, Berechtigung über `auth.RequireClubFunction(...)`-Middleware-Gruppen.
- **`/api/kalender*` und `/api/games*` koexistieren** für dieselbe Domäne. Game-Handler werden über beide Pfade exponiert.
- **`/admin/*` lebt nur im Frontend** (UI-Routen + Nav-Labels).
- **18 OpenSpec-Specs nennen `/api/admin/*`-Pfade**, die in main.go nie exponiert wurden. Diese Specs sind nicht *durch diesen Change* falsch, sondern waren bereits drift-behaftet — der Change bereinigt diese Drift mit auf.

## Schlüsselentscheidungen

### 1. Hart cut für UI-Routen, keine Übergangs-Redirects

**Entschieden.** Bookmarks mit `/admin/*` brechen mit Deploy. Begründung:

- Aktive Nutzer sind ein überschaubarer Vorstands-/Trainerkreis.
- Doppel-Routing mit `<Navigate>`-Wrappern müsste später noch einmal weggeräumt werden — Folge-Change ohne Mehrwert.
- Risiko ist beschränkt auf "URL nochmal anklicken / aus Sidebar wählen".

**Konsequenz:** `CHANGELOG.md`-Eintrag + Hinweis im Vorstands-Chat (manuell, kein Code-Artefakt).

### 2. Tests gegen echte Production-Routerstruktur

**Entschieden.** Statt Mini-Router in jedem Test umstellen wir auf einen geteilten Aufbau.

**Vorgehen:**

1. `cmd/teamwerk/main.go` wird umstrukturiert: die Routen-Konfiguration wandert in eine eigene Funktion `buildRouter(deps Dependencies) chi.Router`. `main()` und Tests rufen beide diese Funktion auf.
2. `internal/testutil/` bekommt einen neuen Helper `NewProductionServer(t, db) *httptest.Server`, der alle Handler-Constructors mit Test-Dependencies aufruft und `buildRouter` mountet.
3. Bestehende Tests rufen statt `r.Post("/api/admin/...", h.X)` jetzt `srv := testutil.NewProductionServer(t, db)` und sprechen den echten Pfad an.

**Vorteile:**

- Tests fangen Routing-Regressionen (vergessenes Mount, falsche Middleware-Gruppe).
- Authentifizierungs-/Autorisierungs-Tests testen tatsächlich die echte Middleware-Kette.

**Trade-offs:**

- Test-Setup wird teurer (Dependencies wie `Hub`, `Mailer`, `Notif`-Services müssen für jeden Test bereitstehen — viele davon können Stubs sein).
- Die Umstellung passiert in **einem** Stream-A-Commit pro Domäne, nicht in einem riesigen Refactor.

**Alternative verworfen:** Reines String-Replace `/api/admin/X` → `/api/X` in den Mini-Routern. Schneller, fängt aber weiterhin keine Routing-Regressionen.

### 3. Phase 2: Single-Step-Migration, kein Doppelmount

**Entschieden.** Backend-Routen + Frontend-Aufrufer + Tests in einem PR.

**Warum kein Doppelmount?**

- TeamWERK hat keine externen API-Konsumenten. Das einzige Frontend wird zusammen mit dem Backend deployed (`//go:embed`). Es gibt keinen Zeitpunkt, an dem altes Frontend gegen neues Backend spricht.
- Doppelmount würde temporär die Inkonsistenz vergrößern statt sie zu verkleinern.

### 4. SSE-Event-Rename als kritischer Pfad

`useLiveUpdates((event) => if (event === 'kalender-event') reload())` wird in mehreren Pages verwendet. Falls eine vergessen wird, bleibt sie nach Spielplan-Änderungen stumm.

**Risiko-Mitigation:**

- Grep-Audit aller `'kalender-event'`-Vorkommen vor und nach Phase-2-Commit (siehe `tasks.md` Stream E).
- Backend-Broadcast (`hub.Broadcast("kalender-event")`) und Frontend-Subscription müssen in derselben Iteration umgestellt sein.

### 5. UI-Routen bleiben deutsch — API geht englisch

**Bewusste Asymmetrie:**

```
UI-Routen (User-facing):     /kalender, /mitglieder, /nutzer, /diensttypen
API-Routen (intern):          /api/games, /api/members, /api/users, /api/duty-types
```

**Warum?**

- User-facing-Begriffe folgen der Sprache der Domäne ("Kalender", "Mitglieder", "Beitrittsanfragen") — das ist eine Vereins-Verwaltung im deutschsprachigen Raum.
- API-Routen sind Dev-facing und folgen englischen Konventionen (REST-/Plural-/Lowercase). Konsistent mit `/api/training-sessions`, `/api/duty-*`, `/api/membership-requests`.
- Mischformen wie `/api/kalender` waren historisch gewachsen und sind die Ausnahme, nicht die Regel.

**Konsequenz:** Mappings wie `path="kalender"` (UI) → `api.get('/api/games')` (API-Aufruf) sind explizit gewollt.

### 6. `/anfragen` als echte Route, nicht als `<Navigate>`

Die Notification-URL `/admin/mitgliedschaft?id=X` zeigte auf eine Page, die nicht existiert. Statt einen neuen, sauberen Fix-Pfad zu erfinden, nutzen wir die bereits vorhandene Konvention: `/anfragen` wird zur echten Route, die `AdminUsersPage` mit voreingestelltem Tab und ggf. Highlight rendert.

**Vorteil:** Ein semantischer Pfad, der die User-Intention abbildet ("Ich klicke auf eine Anfrage in der E-Mail").
**Folge-Cleanup:** Toter `MembershipRequestsPage.tsx` wird gelöscht (separater kleiner Aufräumschritt, nicht im Scope dieses Proposals — wird in `tasks.md` als optionaler Task aufgeführt).

## Sequenzierung

Die Streams in `tasks.md` sind so geschnitten, dass jeder Stream eigenständig läuft und Pull-Request-fähig ist. Empfohlene Reihenfolge:

```
Stream A (Backend-Tests)  ─┐
                            ├─→ Verifikation läuft auf Production-Router
Stream B (Backend /games) ─┘    bevor das Frontend migriert wird

Stream C (UI /admin raus) ─┐
                            ├─→ UI ist konsistent
Stream D (UI /api/games)  ─┘    sobald Backend stabil ist

Stream E (Verifikation)        Smoke-Tests + manuelles E2E
```

Streams A und B können parallel laufen, weil sie unabhängige Files berühren.
Stream C kann parallel zu A/B laufen.
Stream D **muss** nach Stream B kommen (sonst frontend ruft tote Pfade auf).

## Risiken & Gegenmaßnahmen

| Risiko | Wahrscheinlichkeit | Gegenmaßnahme |
|---|---|---|
| SSE-Event nicht überall umbenannt → Tabs werden stumm | Mittel | Grep-Audit `'kalender-event'` Stream E |
| Test-Production-Router-Umstellung dauert länger | Mittel | Eigener PR pro Domäne (Stream A1–A4) |
| Spec-Updates übersehen einen Hit | Niedrig | Final-Grep `/api/admin\|/api/kalender` über `openspec/specs/` |
| Auto-Duty-Regen-Trigger bricht | Niedrig–Mittel | Spielplan-CRUD + Slot-Regen manuell durchspielen (Stream E) |
| User klickt alten Bookmark → 404 | Hoch (akzeptiert) | CHANGELOG + Vorstands-Chat-Ankündigung |

## Was wir NICHT entscheiden

Diese Punkte bleiben absichtlich offen / werden in Folge-Proposals adressiert:

- **`/api/upload/*` vs `/api/uploads/*` vs `/api/files/*`** — separater Proposal "api-files-konsolidierung". Drei Konzepte (Bilder, Dokumente, SEPA) verdienen eigene Klärung.
- **`/api/profile/kind/*`** — Deutsch-Englisch-Mischung im Profil-Strang. Kleiner eigener Cleanup, gegen Erweiterungs-Risiko nicht in diesen Change ziehen.
- **Verb-im-Pfad** (`/api/auth/login`, `/api/impersonate`, `/api/auth/invite`) — REST-Purismus liefert hier wenig praktischen Mehrwert; bewusst belassen.
- **DB-Tabellen-Namen** — `games`-Tabelle heißt bereits englisch; kein DB-Eingriff in diesem Change.

## Offene Fragen für die Umsetzung

Während Implementation klären, aber nicht Show-Stopper für dieses Proposal:

- `AdminUsersPage`-Tab-State: existiert schon ein Tab-System mit URL-Param oder muss `?tab=anfragen&id=X` neu eingebaut werden?
- `vorstand-vault`-UI-Route (`/admin/tresor-einrichtung`): noch geplant oder Spec-Drift? — bei Stream-C-Beginn checken.
- Welche Page-Komponenten-Dateien sollen umbenannt werden (`AdminUsersPage.tsx` → `NutzerPage.tsx` etc.)? Optional, kosmetisch.
