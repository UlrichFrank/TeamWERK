## Why

Beim Kuchendienst braucht ein Spieltag mit mehreren Heimspielen entsprechend viele Kuchen. Heute erzeugt der Auto-Regen (`internal/games/regen.go`, `regenGameItems`) pro Heimspiel einen Kuchen-Slot **pro Team dieses einzelnen Spiels** (`game_teams`) — jedes Team bäckt ausschließlich für sein eigenes Spiel. Der Verein möchte stattdessen eine faire Rotation: die Kuchen eines gesamten Spieltags werden reihum unter den an dem Tag spielenden Mannschaften verteilt, gedeckelt auf maximal N Kuchen pro Mannschaft — unabhängig davon, welches Team welches konkrete Spiel bestreitet. Das heutige Modell (Team = Spielteilnehmer) kann diese tagesweite, mannschaftsübergreifende Verteilung strukturell nicht abbilden.

## What Changes

- Neue vereinsweite Einstellung „Spiele-zu-Kuchen-Verhältnis" (Dezimalzahl, z. B. `1.0`, `0.5`, `1.5`) unter `/einstellungen` im neuen Tab **Bewirtung**. Der Kuchenbedarf eines Spieltags errechnet sich als `aufgerundet(Anzahl Heimspiele des Tages × Verhältnis)`.
- Neues, optionales Feld **„Max. Kuchen pro Mannschaft"** auf Vorlagen-Items (`game_template_items`). Gesetzt aktiviert es den Rotations-Modus für dieses Item; `NULL` (Default) belässt das bisherige Verhalten (ein Slot pro Team des jeweiligen Spiels) unverändert.
- Neue Zuweisungslogik im Auto-Regen: für rotations-aktivierte Items entsteht **ein** Kuchen-Slot pro Heimspiel des Tages (statt einer pro Team des Spiels). Die verantwortliche Mannschaft wird aus einer pro Spieltag frisch aufgebauten Warteschlange gezogen — Reihenfolge nach chronologischem Anpfiff, jede Mannschaft einmalig an der Position ihres ersten Heimspiels des Tages. Zuteilung erfolgt greedy: Mannschaft 1 erhält bis zu N Kuchen, danach Mannschaft 2, usw.
- Übersteigt der Kuchenbedarf das, was `Anzahl teilnehmender Mannschaften × Cap` deckt, bleiben die überzähligen Slots **unzugeordnet** (`team_id = NULL`, offen für alle berechtigten Nutzer) statt den Cap zu verletzen oder eine Mannschaft zu überlasten. Der Regen-Summary weist das aus.
- **Verhaltensänderung (nur für rotations-aktivierte Items):** die Zuweisungs-Wiederherstellung nach einem erneuten Regen (`restoreAssignments`) matched bestehende Zusagen nur noch über `(duty_type_id, event_time)`, ohne `team_id` — weil `team_id` bei Rotations-Items aus der tagesweiten Reihenfolge abgeleitet und damit durch Spielplanänderungen verschiebbar ist. Nicht-rotations-aktivierte Items behalten den bisherigen Drei-Feld-Match unverändert.

## Capabilities

### New Capabilities
- `bewirtungsrotation`: Konfigurierbares Spiele-zu-Kuchen-Verhältnis, Mannschafts-Warteschlange und Cap-gesteuerte Zuweisung für Bewirtungs-/Kuchendienste an Spieltagen, inklusive Einstellungen-UI (Tab „Bewirtung").

### Modified Capabilities
- `duties`: neue Anforderung, dass Rotations-Slots ohne Team-Zuordnung (Cap-Überlauf) über den bestehenden `team_id IS NULL`-Fallback weiterhin für die Spielteams sichtbar/übernehmbar bleiben (keine Änderung an bestehenden Requirements, ergänzende Anforderung für die neue Slot-Variante).

## Impact

- **DB:** neue Migration `045_*` — Spalte `game_template_items.rotation_max_per_team` (nullable INTEGER), `system_settings`-Row `bewirtung_verhaeltnis` (Default `'1'`).
- **Backend:** `internal/games/regen.go` (`regenGameItems`, `regenSingleDay`, `restoreAssignments`/`customKey`), `internal/settings/` (Store-Erweiterung analog `maintenance_mode`), `internal/games/handler.go` (Template-Item-CRUD um neues Feld), neue/erweiterte Routen unter Vorstand-Berechtigung.
- **Frontend:** `web/src/pages/AdminSettingsPage.tsx` (neuer Tab „Bewirtung"), Vorlagen-Editor (Cap-Eingabefeld pro Item), Dienstbörse (Anzeige unzugeordneter Slots unverändert nutzbar, da `team_id=NULL` bereits ein bekannter Zustand ist).
- **Tests:** Unit-Tests für die Warteschlangen-/Zuweisungslogik in `internal/games/regen_test.go`, Route-Tests (Happy-Path + Fehlerfall) für neue/geänderte Settings- und Template-Item-Routen.
