package group

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kamir/gomikrobot/internal/config"
	"github.com/kamir/gomikrobot/internal/timeline"
)

// Manager handles group lifecycle: join, leave, heartbeat, roster management.
type Manager struct {
	cfg       config.GroupConfig
	lfs       *LFSClient
	timeline  *timeline.TimelineService
	identity  AgentIdentity
	topics    TopicNames
	roster    map[string]*GroupMember
	rosterMu  sync.RWMutex
	active    bool
	activeMu  sync.RWMutex
	cancelHB  context.CancelFunc
}

// NewManager creates a new group manager.
func NewManager(cfg config.GroupConfig, timeSvc *timeline.TimelineService, identity AgentIdentity) *Manager {
	lfs := NewLFSClient(cfg.LFSProxyURL, cfg.LFSProxyAPIKey)
	topics := Topics(cfg.GroupName)

	return &Manager{
		cfg:      cfg,
		lfs:      lfs,
		timeline: timeSvc,
		identity: identity,
		topics:   topics,
		roster:   make(map[string]*GroupMember),
	}
}

// Join announces this agent to the group and starts heartbeat.
func (m *Manager) Join(ctx context.Context) error {
	m.activeMu.Lock()
	defer m.activeMu.Unlock()

	if m.active {
		return fmt.Errorf("already joined group %s", m.cfg.GroupName)
	}

	// Announce join
	env := &GroupEnvelope{
		Type:          EnvelopeAnnounce,
		CorrelationID: fmt.Sprintf("join-%d", time.Now().UnixNano()),
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload: AnnouncePayload{
			Action:   "join",
			Identity: m.identity,
		},
	}

	if err := m.lfs.ProduceEnvelope(ctx, m.topics.Announce, env); err != nil {
		return fmt.Errorf("join announce failed: %w", err)
	}

	// Persist self as a member
	if m.timeline != nil {
		caps, _ := json.Marshal(m.identity.Capabilities)
		chs, _ := json.Marshal(m.identity.Channels)
		_ = m.timeline.UpsertGroupMember(&timeline.GroupMemberRecord{
			AgentID:      m.identity.AgentID,
			AgentName:    m.identity.AgentName,
			SoulSummary:  m.identity.SoulSummary,
			Capabilities: string(caps),
			Channels:     string(chs),
			Model:        m.identity.Model,
			Status:       "active",
		})
	}

	m.active = true

	// Start heartbeat
	hbCtx, cancel := context.WithCancel(context.Background())
	m.cancelHB = cancel
	go m.startHeartbeat(hbCtx)

	slog.Info("Joined group", "group", m.cfg.GroupName, "agent_id", m.identity.AgentID)
	return nil
}

// Leave announces departure and stops heartbeat.
func (m *Manager) Leave(ctx context.Context) error {
	m.activeMu.Lock()
	defer m.activeMu.Unlock()

	if !m.active {
		return fmt.Errorf("not in a group")
	}

	// Stop heartbeat
	if m.cancelHB != nil {
		m.cancelHB()
	}

	// Announce leave
	env := &GroupEnvelope{
		Type:          EnvelopeAnnounce,
		CorrelationID: fmt.Sprintf("leave-%d", time.Now().UnixNano()),
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload: AnnouncePayload{
			Action:   "leave",
			Identity: m.identity,
		},
	}

	if err := m.lfs.ProduceEnvelope(ctx, m.topics.Announce, env); err != nil {
		slog.Warn("Leave announce failed", "error", err)
	}

	// Remove self from roster db
	if m.timeline != nil {
		_ = m.timeline.RemoveGroupMember(m.identity.AgentID)
	}

	m.active = false
	m.rosterMu.Lock()
	m.roster = make(map[string]*GroupMember)
	m.rosterMu.Unlock()

	slog.Info("Left group", "group", m.cfg.GroupName)
	return nil
}

// Active returns whether this agent is in a group.
func (m *Manager) Active() bool {
	m.activeMu.RLock()
	defer m.activeMu.RUnlock()
	return m.active
}

// GroupName returns the current group name.
func (m *Manager) GroupName() string {
	return m.cfg.GroupName
}

// Members returns the current roster.
func (m *Manager) Members() []GroupMember {
	m.rosterMu.RLock()
	defer m.rosterMu.RUnlock()

	out := make([]GroupMember, 0, len(m.roster))
	for _, member := range m.roster {
		out = append(out, *member)
	}
	return out
}

// MemberCount returns the number of known members.
func (m *Manager) MemberCount() int {
	m.rosterMu.RLock()
	defer m.rosterMu.RUnlock()
	return len(m.roster)
}

// Status returns a summary of the group state.
func (m *Manager) Status() map[string]any {
	m.activeMu.RLock()
	active := m.active
	m.activeMu.RUnlock()

	healthy := m.lfs.Healthy(context.Background())

	return map[string]any{
		"active":        active,
		"group_name":    m.cfg.GroupName,
		"agent_id":      m.identity.AgentID,
		"member_count":  m.MemberCount(),
		"lfs_proxy_url": m.cfg.LFSProxyURL,
		"lfs_healthy":   healthy,
	}
}

// HandleAnnounce processes an incoming announce message and updates the roster.
func (m *Manager) HandleAnnounce(env *GroupEnvelope) {
	data, err := json.Marshal(env.Payload)
	if err != nil {
		slog.Warn("HandleAnnounce: marshal payload", "error", err)
		return
	}
	var payload AnnouncePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		slog.Warn("HandleAnnounce: unmarshal payload", "error", err)
		return
	}

	id := payload.Identity
	switch payload.Action {
	case "join", "heartbeat":
		member := &GroupMember{
			AgentID:      id.AgentID,
			AgentName:    id.AgentName,
			SoulSummary:  id.SoulSummary,
			Capabilities: id.Capabilities,
			Channels:     id.Channels,
			Model:        id.Model,
			Status:       id.Status,
			LastSeen:     time.Now(),
		}
		m.rosterMu.Lock()
		m.roster[id.AgentID] = member
		m.rosterMu.Unlock()

		// Persist to DB
		if m.timeline != nil {
			caps, _ := json.Marshal(id.Capabilities)
			chs, _ := json.Marshal(id.Channels)
			_ = m.timeline.UpsertGroupMember(&timeline.GroupMemberRecord{
				AgentID:      id.AgentID,
				AgentName:    id.AgentName,
				SoulSummary:  id.SoulSummary,
				Capabilities: string(caps),
				Channels:     string(chs),
				Model:        id.Model,
				Status:       "active",
			})
		}

		if payload.Action == "join" {
			slog.Info("Member joined", "agent_id", id.AgentID, "agent_name", id.AgentName)
			// Reply with our own heartbeat so the joiner learns about us
			go m.sendHeartbeat(context.Background())
		}

	case "leave":
		m.rosterMu.Lock()
		delete(m.roster, id.AgentID)
		m.rosterMu.Unlock()

		if m.timeline != nil {
			_ = m.timeline.RemoveGroupMember(id.AgentID)
		}
		slog.Info("Member left", "agent_id", id.AgentID)
	}
}

// PublishTrace publishes a trace span to the group traces topic.
func (m *Manager) PublishTrace(ctx context.Context, tracePayload TracePayload) error {
	if !m.Active() {
		return nil
	}
	env := &GroupEnvelope{
		Type:          EnvelopeTrace,
		CorrelationID: tracePayload.TraceID,
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload:       tracePayload,
	}
	return m.lfs.ProduceEnvelope(ctx, m.topics.Traces, env)
}

// SubmitTask sends a task request to the group.
func (m *Manager) SubmitTask(ctx context.Context, taskID, description, content string) error {
	if !m.Active() {
		return fmt.Errorf("not in a group")
	}
	env := &GroupEnvelope{
		Type:          EnvelopeRequest,
		CorrelationID: taskID,
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload: TaskRequestPayload{
			TaskID:      taskID,
			Description: description,
			Content:     content,
			RequesterID: m.identity.AgentID,
		},
	}
	return m.lfs.ProduceEnvelope(ctx, m.topics.Requests, env)
}

// RespondTask sends a task response to the group.
func (m *Manager) RespondTask(ctx context.Context, taskID, content, status string) error {
	if !m.Active() {
		return fmt.Errorf("not in a group")
	}
	env := &GroupEnvelope{
		Type:          EnvelopeResponse,
		CorrelationID: taskID,
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload: TaskResponsePayload{
			TaskID:      taskID,
			ResponderID: m.identity.AgentID,
			Content:     content,
			Status:      status,
		},
	}
	return m.lfs.ProduceEnvelope(ctx, m.topics.Responses, env)
}

// Config returns the current group configuration.
func (m *Manager) Config() config.GroupConfig {
	return m.cfg
}

// AgentID returns this agent's ID.
func (m *Manager) AgentID() string {
	return m.identity.AgentID
}

// LFSHealthy returns whether the LFS proxy is reachable.
func (m *Manager) LFSHealthy() bool {
	return m.lfs.Healthy(context.Background())
}

// Topics returns the topic names for the current group.
func (m *Manager) Topics() TopicNames {
	return m.topics
}

func (m *Manager) startHeartbeat(ctx context.Context) {
	interval := 30 * time.Second
	if m.cfg.PollIntervalMs > 0 {
		// Heartbeat interval = 15x poll interval (30s default at 2000ms poll)
		interval = time.Duration(m.cfg.PollIntervalMs*15) * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Also periodically mark stale members
	staleTicker := time.NewTicker(interval * 3)
	defer staleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sendHeartbeat(ctx)
		case <-staleTicker.C:
			if m.timeline != nil {
				cutoff := time.Now().Add(-interval * 3)
				if n, err := m.timeline.MarkStaleMembers(cutoff); err == nil && n > 0 {
					slog.Info("Marked stale members", "count", n)
				}
			}
		}
	}
}

func (m *Manager) sendHeartbeat(ctx context.Context) {
	if !m.Active() {
		return
	}
	env := &GroupEnvelope{
		Type:          EnvelopeAnnounce,
		CorrelationID: fmt.Sprintf("hb-%d", time.Now().UnixNano()),
		SenderID:      m.identity.AgentID,
		Timestamp:     time.Now(),
		Payload: AnnouncePayload{
			Action:   "heartbeat",
			Identity: m.identity,
		},
	}
	if err := m.lfs.ProduceEnvelope(ctx, m.topics.Announce, env); err != nil {
		slog.Debug("Heartbeat failed", "error", err)
	}
}
