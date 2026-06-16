## Design

### Designentscheidung 1: Status quo wird festgeschrieben, nicht korrigiert

**Frage:** Sollen die heutigen Designlöcher (`vorstand_beisitzer` ohne Wirkung, `kassierer` ohne Wirkung, `/anfragen`-Route nur für admin/vorstand obwohl Backend trainer erlaubt) bereits hier dokumentiert werden?

**Entscheidung:** Ja, dokumentieren — aber als „heute geltend". Die Spec sagt z. B. „`vorstand_beisitzer` hat heute exakt die Rechte eines `standard`-Users". Ein Folge-Change `permissions-cleanup` darf dann diese Requirements gezielt MODIFIED markieren.

**Begründung:** Der Wert dieses Proposals liegt darin, **die Realität festzuhalten**, bevor wir sie ändern. Wenn wir hier schon korrigieren, vermischen sich zwei Diskussionen (Was ist? Was sollte sein?) und der Proposal wird unrund. Außerdem fängt der Test in seiner aktuellen Form Regressionen — auch in einer „suboptimalen" Konfiguration ist konsistente Konfiguration testbar.

**Konsequenz:** Die Spec wird stellenweise wie eine Anti-Spec klingen („Persona X hat heute keinen Sonderzugriff auf Endpoint Y"). Das ist Absicht.

### Designentscheidung 2: Persona-Definitionen sind Code, nicht Tabellen

**Frage:** Wie werden Personas in Tests fixiert?

**Entscheidung:** Eine geteilte TypeScript/Go-Quelle pro Stack:

```typescript
// web/src/test/personas.ts
export type Persona = {
  id: string
  label: string
  role: 'admin' | 'standard'
  clubFunctions: string[]
  isParent: boolean
}

export const PERSONAS: Persona[] = [
  { id: 'admin', label: 'Admin', role: 'admin', clubFunctions: [], isParent: false },
  { id: 'vorstand', label: 'Vorstand', role: 'standard', clubFunctions: ['vorstand'], isParent: false },
  { id: 'vorstand_elternteil', label: 'Vorstand-Elternteil', role: 'standard', clubFunctions: ['vorstand'], isParent: true },
  { id: 'vorstand_beisitzer', label: 'Vorstand Beisitzer', role: 'standard', clubFunctions: ['vorstand_beisitzer'], isParent: false },
  { id: 'kassierer', label: 'Kassierer', role: 'standard', clubFunctions: ['kassierer'], isParent: false },
  { id: 'trainer', label: 'Trainer', role: 'standard', clubFunctions: ['trainer'], isParent: false },
  { id: 'trainer_elternteil', label: 'Trainer-Elternteil', role: 'standard', clubFunctions: ['trainer'], isParent: true },
  { id: 'sportliche_leitung', label: 'Sportliche Leitung', role: 'standard', clubFunctions: ['sportliche_leitung'], isParent: false },
  { id: 'sportliche_leitung_elternteil', label: 'Sportliche Leitung-Elternteil', role: 'standard', clubFunctions: ['sportliche_leitung'], isParent: true },
  { id: 'spieler', label: 'Spieler', role: 'standard', clubFunctions: ['spieler'], isParent: false },
  { id: 'elternteil', label: 'Elternteil', role: 'standard', clubFunctions: [], isParent: true },
]
```

```go
// internal/permissions/personas_test.go
type Persona struct {
    ID            string
    Role          string
    ClubFunctions []string
    IsParent      bool
}

var Personas = []Persona{ /* identische 11 Einträge */ }
```

**Begründung:** Personas sind im Code Top-of-Mind. Ein zentrales Tabellenformat im Repo (z. B. JSON) wäre eleganter, aber bricht die Type Safety in Tests und erzeugt einen Build-Schritt. Die zweifache Definition ist Drift-anfällig, aber die Spec gibt vor, was richtig ist — Diff zwischen den Personas-Listen ist trivial im Code-Review zu erkennen.

### Designentscheidung 3: Frontend-API-Mocking — axios-mock-adapter, nicht MSW

**Frage:** Wie werden API-Calls in vitest gemockt?

**Optionen:**
- **axios-mock-adapter** — kleine Library, hängt an die `api`-Axios-Instanz, einfacher Setup, kein Service Worker nötig.
- **MSW** (Mock Service Worker) — modernster Stand der Technik, mockt auf Netzwerk-Layer, funktioniert in jsdom und Browser, mächtiger.

**Entscheidung:** **axios-mock-adapter**.

**Begründung:** Die Permission-Tests müssen nur prüfen, welche Endpoints aufgerufen werden und welche Komponenten gerendert/versteckt sind. Echte Netzwerk-Semantik (CORS, Streaming, SSE) ist nicht im Scope. axios-mock-adapter ist signifikant einfacher zu setupen — ein API-Test braucht keinen Worker-Stub. MSW könnte später nachgezogen werden, wenn integrationsähnliche Tests dazukommen.

### Designentscheidung 4: Backend-Matrix-Test als Table-Test in eigenem Package

**Frage:** Wo lebt der Backend-Matrix-Test?

**Optionen:**
- (a) `internal/auth/permissions_matrix_test.go`
- (b) `internal/app/router_test.go`
- (c) **`internal/permissions/matrix_test.go`** — neues Package nur für die Matrix.

**Entscheidung:** (c).

**Begründung:** Das Test-Package importiert `internal/app` (Router), `internal/auth` (Tokens), `internal/testutil` (DB, Fixtures). In Option (a)/(b) gäbe es Import-Zyklen oder das Vermischen von Unit- und Integration-Tests. Ein dediziertes `internal/permissions`-Test-Package macht den Zweck explizit und liegt am richtigen Abstraktionslevel.

```go
// internal/permissions/matrix_test.go
type endpointCase struct {
    method   string
    path     string
    // Erwartete Status pro Persona-ID
    expected map[string]int
}

var matrix = []endpointCase{
    {method: "GET", path: "/api/members", expected: map[string]int{
        "admin": 200, "vorstand": 200, "vorstand_elternteil": 200,
        "vorstand_beisitzer": 403, "kassierer": 403,
        "trainer": 403, "trainer_elternteil": 403,
        "sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
        "spieler": 403, "elternteil": 403,
    }},
    // … pro Endpoint ein Eintrag
}
```

**Trade-off:** Eine Matrix-Eintrag-Liste ist sehr explizit (gut für Spec-Treue) aber lang (~200 Einträge). Wir akzeptieren das. Pflege per Codegen ist optional; lieber explicit-as-truth.

### Designentscheidung 5: Welche Inline-Gates testen wir genau?

**Frage:** Wo ist die Grenze zwischen „in-Scope" und „too detailed"?

**Entscheidung:** Wir testen die in §7 der Explore-Notiz aufgelisteten **expliziten `const isXxx = …`-Variablen** in Pages und ihre direkten UI-Konsequenzen (Render-Sichtbarkeit oder Disable). Konkret:

| Page | Variable | Sichtbares Element |
|---|---|---|
| MembersPage | `isAdmin` | „Mitglied anlegen"-Button |
| MemberDetailPage | `isAdmin` | „Bearbeiten"-Button |
| TermineDetailPage | `isTrainer` | Edit-Actions im Training-Header |
| TerminePage | `isTrainer` | „Training anlegen"-Button |
| SpieltagDetailPage | `canEdit` | „Spiel bearbeiten"-Button |
| DutyPage | `isAdminOrTrainer` | Slot-Mutation-Actions |
| ChatPage | `canBroadcast` | „Broadcast schreiben"-Button |
| ChatPage | `isAdmin` (UserPicker) | erweiterte User-Auswahl |
| KalenderPage | `canEdit` | „Spiel anlegen"-Button |
| KalenderPage | `canCreateAbsence` | „Abwesenheit anlegen"-Button |
| MemberDatenschutzTab | `isVorstand` | SEPA-Mandat-Aktionen |

Alle anderen Inline-Bedingungen (z. B. „nur Owner sieht Lösch-Icon") sind Ownership-basiert und gehören in dedizierte Page-Tests (out-of-scope hier).

### Designentscheidung 6: Personas decken keine Ownership ab

**Frage:** Wie testen wir Endpoints wie `DELETE /api/absences/{id}`, wo Ownership entscheidet?

**Entscheidung:** Die Matrix testet pro Persona nur, ob die **Route prinzipiell erreichbar** ist (kein 403 durch Middleware). Inhalts-Filterung (Persona darf Endpoint aufrufen, sieht aber nur eigene Daten) wird in der `permissions`-Spec als Requirement formuliert, aber nicht im Matrix-Test geprüft — sie ist Sache der jeweiligen Handler-Tests in den Domain-Packages (existieren teilweise schon).

**Begründung:** Die Matrix-Test prüft Middleware-Verhalten, nicht Geschäftslogik. Sonst werden die Test-Fixtures überkomplex (jede Persona braucht Beispieldaten in der DB).

### Designentscheidung 7: Tests sind „Read-only" gegen Backend

**Frage:** Wie verhindern wir, dass POST/PUT/DELETE-Tests im Matrix-Test Daten in der Test-DB ändern und Folge-Tests beeinflussen?

**Entscheidung:** Pro Test-Case eine eigene `testutil.NewDB(t)`-Instanz. Eintrag der Matrix ist die kleinste Test-Einheit. Setup kostet Zeit, ist aber für die Klarheit und Isolation wert.

**Alternative überlegt:** Eine geteilte DB mit Transaktion-Rollback pro Test. Verworfen, weil SQLite + WAL + verschachtelte Transaktionen subtle Probleme machen, und der Matrix-Test bewusst auch DB-Reads testen soll.

### Designentscheidung 8: Vitest-Konfig — separates `vitest.config.ts` statt Inline

**Frage:** Vitest-Konfig in `vite.config.ts` (inline `test:` Block) oder separate `vitest.config.ts`?

**Entscheidung:** **separates `vitest.config.ts`** mit `defineConfig({ test: { … } })`.

**Begründung:** Vite-Build und Vitest-Run teilen 90% Konfig (Plugins, Alias, JSX), aber Vitest braucht zusätzlich `environment: 'jsdom'`, `setupFiles`, `globals: true`. Eine separate Datei macht das offensichtlich und vermeidet Konditionalen in `vite.config.ts`. Vite-Standard-Pattern.

## Open Questions

- **Coverage-Ziel:** Soll der Matrix-Test bei einer neuen Route den Build blocken (z. B. via Test, der über die Router-Definition iteriert und vergleicht)? Stark abhängig davon, wie groß die Bremse für Folge-Changes sein darf. Vorschlag: erste Version ohne Auto-Blocker, dann nachziehen.
- **CI-Integration:** Aktuell gibt es noch keine CI in diesem Repo. Wir setzen den Frontend-Test-Schritt erstmal nur lokal über `make test` voraus; CI-Pipeline ist out-of-scope.
- **Persona für „Eingeloggt ohne Member":** Existiert in der Realität (frisch registrierter Nutzer, ehemaliger Spieler ohne Funktion). Sollte als 12. Persona aufgenommen werden? Vorschlag: Ja, als `standard_no_function`, eigene Zeile in der Matrix.
- **Wegfall von `spieler_trainer` und `spieler_elternteil`:** Der Realfall „Trainer der auch Spieler ist" wird jetzt nicht abgedeckt — relevant z. B. für die Priorität-Logik in `vereinsfunktion`-Spec (trainer > spieler bei Duty-Soll). Der Realfall „spielendes Elternteil eines Kindes im Verein" wird ebenfalls nicht abgedeckt. Diese Lücken sind bewusst akzeptiert; falls sie später schmerzen, kommen die zwei Personas zurück.
