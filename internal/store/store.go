// Package store provides durable persistence for scheduler entities using an
// embedded SQLite database (pure-Go modernc.org/sqlite driver). All timestamps
// are stored in UTC (RFC 3339). The store is the only writer; clients reach it
// through the daemon's API.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// Store wraps a SQLite database connection.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at path, applies migrations, and
// enables foreign keys. Use ":memory:" for an in-memory database in tests.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// modernc.org/sqlite is safe for concurrent use, but a single writer avoids
	// SQLITE_BUSY under our daemon's serialized write path.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: pragmas: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// migration is one ordered schema change.
type migration struct {
	version int
	stmts   string
}

var migrations = []migration{
	{
		version: 1,
		stmts: `
CREATE TABLE IF NOT EXISTS groups (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	parent_id   TEXT REFERENCES groups(id) ON DELETE CASCADE,
	enabled     INTEGER NOT NULL DEFAULT 1,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_groups_parent ON groups(parent_id);

CREATE TABLE IF NOT EXISTS schedules (
	id            TEXT PRIMARY KEY,
	kind          TEXT NOT NULL,
	rrule         TEXT,
	anchor        TEXT,
	run_at        TEXT,
	trigger_id    TEXT,
	human_summary TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS tasks (
	id              TEXT PRIMARY KEY,
	name            TEXT NOT NULL,
	group_id        TEXT REFERENCES groups(id) ON DELETE SET NULL,
	command         TEXT NOT NULL,
	args_json       TEXT NOT NULL DEFAULT '[]',
	working_dir     TEXT NOT NULL DEFAULT '',
	env_json        TEXT NOT NULL DEFAULT '{}',
	run_as          TEXT NOT NULL DEFAULT '',
	enabled         INTEGER NOT NULL DEFAULT 1,
	timezone        TEXT NOT NULL DEFAULT 'Local',
	schedule_id     TEXT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
	overlap_policy  TEXT NOT NULL DEFAULT 'queue_one',
	catchup_policy  TEXT NOT NULL DEFAULT 'one',
	state           TEXT NOT NULL DEFAULT 'active',
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_group ON tasks(group_id);
CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state);

CREATE TABLE IF NOT EXISTS runs (
	id            TEXT PRIMARY KEY,
	task_id       TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	scheduled_for TEXT NOT NULL,
	started_at    TEXT,
	ended_at      TEXT,
	outcome       TEXT NOT NULL,
	exit_code     INTEGER,
	output        TEXT NOT NULL DEFAULT '',
	trigger       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runs_task ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_scheduled ON runs(scheduled_for);

CREATE TABLE IF NOT EXISTS alerts (
	id           TEXT PRIMARY KEY,
	task_id      TEXT,
	severity     TEXT NOT NULL,
	kind         TEXT NOT NULL,
	message      TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	acknowledged INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_alerts_ack ON alerts(acknowledged);
`,
	},
}

// migrate applies any migrations newer than the recorded schema version.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("store: schema_version table: %w", err)
	}
	var current int
	row := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("store: begin migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(m.stmts); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("store: apply migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_version(version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("store: record migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: commit migration %d: %w", m.version, err)
		}
	}
	return nil
}
