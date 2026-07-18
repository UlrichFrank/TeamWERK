## Why

Welle 1 der `test-coverage-roadmap` adressiert das höchste Bug-Kosten-Risiko: PII-tragende
Routen, deren Autorisierung heute nur teilweise mechanisch abgesichert ist. Ein Regressions-
Bug in dieser Schicht bedeutet ein DSGVO-relevantes Datenleck (fremde Dokumente, Abwesenheiten,
Spielbericht-Fotos), nicht bloß einen UX-Defekt — genau der Fall, den die `test-strategy`-
Capability vor Coverage-Prozentpunkten priorisiert.

Der Ist-Stand (gegen den realen Code geprüft, nicht gegen Roadmap-Annahmen):

- **files**: Nur `FolderContents` ist route-getestet (403 auf gesperrten Unterordner).
  `resolveAccess`/`checkAntiEscalation` sind als Unit abgedeckt, aber die schreibenden und
  ausliefernden Routen (`CreateFolder`, `DeleteFolder`, `UploadFile`, `AddPermission`,
  `download-token`, `DeletePermission`) haben **keinen** Route-Ebenen-Authz-Test.
- **matchreports `ServeImage`** (`images.go:215`, „nur Autor/Reviewer"): **kein Test** — das
  einzige Bild-Ausliefer-Leck ist ungeschützt gegen Regression.
- **duties `assertSlotTakePermitted`** (`match_report_guard.go`): der Spielbericht-Slot-Guard
  (nur `presseteam`/`admin`) hat keinen Test, inkl. der Proxy-Kind-Rollenverschiebung.
- **attendance-Recording** (`Training.SaveAttendances`, `Games.SaveAttendances`): das Package
  `internal/training` hat **gar keine** Testdatei — die Recording-Authz (Trainer des richtigen
  Teams) ist völlig ungetestet. (Die *Stats*-Routen sind bereits gut abgedeckt.)
- **absences**: `Create` und ein Scoping-Test existieren; `Calendar?show_team`, `Update`/
  `Delete`-Authz und `List`-Fremdzugriff fehlen.

## What Changes

Ausschließlich **Tests** (plus, falls ein Nil-/Fail-Closed-Loch auffällt, ein minimaler Guard —
verifiziert vor dem Festnageln, gemäß `test-strategy` „Bug-Verdacht vor Charakterisierung").
**Keine** Geschäftslogik-Änderung, keine neue Route, keine Migration. Alle Tests nutzen die
Welle-0-Fixtures (`internal/testutil`: `PostMultipart`, `CreateFolder`, `SetFolderPermission`,
`CreateFile`, `CreateAbsence`, `RecordTrainingAttendance`/`RecordGameAttendance`,
`CreateMemberWithFields`) und laufen über den echten `prodserver`-Router bzw. den jeweiligen
Handler-Testserver.

- **files** — Route-Ebenen-Tests: `CreateFolder`, `DeleteFolder`, `UploadFile` (multipart),
  `AddPermission` (HTTP-403-Eskalation, ergänzt die vorhandenen `checkAntiEscalation`-Units),
  `HandleDownloadToken` (fail-closed), `DeletePermission`.
- **matchreports** — `ServeImage`-Authz: Autor 200, Reviewer 200, Fremder 403, unbekannt 404,
  ohne Auth 401; plus Router-Tier-Absicherung.
- **duties** — `assertSlotTakePermitted`: Spielbericht-Slot durch Nicht-Presseteam → 403,
  durch `presseteam`/`admin` → ok, Nicht-Spielbericht-Slot unberührt, Proxy-Kind-Fall.
- **attendance-Recording** — `SaveAttendances` (training + games): Trainer des Teams → ok,
  Trainer eines fremden Teams → 403, Nicht-Staff → 403.
- **absences** — `Calendar?show_team`-Scoping (nur vorstand/trainer-like), `Update`/`Delete`
  durch Fremden → 403, `List`-Fremdzugriff.

## Capabilities

### New Capabilities

- `pii-route-authz`: dokumentiert die strukturelle Invariante „jede PII-tragende Route ist auf
  Autorisierungs-Ebene mechanisch getestet (Owner/Eltern/Staff erlaubt, Fremd → 401/403/404,
  fail-closed)" und benennt die abgedeckten Routen-Cluster als Anforderungen mit Szenarien.

### Modified Capabilities

_(keine — es ändern sich keine Requirements der Fach-Capabilities; die getesteten Authz-
Invarianten bestehen bereits im Code und werden hier nur mechanisch festgenagelt.)_

## Impact

- **Tests (neu):** `internal/files/*_test.go`, `internal/matchreports/*_test.go`,
  `internal/duties/*_test.go`, `internal/attendance` **oder** `internal/training`/`internal/games`
  (Recording liegt dort — Package-Grenze im Design geklärt), `internal/absences/*_test.go`.
- **Code:** grundsätzlich keiner. Falls ein Test einen echten Fail-Open-Bug aufdeckt, ein
  eigener minimaler `fix(...)`-Commit **vor** dem Test (Reihenfolge lt. `test-strategy`).
- **Kein** Backend-Vertrag, keine API-, Schema- oder Env-Änderung; SSE unberührt.
- **Harness:** der Welle-0-Authz-Drift-Detektor (`internal/arch/authz_test.go`) profitiert
  indirekt — neu getestete Routen sind in der Persona-Matrix bereits erfasst.
