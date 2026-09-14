package duties_test

// HTTP-Tests für das Video-Upload-Kennzeichen am Diensttyp
// (video-download-duty-upload): `grants_video_upload` ist wie `end_at_next_duty`
// ein reines bool-Flag ohne eigene Validierung — geprüft wird nur die Persistenz
// über Create/Update/List.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func typeGrantsVideoUpload(t *testing.T, srv *httptest.Server, token string, id int) bool {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-types", token)
	defer res.Body.Close()
	var list []struct {
		ID                int  `json:"id"`
		GrantsVideoUpload bool `json:"grants_video_upload"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode duty-types: %v", err)
	}
	for _, dt := range list {
		if dt.ID == id {
			return dt.GrantsVideoUpload
		}
	}
	t.Fatalf("Diensttyp %d nicht in der Liste", id)
	return false
}

func TestCreateType_GrantsVideoUploadIsPersisted(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
		"name": "Video", "hours_value": 2.0, "default_anchor": "start",
		"grants_video_upload": true,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekam %d", res.StatusCode)
	}

	var id int
	var stored bool
	if err := db.QueryRow(
		`SELECT id, grants_video_upload FROM duty_types WHERE name='Video'`).Scan(&id, &stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !stored {
		t.Error("grants_video_upload soll gespeichert sein")
	}
	if got := typeGrantsVideoUpload(t, srv, token, id); !got {
		t.Error("ListTypes soll grants_video_upload=true liefern")
	}
}

func TestCreateType_GrantsVideoUploadDefaultsFalse(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
		"name": "Kuchen", "hours_value": 1.0, "default_anchor": "start",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekam %d", res.StatusCode)
	}

	var stored bool
	if err := db.QueryRow(
		`SELECT grants_video_upload FROM duty_types WHERE name='Kuchen'`).Scan(&stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stored {
		t.Error("grants_video_upload soll ohne explizite Angabe false bleiben")
	}
}

func TestUpdateType_GrantsVideoUploadRoundtrip(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	dtID := createDutyType(t, db, "Video", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	body := map[string]any{
		"name": "Video", "hours_value": 2.0, "default_anchor": "start",
		"grants_video_upload": true,
	}
	res := testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekam %d", res.StatusCode)
	}
	if got := typeGrantsVideoUpload(t, srv, token, dtID); !got {
		t.Fatal("erwartet grants_video_upload=true nach dem Setzen")
	}

	body["grants_video_upload"] = false
	res = testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204 beim Zurücksetzen, bekam %d", res.StatusCode)
	}
	if got := typeGrantsVideoUpload(t, srv, token, dtID); got {
		t.Error("erwartet grants_video_upload=false nach dem Zurücksetzen")
	}
}
