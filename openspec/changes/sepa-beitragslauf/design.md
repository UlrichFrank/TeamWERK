# Design: SEPA-Beitragslauf

## 1. Data Model

### 1.1 `clubs` — Erweiterung

```sql
ALTER TABLE clubs ADD COLUMN glaeubiger_id   TEXT;
ALTER TABLE clubs ADD COLUMN iban            TEXT;
ALTER TABLE clubs ADD COLUMN bic             TEXT;
ALTER TABLE clubs ADD COLUMN kontoinhaber    TEXT;
```

Validierung im Handler vor `PUT /api/club`:
- `glaeubiger_id` Format: `DE\d{2}[A-Z0-9]{3}\d{11}` (z.B. `DE98ZZZ09999999999`)
- `iban` Format-Check (Land + Prüfsumme + Länge nach IBAN-Registry, hier nur DE/AT/CH praktisch relevant)
- `bic` 8 oder 11 Zeichen
- Alle vier Felder müssen vor Export-Versuch gesetzt sein → Vorab-Check in `POST /api/beitragslauf/export`, sonst HTTP 400 mit klarem Fehler.

### 1.2 `members` — Erweiterung

```sql
ALTER TABLE members ADD COLUMN in_ausbildung        INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN last_sepa_einzug_am  DATETIME;
```

- `in_ausbildung` = 1 wenn „Ausbildung oder Freiwilligendienst" zutrifft. Manuell vom Vorstand gepflegt (kein Auto-Reset bei Saisonwechsel).
- `last_sepa_einzug_am` = Zeitpunkt der letzten **bestätigten** Bank-Einreichung. Steuert FRST/RCUR im XML.

### 1.3 `beitrags_saetze` — neu

```sql
CREATE TABLE beitrags_saetze (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kategorie   TEXT NOT NULL CHECK (kategorie IN (
        'aktiv_volljaehrig_ohne',
        'aktiv_volljaehrig_mit',
        'aktiv_volljaehrig_ausb_ohne',
        'aktiv_volljaehrig_ausb_mit',
        'aktiv_minderj_ohne',
        'aktiv_minderj_mit',
        'passiv'
    )),
    betrag_eur  INTEGER NOT NULL,    -- in Cent gespeichert (Integer)
    valid_from  DATE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_beitrags_saetze_kat_valid ON beitrags_saetze(kategorie, valid_from);
```

**Seed (idempotent via `INSERT OR IGNORE`):**

| Kategorie | Betrag (Cent) | valid_from |
|---|---:|---|
| aktiv_volljaehrig_ohne | 30000 | 2026-07-01 |
| aktiv_volljaehrig_mit | 14000 | 2026-07-01 |
| aktiv_volljaehrig_ausb_ohne | 22600 | 2026-07-01 |
| aktiv_volljaehrig_ausb_mit | 9600 | 2026-07-01 |
| aktiv_minderj_ohne | 22600 | 2026-07-01 |
| aktiv_minderj_mit | 9600 | 2026-07-01 |
| passiv | 6000 | 2027-01-01 |

Cent statt Float gegen Rundungsdrift. UI rechnet beim Lesen `/100` und beim Schreiben `*100`.

Lookup-Logik: „letzter Satz vor Effective Start einer Saison" =
```sql
SELECT betrag_eur FROM beitrags_saetze
WHERE kategorie = ? AND valid_from <= ?
ORDER BY valid_from DESC LIMIT 1;
```

## 2. Kategorisierungs-Logik

### 2.1 Eingangsfilter (vor Kategorie-Berechnung)

```go
type ExclusionReason string

const (
    ExclStatusInactive   ExclusionReason = "status_inaktiv"      // ausgetreten/honorar/anwaerter
    ExclBeitragsfrei     ExclusionReason = "beitragsfrei"
    ExclNoMandate        ExclusionReason = "kein_sepa_mandat"
    ExclNoIBAN           ExclusionReason = "iban_fehlt"
    ExclInvalidIBAN      ExclusionReason = "iban_ungueltig"
    ExclNoMemberNumber   ExclusionReason = "mitgliedsnummer_fehlt"
    ExclNoAddress        ExclusionReason = "adresse_unvollstaendig"
)
```

### 2.2 Status → Beitragsgruppe

```go
func beitragsGruppe(status string) string {
    switch status {
    case "aktiv", "verletzt":
        return "aktiv"
    case "pausiert", "passiv":
        return "passiv"
    default:
        return "" // ausgeschlossen
    }
}
```

### 2.3 Kategorie-Bestimmung (Aktiv-Gruppe)

```go
func aktivKategorie(volljaehrig, inAusb, mitStammverein bool) string {
    if !volljaehrig {
        if mitStammverein {
            return "aktiv_minderj_mit"
        }
        return "aktiv_minderj_ohne"
    }
    if inAusb {
        if mitStammverein {
            return "aktiv_volljaehrig_ausb_mit"
        }
        return "aktiv_volljaehrig_ausb_ohne"
    }
    if mitStammverein {
        return "aktiv_volljaehrig_mit"
    }
    return "aktiv_volljaehrig_ohne"
}
```

### 2.4 Volljährigkeit

```go
func istVolljaehrigAmSaisonstart(dob, saisonStart time.Time) bool {
    achtzehn := dob.AddDate(18, 0, 0)
    return !achtzehn.After(saisonStart) // achtzehn <= saisonStart
}
```

### 2.5 Stammverein-Match

Hardcodierte Whitelist (case-insensitive, Vergleich nach Normalisierung — alle Klein, Whitespace-collapse, Punkte/Bindestriche raus):

```go
var mitgliedsvereine = []string{
    "SKG Gablenberg 1884",
    "SKG Stuttgart Max-Eyth-See 1898",
    "SportKultur Stuttgart",
    "Spvgg 1897 Cannstatt",
    "TB Gaisburg 1886",
    "TB Untertürkheim 1888",
    "TSV Stuttgart-Münster 1875/99",
    "TV Cannstatt 1846",
}

type ClubMatch struct {
    Matched bool
    Canonical string  // gefundener Eintrag aus Whitelist
    Warning string    // "unklarer Treffer: home_club='X', nächster Match=Y"
}

func matchHomeClub(homeClub string) ClubMatch
// - leerer/NULL home_club → Matched=false, Warning="" (= "ohne Stammverein", kein Warnhinweis)
// - exakter Normalisierungs-Match → Matched=true, kein Warning
// - Levenshtein-Distance ≤ 3 ODER Teilstring-Match auf normalisierte Form → Matched=true, Warning gesetzt
// - sonst → Matched=false, Warning="home_club='X' konnte keinem Mitgliedsverein zugeordnet werden"
```

Bei `Warning != ""` zeigt das UI ein gelbes Hinweis-Icon in der Vorschau, der Vorstand kann manuell den Haken setzen/entfernen, die Berechnung läuft wie bei Match.

### 2.6 Pro-rata-Logik

```go
// effectiveStart = MAX(saisonStart, validFromKategorie, joinDate)
// monate = Anzahl voller Kalendermonate ab Folgemonat von effectiveStart bis Saisonende
//
// Beispiel: saisonStart=2026-07-01, joinDate=2026-09-15, validFromKategorie=2026-07-01
//   effectiveStart = 2026-09-15
//   FolgeMonat-Anfang = 2026-10-01
//   Saisonende = 2027-06-30
//   monate = 9 (Oktober bis Juni einschl.)

func proRataMonate(effectiveStart, saisonEnde time.Time) int {
    folgeMonat := time.Date(effectiveStart.Year(), effectiveStart.Month()+1, 1, 0, 0, 0, 0, time.UTC)
    if !folgeMonat.Before(saisonEnde) {
        return 0
    }
    monate := (saisonEnde.Year()-folgeMonat.Year())*12 + int(saisonEnde.Month()-folgeMonat.Month()) + 1
    if monate < 0 { return 0 }
    if monate > 12 { return 12 }
    return monate
}

func proRataBetrag(jahresbeitragCent int, monate int) int {
    // kaufmännische Rundung auf Cent
    raw := float64(jahresbeitragCent) * float64(monate) / 12.0
    return int(math.Round(raw))
}
```

**Sonderfall Saisonbeitritt am 1.:** join_date=2026-07-01 → folgeMonat=2026-08-01 → monate=11. Bewusst so: wer am Ersten kommt, zählt den Monat nicht voll (Konsistenz mit der "angefangene Monate zählen nicht"-Regel). Falls Vorstand das anders will: in den Tasks als manuelle Anpassung dokumentieren.

### 2.7 FRST/RCUR-Bestimmung

```go
func sequenceType(lastEinzug *time.Time) string {
    if lastEinzug == nil {
        return "FRST"
    }
    return "RCUR"
}
```

Hinweis: Bank kann bei Rejekt verlangen, dass nächster Einzug wieder FRST ist. Dafür existiert `PUT /api/members/{id}/sepa-sequence-reset`.

## 3. Vorschau-Endpoint

### 3.1 Request

```
GET /api/beitragslauf/preview?saison_id=42
```

### 3.2 Response-Schema

```json
{
  "saison_id": 42,
  "saison_label": "2026/27",
  "faelligkeit": "2026-06-23",
  "items": [
    {
      "member_id": 17,
      "name": "Max Müller",
      "status": "aktiv",
      "kategorie": "aktiv_volljaehrig_mit",
      "kategorie_label": "Aktiv volljährig (mit Stammverein)",
      "monate": 12,
      "jahresbeitrag_cent": 14000,
      "betrag_cent": 14000,
      "seq_tp": "FRST",
      "included": true,
      "warnings": [],
      "exclusions": []
    },
    {
      "member_id": 18,
      "name": "Anna Schmidt",
      "status": "aktiv",
      "kategorie": "aktiv_minderj_mit",
      "monate": 9,
      "jahresbeitrag_cent": 9600,
      "betrag_cent": 7200,
      "seq_tp": "RCUR",
      "included": true,
      "warnings": ["home_club_unklar"],
      "exclusions": []
    },
    {
      "member_id": 19,
      "name": "Pete Honorar",
      "status": "honorar",
      "included": false,
      "exclusions": ["status_inaktiv"]
    }
  ],
  "summary": {
    "included_count": 87,
    "excluded_count": 12,
    "warned_count": 3,
    "total_cent": 1865400
  }
}
```

### 3.3 Datenquellen pro Member

Eine SQL-Query pro Member ist verschwenderisch. Stattdessen ein einziger JOIN mit `members LEFT JOIN clubs ON 1=1` (clubs ist Single-Row, in der Praxis konfiguriert mit `id=1`).

Die Sätze (`beitrags_saetze`) werden einmal pro Request geladen — eine Map `kategorie -> []SatzMitValidFrom`, sortiert absteigend. Lookup pro Member dann reine In-Memory-Operation.

## 4. Export-Endpoint

### 4.1 Request

```
POST /api/beitragslauf/export
Content-Type: application/json

{
  "saison_id": 42,
  "member_ids": [17, 18, 21, 22, ...]
}
```

### 4.2 Response

```
200 OK
Content-Type: application/xml
Content-Disposition: attachment; filename="beitragslauf_2026_2027_2026-06-16.xml"

<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.008.001.08">
  <CstmrDrctDbtInitn>
    <GrpHdr>...</GrpHdr>
    <PmtInf>...</PmtInf>
  </CstmrDrctDbtInitn>
</Document>
```

### 4.3 XML-Struktur (Kerngerüst)

```xml
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.008.001.08">
  <CstmrDrctDbtInitn>
    <GrpHdr>
      <MsgId>TW-{saison_kurz}-{YYYYMMDDHHMMSS}</MsgId>
      <CreDtTm>{ISO-Zeitpunkt}</CreDtTm>
      <NbOfTxs>{N}</NbOfTxs>
      <CtrlSum>{Summe in Euro mit Punkt-Dezimalstelle}</CtrlSum>
      <InitgPty>
        <Nm>{clubs.name}</Nm>
        <Id><OrgId><Othr><Id>{clubs.glaeubiger_id}</Id></Othr></OrgId></Id>
      </InitgPty>
    </GrpHdr>

    <!-- Ein PmtInf-Block pro SeqTp (FRST und RCUR getrennt!) -->
    <PmtInf>
      <PmtInfId>TW-{saison_kurz}-FRST-{YYYYMMDD}</PmtInfId>
      <PmtMtd>DD</PmtMtd>
      <BtchBookg>true</BtchBookg>
      <NbOfTxs>{N_FRST}</NbOfTxs>
      <CtrlSum>{Summe FRST}</CtrlSum>
      <PmtTpInf>
        <SvcLvl><Cd>SEPA</Cd></SvcLvl>
        <LclInstrm><Cd>CORE</Cd></LclInstrm>
        <SeqTp>FRST</SeqTp>
      </PmtTpInf>
      <ReqdColltnDt>{heute + 7 Werktage, YYYY-MM-DD}</ReqdColltnDt>
      <Cdtr>
        <Nm>{clubs.kontoinhaber}</Nm>
        <PstlAdr>{strukturierte Vereinsadresse aus clubs.address}</PstlAdr>
      </Cdtr>
      <CdtrAcct>
        <Id><IBAN>{clubs.iban}</IBAN></Id>
      </CdtrAcct>
      <CdtrAgt>
        <FinInstnId><BICFI>{clubs.bic}</BICFI></FinInstnId>
      </CdtrAgt>
      <CdtrSchmeId>
        <Id><PrvtId><Othr>
          <Id>{clubs.glaeubiger_id}</Id>
          <SchmeNm><Prtry>SEPA</Prtry></SchmeNm>
        </Othr></PrvtId></Id>
      </CdtrSchmeId>

      <!-- Pro Member: ein DrctDbtTxInf -->
      <DrctDbtTxInf>
        <PmtId><EndToEndId>TW-{member_number}-{saison_kurz}</EndToEndId></PmtId>
        <InstdAmt Ccy="EUR">{betrag mit Punkt}</InstdAmt>
        <DrctDbtTx>
          <MndtRltdInf>
            <MndtId>{members.member_number}</MndtId>
            <DtOfSgntr>{members.sepa_mandat_date}</DtOfSgntr>
          </MndtRltdInf>
        </DrctDbtTx>
        <DbtrAgt><FinInstnId><Othr><Id>NOTPROVIDED</Id></Othr></FinInstnId></DbtrAgt>
        <Dbtr>
          <Nm>{account_holder ODER first_name+last_name}</Nm>
          <PstlAdr>
            <StrtNm>{street ohne Hausnr}</StrtNm>
            <BldgNb>{Hausnr aus street}</BldgNb>
            <PstCd>{zip}</PstCd>
            <TwnNm>{city}</TwnNm>
            <Ctry>DE</Ctry>
          </PstlAdr>
        </Dbtr>
        <DbtrAcct>
          <Id><IBAN>{members.iban}</IBAN></Id>
        </DbtrAcct>
        <RmtInf>
          <Ustrd>Jahresbeitrag Saison {saison_kurz} – Mitgliedsnr. {member_number}</Ustrd>
        </RmtInf>
      </DrctDbtTxInf>
    </PmtInf>

    <PmtInf>
      <!-- SeqTp=RCUR, sonst identisch -->
    </PmtInf>
  </CstmrDrctDbtInitn>
</Document>
```

### 4.4 Wichtige Constraints

- Ein `PmtInf`-Block je `SeqTp` — pain.008.001.08 erlaubt nicht, FRST und RCUR im selben Block zu mischen.
- `ReqdColltnDt`: für FRST muss Bank ≥ 5 Bankarbeitstage Vorlauf haben, für RCUR ≥ 2. Wir setzen einheitlich `heute + 7 Kalendertage` und verschieben auf den nächsten Werktag, falls Sa/So.
- `CtrlSum` als Euro mit Punkt-Dezimaltrenner (`14000` Cent → `140.00`).
- `MsgId` und `PmtInfId` müssen ≤ 35 Zeichen, ASCII (keine Umlaute).
- `EndToEndId` muss eindeutig pro Buchung innerhalb der Datei.
- Adresse strukturiert (pain.008.001.08-Pflicht) — kein `AdrLine`-Fallback mehr.

### 4.5 Straßen-Parsing

`street` ist im Bestand häufig „Hauptstr. 12", „Am Bach 3a". Regex:

```go
var streetRe = regexp.MustCompile(`^(.+?)\s+(\d+\s*[a-zA-Z]?)$`)
// Match: Group 1 = StrtNm, Group 2 = BldgNb
// No-Match: kompletter String als StrtNm, BldgNb leer (XSD erlaubt fehlen)
```

Edge-Case: Eingaben wie "Postfach 100" → kein Hausnr-Match → komplett in StrtNm. Akzeptabel; ein Postfach für Lastschrift ist unüblich, kommt in Praxis nur theoretisch vor.

### 4.6 Side-Effects

Der Export ist **nicht** selbst der Confirm. Er **setzt nicht** `last_sepa_einzug_am`. Begründung: Vorstand lädt evtl. die XML runter, prüft, korrigiert, lädt nochmal. Erst nach „Bei Bank hochgeladen bestätigen" wird der Status persistiert.

## 5. Confirm-Endpoint

### 5.1 Request

```
POST /api/beitragslauf/confirm-uploaded
Content-Type: application/json

{
  "member_ids": [17, 18, 21, 22, ...]
}
```

### 5.2 Logik

```sql
UPDATE members SET last_sepa_einzug_am = CURRENT_TIMESTAMP
WHERE id IN (?, ?, ...);
```

Keine Idempotenz-Garantie über mehrere Aufrufe hinweg — wenn der Vorstand „Confirm" doppelt klickt, wird das Datum erneut gesetzt. Das ist OK (RCUR bleibt RCUR). Die Test-Invariante „pro Saison max. einmal gesetzt" ist im Sinne von: pro Lauf-Bestätigung wird der SeqTp-Status korrekt fortgeschrieben.

### 5.3 Response

```json
{ "updated_count": 87 }
```

### 5.4 Live-Update

```go
h.hub.Broadcast("members-changed")
```

Damit verbundene Detail-Seiten den „SEPA-Sequenz zurücksetzen"-Button korrekt anzeigen.

## 6. Reset-Endpoint

```
PUT /api/members/{id}/sepa-sequence-reset
```

```sql
UPDATE members SET last_sepa_einzug_am = NULL WHERE id = ?;
```

Wird genutzt, wenn die Bank ein Mandat ablehnt und der nächste Einzug wieder FRST sein muss. Audit-Log nicht in Scope dieses Proposals (separates Cleanup ist möglich, wenn Auditierbarkeit gewünscht).

## 7. Beitragssätze-Pflege

### 7.1 Endpoints

```
GET /api/beitrags-saetze
```

Response: alle Sätze, sortiert nach `kategorie, valid_from DESC`.

```json
{
  "items": [
    { "id": 7, "kategorie": "aktiv_volljaehrig_mit", "betrag_cent": 14000, "valid_from": "2026-07-01" },
    { "id": 14, "kategorie": "aktiv_volljaehrig_mit", "betrag_cent": 15000, "valid_from": "2027-07-01" }
  ]
}
```

```
POST /api/beitrags-saetze
Content-Type: application/json

{ "kategorie": "passiv", "betrag_cent": 7000, "valid_from": "2028-01-01" }
```

Validation:
- `kategorie` in CHECK-Liste
- `betrag_cent > 0`
- `valid_from` parsebar ISO-Datum
- Kein Dedup-Check: zwei Sätze für dieselbe `kategorie + valid_from` sind erlaubt (User-Fehler, der über UI sichtbar wird).

### 7.2 Frontend — neuer Tab in AdminSettingsPage

```
[Verein] [Saisons] [Kader] [Beiträge*] [Dienste] [Nutzer]
```

Tab-Inhalt: pro Kategorie eine Tabelle mit Spalten `valid_from | Betrag €`. Inline-Form unten zum Anlegen eines neuen Eintrags.

## 8. Frontend — `/admin/beitragslauf`

### 8.1 Layout

```
┌──────────────────────────────────────────────────────────────────┐
│ Beitragslauf                                                     │
├──────────────────────────────────────────────────────────────────┤
│ Saison: [2026/27 ▾]                                              │
│                                                                  │
│ ☑ 87 angehakt · ⚠ 3 Warnungen · ⛔ 12 ausgeschlossen           │
│ Summe: 18.654,00 €                                              │
│                                                                  │
│ ┌──────────────────────────────────────────────────────────────┐ │
│ │ ☑ │ Name           │ Status  │ Kategorie         │ Mon │ Be... │
│ ├──────────────────────────────────────────────────────────────┤ │
│ │ ☑ │ Max Müller     │ aktiv   │ Aktiv volljährig… │ 12  │ 140… │
│ │ ☑⚠│ Anna Schmidt   │ aktiv   │ Aktiv minderj.…   │  9  │  72… │
│ │ ⛔│ Pete Honorar   │ honorar │ —                 │  —  │   — │
│ └──────────────────────────────────────────────────────────────┘ │
│                                                                  │
│ [XML herunterladen]                                              │
└──────────────────────────────────────────────────────────────────┘
```

### 8.2 State-Machine

```
[Vorschau geladen]
   │
   ├── User passt Haken an → State.selected_ids ändert sich
   │
   ├── [XML herunterladen] → POST /export mit selected_ids
   │    → Browser-Download
   │    → State wechselt auf "exported"
   │
[exported]
   │
   ├── [Bei Bank hochgeladen bestätigen] → POST /confirm-uploaded
   │    → Success-Toast
   │    → Reload Vorschau (jetzt mit aktualisiertem SeqTp = RCUR)
   │
[idle]
```

### 8.3 Visual Cues

- Roter „⛔" Icon bei ausgeschlossenen Zeilen, mit Tooltip-Liste der Exclusion-Gründe.
- Gelbes „⚠" Icon bei warnungsbehafteten Zeilen.
- Checkbox bei ausgeschlossenen Zeilen disabled.
- Spalte SeqTp zeigt `FRST` (grün) oder `RCUR` (grau).

## 9. Caveats und bewusste Vereinfachungen

1. **Kein Job-Queueing / kein E-Mail-Versand** der XML. Vorstand lädt selbst herunter und reicht im Banking-Portal ein.
2. **Kein Diff zur Vor-Saison.** Falls Beitragssatz innerhalb einer Saison ändert (selten), nimmt der Vorschau-Endpoint den Satz, der zum Effective Start gilt. Mid-Season-Wechsel wird nicht zwei-stufig abgerechnet (Komplexität nicht gerechtfertigt).
3. **Stammverein-Whitelist hardcoded.** Wenn ein neuer Verein hinzukommt (selten), Code-Änderung nötig. Pflegbare Tabelle wäre möglich, lohnt aktuell den Aufwand nicht.
4. **Adresse strukturiert.** Bestandsdaten haben evtl. „Hauptstr. 12 / Hinterhof"-Eingaben — die werden nicht perfekt zerlegt. Im Vorschau-Endpoint wird das als Warnung markiert (`adresse_komplex`), keine harte Exclusion.
5. **Keine Multi-Club-Mandanten.** TeamWERK kennt aktuell nur einen Verein (`clubs.id=1`). Falls jemals mehrere Vereine, müsste die Beitragsmatrix pro Club gehen.
6. **Kein direkter HBCI-/EBICS-Upload.** Out of scope; Vorstand nutzt BW-Bank-Onlinebanking manuell.

## 10. Test-Strategie

### 10.1 Unit-Tests im Package `beitragslauf`

- `proRataMonate_*` — Tabelle mit (effectiveStart, saisonEnde, expectedMonate)
- `proRataBetrag_KaufmaennischeRundung` — z.B. 14000 Cent × 9 / 12 = 10500.0 (rund), 14000 × 7 / 12 = 8166.666… → 8167
- `aktivKategorie_AlleKombinationen` — 2³ = 8 Fälle
- `istVolljaehrigAmSaisonstart_Stichtag` — DOB = saisonstart - 18 Jahre → true; DOB = saisonstart - 18 Jahre + 1 Tag → false
- `matchHomeClub_*` — exakt, fuzzy, leer, Müll
- `sequenceType_FRST_vs_RCUR`

### 10.2 Integration-Tests im Handler

Setup über `testutil.NewServer(t, db, allRoutes)` mit gesäter clubs-Row + beitrags_saetze + members in allen relevanten Konstellationen.

Tests siehe Proposal-Abschnitt „Test-Anforderungen".

### 10.3 XSD-Validierung

```go
//go:embed pain.008.001.08.xsd
var xsdBytes []byte

func TestExport_XSDValid(t *testing.T) {
    xml := exportXML(t, ...)
    if err := validateAgainstXSD(xml, xsdBytes); err != nil {
        t.Fatalf("XML invalid: %v", err)
    }
}
```

Implementierung über `github.com/lestrrat-go/libxml2` oder einfacher: `xmllint` aufrufen, falls auf VPS verfügbar. In CI/lokal über externe Library — siehe Tasks-Eintrag.

## 11. Risiken & Mitigationen

| Risiko | Mitigation |
|---|---|
| Bank lehnt XML aus formellem Grund ab | XSD-Validierung im Test verhindert Schema-Fehler; manueller Probelauf mit Test-IBAN vor Echtbetrieb |
| Falsche FRST/RCUR-Markierung → Bank-Reject | Reset-Endpoint, automatische Vorschau zeigt SeqTp, Vorstand kann pro Member ablesen |
| Vorstand wählt falsche Member-Untermenge | Klare Summe + Anzahl im Header der Vorschau; "Ausgeschlossen"-Liste ausklappbar |
| Doppelte Einreichung | Warnung „last_sepa_einzug_am ≥ Saisonstart" im UI; trotzdem nicht hart geblockt (manche Bankreklamationen erfordern Wiedereinreichung) |
| Stammverein-Fehlklassifizierung | Warnung im UI, manuelle Override-Möglichkeit |
| Cent-Rundungsdrift bei Pro-rata | Cent-Integer in DB, `math.Round` einmalig pro Member, Summe = `Σ betrag_cent` (exakt) |
