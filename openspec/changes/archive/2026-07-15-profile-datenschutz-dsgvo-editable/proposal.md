## Why

Die aktuelle Spec `profile-datenschutz-tab` beschreibt die DSGVO-Schalter im
`/profil`-Tab „Datenschutz" als **gesperrt (read-only)**. Nutzer konnten damit
Änderungen an ihren DSGVO-Einwilligungen nur außerhalb der App anfragen (Mail,
Vorstand ansprechen), was in der Praxis mühsam ist und den ohnehin bestehenden
Change-Request-Draft-Workflow ungenutzt lässt.

Mit Commit `7e1a91e "fix(members): DSGVO-Schalter im Profil editierbar via
Change-Request"` wurde die Implementierung auf editierbare Schalter mit
Draft-Anfrage umgestellt. Die Spec ist seitdem veraltet und weicht vom
implementierten Verhalten ab — dieses Proposal führt sie nach.

Fachlich bleibt der Kern erhalten: **der Nutzer kann nicht direkt schreiben**,
sondern beantragt die Änderung über den bestehenden Draft-Approval-Flow. Was
sich ändert, ist die Bedien-UX (Checkboxen aktiv statt nur Datei-Klick auf einen
externen Prozess).

## What Changes

- **Spec `profile-datenschutz-tab`** — Requirement „DSGVO-Anzeige (read-only)"
  → umbenannt zu „DSGVO-Einwilligungen mit Change-Request" und inhaltlich
  angepasst:
  - Schalter sind aktiv (bedienbar), lokale Änderung ist Draft-Kandidat.
  - „Änderung anfragen"-Button sendet `POST /members/{id}/change-request` mit
    `field_name='dsgvo'`, `new_value={verarbeitung, weitergabe, foto_veroeffentlichung}`.
  - Solange ein Draft aussteht, zeigt der Tab pro geänderter Einwilligung
    „(angefragt: Ja|Nein)" hinter dem Label.
  - „Anfrage zurückziehen" löscht den Draft (`DELETE /members/{id}/change-drafts/{id}`)
    und rollt die lokalen Schalter auf den Server-Stand zurück.
  - Ohne lokale Änderung ist der Anfrage-Button gesperrt (kein Draft ohne Diff).

- **Keine Code-Änderung in diesem Change** — der Code (Frontend + Test) wurde
  bereits mit `7e1a91e` und dem Folge-Test-Fix eingecheckt. Dieses Proposal
  ist reines Spec-Catch-up.

## Capabilities

### Modified Capabilities

- `profile-datenschutz-tab`: Requirement zu DSGVO-Schaltern von „read-only" auf
  „editierbar via Change-Request-Draft" umgestellt.

## Test-Anforderungen

Der bestehende vitest-Suite `web/src/components/profile/__tests__/ProfileDatenschutzTab.test.tsx`
deckt die neuen Szenarien bereits ab (grüner Testlauf 553/553):

| Testname | Erwartung / Invariante |
|---|---|
| `DSGVO-Status wird editierbar via Change-Request angezeigt` | Alle drei Checkboxen sind aktiv (`disabled=false`); Anfrage-Button ohne Diff gesperrt. |
| `Erklärtext zu jedem der drei DSGVO-Schalter` | Erklärtext pro Schalter präsent. |

**Garantierte Invariante:** Ein Klick auf eine DSGVO-Checkbox im `/profil`-Tab
löst niemals ein direktes `PUT /members/{id}` aus — Änderungen fließen
ausschließlich über den Draft-Approval (`POST /change-request` +
`PUT /change-drafts/{id}/approve`).

## Impact

- **Spec:** `openspec/specs/profile-datenschutz-tab/spec.md` — Requirement
  „DSGVO-Anzeige (read-only) im Datenschutz-Tab" wird beim Archivieren durch
  die neue Formulierung ersetzt.
- **Kein Backend-Impact:** die Endpoints (`change-request`, `change-drafts`)
  existieren seit langem, keine neue Route.
- **Kein Migrations-Impact.**
- **Frontend bereits umgesetzt** — dieser Change dokumentiert den Ist-Stand.
