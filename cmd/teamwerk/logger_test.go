package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// Der Default-Handler im Standardformat (json) MUSS valide JSON-Records mit
// level/msg/time schreiben — die neutrale Log-Schnittstelle für beliebige Collector.
func TestLogger_EmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newLogHandler("json", &buf))
	logger.Info("hello", "k", "v")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, buf.String())
	}
	for _, key := range []string{"level", "msg", "time"} {
		if _, ok := rec[key]; !ok {
			t.Errorf("JSON log record missing %q field: %v", key, rec)
		}
	}
	if rec["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", rec["msg"])
	}
}

// LOG_FORMAT=text liefert menschenlesbare Zeilen statt JSON.
func TestLogger_TextFormatNotJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newLogHandler("text", &buf))
	logger.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err == nil {
		t.Errorf("text format unexpectedly parsed as JSON: %s", buf.String())
	}
}
