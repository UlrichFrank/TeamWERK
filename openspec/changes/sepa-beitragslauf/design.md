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

### 1.2 `members` — keine Änderung

Es werden **keine** neuen `members`-Spalten benötigt. Da alle Einzüge `RCUR` sind, gibt es kein FRST/RCUR-Tracking (`last_sepa_einzug_am` entfällt), und Ausbildung/Beruf werden für den Beitrag nicht berücksichtigt (`in_ausbildung` entfällt).

### 1.3 `beitrags_saetze` — neu

```sql
CREATE TABLE beitrags_saetze (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    kategorie   TEXT NOT NULL CHECK (kategorie IN (
        'aktiv_ohne',
        'aktiv_mit',
        'passiv'
    )),
    betrag_eur  INTEGER NOT NULL,    -- in Cent gespeichert (Integer)
    valid_from  DATE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_beitrags_saetze_kat_valid ON beitrags_saetze(kategorie, valid_from);
```

Kategorien:
- `aktiv_ohne` — Aktivbeitrag ohne Stammverein (Kinder-Satz, gilt für alle aktiven Spieler)
- `aktiv_mit` — Aktivbeitrag mit Stammverein (ermäßigt)
- `passiv` — Passivbeitrag

**Seed (idempotent via `INSERT OR IGNORE`):**

| Kategorie | Betrag (Cent) | valid_from |
|---|---:|---|
| aktiv_ohne | 22600 | 2026-07-01 |
| aktiv_mit | 9600 | 2026-07-01 |
| passiv | 6000 | 2027-01-01 |

Cent statt Float gegen Rundungsdrift. UI rechnet beim Lesen `/100` und beim Schreiben `*100`.

Lookup-Logik: „letzter Satz vor Saisonstart (01.07.)" =
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

### 2.3 Kategorie-Bestimmung

Volljährigkeit, Ausbildung und Beruf werden **nicht** geprüft. Die einzige Abstufung innerhalb der Aktiv-Gruppe ist die Stammverein-Zugehörigkeit:

```go
func aktivKategorie(mitStammverein bool) string {
    if mitStammverein {
        return "aktiv_mit"
    }
    return "aktiv_ohne"
}
```

Die Passiv-Gruppe bildet immer Kategorie `passiv`.

### 2.4 Stammverein-Match

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

### 2.5 Beitrag = voller Jahresbeitrag

Es gibt **keine Pro-rata-Berechnung**. Jedes eingeschlossene Mitglied wird mit dem vollen Jahresbeitrag laut Beitragssatz abgerechnet — unabhängig von `join_date` oder Eintrittszeitpunkt:

```go
// betragCent = LookupBetragCent(saetze, kategorie, saisonStart)
// saisonStart = 01.07.YYYY der gewählten Saison
```

`join_date` beeinflusst den Betrag nicht (kein anteiliger Abzug).

### 2.6 Sequenztyp — immer RCUR

Es wird **nicht** zwischen Erst- und Folgelastschrift unterschieden. Alle Buchungen tragen `SeqTp = RCUR`. Es gibt daher kein FRST/RCUR-Tracking, keine `confirm-uploaded`- und keine `sequence-reset`-Logik.

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
  "faelligkeit": "2026-07-01",
  "items": [
    {
      "member_id": 17,
      "name": "Max Müller",
      "status": "aktiv",
      "kategorie": "aktiv_mit",
      "kategorie_label": "Aktiv (mit Stammverein)",
      "betrag_cent": 9600,
      "included": true,
      "warnings": [],
      "exclusions": []
    },
    {
      "member_id": 18,
      "name": "Anna Schmidt",
      "status": "aktiv",
      "kategorie": "aktiv_ohne",
      "betrag_cent": 22600,
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

Die Sätze (`beitrags_saetze`) werden einmal pro Request geladen — eine Map `kategorie -> []SatzMitValidFrom`, sortiert absteigend. Lookup pro Member dann reine In-Memory-Operation gegen den Saisonstart.

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
Content-Disposition: attachment; filename="beitragslauf_2026_2027.xml"

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

    <!-- Genau ein PmtInf-Block, SeqTp immer RCUR -->
    <PmtInf>
      <PmtInfId>TW-{saison_kurz}-RCUR</PmtInfId>
      <PmtMtd>DD</PmtMtd>
      <BtchBookg>true</BtchBookg>
      <NbOfTxs>{N}</NbOfTxs>
      <CtrlSum>{Summe}</CtrlSum>
      <PmtTpInf>
        <SvcLvl><Cd>SEPA</Cd></SvcLvl>
        <LclInstrm><Cd>CORE</Cd></LclInstrm>
        <SeqTp>RCUR</SeqTp>
      </PmtTpInf>
      <ReqdColltnDt>{01.07. der Saison, YYYY-MM-DD}</ReqdColltnDt>
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
  </CstmrDrctDbtInitn>
</Document>
```

### 4.4 Wichtige Constraints

- Genau **ein** `PmtInf`-Block — alle Buchungen `SeqTp = RCUR`. Keine FRST/RCUR-Trennung.
- `ReqdColltnDt`: Fälligkeit ist immer der 01.07. der Saison. Fällt der 01.07. auf Sa/So, wird auf den nächsten Werktag verschoben.
- `CtrlSum` als Euro mit Punkt-Dezimaltrenner (`9600` Cent → `96.00`).
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

Der Export hat **keine** persistenten Side-Effects. Es wird nichts markiert. Der Kassierer kann die XML beliebig oft erzeugen, prüfen und neu herunterladen. Persistiert wird ausschließlich beim separaten Bestätigen (§5).

## 5. Confirm-Endpoint & Saison-Protokoll

### 5.1 Request

```
POST /api/beitragslauf/confirm
Content-Type: application/json

{
  "saison_id": 42,
  "results": [
    { "member_id": 17, "betrag_cent": 9600,  "success": true },
    { "member_id": 18, "betrag_cent": 22600, "success": true },
    { "member_id": 99, "betrag_cent": 22600, "success": false }
  ]
}
```

Die `results` kommen vom UI aus der zuvor exportierten Auswahl. Default im UI: alle `success = true`; der Kassierer hakt einzelne Mitglieder als „nicht eingezogen" ab.

### 5.2 Protokoll-Datei

Das Protokoll ist eine **append-only Textdatei pro Saisonjahr**, kein DB-Eintrag.

- Verzeichnis: `cfg.BeitragslaufDir` (neue Konfig, default `./storage/beitragslauf-protokolle`), beim Start via `os.MkdirAll` angelegt
- Dateiname: `beitragslauf_{saison_kurz}.txt`, z.B. `beitragslauf_2026-2027.txt` (`saison_kurz` aus dem Saison-Label, `/`→`-`)
- Öffnen mit `os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)` — jeder Confirm hängt einen Block an, bestehende Blöcke werden nie verändert

Format eines angehängten Blocks:

```
=== Lauf bestätigt 2026-07-02T14:33:05Z durch anna@team-stuttgart.org (User #5) ===
Erfolgreich (2) — Summe 318,00 €
  Mitgl.-Nr 1042  Max Müller          96,00 €
  Mitgl.-Nr 1099  Anna Schmidt       226,00 €
Nicht erfolgreich (1)
  Mitgl.-Nr 1200  Pete Reject        226,00 €

```

Der Zeitstempel kommt aus `time.Now()` im Handler (nicht aus dem Request). Name/Mitgliedsnummer werden beim Confirm aus der DB nachgeladen, damit das Protokoll auch ohne UI-Mitlieferung vollständig ist.

### 5.3 Response

```json
{ "saison_label": "2026/27", "erfolgreich": 2, "nicht_erfolgreich": 1, "summe_erfolgreich_cent": 31800 }
```

### 5.4 Protokoll lesen

```
GET /api/beitragslauf/protocol?saison_id=42
```

Gibt den kompletten Textdatei-Inhalt als `text/plain` zurück (für Anzeige im UI und Download). Existiert keine Datei, antwortet der Server mit `200` und leerem Body (oder `404` — Implementierung wählt eine Variante, Test deckt sie ab).

### 5.5 Berechtigung & Side-Effects

`POST /confirm` und `GET /protocol` laufen unter `auth.RequireClubFunction("vorstand", "kassierer")` mit Admin-Override. Der Confirm verändert **keine** Mitgliederdaten — er schreibt ausschließlich das Protokoll. Kein FRST/RCUR-Tracking, keine DB-Mutation.

## 6. Kassierer-Zugriff auf Mitglieder

### 6.1 Route-Gruppen-Umbau in `router.go`

Heute liegen die Member-Lese-Routen in der Vorstand-only-Gruppe (`auth.RequireClubFunction("vorstand")`). Folgende **Lese**-Routen wandern in eine neue `vorstand`+`kassierer`-Gruppe:

- `GET /api/members` (`List`)
- `GET /api/members/{id}` (`Get`)
- `GET /api/members/{id}/parents` (`GetMemberParents`)
- `GET /api/members/export` (`Export`)

Zusätzlich für `kassierer` freigegeben (Bankdaten-Pflege):

- `PUT /api/members/{id}/bankdaten` (neuer Handler, s.u.)
- `POST /api/upload/sepa-mandat/{id}` (`Upload.UploadSepaMandat`)
- `DELETE /api/members/{id}/sepa-mandat` (`Upload.DeleteSepaMandat`)

**Unverändert vorstand-only:** `POST /api/members`, `PUT /api/members/{id}` (Voll-Update), `PUT /api/members/{id}/status`, `DELETE`, `Import`, `LinkUser`, Family-Links, Proxy-Account, Welcome-Mail.

Da `admin` alle `RequireClubFunction`-Checks umgeht, bleibt Admin-Zugriff erhalten.

### 6.2 Bankdaten-Endpoint (Feld-Whitelist)

```
PUT /api/members/{id}/bankdaten
Content-Type: application/json

{ "iban": "DE…", "sepa_mandat": true, "sepa_mandat_date": "2026-05-01",
  "account_holder": "Max Müller", "street": "Hauptstr. 12", "zip": "70182", "city": "Stuttgart" }
```

Der Handler aktualisiert **ausschließlich** diese Spalten via gezieltem `UPDATE members SET iban=?, sepa_mandat=?, sepa_mandat_date=?, account_holder=?, street=?, zip=?, city=? WHERE id=?`. Name, Status, Rollen, `beitragsfrei` etc. werden nicht angefasst. IBAN wird wie in §1.1 mit Mod-97 validiert (sonst 400). `h.hub.Broadcast("members-changed")`.

## 7. Beitragssätze-Pflege

### 7.1 Endpoints

```
GET /api/beitrags-saetze
```

Response: alle Sätze, sortiert nach `kategorie, valid_from DESC`.

```json
{
  "items": [
    { "id": 7, "kategorie": "aktiv_mit", "betrag_cent": 9600, "valid_from": "2026-07-01" },
    { "id": 14, "kategorie": "aktiv_mit", "betrag_cent": 10000, "valid_from": "2027-07-01" }
  ]
}
```

```
POST /api/beitrags-saetze
Content-Type: application/json

{ "kategorie": "passiv", "betrag_cent": 7000, "valid_from": "2028-01-01" }
```

Validation:
- `kategorie` in CHECK-Liste (`aktiv_ohne`, `aktiv_mit`, `passiv`)
- `betrag_cent > 0`
- `valid_from` parsebar ISO-Datum
- Kein Dedup-Check: zwei Sätze für dieselbe `kategorie + valid_from` sind erlaubt (User-Fehler, der über UI sichtbar wird).

### 7.2 Frontend — neuer Tab in AdminSettingsPage

```
[Verein] [Saisons] [Kader] [Beiträge*] [Dienste] [Nutzer]
```

Tab-Inhalt: pro Kategorie (3 Stück) eine Tabelle mit Spalten `valid_from | Betrag €`. Inline-Form unten zum Anlegen eines neuen Eintrags.

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
│ │ ☑ │ Name           │ Status  │ Kategorie         │ Betrag    │ │
│ ├──────────────────────────────────────────────────────────────┤ │
│ │ ☑ │ Max Müller     │ aktiv   │ Aktiv (mit Stamm…)│  96,00 €  │ │
│ │ ☑⚠│ Anna Schmidt   │ aktiv   │ Aktiv (ohne Stam…)│ 226,00 €  │ │
│ │ ⛔│ Pete Honorar   │ honorar │ —                 │   —       │ │
│ └──────────────────────────────────────────────────────────────┘ │
│                                                                  │
│ [XML herunterladen]   [Lauf bestätigen]   [Protokoll ansehen]    │
└──────────────────────────────────────────────────────────────────┘
```

### 8.2 Ablauf

```
[Vorschau geladen]
   │
   ├── User passt Haken an → State.selected_ids ändert sich
   │
   ├── [XML herunterladen] → POST /export mit selected_ids → Browser-Download
   │
   ├── [Lauf bestätigen] → Dialog: Liste der exportierten Mitglieder,
   │        Default „erfolgreich", einzelne als „nicht eingezogen" abhakbar
   │        → POST /confirm {saison_id, results} → Protokoll fortgeschrieben, Toast
   │
   └── [Protokoll ansehen] → GET /protocol?saison_id=… → Textinhalt in Modal/Download
```

„Lauf bestätigen" ist erst nach einem Export aktiv. Es gibt keinen Zwang zur Bestätigung — der Kassierer bestätigt erst, wenn die Bank den Lauf angenommen hat.

### 8.3 Visual Cues

- Roter „⛔" Icon bei ausgeschlossenen Zeilen, mit Tooltip-Liste der Exclusion-Gründe.
- Gelbes „⚠" Icon bei warnungsbehafteten Zeilen.
- Checkbox bei ausgeschlossenen Zeilen disabled.

## 9. Caveats und bewusste Vereinfachungen

1. **Keine anteilige Berechnung.** Jedes einzuziehende Mitglied zahlt den vollen Jahresbeitrag, fällig immer zum 01.07. Eintrittsdatum spielt keine Rolle.
2. **Keine Volljährigkeits-, Ausbildungs- oder Berufsprüfung.** Aktive Spieler werden grundsätzlich mit dem Kinder-/Aktiv-Satz abgerechnet.
3. **Keine Erst-/Folge-Unterscheidung.** Alle Lastschriften sind RCUR; kein FRST/RCUR-Tracking, kein Sequence-Reset. Die Bestätigung dient nur dem Protokoll, nicht der SeqTp-Steuerung.
4. **Protokoll ist eine Textdatei, keine DB-Tabelle.** Bewusst einfach gehalten (append-only). Keine strukturierte Auswertung/Filterung; wer Statistiken will, parst die Datei manuell.
5. **Kein automatischer Bank-Rückabgleich.** Rejects (pain.002) werden nicht eingelesen; der Kassierer markiert „nicht erfolgreich" manuell beim Bestätigen.
6. **Kein Job-Queueing / kein E-Mail-Versand** der XML. Kassierer lädt selbst herunter und reicht im Banking-Portal ein.
7. **Stammverein-Whitelist hardcoded.** Wenn ein neuer Verein hinzukommt (selten), Code-Änderung nötig. Pflegbare Tabelle wäre möglich, lohnt aktuell den Aufwand nicht.
8. **Adresse strukturiert.** Bestandsdaten haben evtl. „Hauptstr. 12 / Hinterhof"-Eingaben — die werden nicht perfekt zerlegt. Im Vorschau-Endpoint wird das als Warnung markiert (`adresse_komplex`), keine harte Exclusion.
9. **Keine Multi-Club-Mandanten.** TeamWERK kennt aktuell nur einen Verein (`clubs.id=1`). Falls jemals mehrere Vereine, müsste die Beitragsmatrix pro Club gehen.
10. **Kein direkter HBCI-/EBICS-Upload.** Out of scope; Vorstand/Kassierer nutzt BW-Bank-Onlinebanking manuell.

## 10. Test-Strategie

### 10.1 Unit-Tests im Package `beitragslauf`

- `aktivKategorie_MitOhneStammverein` — 2 Fälle
- `beitragsGruppe_*` — aktiv/verletzt → aktiv, pausiert/passiv → passiv, sonst → ""
- `matchHomeClub_*` — exakt, fuzzy, leer, Müll
- `lookupBetragCent_*` — Satz zum Saisonstart, kein Satz vor valid_from → Error

### 10.2 Integration-Tests im Handler

Setup über `testutil.NewServer(t, db, allRoutes)` mit gesäter clubs-Row + beitrags_saetze + members in allen relevanten Konstellationen.

Tests siehe Proposal-Abschnitt „Test-Anforderungen". Insbesondere `TestPreview_NeumitgliedZahltVollenBeitrag` sichert die „kein Pro-rata"-Invariante ab, `TestExport_EinPmtInfBlockRCUR` die „immer RCUR"-Invariante, `TestConfirm_HaengtProtokollAn` die Append-only-Invariante, `TestBankdaten_KassiererUpdatetNurBankfelder` die Feld-Whitelist. Protokoll-Tests setzen `cfg.BeitragslaufDir` auf ein `t.TempDir()`.

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

Implementierung über `xmllint --schema` als externer Aufruf (Skip wenn nicht im PATH) — siehe Tasks-Eintrag.

## 11. Risiken & Mitigationen

| Risiko | Mitigation |
|---|---|
| Bank lehnt XML aus formellem Grund ab | XSD-Validierung im Test verhindert Schema-Fehler; manueller Probelauf mit Test-IBAN vor Echtbetrieb |
| Kassierer wählt falsche Member-Untermenge | Klare Summe + Anzahl im Header der Vorschau; "Ausgeschlossen"-Liste ausklappbar |
| Stammverein-Fehlklassifizierung | Warnung im UI, manuelle Override-Möglichkeit |
| Fälligkeit 01.07. liegt in der Vergangenheit | Kassierer reicht den Lauf rechtzeitig ein; XML-Datum bleibt 01.07. der Saison, Bank verschiebt ggf. auf nächsten Ausführungstag |
| Protokolldatei geht verloren / nicht im Backup | `BeitragslaufDir` ins Backup aufnehmen (Doku in CLAUDE.md/Deploy); append-only minimiert versehentliches Überschreiben |
| Kassierer ändert versehentlich Nicht-Bankfelder | Dedizierter `bankdaten`-Endpoint mit Feld-Whitelist; voller Member-Update bleibt vorstand-only |
