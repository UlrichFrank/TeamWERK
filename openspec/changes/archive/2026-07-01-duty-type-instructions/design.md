## Context

`duty_types` ist heute rein strukturell (Name, Stunden, Anchor,
Behavior-Varianten). Fachliche Hinweise, *wie* ein Dienst tatsächlich
auszuführen ist, existieren nur in Köpfen und WhatsApp-Verläufen. Wir wollen
diesen Kontext an genau die Stelle bringen, an der der Nutzer ihn braucht — in
den Slot in der Dienstbörse — und ihn dort pflegbar machen, wo er hingehört —
in die Verwaltung des Dienst-Typs.

`/dokumente` bietet bereits ein vollwertiges Datei-System (`file_folders`,
`files`, `folder_permissions`) mit Rechte-Auflösung (`resolveAccess`) und
Signed-Download-Token (`/api/files/{id}/download-token` →
`?token=…`-Query). Der bestehende Router-Link `/dokumente/datei/{fileId}`
(`DocumentFileLinkPage`) wickelt die Rechte-Abklärung serverseitig ab und
leitet auf die Download-URL um. Genau diesen Link nutzen wir in Markdown als
Bild- bzw. Datei-Referenz — ohne neuen Endpoint.

## Goals / Non-Goals

**Goals:**

- Eine gepflegte Kurz-Anleitung pro Dienst-Typ, editierbar in Markdown.
- Ausführende Nutzer öffnen die Anleitung mit einem Klick aus dem Slot heraus.
- Bilder werden aus dem bestehenden Dokumente-Bereich referenziert, kein
  separater Bild-Upload.
- Rendering ist gegen XSS abgesichert.
- Kein Sonderfall-Rechtsprüfungs-Endpoint für Anleitungsbilder.

**Non-Goals:**

- **Keine Anleitung auf Slot- oder Template-Ebene.** Anleitung ist
  Typ-Eigenschaft. Feinkörnigere Varianten („Kasse @ Auswärtsspiel U16") sind
  bewusst out of scope — falls jemals nötig, additiv nachrüstbar über eine
  Override-Spalte in `duty_templates` oder `duty_slots`.
- **Keine Versionierung / Historie.** Es gibt genau eine aktuelle Anleitung
  pro Typ; `instruction_updated_at`/`instruction_updated_by` reichen für
  Nachvollziehbarkeit („wer hat zuletzt was geändert").
- **Kein Bilder-Upload direkt aus dem Editor.** Bilder werden über
  `/dokumente` hochgeladen (bestehender Flow), im Editor per URL verlinkt.
- **Kein WYSIWYG-Editor.** Reine Textarea + Live-Preview. Vorstand ist
  klein genug, Markdown ist zumutbar.
- **Kein Datei-Picker** in dieser Iteration. Hinweistext erklärt das
  URL-Muster. Follow-up möglich.

## Decisions

### 1. Anleitung hängt an `duty_types`, nicht an `duty_templates` oder `duty_slots`

**Wahl:** Ein Markdown-Feld auf `duty_types`.

**Rationale:** Der Ablauf eines Kassendienstes hängt vom Dienst-Typ ab, nicht
vom konkreten Spiel. Redaktionell N-fach reduziert (ein Typ vs. jeder Slot),
Speicherplatz vernachlässigbar (< 1 KB pro Typ realistisch). Slots werden
regelmäßig auto-regeneriert (Auto-Duty-Regen); Anleitungen dort zu speichern,
wäre gegen den Datenlebenszyklus.

**Alternativen erwogen:**

- **`duty_templates.instruction_md`** — feingranularer, aber Redakteur muss
  n·m Templates pflegen. Vom Nutzer explizit abgewählt (Weiche 1: „A").
- **`duty_slots.instruction_md`** — sinnlos, würde bei jeder Regen-Runde
  verloren gehen oder `is_custom=1` erzwingen.

### 2. Bild-Rechte per Konvention „Anleitungen"-Ordner, nicht per Sonder-Route

**Wahl:** Anleitungen verlinken auf `/dokumente/datei/{fileId}` — Auflösung
läuft durch den bestehenden `resolveAccess`/`download-token`-Pfad. Wir
dokumentieren, dass Anleitungs-Bilder im Ordner „Anleitungen" mit
`everyone/can_read=1` liegen.

**Rationale:** Kein neuer Endpoint, kein Markdown-Parser im Backend, keine
zusätzliche Berechtigungs-Achse. Die einzige Kosten-Seite: wenn der Vorstand
ein Bild aus einem `vorstand`-only-Ordner referenziert, sehen Spieler ein
kaputtes Bild statt einer Berechtigungs-Fehlermeldung. Das ist ein
akzeptabler Trade-off; Broken-Image-Icon plus Alt-Text macht das Problem
sichtbar und ist mit einem Blick auf die Ordner-Konfiguration behoben.

**Alternativen erwogen:**

- **Sonder-Route `/api/duty-types/{id}/inline-image/{fileId}`** mit
  Markdown-Parser + fileId-Whitelist. Wasserdicht, aber wesentlich mehr Code
  und Backend-Markdown-Kenntnis. Vom Nutzer explizit abgewählt (Weiche 2:
  „a").
- **Explizite `duty_type_instruction_images`-Whitelist-Tabelle.** Redakteur
  müsste jedes Bild doppelt „verbinden". UX-Regression.

### 3. Sicherer Renderer: `react-markdown` + `rehype-sanitize`, kein `dangerouslySetInnerHTML`

**Wahl:** Standard-Konfiguration von `rehype-sanitize` (github-schema)
whitelistet die üblichen Block-/Inline-Elemente inkl. `img`, blockiert
`<script>`, `<style>`, `<iframe>`, `on*`-Attribute. Relative URLs wie
`/dokumente/datei/123` sind erlaubt; externe `http(s)`-URLs werden **nicht**
gesondert blockiert (Vorstand-Autorenkreis).

**Rationale:** Standard-Konfig ist gut geprüft; wir umgehen aktive
Sanitizer-Pflege. Wir laden aber weder Bilder anonym noch externes JS —
Sicherheitsfläche bleibt klein.

### 4. Server-Flag `has_instruction` statt Client-seitiger Prüfung

**Wahl:** `GET /api/duty-board` schließt pro Slot `has_instruction: boolean`
ein (via `LEFT JOIN` + `instruction_md != ''`). Der Client zeigt den
Link genau dann.

**Rationale:** Wir wollen den Icon-Link nicht mit einer Extra-Runde je Slot
oder mit einer Vorab-Volllast auf `GET /api/duty-types` verkaufen. Ein
Boolean pro Slot ist praktisch kostenlos und hält die Board-Response
kompakt.

### 5. Beispieltext bei leerer Anleitung, aber kein Auto-Save

**Wahl:** Beim Öffnen des Editors mit leerem `instruction_md` wird die
Textarea mit einem festen Beispieltext (siehe unten) vorbelegt.
Speichern-Button ist zunächst disabled — er aktiviert sich, sobald der
User den Text verändert. Falls der User den Editor ohne Änderung schließt,
bleibt `instruction_md` leer und `has_instruction` bleibt `false`.

**Rationale:** Wir geben eine Struktur vor („was gehört rein"), zwingen sie
aber niemanden auf. Kein „ich hab nur mal geklickt, jetzt steht überall
Boilerplate".

**Beispieltext (verbindlich, `web/src/lib/dutyInstructionTemplate.ts`):**

```markdown
## Vorbereitung
Was muss vor Dienstbeginn erledigt sein?

## Ablauf
1. Erster Schritt
2. Zweiter Schritt

## Häufige Fragen
- Frage → Antwort

## Bilder
Bilder aus dem Ordner **Anleitungen** unter /dokumente einbinden:
`![Kurzbeschreibung](/dokumente/datei/DATEI_ID)`
```

### 6. Frontend-Route deutsch, Backend englisch

Gemäß Namenskonvention (`docs/agent/04-api-db.md`):

- Backend: `PUT /api/duty-types/{id}/instruction`.
- Frontend-Route: `/dienste/anleitung/:typeId` (unter `AppShell`).

## Risks / Trade-offs

- **Broken-Image bei Ordner-Fehlkonfiguration:** Verlinkt der Vorstand aus
  Versehen ein Bild aus einem `vorstand`-only-Ordner, sehen Spieler ein
  kaputtes Bild. → Mitigation: Konvention im Wiki + Doku im Editor
  („Bilder aus dem Ordner **Anleitungen** einbinden"). Kein Code-Aufwand.
- **Markdown-Erosion:** Vorstand kopiert Markdown-Fragmente aus dem Web mit
  rohem HTML → Sanitizer verwirft es stumm. → Mitigation: Preview zeigt das
  Ergebnis sofort, User merkt die Diskrepanz beim Speichern-Vorschau.
- **Bundle-Größe:** `react-markdown` + `rehype-sanitize` sind zusammen
  ca. 55 kB gzipped. Auf VPS Linux XS (1 GB RAM) irrelevant, für die Vite-
  Bundlegröße unproblematisch. Wird via Route-Split nur auf
  `DutyInstructionPage` geladen (dynamic `import(...)` in `App.tsx`).
- **`has_instruction` skew bei Reihenfolgen-Wettlauf:** Vorstand ändert
  Anleitung, User hat Board-Snapshot älter als Broadcast. → SSE-Event löst
  Reload aus (`useLiveUpdates("duties")` in `DutyPage`).
- **`duty_types` ohne Anleitung sollen nicht als „Anleitung vorhanden"
  erscheinen, wenn der User schnell zurücknavigiert:** Der Server ist die
  Wahrheit. Wenn `has_instruction=false`, wird kein Link gerendert; die
  Route `/dienste/anleitung/:typeId` zeigt bei leerer Anleitung eine
  Placeholder-Meldung „Für diesen Dienst gibt es noch keine Anleitung".

## Migration Plan

1. `015_duty_type_instruction.up.sql` aufspielen (idempotent, `ADD COLUMN`
   mit `DEFAULT ''` und `NULL`).
2. Backend deployen — bestehende Reads liefern `instruction_md=""`,
   `has_instruction=false`, alles läuft unverändert.
3. Frontend deployen — Editor sichtbar, aber ohne Inhalt keine Icons in der
   Slot-Liste.
4. Vorstand legt Ordner „Anleitungen" in `/dokumente` an, setzt Rechte auf
   `everyone/read`, füllt die ersten Anleitungen. Kein Zwang, wann.
5. Rollback: `015_duty_type_instruction.down.sql` droppt die drei Spalten.
   Kein Datenverlust außerhalb der Anleitungen selbst.

## Open Questions

_keine — Weichen (A / a / minimal + Beispieltext) sind mit dem Nutzer
abgestimmt, Link-Platzierung „je Slot" bestätigt._
