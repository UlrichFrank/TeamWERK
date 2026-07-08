## Why

Die Hard-Rule „**Jede Mutations-Route ruft `h.hub.Broadcast(...)`**" (CLAUDE.md, Gotcha SSE) ist bislang nur dokumentiert, nicht mechanisch erzwungen. Beim Umbau auf gescopte Live-Updates (`scoped-live-updates`) riss sie an mehreren Stellen unbemerkt:

- `games.UpdateGame` benachrichtigte bei Team-Umhängung nur die neuen Teams,
- `kader.UpdateKader` erreichte entfernte Member / das alte Team nicht,
- `duties.Claim`/`Unclaim` broadcasteten **gar nicht** (vorbestehend, jahrelang stumm).

Diese Fehler kosten keinen Datenverlust, erzeugen aber „schleichende Fehler": Seiten bleiben nach fremden Änderungen stumm, bis der Nutzer manuell neu lädt. Klassisches Review fängt sie unzuverlässig — ein fehlender Aufruf ist unsichtbar. Der Architektur-Test (`internal/arch/arch_test.go`) zeigt, dass die nötige statische Analyse im Projekt bereits etabliert und billig ist.

Dieser Change schließt die Lücke mit einem **mechanischen Gate**: ein Go-Test, der jede mutierende Route (`POST`/`PUT`/`PATCH`/`DELETE`) aus `internal/app/router.go` auf einen Broadcast-Aufruf im zugehörigen Handler prüft. Ausnahmen sind nur über eine **explizite, begründete Allowlist** zulässig — so wird jede Nicht-Broadcast-Route zu einer bewussten Entscheidung statt eines stillen Versehens.

## What Changes

- **Neuer Harness-Test** (`internal/arch/` oder `internal/harness/`, stdlib-only, Teil von `make test` und damit des `pre-push`-Gates): 
  1. parst `BuildRouter` in `internal/app/router.go` und extrahiert alle Registrierungen mutierender Methoden (`r.Post`/`r.Put`/`r.Patch`/`r.Delete`) samt referenziertem Handler (z. B. `membH.Update` → Methode `Update` im Package `members`);
  2. parst die Handler-Methode und prüft, ob ihr Rumpf **irgendeinen** Broadcast-Aufruf enthält (Aufrufe, deren Bezeichner `Broadcast` enthält — deckt `h.hub.Broadcast`, `h.hub.BroadcastToUsers` **und** Helfer wie `broadcastMembers`/`broadcastGame`/`broadcastDutySlot` ab);
  3. schlägt fehl, wenn eine mutierende Route weder einen Broadcast enthält noch auf der Allowlist steht.
- **Explizite Allowlist** legitimer Nicht-Broadcast-Mutationen (mit Begründung je Eintrag), z. B. Auth/Token-Ausgabe (`/api/auth/*`), Impersonation, reine Datei-/Export-Downloads, Push-Subscription-Registrierung — Routen, die keinen von anderen Clients beobachtbaren, live-relevanten Zustand ändern.
- **Bestandsaufnahme + Behebung** der beim Aktivieren des Gates aufgedeckten Verstöße (die bekannten aus `scoped-live-updates` sind bereits gefixt; das Gate deckt etwaige Rest-Lücken auf, insb. diverse `auth`-Mutationen und `members.DeleteMember`/`CreateMemberFromUser`).
- **Doku-Verweis:** CLAUDE.md (`08-verification.md`) nennt das neue Gate neben Architektur-Test und `/verify-change`.

Kein Laufzeit-Code am Produktpfad ändert sich; rein additiv (Test + ggf. nachgezogene Broadcasts).
