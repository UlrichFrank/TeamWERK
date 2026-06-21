## Context

TeamWERK ist eine React-SPA, eingebettet als statische Assets in eine Go-Binary. Die App nutzt React Router v6, einen `AuthContext` (Email/Rolle/Vereinsfunktionen/IsParent aus JWT) und ist als PWA installierbar (`vite-plugin-pwa`). Es gibt **kein** clientseitiges Telemetrie- oder Tracking-System; das Backend hat keine Endpoints für Telemetrie und es soll auch keine geben (Out of Scope für diese Change).

Eine **Matomo-Instanz** läuft bereits unter einer mittwald-Domain (für die Vereins-Homepage). Sie ist über HTTPS erreichbar und kann eine zusätzliche Site-ID für TeamWERK aufnehmen. Auftragsverarbeitungsvertrag mit mittwald ist vorhanden.

Die Zielnutzer von TeamWERK sind ca. 50–150 Vereinsmitglieder (Trainer, Vorstand, Eltern, erwachsene Spieler, Kinder über Proxy-Accounts). Die Nutzung ist **eingeloggt** — wir können also bei Bedarf grobe Segmente (Channel, Rolle, Team) anonym mitschicken, ohne PII zu sammeln.

## Goals / Non-Goals

**Goals:**

- Sicht auf **welche Routen** im TeamWERK genutzt werden und **wie oft** (Pageviews pro Pfad, aktive Sessions, Mobile/Desktop-Verteilung).
- Grobe Segmentierung nach **Channel** (PWA vs. Browser), **Rolle** (admin vs. standard) und **Team** (Slug des Haupt-Teams) — anonym über Custom Dimensions.
- DSGVO-konformer Default ohne Cookie-Banner: Cookieless Tracking, IP-Anonymisierung, DoNotTrack respektieren.
- Saubere Konfiguration über Vite-Env: lokale Dev und unkonfigurierte Umgebungen tracken **nichts**.
- Geringer Footprint (zusätzliche Bundle-Größe < 15 KB gzip).

**Non-Goals:**

- Backend-/Server-Monitoring (RAM, CPU, Latenz, 5xx) — separates Thema, andere Lösung.
- Personalisiertes Tracking, User-IDs, Profiling einzelner Mitglieder.
- Tracking fachlicher Custom-Events ("Dienst übernommen", "Spiel angelegt" …) — erst in einer Folge-Change, falls Bedarf entsteht.
- Service-Worker-Hintergrund-Tracking (offline queued events, Background Sync) — Komplexität nicht gerechtfertigt.
- Reverse-Proxy von `/matomo.php` über das eigene Nginx (Adblocker-Umgehung) — erst falls reale Verzerrung beobachtet wird.

## Decisions

### D1 — Eigener schlanker Wrapper um `window._paq` (kein npm-Paket)

**Wahl:** Direkter Aufruf von Matomos Standard-Tracking-API über das globale `_paq`-Array, gekapselt in `web/src/lib/telemetry.ts` (< 100 LoC). Das offizielle `matomo.js` wird per `<script>`-Tag aus dem Tracker-Host nachgeladen, sobald `VITE_MATOMO_URL` gesetzt ist.

**Warum:**
- **Pivot von ursprünglicher Wahl `@jonkoops/matomo-tracker-react`:** Package ist beim `pnpm add` als **deprecated** markiert ("This package is no longer maintained"). Damit fällt der Hauptnutzen weg.
- Wir brauchen aus dem Tracker konkret nur drei Operationen: Init, `setCustomDimension`, `trackPageView`. Die kann ein 20-Zeilen-Wrapper sauberer ausdrücken als ein Provider, der React-Context für eine globale Variable einrichtet.
- Bundle wird kleiner (kein extra npm-Paket, `matomo.js` wird vom Matomo-Server nachgeladen und im Browser gecacht).
- Kein Deprecation-Risiko, keine Fremd-Abhängigkeit, die unsere Sicherheitspolicy bewerten muss.

**Konsequenz:** Statt `<MatomoProvider>` + `useMatomo()`-Hook nutzen wir einen Helper-Modul mit Funktionen `initTelemetry()`, `setChannelDimension()`, `setTeamSlugDimension()`, `setRoleDimension()`, `trackPageview(href, title)`. Die Init wird einmal in `main.tsx` aufgerufen; das Tracking-Effect im `AppShell` ruft die Helper direkt.

**Alternativen verworfen:**
- **`@jonkoops/matomo-tracker-react`:** deprecated (Stand pnpm-Resolve).
- **`@datapunt/matomo-tracker-react`:** archiviert (älter, daher ursprünglich verworfen).
- **Eigene voll-Reimplementierung des Tracking-Protokolls (HTTP-POST von Hand):** Overkill — `matomo.js` macht genau das, ist klein und gut getestet.

### D2 — SPA-Pageviews via Route-Listener im `AppShell`

**Wahl:** Im `AppShell` (der Layout-Komponente, die alle authentifizierten Routen umschließt) wird `useLocation()` beobachtet; bei jedem `pathname`-Wechsel wird `trackPageView({ documentTitle, href })` aufgerufen.

**Warum:** `AppShell` ist die zentrale Stelle, an der jeder Routenwechsel innerhalb der App vorbeikommt. Auth-Status ist hier bekannt → wir können entscheiden, **nicht** zu tracken, bevor `loading` abgeschlossen ist.

**Spezialfall Public-Routes** (Login, Register, Passwort-Reset, Beitrittsantrag): Diese liegen außerhalb des `AppShell`. Für diese Change **tracken wir sie zunächst nicht** — sie haben keinen Team-/Rollen-Kontext und sind selten. Falls sich Bedarf zeigt, in Folge-Change ergänzen.

### D3 — Custom Dimensions: Channel (1), Team-Slug (2), Rolle (3)

**Slot 1 (`channel`) — Erkennung Pflichtfeld pro Pageview:**

```ts
const isPWA =
  window.matchMedia('(display-mode: standalone)').matches ||
  // iOS Safari Homescreen
  (navigator as { standalone?: boolean }).standalone === true
const channel = isPWA ? 'pwa' : 'browser'
```

**Slot 2 (`team_slug`):** Slug des "Haupt-Teams" des eingeloggten Nutzers. Aktuell ist die Team-Zugehörigkeit **nicht** im `AuthContext`/JWT, sondern muss aus dem Member-Kontext abgeleitet werden.
- **Vorgehen:** Bei `AppShell`-Mount einmal `GET /api/me/teams` (existiert: liefert Teams für das aktuelle Member über `team_names-endpoint` Cap) aufrufen, ersten Eintrag als `team_slug` verwenden, in einem `useState` halten. Mehrfach-Team-User → erstes Team gewinnt; Trainer/Vorstand ohne klares "Haupt-Team" → `mixed`. Nutzer ohne Team → `none`. **Wert wird beim Pageview-Tracking als Custom Dim 2 mitgeschickt.**
- Falls Endpoint fehlt oder fehlschlägt: Dimension einfach weglassen (Matomo akzeptiert das).

**Slot 3 (`role`):** Aus `user.role` (`admin` | `standard`). Trivially aus `AuthContext`.

**Warum diese drei und nicht mehr:**
- Vereinsfunktionen (`spieler`, `trainer`, …) sind **n pro Member** → passen schlecht in eine flache Dimension. Falls später nötig: pro-Funktion als boolesche Custom Variable, oder eigene Dimension `primary_function`.
- Eltern-Eigenschaft (`isParent`) wäre interessant, würde aber den Dim-Slot 4 verbrauchen — verschoben auf "falls Bedarf".

### D4 — Datenschutzfreundlicher Default

**Konfiguration des Trackers beim Initialisieren:**

```ts
createInstance({
  urlBase: import.meta.env.VITE_MATOMO_URL,
  siteId: Number(import.meta.env.VITE_MATOMO_SITE_ID),
  disabled: !import.meta.env.VITE_MATOMO_URL || !import.meta.env.VITE_MATOMO_SITE_ID,
  // Cookieless
  configurations: {
    disableCookies: true,
    setSecureCookie: true,
    setRequestMethod: 'POST',
  },
})
```

**Zusätzlich serverseitig in Matomo-Admin (mittwald):**
- Privacy → Anonymize → IP anonymize 2 Bytes (`192.168.xxx.xxx`).
- Privacy → Anonymize → DoNotTrack support aktivieren.
- Privacy → Force anonymous tracking aktivieren (kein Visitor-Profil).

**Warum:** Damit ist kein Cookie-Banner nötig. Wir setzen keine Cookies, die nicht technisch unbedingt erforderlich sind. Matomo's eigene Doku & viele deutsche Aufsichtsbehörden akzeptieren diesen Modus als einwilligungsfrei. **Aufsicht-strikte Auslegung** (z.B. Hamburg) verlangt teils dennoch Einwilligung — Risiko ist aber für ein internes Vereinstool gering und gegen den UX-Nutzen abwägbar.

**Alternative verworfen:** Cookie-basiertes Tracking mit Consent-Banner. Mehr UX-Reibung, mehr Compliance-Aufwand, und für anonyme Nutzungs-Aggregate nicht nötig.

### D5 — Konfiguration über Vite-Env, eingebackene Werte

**Wahl:** `VITE_MATOMO_URL` und `VITE_MATOMO_SITE_ID` werden zum Build-Zeitpunkt eingebacken. Sind sie leer/ungesetzt, ist der Tracker als `disabled: true` initialisiert und macht keine Requests.

**Warum:** Standard-Vite-Mechanik. Matomo-URL und Site-ID sind kein Geheimnis (steckt sowieso in jedem HTTP-Request). Dev-Builds tracken nichts.

**Konsequenz:** Bei Änderung der Matomo-URL muss neu deployed werden. Akzeptabel — das ändert sich extrem selten.

### D6 — `/datenschutz`-Seite: neu anlegen oder existierende erweitern?

**Beobachtung:** Es gibt aktuell **keine** öffentliche `/datenschutz`-Route in `App.tsx` (es existieren nur Datenschutz-Tabs *innerhalb* von Profil und MemberDetail).

**Wahl:** Neue **öffentliche** Route `/datenschutz` (Public-Tier) mit statischer Markdown-/JSX-Seite. Wird auch im Footer/Login verlinkt.

**Warum:** DSGVO verlangt jederzeit erreichbare Datenschutzerklärung — die sollte nicht hinter Login liegen. Diese Change ist der Anlass, das zu lösen; die Seite enthält neben dem Matomo-Absatz die Basis-Infos (Verantwortlicher, Hosting, gespeicherte Daten, Rechte).

**Risiko:** Scope-Creep — eine komplette Datenschutzerklärung zu formulieren ist eine juristische Aufgabe. **Mitigation:** Wir liefern in dieser Change eine **Minimal-Version** mit Platzhaltern für Verantwortlichen und vollständigem Text, der vom Vorstand vor Go-Live ergänzt/freigegeben werden muss (Akzeptanzkriterium der Tasks).

## Risks / Trade-offs

- **[Adblocker blockieren Matomo-Requests]** → Verzerrte Statistiken (echte Nutzung höher als gemessen). **Mitigation:** Für interne Vereinsnutzer eher gering; bei reale Beobachtung in Folge-Change Reverse-Proxy `/matomo.php` einbauen.
- **[Cookieless Tracking unterschätzt Unique Visitors]** → Matomo nutzt im Cookieless-Mode Fingerprinting (IP+UA-Heuristik) für Visitor-Erkennung, das ist ungenauer. **Mitigation:** Wir lesen *Trends*, keine absoluten Visitor-Zahlen. Für unsere Frage ("welche Routen werden genutzt") irrelevant.
- **[`/api/me/teams` für Team-Dim verfügbar oder nicht?]** → Falls Endpoint nicht existiert oder nicht den erwarteten Slug liefert, Dimension fehlt. **Mitigation:** Endpoint-Existenz wird in Task 1 verifiziert; bei Lücke fällt `team_slug` einfach weg, Tracking funktioniert trotzdem.
- **[Aufsichtsbehörden-Risiko Cookieless-Annahme]** → Mögliche strikte Lesart, dass auch anonymes Cross-Site-Tracking Einwilligung braucht. **Mitigation:** Transparenz in `/datenschutz`; falls in Zukunft ein Verbandsverein eine restriktivere Auslegung verlangt, Tracker via Env-Var deaktivierbar (`VITE_MATOMO_URL=""`).
- **[Build-Zeit-Konfig erschwert Multi-Env]** → Pre-Prod und Prod teilen die Binary; wir tracken alle gleich. **Mitigation:** Nicht-Production-Hosts in Matomo-Admin als "Excluded sites" eintragen oder als separate Site-ID konfigurieren — falls jemals nötig.
- **[Datenschutzerklärung juristisch unvollständig]** → Wir liefern Text, der Vorstand prüfen muss. **Mitigation:** Akzeptanzkriterium in Tasks; Go-Live erst nach Vorstand-Freigabe.

## Migration Plan

Keine Datenmigration — reine Frontend-Erweiterung.

**Rollout:**
1. Code-Change mergen mit `VITE_MATOMO_URL=""` im Default → Tracker bleibt deaktiviert.
2. Site in Matomo-Admin (mittwald) anlegen, Datenschutz-Einstellungen (D4) konfigurieren.
3. `VITE_MATOMO_URL` / `VITE_MATOMO_SITE_ID` in CI/Deploy-Pipeline setzen → erster Build mit aktivem Tracking → Deploy.
4. Im Matomo-Dashboard verifizieren: kommen Pageviews + Custom Dimensions an, IP anonymisiert, kein Cookie gesetzt?

**Rollback:** Env-Variablen leeren und neu deployen → Tracker `disabled: true`, kein Request mehr.

## Open Questions

- ~~Existiert `/api/me/teams` schon mit Slug oder muss `team_slug` aus einer anderen Quelle (Member-Detail) abgeleitet werden?~~ **Geklärt (Task 1.1):** `GET /api/teams/my` existiert, liefert `[{id, name, isExtended}]` — Slug ist **nicht** Bestandteil der Response. Wir slugifizieren den Team-Namen clientseitig (lowercase, Umlaute → ae/oe/ue/ss, übrige Sonderzeichen → `-`). Mehrere Teams → erstes (alphabetisch) gewinnt, oder `mixed` falls > 1; kein Team → `none`; Fehler → `unknown`. Saison-aktiv-Filter ist bereits im Endpoint berücksichtigt.
- Soll Public-Tier (`/login`, `/register`, …) ebenfalls getrackt werden? **Empfehlung:** vorerst nein; bei späterem Bedarf Folge-Change.
- Wer im Vorstand prüft/freigibt den Datenschutz-Text vor Go-Live?
