package beitragslauf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ExportItem ist eine einzelne Lastschrift im Beitragslauf.
type ExportItem struct {
	MemberID     int
	Name         string
	Street       string
	Zip          string
	City         string
	IBAN         string
	BetragCent   int
	MandatRef    string // = member_number
	MandatDatum  string // YYYY-MM-DD
	MemberNumber string
}

// BuildInput bündelt alle Daten für die XML-Erzeugung.
type BuildInput struct {
	SaisonKurz   string // z.B. "2026/27"
	ClubName     string
	GlaeubigerID string
	ClubIBAN     string
	BIC          string
	Kontoinhaber string
	Faelligkeit  time.Time // 01.07. der Saison (ggf. auf Werktag verschoben)
	CreatedAt    time.Time // für CreDtTm + MsgId-Stempel
	Items        []ExportItem
}

const painNS = "urn:iso:std:iso:20022:tech:xsd:pain.008.001.08"

// --- XML-Strukturen (pain.008.001.08) ---

type document struct {
	XMLName xml.Name `xml:"Document"`
	NS      string   `xml:"xmlns,attr"`
	Content cstmr    `xml:"CstmrDrctDbtInitn"`
}

type cstmr struct {
	GrpHdr grpHdr `xml:"GrpHdr"`
	PmtInf pmtInf `xml:"PmtInf"`
}

type grpHdr struct {
	MsgID    string   `xml:"MsgId"`
	CreDtTm  string   `xml:"CreDtTm"`
	NbOfTxs  int      `xml:"NbOfTxs"`
	CtrlSum  string   `xml:"CtrlSum"`
	InitgPty initgPty `xml:"InitgPty"`
}

type initgPty struct {
	Nm string `xml:"Nm"`
	ID *party `xml:"Id,omitempty"`
}

type party struct {
	OrgID *orgID `xml:"OrgId,omitempty"`
}

type orgID struct {
	Othr othr `xml:"Othr"`
}

type othr struct {
	ID      string   `xml:"Id"`
	SchmeNm *schmeNm `xml:"SchmeNm,omitempty"`
}

type schmeNm struct {
	Prtry string `xml:"Prtry"`
}

type pmtInf struct {
	PmtInfID     string      `xml:"PmtInfId"`
	PmtMtd       string      `xml:"PmtMtd"`
	BtchBookg    bool        `xml:"BtchBookg"`
	NbOfTxs      int         `xml:"NbOfTxs"`
	CtrlSum      string      `xml:"CtrlSum"`
	PmtTpInf     pmtTpInf    `xml:"PmtTpInf"`
	ReqdColltnDt string      `xml:"ReqdColltnDt"`
	Cdtr         partyName   `xml:"Cdtr"`
	CdtrAcct     acct        `xml:"CdtrAcct"`
	CdtrAgt      agtBIC      `xml:"CdtrAgt"`
	ChrgBr       string      `xml:"ChrgBr"`
	CdtrSchmeId  cdtrSchmeId `xml:"CdtrSchmeId"`
	Txs          []txInf     `xml:"DrctDbtTxInf"`
}

type pmtTpInf struct {
	SvcLvl    code   `xml:"SvcLvl"`
	LclInstrm code   `xml:"LclInstrm"`
	SeqTp     string `xml:"SeqTp"`
}

type code struct {
	Cd string `xml:"Cd"`
}

type partyName struct {
	Nm string `xml:"Nm"`
}

type acct struct {
	ID acctID `xml:"Id"`
}

type acctID struct {
	IBAN string `xml:"IBAN"`
}

type agtBIC struct {
	FinInstnID finInstnBIC `xml:"FinInstnId"`
}

type finInstnBIC struct {
	BICFI string `xml:"BICFI,omitempty"`
	Othr  *othr  `xml:"Othr,omitempty"`
}

type cdtrSchmeId struct {
	ID schmeIDWrap `xml:"Id"`
}

type schmeIDWrap struct {
	PrvtID prvtID `xml:"PrvtId"`
}

type prvtID struct {
	Othr othr `xml:"Othr"`
}

type txInf struct {
	PmtID     pmtID     `xml:"PmtId"`
	InstdAmt  amount    `xml:"InstdAmt"`
	DrctDbtTx drctDbtTx `xml:"DrctDbtTx"`
	DbtrAgt   agtBIC    `xml:"DbtrAgt"`
	Dbtr      dbtr      `xml:"Dbtr"`
	DbtrAcct  acct      `xml:"DbtrAcct"`
	RmtInf    rmtInf    `xml:"RmtInf"`
}

type pmtID struct {
	EndToEndID string `xml:"EndToEndId"`
}

type amount struct {
	Ccy string `xml:"Ccy,attr"`
	Val string `xml:",chardata"`
}

type drctDbtTx struct {
	MndtRltdInf mndtRltdInf `xml:"MndtRltdInf"`
}

type mndtRltdInf struct {
	MndtID    string `xml:"MndtId"`
	DtOfSgntr string `xml:"DtOfSgntr"`
}

type dbtr struct {
	Nm      string   `xml:"Nm"`
	PstlAdr *pstlAdr `xml:"PstlAdr,omitempty"`
}

type pstlAdr struct {
	StrtNm string `xml:"StrtNm,omitempty"`
	BldgNb string `xml:"BldgNb,omitempty"`
	PstCd  string `xml:"PstCd,omitempty"`
	TwnNm  string `xml:"TwnNm,omitempty"`
	Ctry   string `xml:"Ctry"`
}

type rmtInf struct {
	Ustrd string `xml:"Ustrd"`
}

// BuildXML erzeugt das pain.008.001.08-Dokument. Alle Lastschriften sind RCUR
// und liegen in genau einem PmtInf-Block.
func BuildXML(in BuildInput) ([]byte, error) {
	var sumCent int
	txs := make([]txInf, 0, len(in.Items))
	for _, it := range in.Items {
		sumCent += it.BetragCent
		strt, bldg := parseStreet(it.Street)
		txs = append(txs, txInf{
			PmtID:     pmtID{EndToEndID: ascii(fmt.Sprintf("TW-%s-%s", it.MemberNumber, saisonStamp(in.SaisonKurz)))},
			InstdAmt:  amount{Ccy: "EUR", Val: euro(it.BetragCent)},
			DrctDbtTx: drctDbtTx{MndtRltdInf: mndtRltdInf{MndtID: it.MandatRef, DtOfSgntr: it.MandatDatum}},
			DbtrAgt:   agtBIC{FinInstnID: finInstnBIC{Othr: &othr{ID: "NOTPROVIDED"}}},
			Dbtr: dbtr{
				Nm:      it.Name,
				PstlAdr: &pstlAdr{StrtNm: strt, BldgNb: bldg, PstCd: it.Zip, TwnNm: it.City, Ctry: "DE"},
			},
			DbtrAcct: acct{ID: acctID{IBAN: it.IBAN}},
			RmtInf:   rmtInf{Ustrd: fmt.Sprintf("Mitgliedsbeitrag Team Stuttgart %s - Mitglied %s", saisonShort(in.SaisonKurz), it.MemberNumber)},
		})
	}

	stamp := in.CreatedAt.Format("20060102150405")
	doc := document{
		NS: painNS,
		Content: cstmr{
			GrpHdr: grpHdr{
				MsgID:    ascii(fmt.Sprintf("TW-%s-%s", saisonStamp(in.SaisonKurz), stamp)),
				CreDtTm:  in.CreatedAt.UTC().Format("2006-01-02T15:04:05"),
				NbOfTxs:  len(txs),
				CtrlSum:  euro(sumCent),
				InitgPty: initgPty{Nm: in.ClubName, ID: &party{OrgID: &orgID{Othr: othr{ID: in.GlaeubigerID}}}},
			},
			PmtInf: pmtInf{
				PmtInfID:     ascii(fmt.Sprintf("TW-%s-RCUR", saisonStamp(in.SaisonKurz))),
				PmtMtd:       "DD",
				BtchBookg:    true,
				NbOfTxs:      len(txs),
				CtrlSum:      euro(sumCent),
				PmtTpInf:     pmtTpInf{SvcLvl: code{Cd: "SEPA"}, LclInstrm: code{Cd: "CORE"}, SeqTp: "RCUR"},
				ReqdColltnDt: in.Faelligkeit.Format("2006-01-02"),
				Cdtr:         partyName{Nm: in.Kontoinhaber},
				CdtrAcct:     acct{ID: acctID{IBAN: in.ClubIBAN}},
				CdtrAgt:      agtBIC{FinInstnID: finInstnBIC{BICFI: in.BIC}},
				ChrgBr:       "SLEV",
				CdtrSchmeId: cdtrSchmeId{ID: schmeIDWrap{PrvtID: prvtID{Othr: othr{
					ID: in.GlaeubigerID, SchmeNm: &schmeNm{Prtry: "SEPA"},
				}}}},
				Txs: txs,
			},
		},
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

var streetRe = regexp.MustCompile(`^(.+?)\s+(\d+\s*[a-zA-Z]?)$`)

// parseStreet trennt "Hauptstr. 12" in StrtNm="Hauptstr." und BldgNb="12".
// Ohne Hausnummern-Match landet alles in StrtNm.
func parseStreet(street string) (strtNm, bldgNb string) {
	street = strings.TrimSpace(street)
	if m := streetRe.FindStringSubmatch(street); m != nil {
		return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	}
	return street, ""
}

// nextBusinessDay verschiebt Sa/So auf den folgenden Montag.
func nextBusinessDay(t time.Time) time.Time {
	switch t.Weekday() {
	case time.Saturday:
		return t.AddDate(0, 0, 2)
	case time.Sunday:
		return t.AddDate(0, 0, 1)
	default:
		return t
	}
}

// euro formatiert Cent als Euro mit Punkt-Dezimaltrenner: 9600 → "96.00".
func euro(cent int) string {
	if cent < 0 {
		cent = -cent
	}
	return fmt.Sprintf("%d.%02d", cent/100, cent%100)
}

// saisonStamp wandelt "2026/27" in "2026-27" (für IDs).
func saisonStamp(s string) string { return strings.ReplaceAll(s, "/", "-") }

// saisonShort kürzt vierstellige Jahresangaben auf zwei Stellen:
// "2026/27" → "26/27", "2026/2027" → "26/27". Andere Formate bleiben unverändert.
func saisonShort(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		if len(p) == 4 && isAllDigits(p) {
			parts[i] = p[2:]
		}
	}
	return strings.Join(parts, "/")
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

var nonASCII = regexp.MustCompile(`[^\x20-\x7E]`)

// ascii ersetzt Umlaute und entfernt sonstige Nicht-ASCII-Zeichen
// (für MsgId/PmtInfId/EndToEndId, die ASCII sein müssen).
func ascii(s string) string {
	r := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue",
		"Ä", "Ae", "Ö", "Oe", "Ü", "Ue",
		"ß", "ss",
	)
	return nonASCII.ReplaceAllString(r.Replace(s), "")
}
