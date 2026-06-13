## 1. Backend — Datenstrukturen erweitern

- [x] 1.1 `ImportRow.Status` um `"not_found"` ergänzen (Kommentar im Struct)
- [x] 1.2 `ImportReport` um Feld `NotFound int \`json:"not_found"\`` erweitern
- [x] 1.3 Hilfsfunktion `isDBFieldEmpty` für `sql.NullString` und `sql.NullInt64` implementieren

## 2. Backend — Enrich-Modus im Import-Handler

- [x] 2.1 Matching-Logik: wenn CSV keine `Geburtsdatum`-Spalte hat, Fallback auf Vorname+Nachname-Suche mit Ambiguity-Check (≥2 Treffer → `error`)
- [x] 2.2 `enrich`-Branch in der Import-Schleife: bei fehlendem Match Zeile als `not_found` markieren, `report.NotFound++`, weiter zur nächsten Zeile
- [x] 2.3 Schreib-Pfad für `enrich`: für jedes Feld `isDBFieldEmpty` prüfen bevor geschrieben wird
- [x] 2.4 IBAN im `enrich`-Modus: MOD-97-Validierung ausführen, aber nur schreiben wenn DB-Feld leer

## 3. Frontend — Modus-Auswahl

- [x] 3.1 Drittes Radio-Button „Nur leere Felder ergänzen" im Import-Modal (Step 1) ergänzen; `mode`-State auf `'enrich'` setzen
- [x] 3.2 TypeScript-Interface `ImportReport` um `not_found?: number` erweitern
- [x] 3.3 `ImportRow['status']` Union-Type um `'not_found'` erweitern

## 4. Frontend — Report-Darstellung

- [x] 4.1 `rowStatusIcon()` und `rowStatusColor()` um `not_found` erweitern (Icon `—`, Farbe grau/`brand-text-muted`)
- [x] 4.2 Summary-Badges um „X nicht gefunden" (grau) erweitern, nur anzeigen wenn `not_found > 0`
