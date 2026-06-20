package push_test

import (
	"encoding/json"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/push"
)

func TestBuildBadgePayload(t *testing.T) {
	cases := []struct {
		name  string
		title string
		body  string
		url   string
		badge int
	}{
		{"chat-message", "Anna", "Hallo!", "/chat", 3},
		{"zero-badge", "Anna", "leer", "/chat", 0},
		{"large-badge", "Vorstand", "Wichtig", "/chat", 150},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := push.BuildBadgePayload(c.title, c.body, c.url, c.badge)

			var decoded map[string]any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			for _, k := range []string{"title", "body", "url", "badge"} {
				if _, ok := decoded[k]; !ok {
					t.Fatalf("missing key %q in %s", k, raw)
				}
			}

			if got, ok := decoded["badge"].(float64); !ok || int(got) != c.badge {
				t.Fatalf("badge: expected number %d, got %v (%T)", c.badge, decoded["badge"], decoded["badge"])
			}
			if decoded["title"] != c.title || decoded["body"] != c.body || decoded["url"] != c.url {
				t.Fatalf("string fields mismatch: %v", decoded)
			}
		})
	}
}
