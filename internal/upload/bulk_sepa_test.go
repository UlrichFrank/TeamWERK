package upload_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

// Der server-seitige Bulk-SEPA-Mandat-Import ist mit Zero-Knowledge (Modell B) unvereinbar
// (er legte Klartext-PDFs server-verschlüsselt ohne client-DEK ab) und wurde deaktiviert: die
// Route ist nicht mehr gemountet, der Handler antwortet defensiv mit 410 Gone. Der Matching-
// Code bleibt als Grundlage für einen späteren CLIENTseitigen Bulk-Import dormant erhalten.
func TestBulkImport_Disabled(t *testing.T) {
	db := testutil.NewDB(t)
	h := upload.NewHandler(db, t.TempDir(), testutil.TestJWTSecret, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/members/sepa-mandates/import", h.BulkImportSepaMandate)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})
	res := testutil.Post(t, srv, "/api/members/sepa-mandates/import", tok, map[string]any{})
	if res.StatusCode != http.StatusGone {
		t.Fatalf("Bulk-Import deaktiviert: status %d, want 410", res.StatusCode)
	}
}
