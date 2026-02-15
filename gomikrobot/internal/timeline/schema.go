package timeline

import (
	"time"
)

// TimelineEvent represents a single interaction in the history.
type TimelineEvent struct {
	ID             int64     `json:"id"`
	EventID        string    `json:"event_id"`       // Unique ID (e.g. WhatsApp MessageID)
	TraceID        string    `json:"trace_id"`       // End-to-end trace identifier
	SpanID         string    `json:"span_id"`        // Span identifier (optional)
	ParentSpanID   string    `json:"parent_span_id"` // Parent span (optional)
	Timestamp      time.Time `json:"timestamp"`      // When it happened
	SenderID       string    `json:"sender_id"`      // Phone number
	SenderName     string    `json:"sender_name"`    // Display name
	EventType      string    `json:"event_type"`     // TEXT, AUDIO, IMAGE, SYSTEM
	ContentText    string    `json:"content_text"`   // The text or transcript
	MediaPath      string    `json:"media_path"`     // Path to local file if any
	VectorID       string    `json:"vector_id"`      // Qdrant ID
	Classification string    `json:"classification"` // ABM1 Category
	Authorized     bool      `json:"authorized"`     // Whether sender is in AllowFrom list
}

// WebUser represents a user identity in the Web UI.
type WebUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ForceSend bool      `json:"force_send"`
	CreatedAt time.Time `json:"created_at"`
}

// WebLink maps a WebUser to a WhatsApp JID.
type WebLink struct {
	WebUserID   int64     `json:"web_user_id"`
	WhatsAppJID string    `json:"whatsapp_jid"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentTask represents a tracked agent processing task.
type AgentTask struct {
	ID               int64      `json:"id"`
	TaskID           string     `json:"task_id"`
	IdempotencyKey   string     `json:"idempotency_key,omitempty"`
	TraceID          string     `json:"trace_id,omitempty"`
	Channel          string     `json:"channel"`
	ChatID           string     `json:"chat_id"`
	SenderID         string     `json:"sender_id,omitempty"`
	MessageType      string     `json:"message_type,omitempty"`
	Status           string     `json:"status"`
	ContentIn        string     `json:"content_in,omitempty"`
	ContentOut       string     `json:"content_out,omitempty"`
	ErrorText        string     `json:"error_text,omitempty"`
	DeliveryStatus   string     `json:"delivery_status"`
	PromptTokens     int        `json:"prompt_tokens"`
	CompletionTokens int        `json:"completion_tokens"`
	TotalTokens      int        `json:"total_tokens"`
	DeliveryAttempts int        `json:"delivery_attempts"`
	DeliveryNextAt   *time.Time `json:"delivery_next_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

const (
	TaskStatusPending    = "pending"
	TaskStatusProcessing = "processing"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"

	DeliveryPending = "pending"
	DeliverySent    = "sent"
	DeliveryFailed  = "failed"
	DeliverySkipped = "skipped"
)

// TraceNode represents a node in the trace graph.
type TraceNode struct {
	ID           string  `json:"id"`
	SpanID       string  `json:"span_id"`
	ParentSpanID string  `json:"parent_span_id"`
	Type         string  `json:"type"` // INBOUND, LLM, TOOL, OUTBOUND, POLICY
	Title        string  `json:"title"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	DurationMs   int     `json:"duration_ms"`
	AgentID      string  `json:"agent_id"`
	Output       string  `json:"output"`
}

// TraceEdge represents an edge in the trace graph.
type TraceEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// TraceGraph is the full graph response for a trace.
type TraceGraph struct {
	Nodes           []TraceNode    `json:"nodes"`
	Edges           []TraceEdge    `json:"edges"`
	Task            map[string]any `json:"task"`
	PolicyDecisions []map[string]any `json:"policy_decisions"`
}

// GroupTrace represents a trace span from a remote agent.
type GroupTrace struct {
	ID            int64     `json:"id"`
	TraceID       string    `json:"trace_id"`
	SourceAgentID string    `json:"source_agent_id"`
	SpanID        string    `json:"span_id"`
	ParentSpanID  string    `json:"parent_span_id"`
	SpanType      string    `json:"span_type"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	DurationMs    int       `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

// GroupMemberRecord represents a persisted group member in the database.
type GroupMemberRecord struct {
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	SoulSummary  string    `json:"soul_summary"`
	Capabilities string    `json:"capabilities"` // JSON array
	Channels     string    `json:"channels"`     // JSON array
	Model        string    `json:"model"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
}

// GroupTaskRecord represents a group collaboration task.
type GroupTaskRecord struct {
	ID              int64      `json:"id"`
	TaskID          string     `json:"task_id"`
	Description     string     `json:"description"`
	Content         string     `json:"content"`
	Direction       string     `json:"direction"` // "outgoing" | "incoming"
	RequesterID     string     `json:"requester_id"`
	ResponderID     string     `json:"responder_id"`
	ResponseContent string     `json:"response_content"`
	Status          string     `json:"status"` // pending/completed/failed/rejected
	CreatedAt       time.Time  `json:"created_at"`
	RespondedAt     *time.Time `json:"responded_at,omitempty"`
}

// PolicyDecisionRecord represents a logged policy evaluation.
type PolicyDecisionRecord struct {
	ID        int64     `json:"id"`
	TraceID   string    `json:"trace_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Tool      string    `json:"tool"`
	Tier      int       `json:"tier"`
	Sender    string    `json:"sender,omitempty"`
	Channel   string    `json:"channel,omitempty"`
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

const Schema = `
CREATE TABLE IF NOT EXISTS timeline (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	event_id TEXT UNIQUE,
	trace_id TEXT,
	span_id TEXT,
	parent_span_id TEXT,
	timestamp DATETIME,
	sender_id TEXT,
	sender_name TEXT,
	event_type TEXT,
	content_text TEXT,
	media_path TEXT,
	vector_id TEXT,
	classification TEXT,
	authorized BOOLEAN DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_timeline_timestamp ON timeline(timestamp);
CREATE INDEX IF NOT EXISTS idx_timeline_sender ON timeline(sender_id);
CREATE INDEX IF NOT EXISTS idx_timeline_authorized ON timeline(authorized);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT,
	updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS web_users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE,
	force_send BOOLEAN DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS web_links (
	web_user_id INTEGER PRIMARY KEY,
	whatsapp_jid TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_web_links_whatsapp ON web_links(whatsapp_jid);

CREATE TABLE IF NOT EXISTS tasks (
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
	prompt_tokens INTEGER NOT NULL DEFAULT 0,
	completion_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	delivery_status TEXT NOT NULL DEFAULT 'pending',
	delivery_attempts INTEGER NOT NULL DEFAULT 0,
	delivery_next_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	completed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_idempotency ON tasks(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_tasks_trace ON tasks(trace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_delivery ON tasks(delivery_status, delivery_next_at);

CREATE TABLE IF NOT EXISTS policy_decisions (
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
);
CREATE INDEX IF NOT EXISTS idx_policy_trace ON policy_decisions(trace_id);
CREATE INDEX IF NOT EXISTS idx_policy_task ON policy_decisions(task_id);

CREATE TABLE IF NOT EXISTS memory_chunks (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	embedding BLOB,
	source TEXT NOT NULL DEFAULT 'user',
	tags TEXT DEFAULT '',
	version INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_memory_chunks_source ON memory_chunks(source);

CREATE TABLE IF NOT EXISTS group_traces (
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
);
CREATE INDEX IF NOT EXISTS idx_group_traces_trace ON group_traces(trace_id);
CREATE INDEX IF NOT EXISTS idx_group_traces_agent ON group_traces(source_agent_id);

CREATE TABLE IF NOT EXISTS group_members (
	agent_id TEXT UNIQUE NOT NULL,
	agent_name TEXT,
	soul_summary TEXT,
	capabilities TEXT DEFAULT '[]',
	channels TEXT DEFAULT '[]',
	model TEXT,
	status TEXT DEFAULT 'active',
	last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS group_tasks (
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
);
CREATE INDEX IF NOT EXISTS idx_group_tasks_direction ON group_tasks(direction);
CREATE INDEX IF NOT EXISTS idx_group_tasks_status ON group_tasks(status);
`
