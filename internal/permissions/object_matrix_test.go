// TestObjectAuthorizationMatrix — Objektrechte mechanisch geprüft.
//
// Die Tier-Matrix (matrix_test.go) beweist, dass eine Route im richtigen
// Auth-Tier hängt; sie läuft auf leerer DB mit Pfad-ID 1 und kann Objektrechte
// per Konstruktion nicht sehen. Dieser Test schließt die Lücke: zu jeder Route
// mit Pfadparameter entsteht ein echtes Objekt, das **User B** gehört, und ein
// Unbeteiligter fragt es an: A (Standard/spieler), T (Trainer eines fremden
// Kaders) oder E (Elternteil eines fremden Kindes). Erwartet wird 401, 403 oder
// 404 — nie 2xx.
//
// Vier Klassifikationen, jede Route muss in genau einer stehen:
//   - objectFixtures — echtes Fixture, Erwartung 401/403/404
//   - tierOnly       — schon die Middleware blockt A/T (Vorstand-/Admin-Tier);
//     geprüft wird genau das (403). Fällt das Tier weg, wird der Test rot und
//     verlangt ein echtes Fixture.
//   - openByDesign   — bewusst vereinsweit offen (mit Begründung)
//   - todoFixtures   — noch ohne Fixture (mit Begründung); Befund-Liste im Bericht
//
// Zwei Befund-Listen laufen quer dazu, beide über objectFixtures-Routen:
//   - knownGaps            — heute 2xx auf ein fremdes Objekt (mit Schwere)
//   - validationBeforeAuthz — heute 400, bevor das Objektrecht geprüft wird
//
// Ein Befund-Eintrag macht den Test nicht rot, schlägt aber fehl, sobald sich
// der Status ändert — damit der Eintrag beim Beheben verschwindet statt als
// stille Ausnahme stehenzubleiben.
//
// Neue Route mit Pfadparameter ⇒ eine der vier Klassifikationen ergänzen, sonst
// failt der Drift-Check.
package permissions_test

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// persona wählt den Anfragenden.
type persona int

const (
	// personaA: Standard-Rolle, Vereinsfunktion `spieler`, eigener Kader — der
	// Normalfall im Authenticated-Tier.
	personaA persona = iota
	// personaT: Standard-Rolle, Vereinsfunktion `trainer`, aber Trainer eines
	// FREMDEN Kaders — nötig, damit Routen im Trainer-/Vorstand+Trainer-Tier den
	// Objekt-Check im Handler überhaupt erreichen.
	personaT
	// personaE: Standard-Rolle mit `IsParent`, Elternteil eines EIGENEN Kindes —
	// für die Kind-Routen. A (kein Elternteil) bewiese dort nur die schwächere
	// Aussage; E prüft „Elternteil, aber nicht dieses Kindes".
	personaE
)

func (p persona) String() string {
	switch p {
	case personaT:
		return "T (fremder Trainer)"
	case personaE:
		return "E (Elternteil eines fremden Kindes)"
	case personaA:
		return "A (Standard/spieler)"
	default:
		return "unbekannte Persona"
	}
}

// objFixture beschreibt, wie das fremde Objekt für eine Route entsteht.
type objFixture struct {
	who persona
	// params liefert die Pfadparameter (ohne geschweifte Klammern).
	params func(w *objWorld) map[string]string
	// body überschreibt den Default-Body ({} bei POST/PUT/PATCH).
	body any
	// bodyFn ist body für Fälle, die eine echte ID aus der objWorld brauchen
	// (z. B. newOwnerId). Hat Vorrang vor body.
	bodyFn func(w *objWorld) any
	// noOwnerProbe unterdrückt die Gegenprobe „Eigentümer B bekommt kein 404".
	// Nötig, wenn die Route auch für B nicht 2xx liefern kann (Token im Query,
	// Datei fehlt auf Platte, Tier-Gate greift auch für B).
	noOwnerProbe bool
}

// objGap hält einen Befund fest: erwarteter (= heute beobachteter) Status.
type objGap struct {
	status int
	note   string
}

// ── Bewusst offene Routen ────────────────────────────────────────────────────

var openByDesign = map[string]string{
	"POST /api/duty-board/{slotId}/claim":                "Dienstbörse ist vereinsweit offen — jeder Eingeloggte darf jeden freien Slot übernehmen; das ist der Zweck der Börse, kein Objektrecht.",
	"GET /api/duty-types/{id}/instruction":               "Dienst-Anleitung ist bewusst für alle Eingeloggten lesbar (das Board verlinkt sie); der Diensttyp ist Vereinsstammdatum ohne Personenbezug.",
	"GET /api/duty-slots/{id}/assignments":               "Die Dienstbörse zeigt die Belegung eines Slots allen Eingeloggten — sonst wäre nicht erkennbar, ob noch Plätze frei sind.",
	"GET /api/ausrichter/{id}/usage":                     "Ausrichter sind Vereinsstammdaten ohne Personenbezug; die Verwendungsliste braucht der Termin-Wizard für alle Eingeloggten.",
	"GET /api/game-days/{date}/host":                     "Der Ausrichter eines Spieltags ist Vereinsstammdatum; {date} ist kein Objekt-Identifier, sondern ein Datum.",
	"OPTIONS /api/videos/{id}/hls/master.m3u8":           "CORS-Preflight läuft per Definition ohne Auth (HLSPreflight); die Auslieferung selbst hängt am ?st=-Stream-Token.",
	"OPTIONS /api/videos/{id}/hls/{rendition}/{segment}": "CORS-Preflight läuft per Definition ohne Auth (HLSPreflight); die Auslieferung selbst hängt am ?st=-Stream-Token.",

	// Vereinsstammdaten ohne Eigentümer: das Tier IST die Berechtigung.
	"PUT /api/venues/{id}":                            "Veranstaltungsorte sind Vereinsstammdaten ohne Eigentümer; CRUD liegt laut Auth-Tier-Tabelle (docs/agent/04-api-db.md) bei Vorstand + Trainer/sportliche Leitung. Ein Objektrecht existiert nicht — jeder Trainer pflegt dieselbe Hallenliste.",
	"DELETE /api/venues/{id}":                         "Wie PUT: Hallenliste ist vereinsweites Stammdatum (Auth-Tier Vorstand + Trainer/sL), kein personenbezogenes Objekt.",
	"GET /api/duty-templates/{id}":                    "Dienstvorlagen sind Vereinsstammdaten; der Lesezugriff hängt bewusst am Tier „Vorstand + Trainer + sportliche Leitung (read-only)\" (internal/app/router.go), damit jeder Planende die Vorlage eines Spieltags nachsehen kann.",
	"GET /api/duty-templates/{id}/preview":            "Vorschau derselben Stammdaten-Vorlage gegen ein Datum — rein rechnend, ohne Personenbezug und ohne Schreibwirkung.",
	"GET /api/kader/{id}":                             "Trainer und sportliche Leitung sehen bewusst alle Kader: Spieler werden über Mannschaftsgrenzen hinweg zugeordnet (Aushilfe, erweiterter Kader, Übungsgruppen). Der Kader selbst trägt keine Kontaktdaten.",
	"GET /api/kader/{id}/member-suggestions":          "Vorschlagsliste für die Kaderpflege — vereinsweit gewollt, sonst könnte niemand einen noch nicht zugeordneten Spieler finden. Gleiches Tier wie GET /api/kader/{id}.",
	"GET /api/kader/{id}/extended-member-suggestions": "Wie member-suggestions, nur für den erweiterten Kader (Förderkinder aus anderen Mannschaften) — die Zielgruppe liegt per Definition außerhalb des eigenen Kaders.",
	"GET /api/users/{id}/contact":                     "Das Kontaktprofil ist vereinsweit lesbar und pro Feld opt-in: openspec/specs/person-contact/spec.md legt fest, dass jeder Eingeloggte Name + die selbst freigegebenen Felder (Telefon/Adresse/E-Mail/Foto) sieht. Ohne Freigabe kommt nur der Name.",
	"POST /api/chat/broadcasts/{id}/read":             "Die Route markiert ausschließlich die eigene Lese-Zeile (UPDATE broadcast_reads … WHERE user_id = <self>) und legt keine an. Für einen Nicht-Empfänger trifft das UPDATE keine Zeile: 204 ohne Wirkung, ohne Lesezugriff auf die Mitteilung.",
	"POST /api/duty-assignments/{id}/fulfill":         "Das Abhaken eines Dienstes ist ein Rollenrecht, kein Objektrecht: policy.CanFulfillAssignment lässt jeden Trainer/sportliche Leitung quittieren — wer am Spieltag vor Ort ist, bestätigt, nicht der Trainer der eingeteilten Mannschaft. Welle 1 hat die Route bewusst auf „Rolle + RowsAffected\" gehärtet (docs/agent/06-gotchas.md).",
	"POST /api/duty-assignments/{id}/cash-substitute": "Gegenstück zu fulfill: Ablösesumme statt Dienst, gleiche Rollenlogik (policy.CanFulfillAssignment), gleiche Welle-1-Härtung.",
}

// ── Noch ohne Fixture ────────────────────────────────────────────────────────

var todoFixtures = map[string]string{}

// ── Befunde Welle 2 ──────────────────────────────────────────────────────────
//
// Jeder Eintrag ist ein offener Befund, kein Freibrief: der Test prüft den
// festgehaltenen Status exakt. Sobald die Route 403/404 liefert, wird er rot —
// dann gehört der Eintrag gelöscht.
//
// Schwere: hoch = fremde Personen-/Stammdaten werden verändert · mittel =
// fremde Planungsdaten werden verändert oder gelöscht · niedrig = Wirkung auf
// ein Kennzeichen ohne Personenbezug.
var knownGaps = map[string]objGap{
	// ── Kader (fremder Trainer schreibt auf fremden Kader) ───────────────
	"PUT /api/kader/{id}":                    {http.StatusNoContent, "hoch — Befund Welle 2, offen: UpdateKader prüft nur das Tier (Vorstand/Trainer/sL). Ein fremder Trainer fügt Mitglieder und Trainer eines fremden Kaders hinzu oder entfernt sie; über kader_trainers verschafft er sich damit selbst Zugriff auf dessen Trainings, Anwesenheit und Videos."},
	"DELETE /api/kader/{id}":                 {http.StatusNoContent, "hoch — Befund Welle 2, offen: DeleteKader prüft nur die Mitgliederzahl (409 bei besetztem Kader), nicht die Zugehörigkeit. Ein leerer fremder Kader — der Normalfall zu Saisonbeginn — lässt sich von jedem Trainer löschen."},
	"PATCH /api/kader/{id}/games-per-season": {http.StatusNoContent, "mittel — Befund Welle 2, offen: kein Objekt-Check; die Spielzahl je Saison geht in die Dienst-Soll-Rechnung eines fremden Kaders ein."},

	// ── Änderungsanträge (Vorstand+Trainer+sL-Tier, ohne Objektbezug) ────
	"POST /api/members/{id}/change-drafts/{draftId}/accept": {http.StatusOK, "hoch — Befund Welle 2, offen: AcceptChangeRequestHandler autorisiert nur über das Tier. Ein fremder Trainer übernimmt den Änderungsantrag eines beliebigen Mitglieds (Name/Adresse/DSGVO/SEPA) in dessen Stammdaten — die {id} im Pfad wird dabei nicht einmal gegen den Entwurf geprüft."},
	"DELETE /api/members/{id}/change-drafts/{draftId}":      {http.StatusOK, "mittel — Befund Welle 2, offen: Gegenstück zum Accept; ein fremder Trainer verwirft den Änderungsantrag eines beliebigen Mitglieds, der Antragsteller bekommt die Ablehnung."},

	// ── Dienst-Slots (team-gebunden, aber nur Tier-geprüft) ──────────────
	"PUT /api/duty-slots/{id}":    {http.StatusNoContent, "niedrig — Befund Welle 2, offen: UpdateSlot prüft nur das Tier. Ein fremder Trainer überschreibt Zeiten, Platzzahl und Stundenwert eines Dienst-Slots einer anderen Mannschaft (und setzt ihn dabei auf is_custom=1, womit ihn der nächste Regen-Lauf nicht mehr korrigiert)."},
	"DELETE /api/duty-slots/{id}": {http.StatusNoContent, "mittel — Befund Welle 2, offen: DeleteSlot prüft nur das Tier. Ein fremder Trainer löscht den Dienst einer anderen Mannschaft samt Zuweisungen; die eingeteilten Personen bekommen die Absage-Meldung."},

	// ── Anwesenheits-Kennzeichen und Regen (nur Tier-geprüft) ───────────
	"POST /api/games/{id}/attendance-excluded":               {http.StatusNoContent, "niedrig — Befund Welle 2, offen: SetGameExcluded prüft nur das Tier und die RowsAffected. Ein fremder Trainer nimmt ein Spiel einer anderen Mannschaft aus der Anwesenheitsstatistik."},
	"DELETE /api/games/{id}/attendance-excluded":             {http.StatusNoContent, "niedrig — Befund Welle 2, offen: Gegenrichtung derselben Route (derselbe Handler, val=0)."},
	"POST /api/training-sessions/{id}/attendance-excluded":   {http.StatusNoContent, "niedrig — Befund Welle 2, offen: SetTrainingExcluded prüft nur das Tier; ein fremder Trainer nimmt einen Trainingstermin einer anderen Mannschaft aus der Statistik."},
	"DELETE /api/training-sessions/{id}/attendance-excluded": {http.StatusNoContent, "niedrig — Befund Welle 2, offen: Gegenrichtung derselben Route."},
	"POST /api/games/{id}/regenerate":                        {http.StatusOK, "mittel — Befund Welle 2, offen: RegenerateSlots prüft nur das Tier und startet für ein fremdes Spiel den vollen Regen-Lauf über den Spieltag (±1 Tag) — Slots anderer Mannschaften entstehen, verschwinden und lösen Absage-Meldungen aus."},
}

// ── Validierung vor Autorisierung ────────────────────────────────────────────
//
// Routen, die einen fremden Objektzugriff mit 400 quittieren, bevor sie das
// Objektrecht geprüft haben. Das ist kein Bestehen: der Statuscode beweist nur,
// dass der Body nicht passte. Der Test erwartet für diese Einträge exakt 400 —
// verschwindet der Befund (403/404) oder ändert er sich, wird er rot.
//
// Die Liste ist derzeit leer, und das ist das Ergebnis, nicht der Ausgangspunkt:
// alle sechs Kandidaten aus dem ersten Lauf (chat-members, transfer-ownership,
// kind/recovery-email, training-sessions/respond, training-series PUT und
// dessen unavailabilities) waren Fixture-Fehler — mit einem minimal gültigen
// Body erreichen sie das Objektrecht und antworten 403. Wer hier einen Eintrag
// ergänzt, muss vorher belegen, dass es keinen gültigen Body gibt, der die
// Autorisierung erreicht.
var validationBeforeAuthz = map[string]string{}

// ── Tier-geschützte Routen ───────────────────────────────────────────────────
//
// A bzw. T erreichen den Handler nicht; geprüft wird, dass die Middleware mit
// 403 blockt. Entfällt das Tier-Gate, wird der Test rot und verlangt ein
// echtes Objekt-Fixture.
var tierOnly = map[string]string{
	// Vorstand
	"GET /api/practice-groups/{id}":                    "Vorstand-Tier",
	"PUT /api/practice-groups/{id}":                    "Vorstand-Tier",
	"DELETE /api/practice-groups/{id}":                 "Vorstand-Tier",
	"GET /api/practice-groups/{id}/member-suggestions": "Vorstand-Tier",
	"PUT /api/members/{id}":                            "Vorstand-Tier",
	"PUT /api/members/{id}/status":                     "Vorstand-Tier",
	"DELETE /api/members/{id}":                         "Vorstand-Tier",
	"PUT /api/members/{id}/user":                       "Vorstand-Tier",
	"POST /api/members/{id}/proxy-account":             "Vorstand-Tier",
	"POST /api/members/{id}/welcome-email":             "Vorstand-Tier",
	"POST /api/users/{id}/create-member":               "Vorstand-Tier",
	"PUT /api/seasons/{id}":                            "Vorstand-Tier",
	"PUT /api/seasons/{id}/activate":                   "Vorstand-Tier",
	"DELETE /api/seasons/{id}":                         "Vorstand-Tier",
	"PUT /api/seasons/{id}/duty-targets":               "Vorstand-Tier",
	"PUT /api/teams/{id}":                              "Vorstand-Tier",
	"PUT /api/users/{id}":                              "Vorstand-Tier",
	"PUT /api/users/{id}/role":                         "Vorstand-Tier",
	"PUT /api/users/{id}/recovery-email":               "Vorstand-Tier",
	"DELETE /api/users/{id}":                           "Vorstand-Tier",
	"DELETE /api/invitations/{id}":                     "Vorstand-Tier",
	"POST /api/invitations/{id}/send":                  "Vorstand-Tier",
	"PUT /api/invitations/{id}/member":                 "Vorstand-Tier",
	"POST /api/membership-requests/{id}/approve":       "Vorstand-Tier",
	"POST /api/membership-requests/{id}/reject":        "Vorstand-Tier",
	"DELETE /api/membership-requests/{id}":             "Vorstand-Tier",
	"PUT /api/duty-types/{id}":                         "Vorstand-Tier",
	"PUT /api/duty-types/{id}/instruction":             "Vorstand-Tier",
	"DELETE /api/duty-types/{id}":                      "Vorstand-Tier",
	"PUT /api/ausrichter/{id}":                         "Vorstand-Tier",
	"DELETE /api/ausrichter/{id}":                      "Vorstand-Tier",
	"PUT /api/duty-templates/{id}":                     "Vorstand-Tier",
	"DELETE /api/duty-templates/{id}":                  "Vorstand-Tier",
	"POST /api/upload/member-photo/{id}":               "Vorstand-Tier",
	"DELETE /api/upload/member-photo/{id}":             "Vorstand-Tier",
	"PUT /api/age-class-rules/{ageClass}":              "Vorstand-Tier (Stammdaten, kein Objektbesitz)",
	"DELETE /api/training-group-categories/{name}":     "Vorstand-Tier (Stammdaten, kein Objektbesitz)",
	"PUT /api/stammvereine/{id}":                       "Vorstand-Tier (Stammdaten, kein Objektbesitz)",
	"DELETE /api/stammvereine/{id}":                    "Vorstand-Tier (Stammdaten, kein Objektbesitz)",

	// Vorstand + Kassierer
	"GET /api/members/{id}":              "Vorstand+Kassierer-Tier",
	"GET /api/members/{id}/parents":      "Vorstand+Kassierer-Tier",
	"PUT /api/members/{id}/bank-details": "Vorstand+Kassierer-Tier (Bank-PII)",
	"POST /api/upload/sepa-mandat/{id}":  "Vorstand+Kassierer-Tier (Bank-PII)",
	"DELETE /api/fee-rates/{id}":         "Vorstand+Kassierer-Tier",

	// Admin
	"POST /api/impersonate/{id}": "Admin-Tier",

	// Match-Report-Freigeber (medien/vorstand)
	"POST /api/match-reports/{id}/publish": "Freigeber-Tier (medien/vorstand)",
}

// ── Fixture-Katalog ──────────────────────────────────────────────────────────

// objectFixtures ist die eigentliche Matrix: Route → fremdes Objekt.
// Der Katalog wird als Funktion gebaut, weil die Fixtures die objWorld brauchen.
func objectFixtures() map[string]objFixture {
	// Kurzschreibweisen für die häufigen Parameterformen.
	p := func(k string, v int) map[string]string { return map[string]string{k: strconv.Itoa(v)} }

	m := map[string]objFixture{}
	add := func(key string, f objFixture) {
		if _, dup := m[key]; dup {
			panic("doppelter Fixture-Eintrag: " + key)
		}
		m[key] = f
	}

	// ── Chat: Konversationen ─────────────────────────────────────────────────
	convFixture := func(w *objWorld) map[string]string { return p("id", w.newConversation()) }
	add("GET /api/chat/conversations/{id}/messages", objFixture{params: convFixture})
	add("POST /api/chat/conversations/{id}/messages", objFixture{params: convFixture, body: map[string]any{"body": "Hallo"}})
	add("POST /api/chat/conversations/{id}/read", objFixture{params: convFixture, noOwnerProbe: true})
	add("DELETE /api/chat/conversations/{id}/members/me", objFixture{params: convFixture})
	add("DELETE /api/chat/conversations/{id}/everyone", objFixture{params: convFixture})
	add("PUT /api/chat/conversations/{id}", objFixture{params: convFixture, body: map[string]any{"name": "Neu"}})
	add("DELETE /api/chat/conversations/{id}", objFixture{params: convFixture})
	add("POST /api/chat/conversations/{id}/members", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.newConversation()) },
		// AddMember verlangt ein nicht-leeres userId — mit 0 endet die Route im
		// 400 der Body-Prüfung und erreicht das Objektrecht nie.
		bodyFn: func(w *objWorld) any { return map[string]any{"userId": w.aUserID} },
	})
	add("DELETE /api/chat/conversations/{id}/members/{uid}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.newConversation()), "uid": strconv.Itoa(w.bUserID)}
		},
	})
	add("POST /api/chat/conversations/{id}/transfer-ownership", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.newConversation()) },
		// Feldname ist newOwnerId (nicht userId) und muss ≠ 0 sein.
		bodyFn: func(w *objWorld) any { return map[string]any{"newOwnerId": w.aUserID} },
	})
	add("POST /api/chat/conversations/{id}/polls", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.newConversation()) },
		body:   map[string]any{"question": "Frage?", "options": []string{"Ja", "Nein"}},
	})

	// ── Chat: Nachrichten ────────────────────────────────────────────────────
	msgFixture := func(w *objWorld) map[string]string {
		_, msgID := w.newMessage()
		return p("id", msgID)
	}
	add("GET /api/chat/messages/{id}", objFixture{params: msgFixture})
	add("GET /api/chat/messages/{id}/reads", objFixture{params: msgFixture, noOwnerProbe: true})
	add("PUT /api/chat/messages/{id}", objFixture{params: msgFixture, body: map[string]any{"body": "Geändert"}})
	add("DELETE /api/chat/messages/{id}", objFixture{params: msgFixture})
	add("POST /api/chat/messages/{id}/reactions", objFixture{params: msgFixture, body: map[string]any{"emoji": "👍"}})

	pollFixture := func(w *objWorld) map[string]string {
		msgID, _ := w.newPoll()
		return p("id", msgID)
	}
	add("GET /api/chat/messages/{id}/poll", objFixture{params: pollFixture})
	add("PUT /api/chat/messages/{id}/poll/vote", objFixture{
		params: func(w *objWorld) map[string]string {
			msgID, optID := w.newPoll()
			return map[string]string{"id": strconv.Itoa(msgID), "__opt": strconv.Itoa(optID)}
		},
		body: map[string]any{"optionIds": []int{}},
	})
	add("POST /api/chat/messages/{id}/poll/close", objFixture{params: pollFixture})

	// ── Chat: Mitteilungen (Broadcasts) ──────────────────────────────────────
	bcFixture := func(w *objWorld) map[string]string { return p("id", w.newBroadcast()) }
	add("POST /api/chat/broadcasts/{id}/read", objFixture{params: bcFixture, noOwnerProbe: true})
	add("PUT /api/chat/broadcasts/{id}", objFixture{params: bcFixture, body: map[string]any{"body": "Geändert"}})
	add("DELETE /api/chat/broadcasts/{id}", objFixture{params: bcFixture})

	// ── Chat: Gruppen-Auflösung ──────────────────────────────────────────────
	add("GET /api/chat/team-groups/{teamId}/{kind}/members", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"teamId": strconv.Itoa(w.bTeamID), "kind": "spieler"}
		},
	})
	add("GET /api/chat/practice-groups/{id}/{kind}/members", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bPracticeGroupID), "kind": "spieler"}
		},
	})

	// ── Media ────────────────────────────────────────────────────────────────
	add("GET /api/media/{id}", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.newMedia()) },
		noOwnerProbe: true, // Datei liegt nicht auf Platte — B bekommt 500, nicht 2xx.
	})

	// ── Mitglieder / Profil ──────────────────────────────────────────────────
	add("GET /api/users/{id}/contact", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bUserID) },
	})
	add("GET /api/members/{id}/change-drafts", objFixture{
		params: func(w *objWorld) map[string]string {
			memberID, _ := w.newChangeDraft()
			return p("id", memberID)
		},
	})
	add("POST /api/members/{id}/change-request", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
		body:   map[string]any{"field_name": "street", "new_value": "Neue Straße 1"},
	})
	add("GET /api/members/{id}/sepa-mandat/download-token", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
		noOwnerProbe: true, // B hat kein Mandat auf Platte.
	})
	add("DELETE /api/members/{id}/sepa-mandat", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
	})
	add("GET /api/members/{id}/sepa-mandat/download", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
		noOwnerProbe: true, // Public-Tier, Zugang nur über den Download-Token im Query.
	})

	phoneFixture := func(w *objWorld) map[string]string { return p("id", w.newPhone(w.bMemberID)) }
	add("PUT /api/profile/phones/{id}", objFixture{params: phoneFixture, body: map[string]any{"label": "privat", "number": "0711999"}})
	add("DELETE /api/profile/phones/{id}", objFixture{params: phoneFixture})

	add("PUT /api/members/{id}/cross-team-visible", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
		body:   map[string]any{"visible": true},
	})
	add("PUT /api/members/{id}/chat-visible", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
		body:   map[string]any{"visible": true},
	})

	// ── Kinder-Profile (family_links von B) ──────────────────────────────────
	//
	// Anfragender ist E: selbst Elternteil, aber eines anderen Kindes. Damit
	// prüft der 403 die Kind-Bindung (family_links), nicht nur die Eltern-Rolle.
	childFixture := func(w *objWorld) map[string]string { return p("memberId", w.bChildMemberID) }
	add("GET /api/profile/kind/{memberId}", objFixture{who: personaE, params: childFixture})
	add("PUT /api/profile/kind/{memberId}/account", objFixture{who: personaE, params: childFixture})
	add("PUT /api/profile/kind/{memberId}/member", objFixture{who: personaE, params: childFixture})
	add("PUT /api/profile/kind/{memberId}/bank", objFixture{who: personaE, params: childFixture})
	add("POST /api/profile/kind/{memberId}/photo", objFixture{who: personaE, params: childFixture})
	add("DELETE /api/profile/kind/{memberId}/photo", objFixture{who: personaE, params: childFixture})
	add("POST /api/profile/kind/{memberId}/phones", objFixture{who: personaE, params: childFixture, body: map[string]any{"label": "privat", "number": "0711888"}})
	add("DELETE /api/profile/kind/{memberId}/phones/{phoneId}", objFixture{
		who: personaE,
		params: func(w *objWorld) map[string]string {
			return map[string]string{
				"memberId": strconv.Itoa(w.bChildMemberID),
				"phoneId":  strconv.Itoa(w.newPhone(w.bChildMemberID)),
			}
		},
	})
	// Feldname ist new_email; „email" liefe in die 400-Validierung vor dem
	// Eltern-Check.
	add("POST /api/profile/kind/{memberId}/recovery-email", objFixture{who: personaE, params: childFixture, body: map[string]any{"new_email": "kind@test.local"}})
	add("PUT /api/profile/kind/{memberId}/visibility", objFixture{who: personaE, params: childFixture})
	add("GET /api/profile/kind/{memberId}/calendar-token", objFixture{who: personaE, params: childFixture, noOwnerProbe: true})
	add("POST /api/profile/kind/{memberId}/calendar-token", objFixture{who: personaE, params: childFixture})
	add("DELETE /api/profile/kind/{memberId}/calendar-token", objFixture{
		who: personaE,
		params: func(w *objWorld) map[string]string {
			w.newCalendarToken(w.bChildUserID)
			return p("memberId", w.bChildMemberID)
		},
	})

	// ── Kalender-Feed (Token im Pfad) ────────────────────────────────────────
	add("GET /api/calendar/feed/{token}", objFixture{
		params: func(w *objWorld) map[string]string {
			w.newCalendarToken(w.bUserID)
			// A kennt B's Token nicht — geraten wird ein falscher.
			return map[string]string{"token": "geratener-fremder-token"}
		},
		noOwnerProbe: true,
	})

	// ── Dienste ──────────────────────────────────────────────────────────────
	add("DELETE /api/duty-board/{slotId}/claim", objFixture{
		params: func(w *objWorld) map[string]string {
			slotID, _ := w.newDutyAssignment()
			return p("slotId", slotID)
		},
		noOwnerProbe: true,
	})
	add("POST /api/duty-assignments/{id}/fulfill", objFixture{
		who: personaT,
		params: func(w *objWorld) map[string]string {
			_, assignID := w.newDutyAssignment()
			return p("id", assignID)
		},
		noOwnerProbe: true,
	})
	add("POST /api/duty-assignments/{id}/cash-substitute", objFixture{
		who: personaT,
		params: func(w *objWorld) map[string]string {
			_, assignID := w.newDutyAssignment()
			return p("id", assignID)
		},
		body:         map[string]any{"amount": 10},
		noOwnerProbe: true,
	})
	add("PUT /api/duty-slots/{id}", objFixture{
		who:    personaT,
		params: func(w *objWorld) map[string]string { return p("id", w.newDutySlot()) },
		// hours_value > 0 ist Pflicht — ohne den Wert endet die Route im 400
		// und der fehlende Objekt-Check bliebe unsichtbar.
		body: map[string]any{
			"event_name": "Dienst", "event_date": "2026-03-14", "event_time": "10:00",
			"role_desc": "Kasse", "slots_total": 1, "hours_value": 2,
		},
		noOwnerProbe: true,
	})
	add("DELETE /api/duty-slots/{id}", objFixture{
		who:          personaT,
		params:       func(w *objWorld) map[string]string { return p("id", w.newDutySlot()) },
		noOwnerProbe: true,
	})

	// ── Mitfahrgelegenheiten ─────────────────────────────────────────────────
	add("DELETE /api/mitfahrgelegenheiten/{id}", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.newCarpoolOffer()) },
		noOwnerProbe: true,
	})
	add("POST /api/mitfahrt-paarungen/{id}/confirm", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.newCarpoolPairing()) },
		noOwnerProbe: true,
	})
	add("POST /api/mitfahrt-paarungen/{id}/reject", objFixture{
		params:       func(w *objWorld) map[string]string { return p("id", w.newCarpoolPairing()) },
		noOwnerProbe: true,
	})

	// ── Abwesenheiten ────────────────────────────────────────────────────────
	absFixture := func(w *objWorld) map[string]string { return p("id", w.newAbsence()) }
	add("PUT /api/absences/{id}", objFixture{
		params: absFixture,
		body:   map[string]any{"type": "vacation", "start_date": "2026-03-02", "end_date": "2026-03-09"},
	})
	add("DELETE /api/absences/{id}", objFixture{params: absFixture})

	// ── Dokumente ────────────────────────────────────────────────────────────
	folderFixture := func(w *objWorld) map[string]string { return p("id", w.newFolder()) }
	add("GET /api/folders/{id}/contents", objFixture{params: folderFixture})
	add("PUT /api/folders/{id}", objFixture{params: folderFixture, body: map[string]any{"name": "Neu"}})
	add("DELETE /api/folders/{id}", objFixture{params: folderFixture})
	add("GET /api/folders/{id}/permissions", objFixture{params: folderFixture})
	add("POST /api/folders/{id}/permissions", objFixture{
		params: folderFixture,
		body:   map[string]any{"principal_type": "everyone", "can_read": true, "can_write": false},
	})
	add("DELETE /api/folders/{id}/permissions/{permId}", objFixture{
		params: func(w *objWorld) map[string]string {
			folderID := w.newFolder()
			return map[string]string{
				"id":     strconv.Itoa(folderID),
				"permId": strconv.Itoa(w.newFolderPermission(folderID)),
			}
		},
	})
	add("POST /api/folders/{folderId}/files", objFixture{
		params: func(w *objWorld) map[string]string { return p("folderId", w.newFolder()) },
	})
	fileFixture := func(w *objWorld) map[string]string {
		_, fileID := w.newFile()
		return p("id", fileID)
	}
	add("GET /api/files/{id}/download-token", objFixture{params: fileFixture})
	add("PUT /api/files/{id}", objFixture{params: fileFixture, body: map[string]any{"name": "neu.pdf"}})
	add("DELETE /api/files/{id}", objFixture{params: fileFixture})
	add("GET /api/files/{id}/download", objFixture{
		params:       fileFixture,
		noOwnerProbe: true, // Public-Tier, Zugang nur über den Download-Token im Query.
	})

	// ── Spiele ───────────────────────────────────────────────────────────────
	gameFixture := func(w *objWorld) map[string]string { return p("id", w.newGame()) }
	add("GET /api/games/{id}", objFixture{params: gameFixture})
	add("POST /api/games/{id}/respond", objFixture{params: gameFixture, body: map[string]any{"status": "zu"}})
	add("GET /api/games/{id}/responses", objFixture{params: gameFixture})
	add("GET /api/games/{id}/participants", objFixture{params: gameFixture})
	add("GET /api/games/{id}/attendances", objFixture{params: gameFixture})
	add("POST /api/games/{id}/lineup", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("POST /api/games/{id}/attendances", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("DELETE /api/games/{id}/attendance-tracking", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("POST /api/games/{id}/attendance-excluded", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("DELETE /api/games/{id}/attendance-excluded", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("PUT /api/games/{id}", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("PUT /api/games/{id}/note", objFixture{who: personaT, params: gameFixture, body: map[string]any{"note": "x"}, noOwnerProbe: true})
	add("DELETE /api/games/{id}", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})
	add("POST /api/games/{id}/regenerate", objFixture{who: personaT, params: gameFixture, noOwnerProbe: true})

	// ── Anwesenheits-Statistik ───────────────────────────────────────────────
	add("GET /api/teams/{id}/attendance-stats", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bTeamID) },
	})
	add("GET /api/teams/{id}/attendance-open", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bTeamID) },
	})
	add("GET /api/members/{id}/attendance-stats", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
	})

	// ── Trainingstagebuch ────────────────────────────────────────────────────
	diaryFixture := func(w *objWorld) map[string]string { return p("id", w.newDiaryEntry()) }
	add("PUT /api/training-diary/{id}", objFixture{
		params: diaryFixture,
		body:   map[string]any{"trained_on": "2026-03-02", "kind": "kraft", "duration_min": 30, "rpe": 4},
	})
	add("DELETE /api/training-diary/{id}", objFixture{params: diaryFixture})
	add("POST /api/training-diary/{id}/proof", objFixture{params: diaryFixture, noOwnerProbe: true})
	add("DELETE /api/training-diary/{id}/proof", objFixture{params: diaryFixture})
	add("GET /api/training-diary/{id}/proof", objFixture{params: diaryFixture, noOwnerProbe: true})
	add("GET /api/members/{id}/training-diary", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bMemberID) },
	})
	add("GET /api/teams/{id}/training-diary-stats", objFixture{
		params: func(w *objWorld) map[string]string { return p("id", w.bTeamID) },
	})

	// ── Trainings ────────────────────────────────────────────────────────────
	sessionFixture := func(w *objWorld) map[string]string { return p("id", w.newTrainingSession()) }
	add("GET /api/training-sessions/{id}", objFixture{params: sessionFixture})
	// RSVP-Status ist confirmed/declined/maybe — „zu" endet in der Validierung.
	add("POST /api/training-sessions/{id}/respond", objFixture{params: sessionFixture, body: map[string]any{"status": "confirmed"}})
	add("GET /api/training-sessions/{id}/attendances", objFixture{params: sessionFixture})
	add("PUT /api/training-sessions/{id}", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("DELETE /api/training-sessions/{id}", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("POST /api/training-sessions/{id}/attendances", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("DELETE /api/training-sessions/{id}/attendance-tracking", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("POST /api/training-sessions/{id}/attendance-excluded", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("DELETE /api/training-sessions/{id}/attendance-excluded", objFixture{who: personaT, params: sessionFixture, noOwnerProbe: true})
	add("PUT /api/trainings/{id}/note", objFixture{who: personaT, params: sessionFixture, body: map[string]any{"note": "x"}, noOwnerProbe: true})

	seriesFixture := func(w *objWorld) map[string]string { return p("id", w.newTrainingSeries()) }
	// scope ist Pflichtfeld (all|this_and_following) und wird vor dem
	// Kader-Gate geprüft.
	add("PUT /api/training-series/{id}", objFixture{who: personaT, params: seriesFixture, body: map[string]any{"scope": "all"}, noOwnerProbe: true})
	add("DELETE /api/training-series/{id}", objFixture{who: personaT, params: seriesFixture, noOwnerProbe: true})
	add("GET /api/training-series/{id}/unavailabilities", objFixture{who: personaT, params: seriesFixture, noOwnerProbe: true})
	// member_id > 0 ist Pflicht; das Kader-Gate sitzt dahinter.
	add("POST /api/training-series/{id}/unavailabilities", objFixture{
		who:    personaT,
		params: seriesFixture,
		bodyFn: func(w *objWorld) any {
			return map[string]any{"member_id": w.bMemberID, "reason": "krank"}
		},
		noOwnerProbe: true,
	})
	add("DELETE /api/training-series/{id}/unavailabilities/{uid}", objFixture{
		who: personaT,
		params: func(w *objWorld) map[string]string {
			seriesID := w.newTrainingSeries()
			unavailID := testutil.CreateSeriesUnavailability(w.t, w.db, w.bMemberID, seriesID, "", "", "krank", w.bUserID)
			return map[string]string{"id": strconv.Itoa(seriesID), "uid": strconv.Itoa(unavailID)}
		},
		noOwnerProbe: true,
	})

	// ── Mannschaft (team-interne Flächen) ────────────────────────────────────
	teamFixture := func(w *objWorld) map[string]string { return p("id", w.bTeamID) }
	add("GET /api/teams/{id}/roster", objFixture{params: teamFixture})
	add("GET /api/teams/{id}/responsibility-types", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/responsibility-types", objFixture{params: teamFixture, body: map[string]any{"label": "Bälle"}, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/responsibility-types/{typeId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "typeId": strconv.Itoa(w.newResponsibilityType())}
		},
		noOwnerProbe: true,
	})
	add("POST /api/teams/{id}/responsibilities", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/responsibilities/{respId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "respId": strconv.Itoa(w.newResponsibility())}
		},
		noOwnerProbe: true,
	})
	add("GET /api/teams/{id}/penalties", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/penalties", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/penalties", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/penalties/{penaltyId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "penaltyId": strconv.Itoa(w.newPenalty())}
		},
		noOwnerProbe: true,
	})
	add("GET /api/teams/{id}/penalty-types", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/penalty-types", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/penalty-types/{typeId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "typeId": strconv.Itoa(w.newPenaltyType())}
		},
		noOwnerProbe: true,
	})
	add("GET /api/teams/{id}/penalty-wardens", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/penalty-wardens", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/penalty-wardens/{memberId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "memberId": strconv.Itoa(w.newStrafenwart())}
		},
		noOwnerProbe: true,
	})
	add("GET /api/teams/{id}/penalty-settings", objFixture{params: teamFixture, noOwnerProbe: true})
	add("GET /api/teams/{id}/penalty-settings/preview", objFixture{params: teamFixture, noOwnerProbe: true})
	add("PUT /api/teams/{id}/penalty-settings", objFixture{params: teamFixture, body: map[string]any{"unit": "striche"}, noOwnerProbe: true})
	add("GET /api/teams/{id}/cashbook", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/cashbook", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/cashbook/{entryId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "entryId": strconv.Itoa(w.newCashbookEntry())}
		},
		noOwnerProbe: true,
	})
	add("GET /api/teams/{id}/treasurers", objFixture{params: teamFixture, noOwnerProbe: true})
	add("POST /api/teams/{id}/treasurers", objFixture{params: teamFixture, noOwnerProbe: true})
	add("DELETE /api/teams/{id}/treasurers/{memberId}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.bTeamID), "memberId": strconv.Itoa(w.newKassenwart())}
		},
		noOwnerProbe: true,
	})

	// ── Videos ───────────────────────────────────────────────────────────────
	videoFixture := func(w *objWorld) map[string]string { return p("id", w.newVideo()) }
	add("GET /api/videos/{id}", objFixture{params: videoFixture})
	add("GET /api/videos/{id}/play", objFixture{params: videoFixture, noOwnerProbe: true})
	add("PATCH /api/videos/{id}", objFixture{params: videoFixture, noOwnerProbe: true})
	add("DELETE /api/videos/{id}", objFixture{params: videoFixture, noOwnerProbe: true})
	add("GET /api/videos/{id}/hls/master.m3u8", objFixture{params: videoFixture, noOwnerProbe: true})
	add("GET /api/videos/{id}/hls/{rendition}/{segment}", objFixture{
		params: func(w *objWorld) map[string]string {
			return map[string]string{"id": strconv.Itoa(w.newVideo()), "rendition": "720p", "segment": "seg_001.ts"}
		},
		noOwnerProbe: true,
	})

	// ── Spielberichte ────────────────────────────────────────────────────────
	reportFixture := func(w *objWorld) map[string]string { return p("id", w.newMatchReport()) }
	add("GET /api/match-reports/{id}", objFixture{params: reportFixture})
	add("PUT /api/match-reports/{id}", objFixture{params: reportFixture, noOwnerProbe: true})
	add("DELETE /api/match-reports/{id}", objFixture{params: reportFixture})
	add("POST /api/match-reports/{id}/submit-for-review", objFixture{params: reportFixture, noOwnerProbe: true})
	add("POST /api/match-reports/{id}/images", objFixture{params: reportFixture, noOwnerProbe: true})
	add("GET /api/match-reports/{id}/images/{imgId}/blob", objFixture{
		params: func(w *objWorld) map[string]string {
			reportID, imgID := w.newMatchReportImage()
			return map[string]string{"id": strconv.Itoa(reportID), "imgId": strconv.Itoa(imgID)}
		},
		noOwnerProbe: true, // Bilddatei liegt nicht auf Platte.
	})
	add("DELETE /api/match-reports/{id}/images/{imgId}", objFixture{
		params: func(w *objWorld) map[string]string {
			reportID, imgID := w.newMatchReportImage()
			return map[string]string{"id": strconv.Itoa(reportID), "imgId": strconv.Itoa(imgID)}
		},
		noOwnerProbe: true,
	})

	// ── Kader / Vorlagen (Vorstand+Trainer+sL-Tier, aber objektgebunden) ─────
	kaderFixture := func(w *objWorld) map[string]string { return p("id", w.bKaderID) }
	add("GET /api/kader/{id}", objFixture{who: personaT, params: kaderFixture, noOwnerProbe: true})
	add("PUT /api/kader/{id}", objFixture{who: personaT, params: kaderFixture, noOwnerProbe: true})
	// Leerer Kader: der besetzte endet schon an der Mitglieder-Zählung mit 409
	// und verdeckte damit, dass DeleteKader kein Objektrecht prüft.
	add("DELETE /api/kader/{id}", objFixture{
		who:          personaT,
		params:       func(w *objWorld) map[string]string { return p("id", w.newEmptyKader()) },
		noOwnerProbe: true,
	})
	add("GET /api/kader/{id}/member-suggestions", objFixture{who: personaT, params: kaderFixture, noOwnerProbe: true})
	add("GET /api/kader/{id}/extended-member-suggestions", objFixture{who: personaT, params: kaderFixture, noOwnerProbe: true})
	add("PATCH /api/kader/{id}/games-per-season", objFixture{who: personaT, params: kaderFixture, noOwnerProbe: true})

	// ── Änderungsanträge (Vorstand+Trainer+sL-Tier) ──────────────────────────
	draftFixture := func(w *objWorld) map[string]string {
		memberID, draftID := w.newChangeDraft()
		return map[string]string{"id": strconv.Itoa(memberID), "draftId": strconv.Itoa(draftID)}
	}
	add("POST /api/members/{id}/change-drafts/{draftId}/accept", objFixture{who: personaT, params: draftFixture, noOwnerProbe: true})
	add("DELETE /api/members/{id}/change-drafts/{draftId}", objFixture{who: personaT, params: draftFixture, noOwnerProbe: true})

	// ── Veranstaltungsorte / Dienstvorlagen (Vereinsstammdaten im Trainer-Tier)
	venueFixture := func(w *objWorld) map[string]string {
		return p("id", w.ins(
			`INSERT INTO venues (name, street, city, postal_code) VALUES (?, 'Musterweg 1', 'Stuttgart', '70173')`,
			uniq("Halle")))
	}
	add("PUT /api/venues/{id}", objFixture{who: personaT, params: venueFixture, noOwnerProbe: true})
	add("DELETE /api/venues/{id}", objFixture{who: personaT, params: venueFixture, noOwnerProbe: true})

	templateFixture := func(w *objWorld) map[string]string {
		return p("id", w.ins(`INSERT INTO game_templates (name, template_type) VALUES (?, 'heim')`, uniq("Vorlage")))
	}
	add("GET /api/duty-templates/{id}", objFixture{who: personaT, params: templateFixture, noOwnerProbe: true})
	add("GET /api/duty-templates/{id}/preview", objFixture{who: personaT, params: templateFixture, noOwnerProbe: true})

	return m
}

// ── Testlauf ─────────────────────────────────────────────────────────────────

func TestObjectAuthorizationMatrix(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	w := newObjWorld(t, db, srv)

	fixtures := objectFixtures()

	// 1. Alle Routen mit Pfadparameter einsammeln (wie die Tier-Matrix per chi.Walk).
	type route struct{ method, pattern string }
	var routes []route
	chiRouter, ok := srv.Config.Handler.(chi.Routes)
	if !ok {
		t.Fatal("Test-Server-Handler ist kein chi.Routes — chi.Walk nicht möglich")
	}
	if err := chi.Walk(chiRouter, func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.Contains(pattern, "{") {
			return nil
		}
		routes = append(routes, route{method, pattern})
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].pattern != routes[j].pattern {
			return routes[i].pattern < routes[j].pattern
		}
		return routes[i].method < routes[j].method
	})

	seen := map[string]bool{}
	var unclassified []string
	var checked, tierChecked int

	for _, rt := range routes {
		key := rt.method + " " + rt.pattern
		seen[key] = true

		switch {
		case openByDesign[key] != "":
			continue
		case todoFixtures[key] != "":
			continue
		case tierOnly[key] != "":
			tierChecked++
			runTierCase(t, w, rt.method, rt.pattern)
		default:
			f, has := fixtures[key]
			if !has {
				unclassified = append(unclassified, key)
				continue
			}
			checked++
			runObjectCase(t, w, rt.method, rt.pattern, f)
		}
	}

	if len(unclassified) > 0 {
		for _, key := range unclassified {
			t.Errorf("Route %s ist in der Objekt-Matrix nicht klassifiziert — "+
				"Fixture-Erzeuger in objectFixtures() ergänzen oder mit Begründung in "+
				"tierOnly/openByDesign/todoFixtures eintragen "+
				"(internal/permissions/object_matrix_test.go)", key)
		}
	}

	// 2. Verwaiste Einträge in allen Listen → Fehler (Pflege erzwingen).
	orphan := func(name string, keys []string) {
		for _, k := range keys {
			if !seen[k] {
				t.Errorf("Verwaister %s-Eintrag %q — Route existiert nicht mehr, Eintrag entfernen", name, k)
			}
		}
	}
	orphan("openByDesign", mapKeys(openByDesign))
	orphan("todoFixtures", mapKeys(todoFixtures))
	orphan("tierOnly", mapKeys(tierOnly))
	orphan("objectFixtures", fixtureKeys(fixtures))
	orphan("validationBeforeAuthz", mapKeys(validationBeforeAuthz))
	for k := range knownGaps {
		if !seen[k] {
			t.Errorf("Verwaister knownGaps-Eintrag %q — Route existiert nicht mehr, Eintrag entfernen", k)
		}
	}
	// Ein Befund ohne Fixture wäre nie geprüft worden — die Liste muss auf
	// tatsächlich aufgerufene Routen zeigen.
	for _, list := range []struct {
		name string
		keys []string
	}{
		{"knownGaps", mapKeys(gapKeys(knownGaps))},
		{"validationBeforeAuthz", mapKeys(validationBeforeAuthz)},
	} {
		for _, k := range list.keys {
			if _, has := fixtures[k]; !has {
				t.Errorf("%s-Eintrag %q hat kein Fixture — der Befund wird gar nicht geprüft", list.name, k)
			}
		}
	}

	t.Logf("Objekt-Matrix: %d Routen mit Pfadparameter · %d mit echtem Fixture geprüft · "+
		"%d nur Tier-geprüft · %d openByDesign · %d todoFixtures · %d knownGaps · "+
		"%d validationBeforeAuthz",
		len(routes), checked, tierChecked, len(openByDesign), len(todoFixtures),
		len(knownGaps), len(validationBeforeAuthz))
}

// runTierCase prüft, dass die Middleware A (bzw. T) vor dem Handler abweist.
func runTierCase(t *testing.T, w *objWorld, method, pattern string) {
	t.Helper()
	key := method + " " + pattern
	url := fillParams(pattern, map[string]string{})
	got := call(t, w, method, url, w.aToken, defaultBody(method, nil))
	if got != http.StatusForbidden && got != http.StatusUnauthorized {
		t.Errorf("%s: Tier-Gate greift nicht mehr (Status %d statt 403) — "+
			"die Route braucht jetzt ein echtes Objekt-Fixture statt des tierOnly-Eintrags", key, got)
	}
}

// runObjectCase legt das fremde Objekt an und prüft die Antwort auf A/T.
func runObjectCase(t *testing.T, w *objWorld, method, pattern string, f objFixture) {
	t.Helper()
	key := method + " " + pattern

	// Gegenprobe: der Eigentümer B darf das Objekt finden. Ein 404 hier hieße,
	// dass das Fixture gar nicht greift und der 404 auf A nichts beweist.
	if !f.noOwnerProbe && method == http.MethodGet {
		ownerURL := fillParams(pattern, f.params(w))
		if got := call(t, w, method, ownerURL, w.bToken, nil); got == http.StatusNotFound {
			t.Errorf("%s: Fixture greift nicht — schon der Eigentümer B bekommt 404. "+
				"Solange das so ist, beweist der 404 auf den Anfragenden nichts.", key)
			return
		}
	}

	url := fillParams(pattern, f.params(w))
	token := w.aToken
	switch f.who {
	case personaT:
		token = w.tToken
	case personaE:
		token = w.eToken
	}
	body := f.body
	if f.bodyFn != nil {
		body = f.bodyFn(w)
	}
	got := call(t, w, method, url, token, defaultBody(method, body))

	if note, isValidationFirst := validationBeforeAuthz[key]; isValidationFirst {
		if got != http.StatusBadRequest {
			t.Errorf("%s (Persona %s): validationBeforeAuthz erwartet 400, bekommen %d — "+
				"der Befund hat sich geändert. Wenn die Route jetzt 403/404 liefert: Eintrag entfernen. (%s)",
				key, f.who, got, note)
		}
		return
	}

	if gap, isGap := knownGaps[key]; isGap {
		if got != gap.status {
			t.Errorf("%s (Persona %s): knownGaps erwartet %d, bekommen %d — "+
				"der Befund hat sich geändert. Wenn er behoben ist: Eintrag aus knownGaps entfernen. (%s)",
				key, f.who, gap.status, got, gap.note)
		}
		return
	}

	switch got {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		// erwartet
	case http.StatusBadRequest:
		t.Errorf("%s (Persona %s): 400 — Validierung läuft vor der Autorisierung. "+
			"Erwartet 401/403/404 auf ein fremdes Objekt.", key, f.who)
	default:
		t.Errorf("%s (Persona %s): Status %d auf ein fremdes Objekt — erwartet 401/403/404. "+
			"BEFUND: entweder im Handler beheben oder mit Begründung in knownGaps/openByDesign eintragen.",
			key, f.who, got)
	}
}

// call sendet die Anfrage und liefert den Status-Code.
func call(t *testing.T, w *objWorld, method, url, token string, body any) int {
	t.Helper()
	res := testutil.Do(t, w.srv, method, url, token, body)
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode
}

// defaultBody liefert für mutierende Methoden einen leeren JSON-Body, damit die
// Autorisierung (die vor der Validierung laufen soll) überhaupt erreicht wird.
func defaultBody(method string, override any) any {
	if override != nil {
		return override
	}
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return map[string]any{}
	default:
		return nil
	}
}

// fillParams ersetzt {name}-Segmente durch die Fixture-Werte; nicht belegte
// Parameter fallen auf "1" zurück (Tier-Fälle, die den Handler nie erreichen).
func fillParams(pattern string, params map[string]string) string {
	out := pattern
	for strings.Contains(out, "{") {
		start := strings.Index(out, "{")
		end := strings.Index(out[start:], "}")
		if end == -1 {
			break
		}
		end += start
		name := out[start+1 : end]
		value, ok := params[name]
		if !ok {
			value = "1"
		}
		out = out[:start] + value + out[end+1:]
	}
	return out
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// gapKeys reduziert knownGaps auf eine Form, die mapKeys lesen kann.
func gapKeys(m map[string]objGap) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.note
	}
	return out
}

func fixtureKeys(m map[string]objFixture) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
