# Design — Entbranding & Instanz-Konfiguration

## Leitprinzip: Self-Hosted Single-Tenant

Jeder Verein betreibt **seine eigene** TeamWERK-Instanz (eigene Binary, eigene SQLite-DB, eigene `/etc/teamwerk/env`). Es gibt **keine** geteilte Instanz für mehrere Vereine. Folge: Konfiguration ist instanz-global, keine `club_id`-Spalten, keine Mandanten-Isolation nötig. Passt zu AGPL: „nimm den Code, hoste selbst, teile Verbesserungen".

## Konfigurationsebenen

| Wert | Quelle | Begründung |
|---|---|---|
| Vereinsname, Kurzname | ENV (`CLUB_NAME`, `CLUB_SHORT`) | beim Deploy bekannt, selten geändert |
| Produktions-Domain / CORS-Origin | ENV (`PUBLIC_URL`) | deploy-spezifisch, sicherheitskritisch |
| Absender-/Support-E-Mail | ENV (`MAIL_FROM`, `SUPPORT_EMAIL`) | SMTP-gebunden |
| Markenfarben | Build-Variablen → Tailwind-Theme | Frontend wird gebaut/embedded |
| Logo | austauschbare Asset-Datei mit neutralem Default | binär, nicht ENV-tauglich |
| Texte (Welcome, Login, Beitritt) | optionale Override-Dateien, sonst neutraler Default | i18n-/vereins-spezifisch |
| PDF-Anhänge (Satzung etc.) | optionale Dateien je Instanz | rechtlich vereins-spezifisch |
| SEPA-Stammdaten | bereits DB-Config (Einstellungen → Verein) | personenbezogen, gehört nicht in den Repo |

## Theming-Strategie (Farben)

Markenfarben sind heute fast vollständig in `tailwind.config.js` zentralisiert (nur 2 Streu-Hex im Code). Optionen:

- **A) Build-Zeit-Theme** (empfohlen für Start): Farben aus ENV beim `pnpm build` in die Tailwind-Config injizieren. Einfach, kein Runtime-Overhead, passt zum embed.FS-Modell.
- **B) Runtime-CSS-Variablen**: `--brand-*` als CSS-Custom-Properties, zur Laufzeit setzbar. Flexibler (Theme ohne Rebuild), aber Umbau aller Tokens nötig.

Start mit A; B als spätere Option dokumentieren. Die 2 Streu-Hex-Stellen werden auf Tokens umgestellt (CLAUDE.md-Regel „nur `brand-*`").

## Neutraler Default

Default-Identität = generischer Demo-Verein („Beispielverein", neutrales Logo, Platzhalterfarben). Niemand bekommt versehentlich Team-Stuttgart-Branding ausgeliefert. Team Stuttgart wird zu *einer* Beispiel-Konfiguration (nicht eingecheckt, da `PUBLIC_URL`/Mail personenbezogen).

## Welcome-Mail-Anhänge: aus /dokumente statt Embed

Heute lädt `welcome_email.go` eine **feste Liste** (`satzung.pdf`, `gebuehrenordnung.pdf`, `leitbild.pdf`, `logo.svg`) aus `mailer.AttachmentFS` (eingebettet). Das ist vereins-spezifisch und PII-/Branding-relevant.

**Neu:** Der bereits existierende Dokumente-Bereich (`/dokumente`, `internal/files` mit Ordnern + Permissions) wird zur Quelle. Eine Datei trägt ein Flag „Welcome-Anhang"; der Vorstand setzt es in der UI. Die Welcome-Mail liest beim Versand die markierten Dateien aus dem Store.

- **Minimaler Schema-Eingriff:** ein boolesches Flag/Verknüpfung pro Datei (Migration mit nächster freier Nummer)
- **Berechtigung:** Markieren nur `vorstand`/`admin`; Mutation → `Broadcast` + `useLiveUpdates` (SSE-Pflicht)
- **Folge für ①:** Die eingebetteten Club-PDFs entfallen aus dem Repo — kein neutraler Default-PDF nötig, da „keine Markierung = kein Anhang" ein valider Zustand ist.

## Abgrenzung

Test-Fixtures dürfen weiterhin „Team Stuttgart" o. Ä. als Beispieldaten nutzen — sie sind nicht Teil des ausgelieferten Defaults und kein PII (synthetische Namen, siehe ①).
