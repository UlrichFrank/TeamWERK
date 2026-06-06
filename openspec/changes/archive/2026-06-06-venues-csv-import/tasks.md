## 1. Backend: Import-Handler

- [x] 1.1 In `internal/venues/handler.go` Methode `Import(w http.ResponseWriter, r *http.Request)` hinzufügen
- [x] 1.2 Multipart-Form parsen (`r.ParseMultipartForm(10 << 20)`), Datei aus Feld `file` lesen
- [x] 1.3 CSV mit `encoding/csv` einlesen; erste Zelle auf BOM trimmen (`strings.TrimPrefix(cell, "\xEF\xBB\xBF")`)
- [x] 1.4 Header-Zeile finden (erste Zeile, deren erste Zelle nach Trim `"Name"` ist); alle Zeilen davor überspringen
- [x] 1.5 Spalten-Mapping implementieren: col 0=name, 2=street, 3=postal_code, 4=city, 5=note, 6=note-Anhang (falls nicht leer); col 1 ignorieren
- [x] 1.6 Upsert-Logik: `SELECT id FROM venues WHERE name = ?` → wenn gefunden UPDATE, sonst INSERT; `is_home_venue` beim UPDATE nie anfassen
- [x] 1.7 Zeilen ohne Namen in `errors`-Slice sammeln und überspringen (Import läuft weiter)
- [x] 1.8 Gesamten Import in einer Transaktion ausführen; bei Commit `hub.Broadcast("venues")`
- [x] 1.9 Response: `{ "imported": N, "updated": N, "skipped": N, "errors": [...] }` als JSON mit HTTP 200

## 2. Backend: Route registrieren

- [x] 2.1 In `cmd/teamwerk/main.go` Route `r.Post("/api/admin/venues/import", venueH.Import)` unter Admin-only-Gruppe eintragen

## 3. Frontend: Split-Button

- [x] 3.1 In `AdminVenuesPage.tsx` Import ergänzen: `ChevronDown` aus lucide-react, `useRef` + `useEffect` für Click-outside
- [x] 3.2 State hinzufügen: `showActionsMenu`, `showImport`
- [x] 3.3 Bestehenden "+ Neuer Ort"-Button durch Split-Button ersetzen (linke Hälfte: `onClick={openCreate}`, rechte Hälfte: ChevronDown öffnet Dropdown)
- [x] 3.4 Dropdown-Eintrag "Import CSV" implementiert `() => { setShowActionsMenu(false); setShowImport(true) }`
- [x] 3.5 Click-outside-Handler mit `useRef` auf den Split-Button-Container

## 4. Frontend: Import-Modal

- [x] 4.1 State hinzufügen: `importFile`, `importing`, `importResult` (Typ: `{ imported, updated, skipped, errors }`)
- [x] 4.2 Modal-JSX anlegen: File-Input (`accept=".csv"`), "Importieren"-Button, Lade-Zustand
- [x] 4.3 `handleImport`-Funktion: FormData mit `file` aufbauen, `api.post('/admin/venues/import', fd)` aufrufen
- [x] 4.4 Ergebnis-Anzeige im Modal: "X neu importiert, Y aktualisiert, Z übersprungen" + Fehler-Liste falls vorhanden
- [x] 4.5 Nach erfolgreichem Import: `load()` aufrufen, Modal bleibt offen mit Ergebnis; separater "Schließen"-Button
