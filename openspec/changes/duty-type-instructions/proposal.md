## Why

Die Dienstbörse zeigt heute *was* zu tun ist (Kasse, Wischer, Zeitnahme …), aber
nirgends *wie*. Neue Elternteile bekommen bei jedem Dienst wieder dieselben
Rückfragen. Ein einfacher Weg, pro Dienst-Typ eine gepflegte Kurz-Anleitung
(Markdown, gerne mit Bildern aus dem Dokumente-Bereich) an den Dienst zu hängen,
macht die Arbeit für die Ausführenden selbsterklärend und entlastet Vorstand /
sportliche Leitung.

## What Changes

- **Schema (Migration 015):** `duty_types` bekommt drei Spalten
  `instruction_md TEXT NOT NULL DEFAULT ''`, `instruction_updated_at TEXT`,
  `instruction_updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL`.
- **Backend – neue Route** `PUT /api/duty-types/{id}/instruction`
  (`vorstand` / `admin`), Body `{markdown: string}`. Setzt `instruction_md`,
  `instruction_updated_*`, ruft `h.hub.Broadcast("duties")`.
- **Backend – bestehende Reads erweitern:**
  - `GET /api/duty-types` liefert `instruction_md` sowie
    `instruction_updated_at`/`instruction_updated_by` mit.
  - `GET /api/duty-board` (BoardSlot) liefert zusätzlich `duty_type_id` und
    `has_instruction: boolean` (Server-Vorberechnung `instruction_md != ''`,
    damit der Client kein Markdown lädt, nur um festzustellen: da ist nichts).
- **Frontend – Editor:** Auf `AdminDutyTypesPage` bekommt jede Dienst-Typ-Zeile
  eine neue Aktion **„Anleitung"** → Modal (oder Detail-Sektion) mit
  Markdown-Textarea + Live-Preview darunter. Ist das Feld leer, wird die
  Textarea mit einem **Beispieltext** vorbelegt (siehe `design.md`),
  Speichern erst nach expliziter Bestätigung „Speichern".
- **Frontend – Viewer:** Auf `DutySlotList` erscheint pro Slot **neben dem
  Dienst-Namen** ein Icon-Link `<BookOpen>` mit `aria-label="Anleitung ansehen"`,
  **nur wenn** `has_instruction` gesetzt ist. Klick öffnet
  `/dienste/anleitung/{dutyTypeId}` (neue Seite `DutyInstructionPage.tsx`,
  Rendering via `react-markdown` + `rehype-sanitize`, kein rohes HTML).
- **Frontend – Bild-Referenzen:** Markdown-Bilder in Form
  `![Alt](/dokumente/datei/{fileId})` funktionieren, weil der Sanitizer
  relative URLs zulässt und das `DocumentFileLinkPage` bereits die
  Rechte-Abklärung per Download-Token übernimmt. **Kein neuer Bild-Endpoint.**
- **Konvention (Doku, kein Code):** Der Vorstand legt in `/dokumente` einen
  Ordner **„Anleitungen"** mit `folder_permissions.everyone.can_read=1` an.
  Bilder in Anleitungen liegen dort. Für Bilder außerhalb dieses Ordners
  greift die normale Ordner-Rechteprüfung — Nutzer ohne Zugriff sehen das
  Standard-Broken-Image des Browsers (bewusster Trade-off, kein Sonderfall im
  Code).
- **Live-Updates:** Der Editor speichert → Backend broadcastet
  `duties` → `DutyPage`/`DutyInstructionPage` reloaden via
  `useLiveUpdates`.
- **Tests:** siehe `## Test-Anforderungen`.

## Test-Anforderungen

| Route / Verhalten | Test | Status | Invariante |
|---|---|---|---|
| `PUT /api/duty-types/{id}/instruction` | `TestPutInstruction_HappyPath` | 200 | `instruction_md` in DB gesetzt, `updated_at` gefüllt, Broadcast `duties` |
| `PUT /api/duty-types/{id}/instruction` | `TestPutInstruction_Unauthenticated` | 401 | keine DB-Änderung |
| `PUT /api/duty-types/{id}/instruction` | `TestPutInstruction_ForbiddenForStandard` | 403 | Standard-User (ohne `vorstand`/`admin`) darf nicht schreiben |
| `PUT /api/duty-types/{id}/instruction` | `TestPutInstruction_NotFound` | 404 | unbekannte `id` liefert 404, kein Insert |
| `PUT /api/duty-types/{id}/instruction` | `TestPutInstruction_MissingBody` | 400 | Body ohne `markdown`-Feld wird abgelehnt |
| `GET /api/duty-types` | `TestListTypes_IncludesInstructionFields` | 200 | `instruction_md`, `instruction_updated_at` sind Teil der Antwort |
| `GET /api/duty-board` | `TestBoard_ExposesHasInstruction` | 200 | Slot mit Type-Anleitung liefert `has_instruction=true`, ohne Anleitung `false` |
| Sanitizer-Verhalten (Vitest) | `sanitizes disallowed html` | – | `<script>alert(1)</script>` im Markdown wird beim Rendern verworfen |
| Editor-Vorbelegung (Vitest) | `prefills example on empty instruction` | – | Öffnen mit leerem Feld setzt Textarea auf Beispieltext, Save erst nach Änderung |

## Capabilities

### New Capabilities

- `duty-type-instructions`: Modell, Berechtigungen und UX-Anforderungen für die
  gepflegte Kurz-Anleitung pro Dienst-Typ.

### Modified Capabilities

_keine — der bestehende `duties`-Capability bleibt unverändert; die neuen
Requirements leben eigenständig._

## Impact

- **Code (Backend):** `internal/duties/handler.go` (neue Route + Fields in
  `ListTypes` + `Board`-Query), `internal/app/router.go` (Route-Mount),
  `internal/duties/handler_test.go` (neue Tests). SSE nutzt vorhandenen
  `duties`-Event.
- **Code (Frontend):** `web/src/pages/AdminDutyTypesPage.tsx` (Editor-Modal),
  neue `web/src/pages/DutyInstructionPage.tsx`, `web/src/components/DutySlotList.tsx`
  (Icon-Link), `web/src/App.tsx` (neue Route `/dienste/anleitung/:typeId`).
- **DB-Migration:** `015_duty_type_instruction.up.sql` /
  `.down.sql`. Bestehende Zeilen bekommen `instruction_md=''`, `updated_at=NULL`.
- **Dependencies:** `pnpm add react-markdown rehype-sanitize` in `web/`.
- **CHANGELOG:** `[feat] duties: Anleitung pro Dienst-Typ (Markdown,
  gerendert für die Ausführenden)`.
- **Berechtigungen / RAM / Deploy:** unverändert. Kein CGo, kein neues
  Verzeichnis, kein Scheduler-Job.
- **Zero-Knowledge-Grenze:** Anleitung ist **kein** PII → wird als Klartext
  in der DB gehalten. (Nur Bank-/SEPA-Felder sind clientseitig verschlüsselt.)
