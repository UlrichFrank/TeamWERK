## Context

Der bestehende CSV-Export (`GET /api/members/export`) liefert nur 6 von 12 relevanten Mitgliedsfeldern und enthält keine Verknüpfungs-Emails. Ein Import-Endpunkt existiert nicht. Beide Funktionen sollen auf der Mitglieder-Seite (/mitglieder) für Admins zugänglich sein. Das System läuft auf SQLite (kein ORM, kein RETURNING), Go 1.23, React 18. Der VPS hat 1 GB RAM — speicherintensive Verarbeitung (z.B. gesamte CSV im Memory halten) ist bei max. ~500 Mitgliedern kein Problem.

## Goals / Non-Goals

**Goals:**
- Export: Alle 12 Felder + Erziehungsberechtigte-Emails, alle Mitglieder inkl. `ausgetreten`
- Import: Idempotent, non-destructive, mit detailliertem Zeilenbericht
- Import-Modi: "nur ergänzen" und "fehlende + geänderte Felder aktualisieren"
- Keine neuen Go-Dependencies

**Non-Goals:**
- Saison- oder Mannschaftsdaten im CSV
- Automatisches Anlegen von Usern beim Import
- Entfernen von Feldinhalten oder Links via Import
- Bulk-Löschen via Import

## Decisions

### D1: Idempotenz-Schlüssel — Vorname + Nachname (+ Geburtsdatum als Tiebreaker)

**Gewählt:** `lower(first_name) + lower(last_name)` als Primärschlüssel; wenn Geburtsdatum in der CSV-Zeile vorhanden, wird es als Tiebreaker für Disambiguierung bei Namensgleichheit verwendet.

**Alternativ überlegt:** Passnummer als eindeutiger Schlüssel — abgelehnt, weil nullable und beim Erstimport häufig leer.

**Konsequenz:** Zwei Mitglieder mit identischem Vor- und Nachnamen UND identischem oder fehlendem Geburtsdatum können nicht gleichzeitig importiert werden. Bei 200 Mitgliedern eines Handballvereins akzeptabel.

### D2: Import verarbeitet die gesamte CSV serverseitig

Der Client sendet die Datei als `multipart/form-data`. Der Server parst, validiert und führt alle DB-Operationen durch, gibt einen strukturierten Importbericht als JSON zurück.

**Alternativ überlegt:** Clientseitiger Pre-Check im Browser — abgelehnt, da DB-Lookup (Existenz-Check, Email-Lookup) zwingend serverseitig.

### D3: Non-Destructive-Policy im Überschreiben-Modus

Leere CSV-Zellen überschreiben keine bestehenden DB-Werte. Nur Felder mit einem nicht-leeren CSV-Wert werden aktualisiert. `user_id` und `family_links` werden nur hinzugefügt, nie entfernt.

**Begründung:** Verhindert versehentliche Datenverluste bei unvollständigen CSV-Exporten (z.B. wenn Erziehungsberechtigten-Spalten absichtlich leer gelassen werden).

### D4: Importbericht als JSON-Response, Frontend rendert Modal

Der Endpunkt gibt immer HTTP 200 zurück (auch bei Zeilenfehlern) mit einem strukturierten JSON-Bericht. Nur echte Server-/Parse-Fehler geben 4xx/5xx zurück.

```json
{
  "total": 42,
  "created": 3,
  "updated": 7,
  "unchanged": 31,
  "errors": 1,
  "rows": [
    { "line": 2, "status": "created", "name": "Müller, Hans", "dob": "2010-03-02" },
    { "line": 8, "status": "updated", "name": "Schmidt, Anna",
      "changes": ["Passnummer: '' → 'DE-12345'"] },
    { "line": 15, "status": "error", "name": "Maier, Franz",
      "message": "Mehrfach in CSV (Zeile 8 und 15)" }
  ]
}
```

### D5: CSV-Encoding UTF-8, Semikolon als Trennzeichen

Semikolon als Standard-Trennzeichen (kompatibel mit deutschem Excel). Der Import akzeptiert Komma oder Semikolon (Auto-Detection anhand der Header-Zeile).

## Risks / Trade-offs

- **Namensgleichheit** → Mitigation: Geburtsdatum als Tiebreaker; Duplikat in CSV → Fehler im Bericht
- **Falsche Email in Erziehungsberechtigten-Spalte** → Mitigation: Still ignorieren wenn Email nicht in `users` gefunden (Bericht enthält Hinweis "nicht gefunden")
- **Großes CSV (> 500 Zeilen)** → Kein Problem bei 1 GB RAM; keine Pagination nötig
- **Excel-BOM (UTF-8 mit BOM)** → Mitigation: BOM beim Einlesen strippen (`strings.TrimPrefix(header, "\xef\xbb\xbf")`)
