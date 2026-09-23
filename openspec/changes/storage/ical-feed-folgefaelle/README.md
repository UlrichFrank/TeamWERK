# iCal-Feed — Folgefälle

Aus derselben Analyse wie `ical-feed-dienste-und-rfc`, dort bewusst **nicht** umgesetzt. Mit
dem Archiv jenes Changes gelten sie **nicht** als erledigt; jeder Punkt braucht einen eigenen
Change.

1. **Generische Termine ohne Kader-Zusatz und ohne Namens-Fallback.** `gameTitle` gibt im
   `default`-Zweig nur `opponent` zurück: das Kader-Label (inkl. `erw. Kader`) fehlt, und bei
   leerem `opponent` entsteht ein Event mit leerem `SUMMARY`. Ändert Titel-Zusagen der Spec
   `ical-feed` → eigener Change.
2. **`SEQUENCE` / `LAST-MODIFIED`.** Braucht `updated_at` auf `games`, `training_sessions` und
   `duty_slots` (Migration) und Nachziehen an jedem Schreibpfad, inklusive der Regen-Engine,
   die Slots löscht und neu anlegt. Bis dahin ist `DTSTAMP` = `created_at` kein
   Änderungszeitstempel (siehe `ical-feed-dienste-und-rfc/design.md`, Entscheidung 2).
3. **Anrede im Kind-Feed.** Die Aufstellungssätze in der `DESCRIPTION` sprechen mit „Du" an —
   im Kind-Token liest aber meist ein Elternteil. Offene Produktentscheidung.
4. **Kleinkram ohne Handlungsbedarf** (nur festgehalten): der `Team (…)`-Wrapper steht nur bei
   Spielen, generische Termine tragen kein Gattungswort — bewusst so bzw. Geschmacksfrage.
