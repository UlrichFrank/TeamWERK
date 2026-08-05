package h4aimport

import "testing"

func TestBuildXajaxArgs_ExactPayload(t *testing.T) {
	wantArg1 := `xjxargs[]=<xjxobj><e><k>ge_statsel</k><v>S0</v></e><e><k>ge_dasel</k><v>S<![CDATA[all;all]]></v></e><e><k>ge_gameno</k><v>S</v></e><e><k>sbdasel</k><v>SLos</v></e><e><k>ge_periods</k><v>S142</v></e></xjxobj>`
	wantArg2 := `xjxargs[]=<xjxobj><e><k>dummy</k><v>S1</v></e><e><k>opOwnGames</k><v>Son</v></e></xjxobj>`

	a1, a2 := BuildXajaxArgs("142")
	if got := "xjxargs[]=" + a1; got != wantArg1 {
		t.Errorf("arg1 falsch\n got: %s\nwant: %s", got, wantArg1)
	}
	if got := "xjxargs[]=" + a2; got != wantArg2 {
		t.Errorf("arg2 falsch\n got: %s\nwant: %s", got, wantArg2)
	}
}

func TestEncodeValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "S"},
		{"0", "S0"},
		{"Los", "SLos"},
		{"all;all", "S<![CDATA[all;all]]>"},
		{"a<b", "S<![CDATA[a<b]]>"},
		{"x&y", "S<![CDATA[x&y]]>"},
	}
	for _, c := range cases {
		if got := EncodeValue(c.in); got != c.want {
			t.Errorf("EncodeValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildGamesFormBody(t *testing.T) {
	body := BuildGamesFormBody("142", 1700000000000)
	// CDATA/XML-Sonderzeichen müssen URL-kodiert sein.
	if !contains(body, "xjxfun=xajax_update") || !contains(body, "xjxr=1700000000000") {
		t.Fatalf("Body ohne xjxfun/xjxr: %s", body)
	}
	if contains(body, "<xjxobj>") {
		t.Errorf("Body nicht URL-kodiert (rohe <xjxobj>): %s", body)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
