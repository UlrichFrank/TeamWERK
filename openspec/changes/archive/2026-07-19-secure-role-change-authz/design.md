## Context

`UpdateUserRole` sitzt heute im Vorstand-Tier (`RequireClubFunction("vorstand")`). Vorstände sollen legitim Nicht-Admin-Rollen pflegen können (z.B. Funktionszuordnungen über die Nutzerverwaltung), daher ist ein pauschales Verschieben nach admin-only nicht zwingend gewünscht. Die eigentliche Lücke ist eng: Degradierung eines Admins durch einen Nicht-Admin und Selbst-Rollenänderung.

## Goals / Non-Goals

**Goals:**
- Kein Nicht-Admin kann einen Admin herabstufen oder die eigene Rolle ändern.
- Bestehende, legitime Rollenpflege durch Vorstand bleibt möglich.

**Non-Goals:**
- Kein Umbau des Rollenmodells.
- Last-Admin-Schutz (verhindern, dass der letzte Admin entfernt wird) ist optional und kann separat ergänzt werden.

## Decisions

**D1 — Handler-Guard statt Tier-Verschiebung (empfohlen).** Im Handler prüfen: wenn `caller.Role != "admin"` und (Ziel-aktuelle-Rolle == `admin` ODER Ziel == Aufrufer-Selbst) → 403. Vorteil: Vorstand behält legitime Nicht-Admin-Rollenpflege. Alternative „Route nach `RequireRole("admin")`" verworfen, weil sie den legitimen Vorstands-Use-Case mitnimmt — es sei denn, das Produktteam möchte Rollenverwaltung generell admin-only (dann diese Alternative wählen).

**D2 — Invariante in der Spec, Mechanismus im Code.** Die `auth`-Anforderung beschreibt die Garantie (Nicht-Admin kann Admin nicht degradieren / sich selbst nicht ändern), unabhängig vom gewählten Mechanismus.

## Risks / Trade-offs

- **[Vorstand erwartet, Admins verwalten zu können]** → bewusst eingeschränkt; Admin-Verwaltung bleibt admin-only.
- **[Last-Admin-Lücke bleibt offen]** → außerhalb Scope; als Folge-Idee notiert.

## Open Questions

- Soll Rollenverwaltung generell admin-only werden (dann Tier-Verschiebung statt Handler-Guard)? Produktentscheidung.
- Last-Admin-Schutz mit aufnehmen?
