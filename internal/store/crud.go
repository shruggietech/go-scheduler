package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("store: not found")

const rfc3339 = time.RFC3339Nano

// ---- conversion helpers -------------------------------------------------

func newID() string { return uuid.NewString() }

func fmtTime(t time.Time) string { return t.UTC().Format(rfc3339) }

func fmtTimePtr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: fmtTime(*t), Valid: true}
}

func parseTime(s string) (time.Time, error) { return time.Parse(rfc3339, s) }

func parseTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func intPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

func nullInt(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

// ---- Group --------------------------------------------------------------

// CreateGroup inserts g, assigning an ID and timestamps when empty.
func (s *Store) CreateGroup(g *domain.Group) error {
	if g.ID == "" {
		g.ID = newID()
	}
	now := time.Now().UTC()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	g.UpdatedAt = now
	_, err := s.db.Exec(
		`INSERT INTO groups(id,name,parent_id,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?)`,
		g.ID, g.Name, nullStr(g.ParentID), boolToInt(g.Enabled), fmtTime(g.CreatedAt), fmtTime(g.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: create group: %w", err)
	}
	return nil
}

// GetGroup returns the group by id, or ErrNotFound.
func (s *Store) GetGroup(id string) (domain.Group, error) {
	row := s.db.QueryRow(`SELECT id,name,parent_id,enabled,created_at,updated_at FROM groups WHERE id=?`, id)
	return scanGroup(row)
}

// ListGroups returns all groups.
func (s *Store) ListGroups() ([]domain.Group, error) {
	rows, err := s.db.Query(`SELECT id,name,parent_id,enabled,created_at,updated_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list groups: %w", err)
	}
	defer rows.Close()
	var out []domain.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetGroupEnabled toggles a group's enabled flag.
func (s *Store) SetGroupEnabled(id string, enabled bool) error {
	res, err := s.db.Exec(`UPDATE groups SET enabled=?, updated_at=? WHERE id=?`,
		boolToInt(enabled), fmtTime(time.Now().UTC()), id)
	return affected(res, err, "set group enabled")
}

// DeleteGroup removes a group (cascading to children via FK).
func (s *Store) DeleteGroup(id string) error {
	res, err := s.db.Exec(`DELETE FROM groups WHERE id=?`, id)
	return affected(res, err, "delete group")
}

type scanner interface {
	Scan(dest ...any) error
}

func scanGroup(sc scanner) (domain.Group, error) {
	var g domain.Group
	var parent sql.NullString
	var enabled int
	var created, updated string
	if err := sc.Scan(&g.ID, &g.Name, &parent, &enabled, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Group{}, ErrNotFound
		}
		return domain.Group{}, fmt.Errorf("store: scan group: %w", err)
	}
	g.ParentID = parent.String
	g.Enabled = enabled != 0
	g.CreatedAt, _ = parseTime(created)
	g.UpdatedAt, _ = parseTime(updated)
	return g, nil
}

// ---- Schedule -----------------------------------------------------------

// CreateSchedule inserts sch, assigning an ID when empty.
func (s *Store) CreateSchedule(sch *domain.Schedule) error {
	if sch.ID == "" {
		sch.ID = newID()
	}
	_, err := s.db.Exec(
		`INSERT INTO schedules(id,kind,rrule,anchor,run_at,trigger_id,human_summary) VALUES(?,?,?,?,?,?,?)`,
		sch.ID, string(sch.Kind), nullStr(sch.RRULE), fmtTimePtr(sch.Anchor), fmtTimePtr(sch.RunAt),
		nullStr(sch.TriggerID), sch.HumanSummary,
	)
	if err != nil {
		return fmt.Errorf("store: create schedule: %w", err)
	}
	return nil
}

// GetSchedule returns the schedule by id, or ErrNotFound.
func (s *Store) GetSchedule(id string) (domain.Schedule, error) {
	row := s.db.QueryRow(`SELECT id,kind,rrule,anchor,run_at,trigger_id,human_summary FROM schedules WHERE id=?`, id)
	var sch domain.Schedule
	var rrule, trigger sql.NullString
	var anchor, runAt sql.NullString
	var kind string
	if err := row.Scan(&sch.ID, &kind, &rrule, &anchor, &runAt, &trigger, &sch.HumanSummary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Schedule{}, ErrNotFound
		}
		return domain.Schedule{}, fmt.Errorf("store: scan schedule: %w", err)
	}
	sch.Kind = domain.ScheduleKind(kind)
	sch.RRULE = rrule.String
	sch.TriggerID = trigger.String
	var err error
	if sch.Anchor, err = parseTimePtr(anchor); err != nil {
		return domain.Schedule{}, fmt.Errorf("store: schedule anchor: %w", err)
	}
	if sch.RunAt, err = parseTimePtr(runAt); err != nil {
		return domain.Schedule{}, fmt.Errorf("store: schedule run_at: %w", err)
	}
	return sch, nil
}

// ---- Task ---------------------------------------------------------------

// CreateTask inserts t, assigning an ID and timestamps when empty.
func (s *Store) CreateTask(t *domain.Task) error {
	if t.ID == "" {
		t.ID = newID()
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	argsJSON, _ := json.Marshal(t.Args)
	envJSON, _ := json.Marshal(t.Env)
	_, err := s.db.Exec(
		`INSERT INTO tasks(id,name,group_id,command,args_json,working_dir,env_json,run_as,enabled,timezone,schedule_id,overlap_policy,catchup_policy,state,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, nullStr(t.GroupID), t.Command, string(argsJSON), t.WorkingDir, string(envJSON),
		t.RunAs, boolToInt(t.Enabled), t.Timezone, t.ScheduleID, string(t.OverlapPolicy),
		string(t.CatchupPolicy), string(t.State), fmtTime(t.CreatedAt), fmtTime(t.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: create task: %w", err)
	}
	return nil
}

// GetTask returns the task by id, or ErrNotFound.
func (s *Store) GetTask(id string) (domain.Task, error) {
	row := s.db.QueryRow(taskSelect+` WHERE id=?`, id)
	return scanTask(row)
}

// ListTasks returns all tasks, optionally filtered by group and/or state
// (empty string means "any").
func (s *Store) ListTasks(groupID, state string) ([]domain.Task, error) {
	q := taskSelect + ` WHERE 1=1`
	var args []any
	if groupID != "" {
		q += ` AND group_id=?`
		args = append(args, groupID)
	}
	if state != "" {
		q += ` AND state=?`
		args = append(args, state)
	}
	q += ` ORDER BY name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list tasks: %w", err)
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SetTaskState updates a task's lifecycle state.
func (s *Store) SetTaskState(id string, state domain.TaskState) error {
	res, err := s.db.Exec(`UPDATE tasks SET state=?, updated_at=? WHERE id=?`,
		string(state), fmtTime(time.Now().UTC()), id)
	return affected(res, err, "set task state")
}

// SetTaskEnabled toggles a task's enabled flag.
func (s *Store) SetTaskEnabled(id string, enabled bool) error {
	res, err := s.db.Exec(`UPDATE tasks SET enabled=?, updated_at=? WHERE id=?`,
		boolToInt(enabled), fmtTime(time.Now().UTC()), id)
	return affected(res, err, "set task enabled")
}

// DeleteTask removes a task (cascading to its runs and schedule).
func (s *Store) DeleteTask(id string) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return affected(res, err, "delete task")
}

const taskSelect = `SELECT id,name,group_id,command,args_json,working_dir,env_json,run_as,enabled,timezone,schedule_id,overlap_policy,catchup_policy,state,created_at,updated_at FROM tasks`

func scanTask(sc scanner) (domain.Task, error) {
	var t domain.Task
	var group sql.NullString
	var argsJSON, envJSON string
	var enabled int
	var overlap, catchup, state, created, updated string
	if err := sc.Scan(&t.ID, &t.Name, &group, &t.Command, &argsJSON, &t.WorkingDir, &envJSON, &t.RunAs,
		&enabled, &t.Timezone, &t.ScheduleID, &overlap, &catchup, &state, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, ErrNotFound
		}
		return domain.Task{}, fmt.Errorf("store: scan task: %w", err)
	}
	t.GroupID = group.String
	_ = json.Unmarshal([]byte(argsJSON), &t.Args)
	_ = json.Unmarshal([]byte(envJSON), &t.Env)
	t.Enabled = enabled != 0
	t.OverlapPolicy = domain.OverlapPolicy(overlap)
	t.CatchupPolicy = domain.CatchupPolicy(catchup)
	t.State = domain.TaskState(state)
	t.CreatedAt, _ = parseTime(created)
	t.UpdatedAt, _ = parseTime(updated)
	return t, nil
}

// ---- Run ----------------------------------------------------------------

// CreateRun inserts a run record, assigning an ID when empty.
func (s *Store) CreateRun(r *domain.Run) error {
	if r.ID == "" {
		r.ID = newID()
	}
	_, err := s.db.Exec(
		`INSERT INTO runs(id,task_id,scheduled_for,started_at,ended_at,outcome,exit_code,output,trigger)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		r.ID, r.TaskID, fmtTime(r.ScheduledFor), fmtTimePtr(r.StartedAt), fmtTimePtr(r.EndedAt),
		string(r.Outcome), nullInt(r.ExitCode), r.Output, string(r.Trigger),
	)
	if err != nil {
		return fmt.Errorf("store: create run: %w", err)
	}
	return nil
}

// ListRuns returns runs for a task (or all when taskID is empty), newest first,
// up to limit (0 = no limit).
func (s *Store) ListRuns(taskID string, limit int) ([]domain.Run, error) {
	q := `SELECT id,task_id,scheduled_for,started_at,ended_at,outcome,exit_code,output,trigger FROM runs`
	var args []any
	if taskID != "" {
		q += ` WHERE task_id=?`
		args = append(args, taskID)
	}
	q += ` ORDER BY scheduled_for DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list runs: %w", err)
	}
	defer rows.Close()
	var out []domain.Run
	for rows.Next() {
		var r domain.Run
		var started, ended sql.NullString
		var exit sql.NullInt64
		var outcome, trigger, scheduled string
		if err := rows.Scan(&r.ID, &r.TaskID, &scheduled, &started, &ended, &outcome, &exit, &r.Output, &trigger); err != nil {
			return nil, fmt.Errorf("store: scan run: %w", err)
		}
		r.ScheduledFor, _ = parseTime(scheduled)
		r.StartedAt, _ = parseTimePtr(started)
		r.EndedAt, _ = parseTimePtr(ended)
		r.Outcome = domain.RunOutcome(outcome)
		r.Trigger = domain.RunTrigger(trigger)
		r.ExitCode = intPtr(exit)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- Alert --------------------------------------------------------------

// CreateAlert inserts an alert, assigning an ID and timestamp when empty.
func (s *Store) CreateAlert(a *domain.Alert) error {
	if a.ID == "" {
		a.ID = newID()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO alerts(id,task_id,severity,kind,message,created_at,acknowledged) VALUES(?,?,?,?,?,?,?)`,
		a.ID, nullStr(a.TaskID), string(a.Severity), string(a.Kind), a.Message,
		fmtTime(a.CreatedAt), boolToInt(a.Acknowledged),
	)
	if err != nil {
		return fmt.Errorf("store: create alert: %w", err)
	}
	return nil
}

// ListAlerts returns alerts, optionally only unacknowledged ones, newest first.
func (s *Store) ListAlerts(unackedOnly bool) ([]domain.Alert, error) {
	q := `SELECT id,task_id,severity,kind,message,created_at,acknowledged FROM alerts`
	if unackedOnly {
		q += ` WHERE acknowledged=0`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("store: list alerts: %w", err)
	}
	defer rows.Close()
	var out []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var task sql.NullString
		var ack int
		var severity, kind, created string
		if err := rows.Scan(&a.ID, &task, &severity, &kind, &a.Message, &created, &ack); err != nil {
			return nil, fmt.Errorf("store: scan alert: %w", err)
		}
		a.TaskID = task.String
		a.Severity = domain.AlertSeverity(severity)
		a.Kind = domain.AlertKind(kind)
		a.CreatedAt, _ = parseTime(created)
		a.Acknowledged = ack != 0
		out = append(out, a)
	}
	return out, rows.Err()
}

// AckAlert marks an alert acknowledged.
func (s *Store) AckAlert(id string) error {
	res, err := s.db.Exec(`UPDATE alerts SET acknowledged=1 WHERE id=?`, id)
	return affected(res, err, "ack alert")
}

// affected wraps an Exec result, returning ErrNotFound when no row changed.
func affected(res sql.Result, err error, op string) error {
	if err != nil {
		return fmt.Errorf("store: %s: %w", op, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: %s rows: %w", op, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
