// Package settings liefert einen zur Laufzeit umschaltbaren Wartungsmodus.
//
// Der Zustand liegt in der Tabelle system_settings (Key-Value); der Store
// spiegelt ihn in einem atomic.Bool, damit die Middleware auf dem Hot-Path
// ohne DB-Roundtrip auskommt. Ein periodischer Poll fängt Änderungen ab,
// die außerhalb des HTTP-Handlers passiert sind — konkret der CLI-Fallback
// `teamwerk maintenance on|off`.
package settings

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"
)

const (
	keyMaintenanceMode = "maintenance_mode"
	valueOn            = "on"
	valueOff           = "off"

	// defaultPollInterval ist so gewählt, dass ein CLI-Toggle spätestens nach
	// zehn Sekunden im laufenden Server-Prozess sichtbar wird. Kosten: ein
	// SELECT alle 10 s — vernachlässigbar gegen den Nutzen (kein Restart nötig).
	defaultPollInterval = 10 * time.Second
)

// SettingSnapshot beschreibt den Zustand einer system_settings-Row mit
// Metadaten. Wird vom Admin-UI-Endpoint genutzt; die Middleware und der
// Public-Status-Endpoint kommen mit dem atomic.Bool aus.
// UpdatedAt kommt als ISO-8601-String aus SQLite (DEFAULT CURRENT_TIMESTAMP
// liefert 'YYYY-MM-DD HH:MM:SS') — bewusst kein time.Time, weil der SQLite-
// Treiber String-Timestamps nicht direkt in time.Time scannen kann. Der
// Handler reicht den String 1:1 ans Frontend durch.
type SettingSnapshot struct {
	Enabled       bool
	UpdatedAt     sql.NullString
	UpdatedByID   sql.NullInt64
	UpdatedByName sql.NullString
}

// Store cached den Wartungsmodus-Wert in-memory und synchronisiert
// periodisch mit der DB. Sicher für gleichzeitige Reads durch Middleware.
type Store struct {
	db            *sql.DB
	maintenanceOn atomic.Bool
	pollInterval  time.Duration
}

// NewStore lädt den initialen Wert aus system_settings und startet — sofern
// ctx nicht schon abgebrochen ist — einen Polling-Loop, der den Cache alle
// pollInterval Sekunden nachlädt. Ist ctx nil, läuft kein Poll (nützlich für
// Tests, die manuell reloaden).
func NewStore(ctx context.Context, database *sql.DB) *Store {
	s := &Store{db: database, pollInterval: defaultPollInterval}
	if err := s.reload(context.Background()); err != nil {
		slog.Warn("settings: initial reload failed", "error", err)
	}
	if ctx != nil {
		go s.pollLoop(ctx)
	}
	return s
}

// NewStoreForTest baut einen Store ohne Poll-Loop und mit konfigurierbarem
// Poll-Intervall. Für Tests, die den Poll gezielt anstoßen wollen.
func NewStoreForTest(database *sql.DB, pollInterval time.Duration) *Store {
	s := &Store{db: database, pollInterval: pollInterval}
	_ = s.reload(context.Background())
	return s
}

// MaintenanceMode liefert den aktuellen Cache-Wert (atomic-Load, lock-frei).
func (s *Store) MaintenanceMode() bool {
	return s.maintenanceOn.Load()
}

// SetMaintenanceMode schreibt den neuen Zustand in die DB und aktualisiert
// den in-memory-Cache sofort. updatedBy ist die User-ID des umschaltenden
// Admins (0 → NULL, z. B. wenn der CLI-Fallback aufgerufen wird).
func (s *Store) SetMaintenanceMode(ctx context.Context, enabled bool, updatedBy int) error {
	v := valueOff
	if enabled {
		v = valueOn
	}
	var updatedByArg any
	if updatedBy > 0 {
		updatedByArg = updatedBy
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE system_settings
		 SET value = ?, updated_at = CURRENT_TIMESTAMP, updated_by = ?
		 WHERE key = ?`,
		v, updatedByArg, keyMaintenanceMode)
	if err != nil {
		return err
	}
	s.maintenanceOn.Store(enabled)
	return nil
}

// Snapshot liefert den vollen Zustand inklusive Metadaten (updated_at,
// updated_by mit E-Mail-Anzeige aus users). Für die Admin-UI.
func (s *Store) Snapshot(ctx context.Context) (SettingSnapshot, error) {
	var snap SettingSnapshot
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT ss.value, ss.updated_at, ss.updated_by, u.email
		 FROM system_settings ss
		 LEFT JOIN users u ON u.id = ss.updated_by
		 WHERE ss.key = ?`,
		keyMaintenanceMode,
	).Scan(&value, &snap.UpdatedAt, &snap.UpdatedByID, &snap.UpdatedByName)
	if errors.Is(err, sql.ErrNoRows) {
		return SettingSnapshot{}, nil
	}
	if err != nil {
		return SettingSnapshot{}, err
	}
	snap.Enabled = value == valueOn
	return snap, nil
}

// Reload lädt den DB-Wert neu und aktualisiert den Cache. Public gemacht, damit
// Tests den Poll-Zyklus simulieren können, ohne die Goroutine zu warten.
func (s *Store) Reload(ctx context.Context) error {
	return s.reload(ctx)
}

func (s *Store) reload(ctx context.Context) error {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM system_settings WHERE key = ?`, keyMaintenanceMode,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		s.maintenanceOn.Store(false)
		return nil
	}
	if err != nil {
		return err
	}
	s.maintenanceOn.Store(value == valueOn)
	return nil
}

func (s *Store) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.reload(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("settings: poll reload failed", "error", err)
			}
		}
	}
}
