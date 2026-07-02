package videos

import "testing"

func TestH264CodecString(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		level   int
		want    string
		wantErr bool
	}{
		{"BaselineL30", "Baseline", 30, "avc1.42001E", false},
		{"ConstrainedBaselineL31", "Constrained Baseline", 31, "avc1.42001F", false},
		{"MainL40", "Main", 40, "avc1.4D0028", false},
		{"HighL40", "High", 40, "avc1.640028", false},
		{"HighL41", "High", 41, "avc1.640029", false},
		{"High10L40", "High 10", 40, "avc1.6E0028", false},
		{"UnknownProfile", "Fantasy", 40, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := h264CodecString(tc.profile, tc.level)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("h264CodecString(%q, %d) = %q, want %q", tc.profile, tc.level, got, tc.want)
			}
		})
	}
}

func TestAACCodecString(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		want    string
		wantErr bool
	}{
		{"LC", "LC", "mp4a.40.2", false},
		{"HEAAC", "HE-AAC", "mp4a.40.5", false},
		{"HEAACv2", "HE-AACv2", "mp4a.40.29", false},
		{"lowercaseLC", "lc", "mp4a.40.2", false},
		{"UnknownProfile", "SomethingElse", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := aacCodecString(tc.profile)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("aacCodecString(%q) = %q, want %q", tc.profile, got, tc.want)
			}
		})
	}
}
