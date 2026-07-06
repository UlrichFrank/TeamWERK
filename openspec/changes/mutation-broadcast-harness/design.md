# Design — mutation-broadcast-harness

## Ziel

Ein billiger, wartungsarmer statischer Test, der die SSE-Broadcast-Invariante erzwingt, ohne den Produktpfad zu berühren und ohne Laufzeit-Overhead. Vorbild: `internal/arch/arch_test.go` (stdlib `go/parser`, `go/ast`).

## Detektionsstrategie

**Schritt 1 — Routen extrahieren.** `internal/app/router.go` (`BuildRouter`) mit `go/parser` einlesen. Alle `CallExpr` auf einem Chi-Router mit Selektor `Post`/`Put`/`Patch`/`Delete` sammeln. Aus dem zweiten Argument (dem Handler-Ausdruck, z. B. `membH.Update`, `h.Games.UpdateGame`) den **Handler-Empfänger** und den **Methodennamen** ableiten. Die Zuordnung Router-Variable → Domänen-Package erfolgt über die lokalen Variablen-Deklarationen in `BuildRouter` (z. B. `membH := handlers.Members`).

**Schritt 2 — Handler-Rumpf prüfen.** Für jede so gefundene `(Package, Methode)` die Methode im jeweiligen `internal/<pkg>/*.go` per AST finden und den Funktionsrumpf auf einen **Broadcast-Aufruf** absuchen: jeder `CallExpr`, dessen aufgerufener Bezeichner (Selektor- oder Ident-Name) die Teilzeichenkette `Broadcast` enthält. Das erkennt bewusst auch **Helfer** (`broadcastMembers`, `broadcastGame`, `broadcastKaderUpdate`, `broadcastDutySlot`), nicht nur direkte `hub.Broadcast`-Aufrufe.

**Schritt 3 — Urteil.** Route mutierend **und** kein Broadcast im Rumpf **und** nicht auf Allowlist → Testfehler mit Datei/Route/Methode.

## Bewusste Vereinfachungen (und warum vertretbar)

- **Ein-Ebenen-Heuristik, keine Datenflussanalyse.** Der Test prüft nur, *ob* ein Broadcast-artiger Aufruf im Rumpf steht — nicht, ob das *richtige* Event an das *richtige* Publikum geht. Das fängt die reale, häufige Fehlerklasse (Broadcast komplett vergessen) und ist robust gegen Refactoring. Audience-Korrektheit bleibt Sache der Route-spezifischen Verhaltens-Tests (die dieser Change ergänzend fordert, aber nicht ersetzt).
- **Substring `Broadcast` statt fixer Symbolliste.** Neue Broadcast-Helfer funktionieren ohne Test-Anpassung, solange sie der Namenskonvention folgen (die im Projekt durchgängig gilt). Kosten: ein Helfer, der `Broadcast` heißt aber nicht broadcastet, würde durchrutschen — im Projekt nicht real.
- **Allowlist statt Weglassen.** Jede Ausnahme ist ein benannter, kommentierter Eintrag `{pkg, method, grund}`. Wächst die Liste unbemerkt, fällt das im Review auf. Eine veraltete Allowlist-Zeile (Route existiert nicht mehr) SHALL ebenfalls einen Testfehler erzeugen — sonst verrottet sie.

## Allowlist-Kriterium

Auf die Allowlist gehört eine mutierende Route nur, wenn sie **keinen von anderen eingeloggten Clients live beobachtbaren Zustand** ändert. Kandidaten:

| Route(n) | Grund |
|---|---|
| `POST /api/auth/*` (Login, Refresh, Reset) | ändert Session/Token, keine beobachtbare Domänenliste |
| Impersonation (`admin`) | erzeugt Token, kein Domänen-State |
| Push-Subscription-Registrierung | pro-Gerät, kein geteilter State |
| reine Datei-/Export-Downloads mit Nebeneffekt | liefern Blobs, kein Live-Listen-State |

Grenzfälle (z. B. `auth.CreateUser`, das die AdminUsers-Liste ändert) gehören **nicht** auf die Allowlist, sondern brauchen einen Broadcast.

## Verworfene Alternativen

- **Laufzeit-Middleware, die Broadcasts zählt:** Overhead im Produktpfad, und ein „0 Broadcasts"-Signal ist erst zur Laufzeit sichtbar — zu spät. Statisch = im Gate.
- **golangci-lint Custom-Linter:** höhere Einstiegshürde (Plugin-Build), gleicher Nutzen; der stdlib-AST-Test ist konsistent mit dem bestehenden Architektur-Test.
- **Verpflichtender Verhaltens-Test pro Route:** stärker (prüft Audience), aber nicht mechanisch erzwingbar ohne genau diesen Meta-Test — daher hier die Struktur-Prüfung als Fundament.
