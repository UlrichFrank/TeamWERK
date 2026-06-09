## Context

`POST /api/members/import` in `internal/members/handler.go` verarbeitet aktuell 8 von 18 vorhandenen DB-Feldern der `members`-Tabelle. Die Email-Spalten der CSV werden komplett ignoriert; Eltern-Verknüpfungen können nur über speziell benannte Spalten (`Erziehungsberechtigter1_Email`) angelegt werden, die in der tatsächlich exportierten Vereinsverwaltungs-CSV nicht vorkommen. Ein vollständiger Import der aktuellen Mitgliederliste (181 Einträge) erfordert deshalb manuelles Nacharbeiten.

Voruntersuchung der CSV hat ergeben: 81 IBANs vorhanden, davon 5 mit Checksummenfehlern (falsche Prüfziffer oder abgeschnittene Nummern).

## Goals / Non-Goals

**Goals:**
- Fehlende Felder (Adresse, Beitrittsdatum, IBAN, Kontoinhaber, SEPA) vollständig importieren
- IBAN-Korrektheit per MOD-97 sicherstellen, fehlerhafte IBANs als Warning melden
- Email-Spalten automatisch als eigene Email oder Eltern-Email klassifizieren und entsprechende Verknüpfungen anlegen
- Admin kann mit `mode=preview` alle geplanten Änderungen sehen, bevor sie angewendet werden

**Non-Goals:**
- Telefonnummern importieren (nicht zuverlässig zuordenbar, da Elternnummern)
- Neue User-Accounts beim Import anlegen (nur bestehende User verknüpfen)
- Profilbilder oder DSGVO-Felder importieren
- Bankdaten-Verschlüsselung (kein Scope dieser Änderung)

## Decisions

### 1. Email-Klassifizierungsheuristik

**Entscheidung:** Zweistufig — primär Alter, sekundär Vorname im lokalen Teil der Email-Adresse.

```
Alter < 18 UND Vorname NICHT im lokalen Teil  →  ELTERN-Email  →  family_link
Alter < 18 UND Vorname im lokalen Teil        →  KIND-EIGEN    →  kein Link (Notiz)
Alter ≥ 18                                    →  EIGEN         →  user_id-Link
```

Der Vorname wird normalisiert (Kleinbuchstaben, nur a-z) und mit dem normalisierten lokalen Teil der Email verglichen (Substring-Match). Für `Email 2` gilt dieselbe Logik unabhängig von `Email 1`.

**Alternativen verworfen:**
- *Kontoinhaber-Ähnlichkeit allein*: Feld ist bei vielen Erwachsenen leer, kein verlässliches Signal
- *Nur Alter*: Übersieht Kinder mit eigener Email-Adresse (z.B. `leonhard@...` bei 15-Jährigem)
- *Externe E-Mail-Validierung*: unnötige Abhängigkeit, kein Mehrwert

### 2. Preview-Modus

**Entscheidung:** Gleicher Code-Pfad wie `mode=update`, gesteuert durch einen `dryRun bool`-Parameter, der an alle DB-schreibenden Stellen weitergegeben wird. Bei `dryRun=true` werden `ExecContext`-Aufrufe für INSERT/UPDATE übersprungen; der Report wird identisch befüllt.

**Warum nicht separater Endpunkt?** Gleiche Datei, gleiche Logik — ein Parameter ist ausreichend und vermeidet Code-Duplizierung.

### 3. IBAN-Validierung

**Entscheidung:** MOD-97-Algorithmus in einer eigenständigen `validateIBAN(s string) (bool, string)`-Hilfsfunktion in `handler.go`. Rückgabe: gültig/ungültig + Fehlerbeschreibung. Nutzt ausschließlich `math/big` aus der Go-Stdlib.

**Verhalten bei ungültiger IBAN:** IBAN wird nicht gespeichert; alle anderen Felder der Zeile werden normal verarbeitet. Im Report erscheint die Zeile als `updated` (oder `created`) mit einer zusätzlichen `IBANWarning`-Meldung.

### 4. IBAN-Überschreiben

**Entscheidung:** Im `update`-Modus wird die IBAN immer überschrieben, wenn die CSV einen nichtleeren Wert enthält (auch wenn die DB schon einen anderen Wert hat). Begründung: Die CSV ist die autoritative Quelle aus der Vereinsverwaltung; einziger bestehender DB-Eintrag ist ein Test-Dummy.

### 5. Spalten-Mapping für neue Felder

CSV-Spalte → DB-Feld (Alias falls nötig):

| CSV-Spalte   | DB-Feld          | Normalisierung                              |
|-------------|------------------|---------------------------------------------|
| `Adresse`   | `street`         | —                                           |
| `PLZ`       | `zip`            | —                                           |
| `Ort`       | `city`           | —                                           |
| `Mitglied seit` | `join_date`  | `normalizeDate()` (dd.mm.yy + dd.mm.YYYY)   |
| `IBAN`      | `iban`           | Whitespace entfernen, Großbuchstaben        |
| `Kontoinhaber` | `account_holder` | —                                      |
| `SEPA Mandat` | `sepa_mandat`  | `"vorliegend"` → 1, sonst 0               |

## Risks / Trade-offs

**Fehlklassifizierung von Emails bei Erwachsenen mit funktionalen Adressen** (z.B. `info@kanzlei-baisch.de` für Marko Baisch, 56J)
→ Kein Schaden: Ein Erwachsener mit fremder Email führt nur dazu, dass kein User gefunden wird und kein Link gesetzt wird — das ist dasselbe Ergebnis wie vorher. Der Admin sieht es im Report.

**18-jährige mit noch aktiver Eltern-Email** (z.B. Ben Miess, Marwin Kühnel)
→ Werden als EIGEN klassifiziert; die Eltern-Email führt zu keinem User-Fund (da kein User mit dieser Email existiert). Kein falscher Family-Link. Akzeptabel — diese Fälle können manuell nachgepflegt werden.

**Gleiche Email für mehrere Kinder (Familien)** (15 Fälle in der aktuellen CSV)
→ Korrekt: derselbe User-Account wird per `family_links` mit mehreren Member-Einträgen verknüpft. Kein Unique-Constraint-Problem.

**Daten-Vollständigkeit von `Mitglied seit`** — gemischte Datumsformate (`07.04.25` und `22.01.2026`)
→ `normalizeDate()` behandelt bereits beide Formate; zweistellige Jahre werden als 20xx interpretiert. Kein Risiko.

## Migration Plan

Keine DB-Migration erforderlich — alle Zielfelder (`street`, `zip`, `city`, `join_date`, `iban`, `account_holder`, `sepa_mandat`) existieren bereits in der `members`-Tabelle.

Deployment: normaler `make deploy`-Lauf. Die Erweiterung ist vollständig additiv; bestehende Import-Aufrufe mit `mode=append` oder `mode=update` funktionieren unverändert.

## Open Questions

*(keine)*
