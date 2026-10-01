package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func intPtr(v int) *int { return &v }

// TestResolveAlbum deckt die Album-Regeln ab: Kurzform, Grenze 10, Duplikate,
// fremde und bereits zugeordnete media-IDs sowie beide Felder zugleich. Jede
// Verletzung ist derselbe Fehler (errInvalidMedia), damit der Server nicht
// verrät, ob eine fremde ID existiert.
func TestResolveAlbum(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")

	seq := 0
	newMedia := func(uploader int) int {
		t.Helper()
		seq++
		res, err := db.Exec(`INSERT INTO media (disk_name, mime_type, size, uploaded_by) VALUES (?, 'image/png', 1, ?)`,
			fmt.Sprintf("album-%d.png", seq), uploader)
		if err != nil {
			t.Fatalf("media: %v", err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}
	own := make([]int, 11)
	for i := range own {
		own[i] = newMedia(owner)
	}
	foreign := newMedia(other)

	// Ein eigenes Bild, das schon an einer Nachricht hängt, und eins an einer Mitteilung.
	usedInMsg, usedInBc := newMedia(owner), newMedia(owner)
	res, _ := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', 'G', ?)`, owner)
	convID, _ := res.LastInsertId()
	res, _ = db.Exec(`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`, convID, owner, usedInMsg)
	msgID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO message_media (message_id, media_id, position) VALUES (?, ?, 0)`, msgID, usedInMsg)
	res, _ = db.Exec(`INSERT INTO broadcasts (sender_id, body, media_id) VALUES (?, '', ?)`, owner, usedInBc)
	bcID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO broadcast_media (broadcast_id, media_id, position) VALUES (?, ?, 0)`, bcID, usedInBc)

	cases := []struct {
		name    string
		in      albumFields
		want    []int
		wantErr bool
	}{
		{"leer", albumFields{}, nil, false},
		{"leere Liste", albumFields{MediaIDs: []int{}}, nil, false},
		{"Kurzform", albumFields{MediaID: intPtr(own[0])}, []int{own[0]}, false},
		{"eins", albumFields{MediaIDs: []int{own[0]}}, []int{own[0]}, false},
		{"zehn in Reihenfolge", albumFields{MediaIDs: []int{own[9], own[8], own[7], own[6], own[5], own[4], own[3], own[2], own[1], own[0]}},
			[]int{own[9], own[8], own[7], own[6], own[5], own[4], own[3], own[2], own[1], own[0]}, false},
		{"elf", albumFields{MediaIDs: own}, nil, true},
		{"Duplikat", albumFields{MediaIDs: []int{own[0], own[0]}}, nil, true},
		{"fremd", albumFields{MediaIDs: []int{own[0], foreign}}, nil, true},
		{"fremd als Kurzform", albumFields{MediaID: intPtr(foreign)}, nil, true},
		{"unbekannt", albumFields{MediaIDs: []int{999999}}, nil, true},
		{"schon an Nachricht", albumFields{MediaIDs: []int{usedInMsg}}, nil, true},
		{"schon an Mitteilung", albumFields{MediaIDs: []int{usedInBc}}, nil, true},
		{"beide Felder", albumFields{MediaID: intPtr(own[0]), MediaIDs: []int{own[1]}}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveAlbum(context.Background(), db, owner, tc.in)
			if tc.wantErr {
				if !errors.Is(err, errInvalidMedia) {
					t.Fatalf("erwartet errInvalidMedia, bekommen %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestImagePreview(t *testing.T) {
	for n, want := range map[int]string{1: "Bild", 2: "2 Bilder", 10: "10 Bilder"} {
		if got := imagePreview(n); got != want {
			t.Errorf("imagePreview(%d) = %q, want %q", n, got, want)
		}
	}
}

var _ rowQueryer = (*sql.Tx)(nil)
