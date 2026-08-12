## Why

„Max. Kuchen pro Mannschaft" ist heute ein Feld **pro Vorlagen-Item** (`game_template_items.rotation_max_per_team`) — und trägt dort zwei Bedeutungen gleichzeitig: gesetzt = Rotation für dieses Item aktiv, und der Wert = Cap pro Mannschaft. Fachlich ist der Cap aber keine Eigenschaft einer Dienstplan-Vorlage, sondern eine **vereinsweite Bewirtungsregel** — genau wie das bereits vorhandene Spiele-zu-Kuchen-Verhältnis unter `/einstellungen?tab=bewirtung`. Beide Zahlen gehören zusammen: der Bedarf eines Spieltags (Verhältnis) und die Obergrenze je Mannschaft (Cap) ergeben nur gemeinsam ein sinnvolles Bild.

In der Vorlage sucht man das Feld nicht — dort steht es zwischen Anker, Offset und Slot-Anzahl, also lauter Item-Mechanik. Zusätzlich lädt die heutige Trennung dazu ein, denselben Duty-Type in zwei Vorlagen mit **unterschiedlichem** Cap zu konfigurieren; die Engine löst das mit „erster gewinnt" (`buildRotationPlan`), was still das falsche Ergebnis liefern kann.

## What Changes

- **Der Cap zieht in die Einstellungen um.** Neuer `system_settings`-Key `bewirtung_max_per_team` (positive Ganzzahl), gepflegt im bestehenden Tab **Bewirtung** unter `/einstellungen` direkt neben „Kuchen je Spiel". Beide Felder tragen dort den Hinweis, dass sie die **automatische Dienst-Generierung bei Heimspielen** steuern.
- **Im Vorlagen-Item bleibt nur der Schalter.** `game_template_items.rotation_max_per_team` (nullable INTEGER) wird zu `rotation_enabled` (INTEGER NOT NULL DEFAULT 0). Die Checkbox „Bewirtungsrotation" ersetzt das Zahlenfeld in beiden Editoren (Modal auf `/dienstplan-vorlagen`, Detailseite `/dienstplan-vorlagen/:id`) und verweist für den Cap auf die Einstellungen.
- **`GET`/`PUT /api/settings/bewirtung` transportieren beide Werte** (`verhaeltnis`, `max_per_team`). `PUT` akzeptiert jedes Feld einzeln (Pointer-Semantik: fehlendes Feld = unverändert) und lehnt `max_per_team <= 0` mit `400 invalid_max_per_team` ab, ohne zu persistieren.
- **`buildRotationPlan` liest den Cap einmal pro Regen-Lauf aus `system_settings`** (in derselben `tx` wie das Verhältnis, gleiches Muster wie `GetBewirtungVerhaeltnis`) statt aus dem ersten Item der Gruppe. Damit entfällt der „erster gewinnt"-Sonderfall bei divergierenden Caps ersatzlos.
- **Datenerhalt:** Die Migration setzt `rotation_enabled=1` für jedes Item mit bisher gesetztem Cap und seedet `bewirtung_max_per_team` aus dem größten vorhandenen Item-Cap (Fallback `1`). Konfigurierte Rotationen laufen nach dem Deploy unverändert weiter.

Nicht Teil dieses Changes: die Zuteilungslogik selbst (Warteschlange, Bedarf, Greedy-Cap, Restore-Matching) bleibt unverändert — nur die Herkunft des Caps ändert sich.

## Capabilities

### Modified Capabilities
- `bewirtungsrotation`: Der Cap „Max. Kuchen pro Mannschaft" wandert vom Vorlagen-Item in `system_settings`; das Item behält nur noch einen booleschen Rotations-Schalter. Die Settings-Route und der Einstellungen-Tab decken beide Werte ab.

## Impact

- **DB:** neue Migration `046_*` — `game_template_items.rotation_enabled` (ersetzt `rotation_max_per_team`, Daten übernommen), `system_settings`-Row `bewirtung_max_per_team`.
- **Backend:** `internal/settings/bewirtung.go` (Get/Set für den Cap), `internal/settings/handler.go` (beide Felder in `GetBewirtung`/`SetBewirtung`), `internal/games/handler.go` (`templateItemRow`/`templateItem`-Feld, Validierung, INSERT, Scans), `internal/games/regen.go` (`buildRotationPlan` liest den Cap aus den Settings, `rotationGroup.cap` entfällt).
- **Frontend:** `web/src/components/DutyTemplateItemFields.tsx` (`RotationCapField` → `RotationEnabledField`), `AdminDutyTemplatesPage.tsx`, `AdminDutyTemplateDetailPage.tsx`, `AdminSettingsPage.tsx` (`BewirtungTab` mit zweitem Feld + Heimspiel-Hinweis).
- **Tests:** `internal/settings/bewirtung_test.go` (Get/Set Cap, Default, Ablehnung ≤ 0), `internal/settings/handler_test.go` bzw. Route-Tests (beide Felder, Teil-Update), `internal/games/rotation_template_test.go` (Schalter statt Cap), `internal/games/rotation_regen_test.go` (Cap aus Settings), `web/src/pages/__tests__/AdminSettingsPage.bewirtung.test.tsx`, `web/src/pages/__tests__/AdminDutyTemplateDetailPage.rotationCap.test.tsx`.
- **Docs:** Gotcha „Bewirtungs-/Kuchendienst-Rotation" in `docs/agent/06-gotchas.md` (Cap-Herkunft), Spec `openspec/specs/bewirtungsrotation/spec.md`.
