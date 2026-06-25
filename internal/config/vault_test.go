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

	// Einrichtung (Happy-Path): öffentlicher + verschlüsselter privater Schlüssel + Salt + Key-Check
	setup := map[string]any{
		"group_public_key":      "cHViS2V5",
		"group_private_key_enc": "ZW5jUHJpdg==",
		"vorstand_kdf_salt":     "c2FsdA==",
		"vorstand_key_check":    "Y2hlY2s=",
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", tok, setup); res.StatusCode != http.StatusNoContent {
		t.Fatalf("Einrichtung: status %d, want 204", res.StatusCode)
	}

	// Danach: konfiguriert, öffentlicher + verschlüsselter privater Schlüssel werden ausgeliefert
	res = testutil.Get(t, srv, "/api/admin/encryption-config", tok)
	got = map[string]any{}
	json.NewDecoder(res.Body).Decode(&got)
	if got["configured"] != true {
		t.Errorf("nach Einrichtung configured = %v, want true", got["configured"])
	}
	if got["group_public_key"] != "cHViS2V5" {
		t.Errorf("group_public_key = %v", got["group_public_key"])
	}
	if got["group_private_key_enc"] != "ZW5jUHJpdg==" {
		t.Errorf("group_private_key_enc = %v", got["group_private_key_enc"])
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
	setup := map[string]any{
		"group_public_key":      "cHViS2V5",
		"group_private_key_enc": "ZW5jUHJpdg==",
		"vorstand_kdf_salt":     "c2FsdA==",
		"vorstand_key_check":    "Y2hlY2s=",
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/encryption-config", spieler, setup); res.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", res.StatusCode)
	}
}

// Passphrase-Rotation (Normalfall, O(1)): nur privater Schlüssel + Salt + Key-Check werden
// ersetzt; öffentlicher Schlüssel und DEKs bleiben unangetastet.
func TestRotateEncryption_Passphrase(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO clubs (name, group_public_key, group_private_key_enc, vorstand_kdf_salt, vorstand_key_check)
		 VALUES ('Team Stuttgart', 'PUB', 'oldPrivEnc', 'b2xkU2FsdA==', 'b2xkQ2hlY2s=')`,
	); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	m1 := testutil.CreateMember(t, database, 0)
	if _, err := database.Exec(
		`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, 'CT', 'wrap1')`, m1,
	); err != nil {
		t.Fatalf("seed member_sensitive: %v", err)
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/rotate-encryption", h.RotateEncryption)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})

	body := map[string]any{
		"group_private_key_enc": "newPrivEnc",
		"vorstand_kdf_salt":     "bmV3U2FsdA==",
		"vorstand_key_check":    "bmV3Q2hlY2s=",
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/rotate-encryption", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("Passphrase-Rotation: status %d, want 204", res.StatusCode)
	}

	var pub, priv, salt, check, wrap string
	database.QueryRow(`SELECT group_public_key, group_private_key_enc, vorstand_kdf_salt, vorstand_key_check FROM clubs LIMIT 1`).
		Scan(&pub, &priv, &salt, &check)
	database.QueryRow(`SELECT dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, m1).Scan(&wrap)
	if priv != "newPrivEnc" || salt != "bmV3U2FsdA==" || check != "bmV3Q2hlY2s=" {
		t.Errorf("privater Schlüssel/Salt/Check nicht rotiert: priv=%q salt=%q check=%q", priv, salt, check)
	}
	if pub != "PUB" {
		t.Errorf("öffentlicher Schlüssel verändert: %q (Passphrase-Rotation darf ihn nicht anfassen)", pub)
	}
	if wrap != "wrap1" {
		t.Errorf("DEK verändert: %q (Passphrase-Rotation darf DEKs nicht anfassen)", wrap)
	}
}

// Keypair-Rotation (Schlüssel-Leak, O(n)): neuer öffentlicher Schlüssel + neu gewrappte DEKs.
func TestRotateEncryption_Keypair(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO clubs (name, group_public_key, group_private_key_enc, vorstand_kdf_salt, vorstand_key_check)
		 VALUES ('Team Stuttgart', 'oldPUB', 'oldPrivEnc', 'b2xkU2FsdA==', 'b2xkQ2hlY2s=')`,
	); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	m1 := testutil.CreateMember(t, database, 0)
	database.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, 'CT', 'oldWrap')`, m1)
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/admin/rotate-encryption", h.RotateEncryption)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})

	body := map[string]any{
		"group_public_key":      "newPUB",
		"group_private_key_enc": "newPrivEnc",
		"vorstand_kdf_salt":     "bmV3U2FsdA==",
		"vorstand_key_check":    "bmV3Q2hlY2s=",
		"wraps":                 []map[string]any{{"member_id": m1, "dek_enc_vorstand": "newWrap"}},
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/admin/rotate-encryption", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("Keypair-Rotation: status %d, want 204", res.StatusCode)
	}
	var pub, wrap string
	database.QueryRow(`SELECT group_public_key FROM clubs LIMIT 1`).Scan(&pub)
	database.QueryRow(`SELECT dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, m1).Scan(&wrap)
	if pub != "newPUB" || wrap != "newWrap" {
		t.Errorf("Keypair-Rotation unvollständig: pub=%q wrap=%q", pub, wrap)
	}
}
