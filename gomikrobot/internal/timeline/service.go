package timeline

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type TimelineService struct {
	db *sql.DB
}

func NewTimelineService(dbPath string) (*TimelineService, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open timeline db: %w", err)
	}

	// Apply schema
	if _, err := db.Exec(Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}
	// Best-effort migration for existing dbs (no-op if column exists).
	_, _ = db.Exec(`ALTER TABLE web_users ADD COLUMN force_send BOOLEAN DEFAULT 1`)
	// Best-effort migrations for tracing columns.
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN trace_id TEXT`)
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN span_id TEXT`)
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN parent_span_id TEXT`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_timeline_trace ON timeline(trace_id)`)
	// Backfill trace_id for existing rows (best-effort).
	_, _ = db.Exec(`
		UPDATE timeline
		SET trace_id = CASE
			WHEN event_id IS NOT NULL AND event_id != '' THEN 'trace:' || event_id
			ELSE 'trace:' || sender_id || ':' || strftime('%s', timestamp)
		END
		WHERE trace_id IS NULL OR trace_id = ''
	`)
	// Backfill for older rows where force_send is NULL.
	_, _ = db.Exec(`UPDATE web_users SET force_send = 1 WHERE force_send IS NULL`)
	// Best-effort migration: ensure tasks table exists on older DBs.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT UNIQUE NOT NULL,
		idempotency_key TEXT UNIQUE,
		trace_id TEXT,
		channel TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		sender_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		content_in TEXT,
		content_out TEXT,
		error_text TEXT,
		delivery_status TEXT NOT NULL DEFAULT 'pending',
		delivery_attempts INTEGER NOT NULL DEFAULT 0,
		delivery_next_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_idempotency ON tasks(idempotency_key)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_trace ON tasks(trace_id)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_delivery ON tasks(delivery_status, delivery_next_at)`)
	// Best-effort migration: add message_type column to tasks table.
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN message_type TEXT DEFAULT ''`)
	// Best-effort migration: add token columns to tasks table.
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN prompt_tokens INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN completion_tokens INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN total_tokens INTEGER NOT NULL DEFAULT 0`)
	// Best-effort migration: policy_decisions table.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS policy_decisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trace_id TEXT,
		task_id TEXT,
		tool TEXT NOT NULL,
		tier INTEGER NOT NULL,
		sender TEXT,
		channel TEXT,
		allowed BOOLEAN NOT NULL,
		reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_policy_trace ON policy_decisions(trace_id)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_policy_task ON policy_decisions(task_id)`)
	// Best-effort migration: memory_chunks table.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS memory_chunks (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		embedding BLOB,
		source TEXT NOT NULL DEFAULT 'user',
		tags TEXT DEFAULT '',
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_memory_chunks_source ON memory_chunks(source)`)
	// Best-effort migration: span timing columns on timeline.
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN span_started_at DATETIME`)
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN span_ended_at DATETIME`)
	_, _ = db.Exec(`ALTER TABLE timeline ADD COLUMN span_duration_ms INTEGER DEFAULT 0`)
	// Best-effort migration: group tables.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS group_traces (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trace_id TEXT NOT NULL,
		source_agent_id TEXT NOT NULL,
		span_id TEXT,
		parent_span_id TEXT,
		span_type TEXT,
		title TEXT,
		content TEXT,
		started_at DATETIME,
		ended_at DATETIME,
		duration_ms INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_traces_trace ON group_traces(trace_id)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_traces_agent ON group_traces(source_agent_id)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS group_members (
		agent_id TEXT UNIQUE NOT NULL,
		agent_name TEXT,
		soul_summary TEXT,
		capabilities TEXT DEFAULT '[]',
		channels TEXT DEFAULT '[]',
		model TEXT,
		status TEXT DEFAULT 'active',
		last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	// Best-effort migration: group_tasks table.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS group_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT UNIQUE NOT NULL,
		description TEXT,
		content TEXT,
		direction TEXT NOT NULL DEFAULT 'outgoing',
		requester_id TEXT NOT NULL,
		responder_id TEXT,
		response_content TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		responded_at DATETIME
	)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_tasks_direction ON group_tasks(direction)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_group_tasks_status ON group_tasks(status)`)

	return &TimelineService{db: db}, nil
}

// DB returns the underlying *sql.DB for shared access (e.g. memory subsystem).
func (s *TimelineService) DB() *sql.DB { return s.db }

func (s *TimelineService) Close() error {
	return s.db.Close()
}

func (s *TimelineService) AddEvent(evt *TimelineEvent) error {
	query := `
	INSERT INTO timeline (event_id, trace_id, span_id, parent_span_id, timestamp, sender_id, sender_name, event_type, content_text, media_path, vector_id, classification, authorized)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		evt.EventID,
		evt.TraceID,
		evt.SpanID,
		evt.ParentSpanID,
		evt.Timestamp,
		evt.SenderID,
		evt.SenderName,
		evt.EventType,
		evt.ContentText,
		evt.MediaPath,
		evt.VectorID,
		evt.Classification,
		evt.Authorized,
	)
	return err
}

type FilterArgs struct {
	SenderID       string
	TraceID        string
	Limit          int
	Offset         int
	StartDate      *time.Time
	EndDate        *time.Time
	AuthorizedOnly *bool // nil = all, true = authorized only, false = unauthorized only
}

func (s *TimelineService) GetEvents(filter FilterArgs) ([]TimelineEvent, error) {
	query := `SELECT id, event_id, COALESCE(trace_id,''), COALESCE(span_id,''), COALESCE(parent_span_id,''), timestamp, sender_id, sender_name, event_type, content_text, media_path, vector_id, classification, authorized FROM timeline WHERE 1=1`
	args := []interface{}{}

	if filter.SenderID != "" {
		query += " AND sender_id = ?"
		args = append(args, filter.SenderID)
	}
	if filter.StartDate != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.EndDate)
	}
	if filter.AuthorizedOnly != nil {
		query += " AND authorized = ?"
		args = append(args, *filter.AuthorizedOnly)
	}
	if filter.TraceID != "" {
		query += " AND trace_id = ?"
		args = append(args, filter.TraceID)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		err := rows.Scan(
			&e.ID,
			&e.EventID,
			&e.TraceID,
			&e.SpanID,
			&e.ParentSpanID,
			&e.Timestamp,
			&e.SenderID,
			&e.SenderName,
			&e.EventType,
			&e.ContentText,
			&e.MediaPath,
			&e.VectorID,
			&e.Classification,
			&e.Authorized,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GetSetting returns a setting value by key.
func (s *TimelineService) GetSetting(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetSetting persists a setting value.
func (s *TimelineService) SetSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value)
	return err
}

// IsSilentMode checks if silent mode is enabled. Defaults to true (safe default).
func (s *TimelineService) IsSilentMode() bool {
	val, err := s.GetSetting("silent_mode")
	if err != nil {
		return true // Safe default: silent
	}
	return val == "true"
}

// CreateWebUser creates a web user or returns the existing one with the same name.
func (s *TimelineService) CreateWebUser(name string) (*WebUser, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	_, err := s.db.Exec(`INSERT INTO web_users (name, created_at) VALUES (?, datetime('now'))`, name)
	if err != nil {
		// If duplicate, return existing
		return s.GetWebUserByName(name)
	}
	return s.GetWebUserByName(name)
}

// ListWebUsers returns all web users sorted by name.
func (s *TimelineService) ListWebUsers() ([]WebUser, error) {
	rows, err := s.db.Query(`SELECT id, name, COALESCE(force_send, 1), created_at FROM web_users ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []WebUser
	for rows.Next() {
		var u WebUser
		if err := rows.Scan(&u.ID, &u.Name, &u.ForceSend, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetWebUser returns a web user by ID.
func (s *TimelineService) GetWebUser(id int64) (*WebUser, error) {
	var u WebUser
	err := s.db.QueryRow(`SELECT id, name, COALESCE(force_send, 1), created_at FROM web_users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &u.ForceSend, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetWebUserByName returns a web user by name.
func (s *TimelineService) GetWebUserByName(name string) (*WebUser, error) {
	var u WebUser
	err := s.db.QueryRow(`SELECT id, name, COALESCE(force_send, 1), created_at FROM web_users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &u.ForceSend, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// LinkWebUser links a web user to a WhatsApp JID.
func (s *TimelineService) LinkWebUser(webUserID int64, whatsappJID string) error {
	if whatsappJID == "" {
		return fmt.Errorf("whatsapp_jid is required")
	}
	_, err := s.db.Exec(`
		INSERT INTO web_links (web_user_id, whatsapp_jid, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(web_user_id) DO UPDATE SET whatsapp_jid = excluded.whatsapp_jid, updated_at = datetime('now')
	`, webUserID, whatsappJID)
	return err
}

// UnlinkWebUser removes the WhatsApp link for a web user.
func (s *TimelineService) UnlinkWebUser(webUserID int64) error {
	_, err := s.db.Exec(`DELETE FROM web_links WHERE web_user_id = ?`, webUserID)
	return err
}

// GetWebLink returns the WhatsApp JID for a web user, if linked.
func (s *TimelineService) GetWebLink(webUserID int64) (string, bool, error) {
	var jid string
	err := s.db.QueryRow(`SELECT whatsapp_jid FROM web_links WHERE web_user_id = ?`, webUserID).Scan(&jid)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return jid, true, nil
}

// SetWebUserForceSend updates the force_send flag for a web user.
func (s *TimelineService) SetWebUserForceSend(webUserID int64, force bool) error {
	_, err := s.db.Exec(`UPDATE web_users SET force_send = ? WHERE id = ?`, force, webUserID)
	return err
}

// --- Task CRUD ---

func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

// CreateTask inserts a new agent task. TaskID is generated if empty.
func (s *TimelineService) CreateTask(task *AgentTask) (*AgentTask, error) {
	if task.TaskID == "" {
		task.TaskID = newTaskID()
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.DeliveryStatus == "" {
		task.DeliveryStatus = DeliveryPending
	}

	query := `
	INSERT INTO tasks (task_id, idempotency_key, trace_id, channel, chat_id, sender_id, message_type, status, content_in, delivery_status)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Pass NULL for empty idempotency_key to avoid UNIQUE constraint on empty strings.
	var idempKey interface{}
	if task.IdempotencyKey != "" {
		idempKey = task.IdempotencyKey
	}
	result, err := s.db.Exec(query,
		task.TaskID,
		idempKey,
		task.TraceID,
		task.Channel,
		task.ChatID,
		task.SenderID,
		task.MessageType,
		task.Status,
		task.ContentIn,
		task.DeliveryStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	id, _ := result.LastInsertId()
	task.ID = id
	return s.GetTask(task.TaskID)
}

// GetTask returns a task by task_id.
func (s *TimelineService) GetTask(taskID string) (*AgentTask, error) {
	query := `SELECT id, task_id, COALESCE(idempotency_key,''), COALESCE(trace_id,''),
		channel, chat_id, COALESCE(sender_id,''), COALESCE(message_type,''), status,
		COALESCE(content_in,''), COALESCE(content_out,''), COALESCE(error_text,''),
		prompt_tokens, completion_tokens, total_tokens,
		delivery_status, delivery_attempts, delivery_next_at,
		created_at, updated_at, completed_at
	FROM tasks WHERE task_id = ?`

	var t AgentTask
	var deliveryNextAt, completedAt sql.NullTime
	err := s.db.QueryRow(query, taskID).Scan(
		&t.ID, &t.TaskID, &t.IdempotencyKey, &t.TraceID,
		&t.Channel, &t.ChatID, &t.SenderID, &t.MessageType, &t.Status,
		&t.ContentIn, &t.ContentOut, &t.ErrorText,
		&t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
		&t.DeliveryStatus, &t.DeliveryAttempts, &deliveryNextAt,
		&t.CreatedAt, &t.UpdatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if deliveryNextAt.Valid {
		t.DeliveryNextAt = &deliveryNextAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return &t, nil
}

// GetTaskByIdempotencyKey returns a task by its idempotency key.
// Returns (nil, nil) if not found — critical for dedup logic.
func (s *TimelineService) GetTaskByIdempotencyKey(key string) (*AgentTask, error) {
	if key == "" {
		return nil, nil
	}
	query := `SELECT id, task_id, COALESCE(idempotency_key,''), COALESCE(trace_id,''),
		channel, chat_id, COALESCE(sender_id,''), COALESCE(message_type,''), status,
		COALESCE(content_in,''), COALESCE(content_out,''), COALESCE(error_text,''),
		prompt_tokens, completion_tokens, total_tokens,
		delivery_status, delivery_attempts, delivery_next_at,
		created_at, updated_at, completed_at
	FROM tasks WHERE idempotency_key = ?`

	var t AgentTask
	var deliveryNextAt, completedAt sql.NullTime
	err := s.db.QueryRow(query, key).Scan(
		&t.ID, &t.TaskID, &t.IdempotencyKey, &t.TraceID,
		&t.Channel, &t.ChatID, &t.SenderID, &t.MessageType, &t.Status,
		&t.ContentIn, &t.ContentOut, &t.ErrorText,
		&t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
		&t.DeliveryStatus, &t.DeliveryAttempts, &deliveryNextAt,
		&t.CreatedAt, &t.UpdatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task by idempotency key: %w", err)
	}
	if deliveryNextAt.Valid {
		t.DeliveryNextAt = &deliveryNextAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return &t, nil
}

// UpdateTaskStatus updates a task's status, content_out, and error_text.
func (s *TimelineService) UpdateTaskStatus(taskID, status, contentOut, errorText string) error {
	query := `UPDATE tasks SET status = ?, content_out = ?, error_text = ?, updated_at = datetime('now')`
	if status == TaskStatusCompleted || status == TaskStatusFailed {
		query += `, completed_at = datetime('now')`
	}
	query += ` WHERE task_id = ?`
	_, err := s.db.Exec(query, status, contentOut, errorText, taskID)
	return err
}

// UpdateTaskDelivery updates delivery_status, increments delivery_attempts, and sets delivery_next_at.
func (s *TimelineService) UpdateTaskDelivery(taskID, deliveryStatus string, nextAt *time.Time) error {
	query := `UPDATE tasks SET delivery_status = ?, delivery_attempts = delivery_attempts + 1, delivery_next_at = ?, updated_at = datetime('now') WHERE task_id = ?`
	var nextAtVal interface{}
	if nextAt != nil {
		nextAtVal = *nextAt
	}
	_, err := s.db.Exec(query, deliveryStatus, nextAtVal, taskID)
	return err
}

// ListPendingDeliveries returns completed tasks that still need delivery.
func (s *TimelineService) ListPendingDeliveries(limit int) ([]AgentTask, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `SELECT id, task_id, COALESCE(idempotency_key,''), COALESCE(trace_id,''),
		channel, chat_id, COALESCE(sender_id,''), COALESCE(message_type,''), status,
		COALESCE(content_in,''), COALESCE(content_out,''), COALESCE(error_text,''),
		prompt_tokens, completion_tokens, total_tokens,
		delivery_status, delivery_attempts, delivery_next_at,
		created_at, updated_at, completed_at
	FROM tasks
	WHERE status = 'completed' AND delivery_status = 'pending'
		AND (delivery_next_at IS NULL OR delivery_next_at <= datetime('now'))
	ORDER BY created_at ASC
	LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending deliveries: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

// ListTasks returns tasks filtered by optional status and channel.
func (s *TimelineService) ListTasks(status, channel string, limit, offset int) ([]AgentTask, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, task_id, COALESCE(idempotency_key,''), COALESCE(trace_id,''),
		channel, chat_id, COALESCE(sender_id,''), COALESCE(message_type,''), status,
		COALESCE(content_in,''), COALESCE(content_out,''), COALESCE(error_text,''),
		prompt_tokens, completion_tokens, total_tokens,
		delivery_status, delivery_attempts, delivery_next_at,
		created_at, updated_at, completed_at
	FROM tasks WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if channel != "" {
		query += " AND channel = ?"
		args = append(args, channel)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]AgentTask, error) {
	var tasks []AgentTask
	for rows.Next() {
		var t AgentTask
		var deliveryNextAt, completedAt sql.NullTime
		err := rows.Scan(
			&t.ID, &t.TaskID, &t.IdempotencyKey, &t.TraceID,
			&t.Channel, &t.ChatID, &t.SenderID, &t.MessageType, &t.Status,
			&t.ContentIn, &t.ContentOut, &t.ErrorText,
			&t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
			&t.DeliveryStatus, &t.DeliveryAttempts, &deliveryNextAt,
			&t.CreatedAt, &t.UpdatedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}
		if deliveryNextAt.Valid {
			t.DeliveryNextAt = &deliveryNextAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// UpdateTaskTokens adds token usage to a task.
func (s *TimelineService) UpdateTaskTokens(taskID string, prompt, completion, total int) error {
	_, err := s.db.Exec(`UPDATE tasks SET
		prompt_tokens = prompt_tokens + ?,
		completion_tokens = completion_tokens + ?,
		total_tokens = total_tokens + ?,
		updated_at = datetime('now')
	WHERE task_id = ?`, prompt, completion, total, taskID)
	return err
}

// GetDailyTokenUsage returns total tokens used today across all tasks.
func (s *TimelineService) GetDailyTokenUsage() (int, error) {
	var total int
	err := s.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM tasks WHERE created_at >= date('now')`).Scan(&total)
	return total, err
}

// LogPolicyDecision records a policy evaluation result.
func (s *TimelineService) LogPolicyDecision(rec *PolicyDecisionRecord) error {
	_, err := s.db.Exec(`INSERT INTO policy_decisions (trace_id, task_id, tool, tier, sender, channel, allowed, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.TraceID, rec.TaskID, rec.Tool, rec.Tier, rec.Sender, rec.Channel, rec.Allowed, rec.Reason)
	return err
}

// ListPolicyDecisions returns policy decisions matching the given trace_id.
func (s *TimelineService) ListPolicyDecisions(traceID string) ([]PolicyDecisionRecord, error) {
	rows, err := s.db.Query(`SELECT id, COALESCE(trace_id,''), COALESCE(task_id,''), tool, tier,
		COALESCE(sender,''), COALESCE(channel,''), allowed, COALESCE(reason,''), created_at
		FROM policy_decisions WHERE trace_id = ? ORDER BY created_at ASC`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PolicyDecisionRecord
	for rows.Next() {
		var r PolicyDecisionRecord
		if err := rows.Scan(&r.ID, &r.TraceID, &r.TaskID, &r.Tool, &r.Tier,
			&r.Sender, &r.Channel, &r.Allowed, &r.Reason, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetTaskByTraceID returns the first task matching the given trace_id (nil if not found).
func (s *TimelineService) GetTaskByTraceID(traceID string) (*AgentTask, error) {
	row := s.db.QueryRow(`SELECT id, task_id, COALESCE(idempotency_key,''), COALESCE(trace_id,''),
		channel, chat_id, COALESCE(sender_id,''), status, COALESCE(content_in,''), COALESCE(content_out,''),
		COALESCE(error_text,''), COALESCE(delivery_status,'pending'), delivery_attempts,
		delivery_next_at, prompt_tokens, completion_tokens, total_tokens,
		created_at, updated_at, completed_at
		FROM tasks WHERE trace_id = ? LIMIT 1`, traceID)
	var t AgentTask
	var nextAt, completedAt *string
	err := row.Scan(&t.ID, &t.TaskID, &t.IdempotencyKey, &t.TraceID,
		&t.Channel, &t.ChatID, &t.SenderID, &t.Status, &t.ContentIn, &t.ContentOut,
		&t.ErrorText, &t.DeliveryStatus, &t.DeliveryAttempts,
		&nextAt, &t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
		&t.CreatedAt, &t.UpdatedAt, &completedAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	if nextAt != nil {
		parsed, _ := time.Parse("2006-01-02 15:04:05", *nextAt)
		t.DeliveryNextAt = &parsed
	}
	if completedAt != nil {
		parsed, _ := time.Parse("2006-01-02 15:04:05", *completedAt)
		t.CompletedAt = &parsed
	}
	return &t, nil
}

// --- Trace Graph ---

// GetTraceGraph builds a trace graph from timeline events and group traces for a trace ID.
func (s *TimelineService) GetTraceGraph(traceID string) (*TraceGraph, error) {
	// Local events
	events, err := s.GetEvents(FilterArgs{Limit: 500, TraceID: traceID})
	if err != nil {
		return nil, fmt.Errorf("get trace graph events: %w", err)
	}

	var nodes []TraceNode
	edgeMap := make(map[string]string) // child -> parent

	for _, e := range events {
		spanType := "EVENT"
		switch {
		case e.Classification != "" && (contains(e.Classification, "INBOUND") || e.SenderName == "User"):
			spanType = "INBOUND"
		case e.Classification != "" && (contains(e.Classification, "OUTBOUND") || e.SenderName == "Agent"):
			spanType = "OUTBOUND"
		case contains(e.Classification, "LLM"):
			spanType = "LLM"
		case contains(e.Classification, "TOOL"):
			spanType = "TOOL"
		case contains(e.Classification, "POLICY"):
			spanType = "POLICY"
		}
		node := TraceNode{
			ID:           e.EventID,
			SpanID:       e.SpanID,
			ParentSpanID: e.ParentSpanID,
			Type:         spanType,
			Title:        e.Classification,
			StartTime:    e.Timestamp.Format("15:04:05"),
			AgentID:      "local",
		}
		nodes = append(nodes, node)
		if e.ParentSpanID != "" && e.SpanID != "" {
			edgeMap[e.SpanID] = e.ParentSpanID
		}
	}

	// Remote group traces
	remoteTraces, err := s.GetGroupTraces(traceID)
	if err == nil {
		for _, gt := range remoteTraces {
			node := TraceNode{
				ID:           fmt.Sprintf("remote-%d", gt.ID),
				SpanID:       gt.SpanID,
				ParentSpanID: gt.ParentSpanID,
				Type:         gt.SpanType,
				Title:        gt.Title,
				DurationMs:   gt.DurationMs,
				AgentID:      gt.SourceAgentID,
			}
			if gt.StartedAt != nil {
				node.StartTime = gt.StartedAt.Format("15:04:05")
			}
			if gt.EndedAt != nil {
				node.EndTime = gt.EndedAt.Format("15:04:05")
			}
			nodes = append(nodes, node)
			if gt.ParentSpanID != "" && gt.SpanID != "" {
				edgeMap[gt.SpanID] = gt.ParentSpanID
			}
		}
	}

	var edges []TraceEdge
	for child, parent := range edgeMap {
		edges = append(edges, TraceEdge{Source: parent, Target: child})
	}

	// Task info
	var taskInfo map[string]any
	if task, err := s.GetTaskByTraceID(traceID); err == nil && task != nil {
		taskInfo = map[string]any{
			"task_id":           task.TaskID,
			"status":            task.Status,
			"delivery_status":   task.DeliveryStatus,
			"prompt_tokens":     task.PromptTokens,
			"completion_tokens": task.CompletionTokens,
			"total_tokens":      task.TotalTokens,
			"channel":           task.Channel,
			"created_at":        task.CreatedAt,
			"completed_at":      task.CompletedAt,
		}
	}

	// Policy decisions
	var policyDecisions []map[string]any
	if decisions, err := s.ListPolicyDecisions(traceID); err == nil {
		for _, d := range decisions {
			policyDecisions = append(policyDecisions, map[string]any{
				"tool":    d.Tool,
				"tier":    d.Tier,
				"allowed": d.Allowed,
				"reason":  d.Reason,
				"time":    d.CreatedAt.Format("15:04:05"),
			})
		}
	}

	return &TraceGraph{
		Nodes:           nodes,
		Edges:           edges,
		Task:            taskInfo,
		PolicyDecisions: policyDecisions,
	}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- Group Traces ---

// AddGroupTrace inserts a remote agent trace span.
func (s *TimelineService) AddGroupTrace(gt *GroupTrace) error {
	_, err := s.db.Exec(`INSERT INTO group_traces
		(trace_id, source_agent_id, span_id, parent_span_id, span_type, title, content, started_at, ended_at, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		gt.TraceID, gt.SourceAgentID, gt.SpanID, gt.ParentSpanID, gt.SpanType,
		gt.Title, gt.Content, gt.StartedAt, gt.EndedAt, gt.DurationMs)
	return err
}

// GetGroupTraces returns remote trace spans for a trace ID.
func (s *TimelineService) GetGroupTraces(traceID string) ([]GroupTrace, error) {
	rows, err := s.db.Query(`SELECT id, trace_id, source_agent_id,
		COALESCE(span_id,''), COALESCE(parent_span_id,''), COALESCE(span_type,''),
		COALESCE(title,''), COALESCE(content,''), started_at, ended_at,
		COALESCE(duration_ms,0), created_at
		FROM group_traces WHERE trace_id = ? ORDER BY created_at ASC`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupTrace
	for rows.Next() {
		var gt GroupTrace
		var startedAt, endedAt sql.NullTime
		if err := rows.Scan(&gt.ID, &gt.TraceID, &gt.SourceAgentID,
			&gt.SpanID, &gt.ParentSpanID, &gt.SpanType,
			&gt.Title, &gt.Content, &startedAt, &endedAt,
			&gt.DurationMs, &gt.CreatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			gt.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			gt.EndedAt = &endedAt.Time
		}
		out = append(out, gt)
	}
	return out, rows.Err()
}

// --- Group Members ---

// UpsertGroupMember inserts or updates a group member in the local roster.
func (s *TimelineService) UpsertGroupMember(m *GroupMemberRecord) error {
	_, err := s.db.Exec(`INSERT INTO group_members
		(agent_id, agent_name, soul_summary, capabilities, channels, model, status, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(agent_id) DO UPDATE SET
			agent_name = excluded.agent_name,
			soul_summary = excluded.soul_summary,
			capabilities = excluded.capabilities,
			channels = excluded.channels,
			model = excluded.model,
			status = excluded.status,
			last_seen = datetime('now')`,
		m.AgentID, m.AgentName, m.SoulSummary, m.Capabilities, m.Channels, m.Model, m.Status)
	return err
}

// ListGroupMembers returns all group members.
func (s *TimelineService) ListGroupMembers() ([]GroupMemberRecord, error) {
	rows, err := s.db.Query(`SELECT agent_id, COALESCE(agent_name,''), COALESCE(soul_summary,''),
		COALESCE(capabilities,'[]'), COALESCE(channels,'[]'), COALESCE(model,''),
		COALESCE(status,'active'), last_seen
		FROM group_members ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupMemberRecord
	for rows.Next() {
		var m GroupMemberRecord
		if err := rows.Scan(&m.AgentID, &m.AgentName, &m.SoulSummary,
			&m.Capabilities, &m.Channels, &m.Model, &m.Status, &m.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RemoveGroupMember removes a member from the roster.
func (s *TimelineService) RemoveGroupMember(agentID string) error {
	_, err := s.db.Exec(`DELETE FROM group_members WHERE agent_id = ?`, agentID)
	return err
}

// MarkStaleMembers marks members not seen since the given cutoff as "stale".
func (s *TimelineService) MarkStaleMembers(cutoff time.Time) (int64, error) {
	result, err := s.db.Exec(`UPDATE group_members SET status = 'stale' WHERE last_seen < ? AND status = 'active'`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Group Tasks ---

// InsertGroupTask inserts a new group collaboration task.
func (s *TimelineService) InsertGroupTask(task *GroupTaskRecord) error {
	_, err := s.db.Exec(`INSERT INTO group_tasks
		(task_id, description, content, direction, requester_id, responder_id, response_content, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.TaskID, task.Description, task.Content, task.Direction,
		task.RequesterID, task.ResponderID, task.ResponseContent, task.Status)
	return err
}

// UpdateGroupTaskResponse updates a group task with response data.
func (s *TimelineService) UpdateGroupTaskResponse(taskID, responderID, content, status string) error {
	_, err := s.db.Exec(`UPDATE group_tasks SET
		responder_id = ?, response_content = ?, status = ?, responded_at = datetime('now')
		WHERE task_id = ?`,
		responderID, content, status, taskID)
	return err
}

// ListGroupTasks returns group tasks filtered by direction and status.
func (s *TimelineService) ListGroupTasks(direction, status string, limit, offset int) ([]GroupTaskRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, task_id, COALESCE(description,''), COALESCE(content,''),
		direction, requester_id, COALESCE(responder_id,''),
		COALESCE(response_content,''), status, created_at, responded_at
		FROM group_tasks WHERE 1=1`
	args := []interface{}{}

	if direction != "" {
		query += " AND direction = ?"
		args = append(args, direction)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupTaskRecord
	for rows.Next() {
		var t GroupTaskRecord
		var respondedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.TaskID, &t.Description, &t.Content,
			&t.Direction, &t.RequesterID, &t.ResponderID,
			&t.ResponseContent, &t.Status, &t.CreatedAt, &respondedAt); err != nil {
			return nil, err
		}
		if respondedAt.Valid {
			t.RespondedAt = &respondedAt.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListAllGroupTraces returns paginated group traces with optional agent filter.
func (s *TimelineService) ListAllGroupTraces(limit, offset int, agentFilter string) ([]GroupTrace, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, trace_id, source_agent_id,
		COALESCE(span_id,''), COALESCE(parent_span_id,''), COALESCE(span_type,''),
		COALESCE(title,''), COALESCE(content,''), started_at, ended_at,
		COALESCE(duration_ms,0), created_at
		FROM group_traces WHERE 1=1`
	args := []interface{}{}

	if agentFilter != "" {
		query += " AND source_agent_id = ?"
		args = append(args, agentFilter)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupTrace
	for rows.Next() {
		var gt GroupTrace
		var startedAt, endedAt sql.NullTime
		if err := rows.Scan(&gt.ID, &gt.TraceID, &gt.SourceAgentID,
			&gt.SpanID, &gt.ParentSpanID, &gt.SpanType,
			&gt.Title, &gt.Content, &startedAt, &endedAt,
			&gt.DurationMs, &gt.CreatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			gt.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			gt.EndedAt = &endedAt.Time
		}
		out = append(out, gt)
	}
	return out, rows.Err()
}
