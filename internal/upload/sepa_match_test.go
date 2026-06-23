package upload

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"golang.org/x/text/unicode/norm"
)

func TestNormalizeName(t *testing.T) {
	t.Parallel()
	// NFD-Form (decomposed: a + combining diaeresis), wie macOS Finder
	// Dateinamen liefert. Explizit per norm.NFD aufgebaut, damit der Test
	// nicht davon abhängt, in welcher Form das Source-Literal eingebettet ist.
	nfdJaeger := norm.NFD.String("Jäger")
	cases := []struct {
		in, want string
	}{
		{"Max Mustermann", "maxmustermann"},
		{"MaxMustermann", "maxmustermann"},
		{"Jürgen Müller", "juergenmueller"},
		{"Heß", "hess"},
		{"Anna-Lena O'Connor", "annalenaoconnor"},
		{"  Max_Mustermann.PDF ", "maxmustermannpdf"},
		{nfdJaeger, "jaeger"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeName(c.in); got != c.want {
			t.Errorf("normalizeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func insertMatchMember(t *testing.T, db *sql.DB, first, last string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status) VALUES (?, ?, 'aktiv')`,
		first, last)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func TestMatchMemberByFilename(t *testing.T) {
	t.Parallel()
	db := testutil.NewDB(t)

	maxID := insertMatchMember(t, db, "Max", "Mustermann")
	jurgenID := insertMatchMember(t, db, "Jürgen", "Müller")
	annaID := insertMatchMember(t, db, "Anna-Lena", "O'Connor")
	dupA := insertMatchMember(t, db, "Lukas", "Schmidt")
	dupB := insertMatchMember(t, db, "Lukas", "Schmidt")
	// Member mit Zweitname im first_name → erster-Token-Fallback muss greifen
	lucaID := insertMatchMember(t, db, "Luca Marco", "Buric")
	// Member mit Umlaut → NFD-Input (macOS) muss matchen
	felixID := insertMatchMember(t, db, "Felix", "Jäger")

	cases := []struct {
		name     string
		basename string
		wantIDs  []int
	}{
		{"unique forward", "MaxMustermann", []int{maxID}},
		{"unique reverse", "MustermannMax", []int{maxID}},
		{"umlaut", "JuergenMueller", []int{jurgenID}},
		{"hyphen+apostrophe", "AnnaLenaOConnor", []int{annaID}},
		{"reverse hyphen+apostrophe", "OConnorAnnaLena", []int{annaID}},
		{"no match", "Unbekannt", nil},
		{"ambiguous", "LukasSchmidt", []int{dupA, dupB}},
		{"first-token fallback forward", "LucaBuric", []int{lucaID}},
		{"first-token fallback reverse", "BuricLuca", []int{lucaID}},
		{"full name with second name still matches", "LucaMarcoBuric", []int{lucaID}},
		{"NFD basename matcht NFC-DB-Name", norm.NFD.String("JägerFelix"), []int{felixID}},
		{"empty basename", "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := matchMemberByFilename(context.Background(), db, c.basename)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			sort.Ints(got)
			want := append([]int(nil), c.wantIDs...)
			sort.Ints(want)
			if !equalIntSlice(got, want) {
				t.Errorf("matchMemberByFilename(%q) = %v, want %v", c.basename, got, c.wantIDs)
			}
		})
	}
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
