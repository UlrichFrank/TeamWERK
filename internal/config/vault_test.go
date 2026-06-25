package config_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestEncryptionConfig_SetGetConflict(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(`INSERT INTO clubs (name) VALUES ('Team Stuttgart')`); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/admin/encryption-config", h.GetEncryptionConfig)
		r.Put("/api/admin/encryption-config", h.SetEncryptionConfig)
	})
	tok := testutil.Token(t, 1, "admin", []string{"vorstand"})

	// Vor Einrichtung: nicht konfiguriert
	res := testutil.Get(t, srv, "/api/admin/encryption-config", tok)
	var got map[string]any
	json.NewDecoder(res.Body).Decode(&got)
	if got["configured"] != false {
		t.Fatalf("vor Einrichtung configured = %v, want false", got["configured"])
	}

	// Einrichtung (Happy-Path)
	setup := map[string]any{"vorstand_kdf_salt": "c2FsdA==", "vorstand_key_check": "Y2hlY2s="}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", tok, setup); res.StatusCode != http.StatusNoContent {
		t.Fatalf("Einrichtung: status %d, want 204", res.StatusCode)
	}

	// Danach: konfiguriert, Salt + Key-Check werden ausgeliefert
	res = testutil.Get(t, srv, "/api/admin/encryption-config", tok)
	got = map[string]any{}
	json.NewDecoder(res.Body).Decode(&got)
	if got["configured"] != true {
		t.Errorf("nach Einrichtung configured = %v, want true", got["configured"])
	}
	if got["vorstand_kdf_salt"] != "c2FsdA==" {
		t.Errorf("salt = %v", got["vorstand_kdf_salt"])
	}

	// Zweite Einrichtung → 409 (Rotation verwenden)
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", tok, setup); res.StatusCode != http.StatusConflict {
		t.Errorf("doppelte Einrichtung: status %d, want 409", res.StatusCode)
	}

	// Leerer Body → 400
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", tok, map[string]any{}); res.StatusCode != http.StatusBadRequest {
		t.Errorf("leerer Body: status %d, want 400", res.StatusCode)
	}
}

func TestEncryptionConfig_Forbidden(t *testing.T) {
	database := testutil.NewDB(t)
	database.Exec(`INSERT INTO clubs (name) VALUES ('Team Stuttgart')`)
	h := config.NewHandler(database, hub.NewHub())
	// Mit Auth-Tier wie im Router gemountet.
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "kassierer"))
			r.Put("/api/admin/encryption-config", h.SetEncryptionConfig)
		})
	})
	spieler := testutil.Token(t, 2, "standard", []string{"spieler"})
	setup := map[string]any{"vorstand_kdf_salt": "c2FsdA==", "vorstand_key_check": "Y2hlY2s="}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", spieler, setup); res.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", res.StatusCode)
	}
}

func TestRotateEncryption_RewrapsAtomically(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO clubs (name, vorstand_kdf_salt, vorstand_key_check) VALUES ('Team Stuttgart', 'b2xkU2FsdA==', 'b2xkQ2hlY2s=')`,
	); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	m1 := testutil.CreateMember(t, database, 0)
	m2 := testutil.CreateMember(t, database, 0)
	for _, m := range []int{m1, m2} {
		if _, err := database.Exec(
			`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, 'CT', 'oldWrap')`, m,
		); err != nil {
			t.Fatalf("seed member_sensitive: %v", err)
		}
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/rotate-encryption", h.RotateEncryption)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})

	body := map[string]any{
		"vorstand_kdf_salt":  "bmV3U2FsdA==",
		"vorstand_key_check": "bmV3Q2hlY2s=",
		"wraps": []map[string]any{
			{"member_id": m1, "dek_enc_vorstand": "newWrap1"},
			{"member_id": m2, "dek_enc_vorstand": "newWrap2"},
		},
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/rotate-encryption", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("Rotation: status %d, want 204", res.StatusCode)
	}

	// Clubs-Hilfswerte aktualisiert
	var salt, check string
	database.QueryRow(`SELECT vorstand_kdf_salt, vorstand_key_check FROM clubs LIMIT 1`).Scan(&salt, &check)
	if salt != "bmV3U2FsdA==" || check != "bmV3Q2hlY2s=" {
		t.Errorf("clubs nicht rotiert: salt=%q check=%q", salt, check)
	}
	// Alle DEKs neu gewrappt
	var w1, w2 string
	database.QueryRow(`SELECT dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, m1).Scan(&w1)
	database.QueryRow(`SELECT dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, m2).Scan(&w2)
	if w1 != "newWrap1" || w2 != "newWrap2" {
		t.Errorf("DEKs nicht rotiert: w1=%q w2=%q", w1, w2)
	}
}
