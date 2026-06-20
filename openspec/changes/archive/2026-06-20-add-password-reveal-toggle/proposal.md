## Why

Auf der Login-Seite (und allen weiteren Formularen mit Passwort-Eingabe) gibt es bislang keine Möglichkeit, das eingetippte Passwort sichtbar zu machen. Vertipper bleiben unentdeckt, was bei mobilen Geräten mit kleiner Tastatur und Großschreibung/Sonderzeichen-Wechsel häufig zu Login-Fehlversuchen führt.

Ein einfacher „Augen-Toggle" ist Industrie-Standard, hat aber einen Sicherheits-Nachteil: an einem fremden Browser, in dem das Passwort gespeichert ist, kann jeder Anwesende durch Klick auf das Auge das Klartext-Passwort des eigentlichen Eigentümers ablesen. Dieser Change löst beide Probleme, indem der Toggle **nur dann** erscheint, wenn der Nutzer das Passwort **selbst getippt** hat — nicht, wenn es durch Browser-Autofill oder eine Passwort-Manager-Extension eingetragen wurde.

## What Changes

- Neue Shared-Komponente `<PasswordInput>` in `web/src/components/forms/PasswordInput.tsx`, die:
  - das Lucide-`Eye`/`EyeOff`-Icon rechts im Input rendert
  - intern zwischen „selbst getippt" und „injected" unterscheidet (via `keydown`/`paste`/`cut`-Signal)
  - den Reveal-Button nur dann anzeigt, wenn das aktuelle Passwort vom Nutzer getippt wurde **und** das Feld nicht leer ist
  - bei `blur` automatisch wieder auf `type=password` zurückfällt
  - vollständige Tastatur- und Screenreader-Bedienbarkeit (`aria-pressed`, `aria-label`, `type="button"`) bietet
- Vier bestehende Formulare auf `<PasswordInput>` umstellen:
  - `LoginPage.tsx` (1 Feld)
  - `RegisterPage.tsx` (2 Felder: Passwort + Bestätigung)
  - `ResetPasswordPage.tsx` (2 Felder: Passwort + Bestätigung)
  - `components/profile/PasswordChangeModal.tsx` (3 Felder: aktuelles + neues + Bestätigung)
- Neue Capability-Spec `password-reveal-toggle` mit allen Verhaltens-Requirements
- Erweiterung der Capability-Spec `component-standards` um die Pflicht, für jedes Passwort-Feld die neue Shared-Komponente zu verwenden

## Capabilities

### New Capabilities

- `password-reveal-toggle`: Sichtbarkeits-Toggle für Passwort-Felder, der nur bei selbst-getipptem Passwort aktiv ist.

### Modified Capabilities

- `component-standards`: Ergänzt um die Pflichtregel, dass jedes `<input type="password">` in `web/src/` durch die Shared-Komponente `<PasswordInput>` ersetzt wird (analog zu den bestehenden Klassen-String-Pflichten).

## Impact

- **Backend:** keine Änderung — kein neuer Endpoint, keine DB-Migration.
- **Frontend:** Eine neue Komponente plus Umstellung von 4 Dateien (insgesamt 8 Passwort-Felder). Keine neuen Dependencies — `lucide-react` ist bereits installiert.
- **A11y:** Toggle-Button mit klarem ARIA-Label und `aria-pressed`. Tab-Navigation bleibt funktional (Button kommt in den Tab-Index nach dem Feld).
- **Sicherheit:** Reveal-Funktion ist absichtlich nicht für autofilled Werte verfügbar — schützt vor Klartext-Leak fremder Browser-Sessions.
- **Tests:** Keine Frontend-Test-Infrastruktur im Projekt (kein vitest/jest). Verifikation manuell über die in `tasks.md` aufgelisteten Browser-Szenarien. Da kein neuer HTTP-Endpoint entsteht, greift die Backend-Test-Pflicht aus CLAUDE.md nicht.
- **Mobile:** Button hat 44 px Mindest-Touch-Target (`py-2.5`-Pendant), erfüllt die Mobile-Konvention aus CLAUDE.md.
- **Breaking Change Risiko:** keines — die neue Komponente ist im Default-Verhalten ein normaler Passwort-Input.
