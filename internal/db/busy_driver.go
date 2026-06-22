// Package db erweitert den modernc.org/sqlite-Treiber um eine dünne Hülle,
// die SQLITE_BUSY-Returns zentral als teamwerk_sqlite_busy_total zählt.
//
// Architektur: der Wrapper delegiert vollständig an den registrierten "sqlite"-
// Treiber und ruft am Error-Pfad health.CheckSQLiteBusy(err). Handler bleiben
// *sql.DB-typisiert; sie merken vom Wrapping nichts.
package db

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/teamstuttgart/teamwerk/internal/health"
	sqlitedrv "modernc.org/sqlite"
)

// busyDriverName ist der Treibername, unter dem der Wrapper registriert ist.
// Open() in db.go nutzt ihn statt "sqlite".
const busyDriverName = "sqlite-busy-counting"

func init() {
	sql.Register(busyDriverName, &busyDriver{underlying: &sqlitedrv.Driver{}})
}

// busyDriver implementiert driver.Driver durch Delegation.
type busyDriver struct {
	underlying driver.Driver
}

func (d *busyDriver) Open(name string) (driver.Conn, error) {
	c, err := d.underlying.Open(name)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return &busyConn{c}, nil
}

// busyConn spiegelt die Subinterfaces, die modernc.org/sqlite selbst bietet.
// Build-Time-Assertions sichern ab, dass kein Subinterface vergessen wurde —
// sonst würde database/sql auf Slow-Path-Iterationen zurückfallen.
var (
	_ driver.Conn               = (*busyConn)(nil)
	_ driver.ConnBeginTx        = (*busyConn)(nil)
	_ driver.ConnPrepareContext = (*busyConn)(nil)
	_ driver.ExecerContext      = (*busyConn)(nil)
	_ driver.QueryerContext     = (*busyConn)(nil)
	_ driver.Pinger             = (*busyConn)(nil)
	_ driver.SessionResetter    = (*busyConn)(nil)
	_ driver.Validator          = (*busyConn)(nil)
)

type busyConn struct {
	c driver.Conn
}

func (b *busyConn) Prepare(query string) (driver.Stmt, error) {
	s, err := b.c.Prepare(query)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return &busyStmt{s}, nil
}

func (b *busyConn) Close() error {
	err := b.c.Close()
	if err != nil {
		health.CheckSQLiteBusy(err)
	}
	return err
}

func (b *busyConn) Begin() (driver.Tx, error) {
	tx, err := b.c.Begin() //nolint:staticcheck // delegiert an Underlying-Conn; ConnBeginTx wird daneben angeboten
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return tx, nil
}

func (b *busyConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	tx, err := b.c.(driver.ConnBeginTx).BeginTx(ctx, opts)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return tx, nil
}

func (b *busyConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	s, err := b.c.(driver.ConnPrepareContext).PrepareContext(ctx, query)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return &busyStmt{s}, nil
}

func (b *busyConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	res, err := b.c.(driver.ExecerContext).ExecContext(ctx, query, args)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return res, nil
}

func (b *busyConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	rows, err := b.c.(driver.QueryerContext).QueryContext(ctx, query, args)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return rows, nil
}

func (b *busyConn) Ping(ctx context.Context) error {
	err := b.c.(driver.Pinger).Ping(ctx)
	if err != nil {
		health.CheckSQLiteBusy(err)
	}
	return err
}

func (b *busyConn) ResetSession(ctx context.Context) error {
	err := b.c.(driver.SessionResetter).ResetSession(ctx)
	if err != nil {
		health.CheckSQLiteBusy(err)
	}
	return err
}

func (b *busyConn) IsValid() bool {
	return b.c.(driver.Validator).IsValid()
}

// busyStmt spiegelt die Stmt-Subinterfaces.
var (
	_ driver.Stmt             = (*busyStmt)(nil)
	_ driver.StmtExecContext  = (*busyStmt)(nil)
	_ driver.StmtQueryContext = (*busyStmt)(nil)
)

type busyStmt struct {
	s driver.Stmt
}

func (b *busyStmt) Close() error {
	err := b.s.Close()
	if err != nil {
		health.CheckSQLiteBusy(err)
	}
	return err
}

func (b *busyStmt) NumInput() int { return b.s.NumInput() }

func (b *busyStmt) Exec(args []driver.Value) (driver.Result, error) {
	res, err := b.s.Exec(args) //nolint:staticcheck // legacy-Pfad; StmtExecContext wird zusätzlich angeboten
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return res, nil
}

func (b *busyStmt) Query(args []driver.Value) (driver.Rows, error) {
	rows, err := b.s.Query(args) //nolint:staticcheck // legacy-Pfad; StmtQueryContext wird zusätzlich angeboten
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return rows, nil
}

func (b *busyStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	res, err := b.s.(driver.StmtExecContext).ExecContext(ctx, args)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return res, nil
}

func (b *busyStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	rows, err := b.s.(driver.StmtQueryContext).QueryContext(ctx, args)
	if err != nil {
		health.CheckSQLiteBusy(err)
		return nil, err
	}
	return rows, nil
}
