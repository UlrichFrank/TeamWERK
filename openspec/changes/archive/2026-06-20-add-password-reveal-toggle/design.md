## Context

Vier Formulare im Frontend benötigen Passwort-Eingaben:

| Datei | Felder |
|---|---|
| `web/src/pages/LoginPage.tsx` | 1 (Passwort) |
| `web/src/pages/RegisterPage.tsx` | 2 (Passwort + Bestätigung) |
| `web/src/pages/ResetPasswordPage.tsx` | 2 (neues Passwort + Bestätigung) |
| `web/src/components/profile/PasswordChangeModal.tsx` | 3 (aktuell + neu + Bestätigung) |

Aktuell verwenden alle den Standard-Input-Klassen-String aus `component-standards` mit `type="password"`. Es gibt keinen Reveal-Toggle.

Browser-Verhalten beim Befüllen von Passwort-Feldern unterscheidet sich nach Quelle. Die zentrale technische Frage ist: **wie unterscheidet man zuverlässig „User-getippt" von „Browser/Extension-injected"?**

| Quelle | `keydown` | `paste`/`cut` | `input` | `:-webkit-autofill` |
|---|---|---|---|---|
| Tastatur | ✓ | — | ✓ | — |
| Backspace/Delete | ✓ | — | ✓ | — |
| Ctrl+V Paste | ✓ | ✓ | ✓ | — |
| Rechtsklick → Einfügen | — | ✓ | ✓ | — |
| Chrome Autofill | — | — | ✓ | ✓ |
| 1Password / Bitwarden | — | — | ✓ (meist) | — |
| iOS Strong Password | — | — | ✓ | ✓ (teilweise) |

`keydown` ∪ `paste` ∪ `cut` deckt alle echten User-Aktionen ab und enthält keine Autofill-Quelle. Das ist das belastbare Signal.

## Goals / Non-Goals

**Goals:**
- Reveal-Toggle nur bei nachweislich vom Nutzer getipptem (oder gepastetem) Passwort sichtbar.
- Sobald irgendwann ein Injection-Ereignis (Autofill) den Wert verändert hat, bleibt der Toggle versteckt — auch wenn der Nutzer danach noch Zeichen ergänzt (strenge Variante).
- Reset-Pfad: Wenn der Nutzer das Feld vollständig leert und neu tippt, wird der Toggle wieder verfügbar.
- Identisches Verhalten in allen 4 Passwort-Formularen via Shared-Komponente.
- Auto-Hide beim Verlust des Fokus (`onBlur`): zurück zu `type=password`.
- Volle Tastatur-/Screenreader-Bedienbarkeit.

**Non-Goals:**
- Kein Timeout-basiertes Auto-Hide (z. B. „nach 10 s wieder maskieren") — explizit ausgeschlossen.
- Keine Backend-Änderung.
- Keine Erkennung von Drag-&-Drop-Eingaben oder anderen exotischen Quellen (sehr selten in Login-Kontexten).
- Kein Reveal-Toggle in Feldern, die ein Passwort nur „bestätigen" sollen (z. B. Pre-Operation-Prüfung) — gilt für die hier aufgezählten Felder uniformly.

## Decisions

### Entscheidung 1: Detektion via `keydown` + `paste` + `cut` Zeitfenster

Die Komponente speichert in einem `useRef` den `performance.now()`-Zeitstempel der letzten User-Aktion (`onKeyDown`, `onPaste`, `onCut`). Beim `onChange` wird verglichen:

```
now() - lastUserActionAt < 100ms  →  User-driven
sonst                             →  Injected
```

100 ms ist konservativ — `keydown` und das daraus resultierende `input` liegen in der Praxis < 5 ms auseinander. Das Fenster fängt gleichzeitig Composition-Input (IME) ab.

**Alternative A:** `:-webkit-autofill`-CSS-Detektion zusätzlich. **Verworfen**, weil sie nur Chrome/Safari-Browser-Autofill erkennt, aber nicht 1Password/Bitwarden. Das Keydown-Signal deckt alle Fälle ab.

**Alternative B:** `InputEvent.isTrusted`. **Verworfen**: auch Autofill-Events haben `isTrusted=true`. Kein Unterscheidungsmerkmal.

### Entscheidung 2: Streng — Tainted-Flag ist one-way

Sobald einmal ein injizierter Wert das Feld verändert hat, bleibt `tainted=true`, auch wenn der Nutzer danach Zeichen ergänzt oder korrigiert. Der Toggle erscheint erst wieder, wenn der Nutzer das Feld komplett leert (Wert wird `""`) und neu tippt. Dann werden `userTyped` und `tainted` gemeinsam zurückgesetzt.

**Begründung:** Wenn ein Wert teilweise autofilled wurde, weiß der Nutzer nicht zwingend den vollständigen Inhalt. Reveal würde dann fremde Daten preisgeben.

**Trade-off:** Wer aus Versehen einmal Autofill triggert und das Feld dann ergänzt, sieht den Toggle nicht. Workaround: Feld leeren und neu tippen. Akzeptabel für den Sicherheitsgewinn.

### Entscheidung 3: Auto-Hide bei Blur, kein Timeout

`onBlur` setzt den Reveal-Zustand auf `false` (Feld wieder `type=password`). Der `userTyped`/`tainted`-State bleibt erhalten — der Toggle ist beim erneuten Fokus weiterhin sichtbar (falls vorher schon erlaubt), nur das Passwort selbst wird wieder maskiert.

Kein Timeout, weil:
- Der Nutzer beim aktiven Tippen das Passwort sehen will (sonst macht der Toggle keinen Sinn).
- Ein Wegklicken vom Feld ist das einzige sinnvolle Signal für „Aufmerksamkeit weg" — Inaktivitätstimer würden mitten beim Lesen / Vergleichen stören.

### Entscheidung 4: Lucide `Eye` / `EyeOff` Icons

`lucide-react` ist bereits installiert (v1.16.0). Beide Icons sind verfügbar. Größe `w-5 h-5`, Farbe via `currentColor` (geerbt vom `text-brand-text-muted` des Toggle-Buttons). Konform mit dem CLAUDE.md-Verbot von Unicode/Emojis in JSX.

### Entscheidung 5: Komponente kapselt nur das Reveal-Verhalten, nicht das Label

`<PasswordInput>` rendert ausschließlich das `<input>` + den Toggle-Button. Label, Wrapper-`<div>`, Fehlertext bleiben in der jeweiligen Page. Begründung: Die Formulare haben unterschiedliche Layout-Bedürfnisse (Modal vs. Vollseite), unterschiedliche Label-Texte und teilweise begleitende Hilfstexte. Eine zu mächtige Komponente würde die DRY-Ersparnis durch Anpassungs-Props wieder auffressen.

**Schnittstelle (informativ):**
```
<PasswordInput
  value={password}
  onChange={setPassword}
  autoComplete="current-password" | "new-password" | "off"
  required
  id="..."  // für label-htmlFor
/>
```

### Entscheidung 6: Layout via Wrapper-`<div class="relative">` + absolut positionierter Button

Der Input behält den Standard-Klassen-String aus `component-standards`, bekommt aber zusätzlich `pr-10` (Platz für den Button). Der Toggle-Button ist `absolute right-0 top-0 h-full px-3` — das gibt ihm automatisch das 44px-Touch-Target, sobald der Input ≥ 44px hoch ist (gegeben durch `py-2` + Text + Border).

```
┌──────────────────────────────────────────┐
│ relative wrapper                          │
│  ┌─────────────────────────────┬──────┐  │
│  │ input (pr-10)               │ btn  │  │
│  └─────────────────────────────┴──────┘  │
│                              absolute    │
└──────────────────────────────────────────┘
```

### Entscheidung 7: `autoComplete`-Wert wird vom Aufrufer vorgegeben

Login: `autoComplete="current-password"`, Register/Reset/Change-neues-Feld: `autoComplete="new-password"`. Die Komponente hat keinen Default — Aufrufer muss explizit setzen (TypeScript-Pflicht-Prop).

**Begründung:** Verhindert Browser-Autofill am falschen Ort (z. B. das aktuelle Passwort in das „neues Passwort"-Feld).

## Risks / Trade-offs

- **[Risiko]** Browser/Extension umgehen das Keydown-Signal künftig durch synthetisierte Events. **Mitigation:** Akzeptiert — die Komponente schützt vor üblichen Autofill-Quellen, nicht vor maliziösen Extensions, die volles DOM-Access haben. Diese hätten andere Angriffsvektoren.
- **[Risiko]** iOS-PWA in Standalone-Mode könnte `:-webkit-autofill` anders feuern. **Mitigation:** Da wir die CSS-Pseudoklasse gar nicht nutzen, irrelevant. Detektion läuft rein über JS-Events, die plattformeinheitlich verhalten.
- **[Risiko]** Composition-Input (IME, z. B. Chinesisch) löst möglicherweise mehrere `input`-Events ohne zwischenzeitliche `keydown`. **Mitigation:** Das `compositionstart`/`compositionupdate`-Event kann zusätzlich als User-Action gewertet werden. Nicht im MVP — bei Bedarf nachreichbar.
- **[Trade-off]** Strenger Modus = etwas weniger User-Komfort bei Tippfehler-Korrektur nach Autofill. Bewusst gewählt für Sicherheitsgewinn.
- **[Trade-off]** Kein Reveal-Toggle bei Autofill bedeutet: Nutzer können auch bei korrekt autofilled Passwort nicht visuell verifizieren, was eingetragen ist. Bewusst gewählt — bei korrektem Autofill ist die Verifizierung nicht nötig (Submit zeigt, ob es funktioniert).

## Migration Plan

1. Neue Komponente `web/src/components/forms/PasswordInput.tsx` anlegen.
2. Eine Page nach der anderen umstellen, jeweils ein eigener Commit.
3. Lokaler Browser-Smoke-Test pro Page in Chrome (Autofill simulieren via gespeichertes Passwort) und Firefox (privater Modus, kein Autofill verfügbar).
4. `make build` + `pnpm lint` → grün.
5. `make deploy`. Kein DB- oder Backend-Schritt nötig.

**Rollback:** Reverter-Commit oder Page-für-Page-Revert. Kein Datenrisiko, da rein UI.

## Open Questions

- **Composition-Events für IME**: Soll `compositionstart`/`compositionupdate` ebenfalls als User-Action zählen? → Default **nein**, on demand nachreichbar.
- **Zukünftige Felder (z. B. SMTP-Passwort im Vorstandsbereich)**: Sollten ebenfalls die Komponente nutzen? → Die `component-standards`-Regel macht das verbindlich für alle künftigen Passwort-Eingaben.
- **Manuelle Verifikation reicht?** Frontend hat keine Test-Infrastruktur. Falls langfristig Vitest eingeführt wird, sind die Verhaltens-Szenarien aus der Capability-Spec direkt als Test-Cases verwendbar.
