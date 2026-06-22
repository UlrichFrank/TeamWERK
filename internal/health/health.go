// Package health stellt anbieter-neutrale Betriebssignale bereit:
//
//	GET /api/healthz  — Liveness/Readiness (public, 200/503), grobe nicht-sensible Signale
//	GET /api/metrics  — Prometheus-Textformat (Bearer-Token), reichere Metriken
//
// Es wertet selbst NICHTS aus und alarmiert NICHT — Schwellwerte, Auswertung und
// Alerting leben in einem austauschbaren externen Monitor. Die App ist reine
// Signal-Quelle (siehe openspec/changes/monitoring-selfhosted).
package health

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// startTime wird bei Package-Init (Prozessstart) gesetzt und speist die Uptime-Metrik.
var startTime = time.Now()

// Handler bedient /api/healthz und /api/metrics.
type Handler struct {
	db           *sql.DB
	dbPath       string
	metricsToken string
}

// NewHandler erzeugt den Health-Handler. metricsToken == "" deaktiviert /api/metrics (404).
func NewHandler(db *sql.DB, dbPath, metricsToken string) *Handler {
	return &Handler{db: db, dbPath: dbPath, metricsToken: metricsToken}
}

type healthzResponse struct {
	Status          string `json:"status"`            // "ok" | "degraded"
	DB              string `json:"db"`                // "ok" | "fail"
	DiskFreePct     int    `json:"disk_free_pct"`     // -1 = unbekannt
	SchedulerAgeSec int64  `json:"scheduler_age_sec"` // -1 = noch kein Heartbeat
}

// Healthz ist die öffentliche Liveness/Readiness-Prüfung. 200 bei gesunder DB,
// sonst 503. Der Body trägt nur grobe, nicht-sensible Signale.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	dbOK := h.pingDB(r.Context())
	resp := healthzResponse{
		Status:          "ok",
		DB:              "ok",
		DiskFreePct:     diskFreePct(h.dbPath),
		SchedulerAgeSec: h.schedulerAgeSec(),
	}
	code := http.StatusOK
	if !dbOK {
		resp.Status = "degraded"
		resp.DB = "fail"
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resp)
}

// Metrics liefert die Betriebsmetriken im Prometheus-Textformat. Ohne gesetztes
// METRICS_TOKEN ist der Endpoint deaktiviert (404); sonst ist ein passender
// Bearer-Token Pflicht (401).
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if h.metricsToken == "" {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+h.metricsToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var b strings.Builder
	writeMetric(&b, "teamwerk_up", "gauge", "1 = Prozess antwortet", 1)
	dbUp := 0.0
	if h.pingDB(r.Context()) {
		dbUp = 1
	}
	writeMetric(&b, "teamwerk_db_up", "gauge", "1 = Datenbank erreichbar", dbUp)
	writeMetric(&b, "teamwerk_disk_free_ratio", "gauge", "Freier Anteil des DB-Dateisystems (0..1, -1 = unbekannt)", diskFreeRatio(h.dbPath))
	if ratio, ok := memFreeRatio(); ok {
		writeMetric(&b, "teamwerk_mem_free_ratio", "gauge", "Freier RAM-Anteil (0..1, nur Linux)", ratio)
	}
	writeMetric(&b, "teamwerk_scheduler_age_seconds", "gauge", "Sekunden seit letztem Scheduler-Heartbeat (-1 = nie)", float64(h.schedulerAgeSec()))
	writeMetric(&b, "teamwerk_panics_total", "counter", "Abgefangene HTTP-Handler-Panics seit Prozessstart", float64(PanicsTotal()))
	writeMetric(&b, "teamwerk_uptime_seconds", "gauge", "Prozess-Laufzeit in Sekunden", time.Since(startTime).Seconds())

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, b.String())
}

func writeMetric(b *strings.Builder, name, typ, help string, val float64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n%s %g\n", name, help, name, typ, name, val)
}

func (h *Handler) pingDB(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.PingContext(ctx) == nil
}

// schedulerAgeSec liefert das Alter des letzten Scheduler-Heartbeats in Sekunden,
// oder -1 falls noch kein Heartbeat geschrieben wurde.
func (h *Handler) schedulerAgeSec() int64 {
	var ts string
	if err := h.db.QueryRow(`SELECT updated_at FROM monitoring_heartbeat WHERE id = 1`).Scan(&ts); err != nil {
		return -1
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return -1
	}
	age := int64(time.Since(t).Seconds())
	if age < 0 {
		age = 0
	}
	return age
}

// diskFreePct liefert den freien Anteil des Dateisystems der DB-Datei in Prozent
// (-1 = unbekannt). Funktioniert unter Linux und macOS.
func diskFreePct(dbPath string) int {
	ratio := diskFreeRatio(dbPath)
	if ratio < 0 {
		return -1
	}
	return int(ratio * 100)
}

func diskFreeRatio(dbPath string) float64 {
	dir := "."
	if dbPath != "" {
		dir = filepath.Dir(dbPath)
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return -1
	}
	total := st.Blocks * uint64(st.Bsize)
	if total == 0 {
		return -1
	}
	free := st.Bavail * uint64(st.Bsize)
	return float64(free) / float64(total)
}

// memFreeRatio liest den freien RAM-Anteil aus /proc/meminfo (nur Linux).
// ok=false auf Plattformen ohne /proc/meminfo — der Aufrufer lässt die Metrik dann weg.
func memFreeRatio() (float64, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	var total, avail uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total, _ = strconv.ParseUint(fields[1], 10, 64)
		case "MemAvailable:":
			avail, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}
	if total == 0 {
		return 0, false
	}
	return float64(avail) / float64(total), true
}
