package group

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kamir/gomikrobot/internal/bus"
	"github.com/kamir/gomikrobot/internal/timeline"
)

// Consumer reads messages from Kafka topics.
type Consumer interface {
	// Start begins consuming from the configured topics.
	Start(ctx context.Context) error
	// Messages returns a channel of raw messages.
	Messages() <-chan ConsumerMessage
	// Close stops the consumer.
	Close() error
}

// ConsumerMessage is a raw message from Kafka.
type ConsumerMessage struct {
	Topic string
	Key   []byte
	Value []byte
}

// GroupRouter routes incoming Kafka messages to the appropriate handler.
type GroupRouter struct {
	manager  *Manager
	msgBus   *bus.MessageBus
	consumer Consumer
	topics   TopicNames
}

// NewGroupRouter creates a router that bridges Kafka messages into the bus.
func NewGroupRouter(manager *Manager, msgBus *bus.MessageBus, consumer Consumer) *GroupRouter {
	return &GroupRouter{
		manager:  manager,
		msgBus:   msgBus,
		consumer: consumer,
		topics:   manager.Topics(),
	}
}

// Run starts consuming and routing messages. Blocks until context is cancelled.
func (r *GroupRouter) Run(ctx context.Context) error {
	if err := r.consumer.Start(ctx); err != nil {
		return fmt.Errorf("group router: start consumer: %w", err)
	}
	defer r.consumer.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-r.consumer.Messages():
			if !ok {
				return nil
			}
			r.handleMessage(msg)
		}
	}
}

func (r *GroupRouter) handleMessage(msg ConsumerMessage) {
	var env GroupEnvelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.Warn("GroupRouter: unmarshal envelope", "error", err, "topic", msg.Topic)
		return
	}

	// Skip our own messages
	if env.SenderID == r.manager.identity.AgentID {
		return
	}

	switch msg.Topic {
	case r.topics.Announce:
		r.manager.HandleAnnounce(&env)

	case r.topics.Requests:
		r.handleTaskRequest(&env)

	case r.topics.Responses:
		r.handleTaskResponse(&env)

	case r.topics.Traces:
		r.handleTrace(&env)

	default:
		slog.Debug("GroupRouter: unknown topic", "topic", msg.Topic)
	}
}

func (r *GroupRouter) handleTaskRequest(env *GroupEnvelope) {
	data, err := json.Marshal(env.Payload)
	if err != nil {
		return
	}
	var payload TaskRequestPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	// Route into the agent's inbound bus as a "group" channel message
	r.msgBus.PublishInbound(&bus.InboundMessage{
		Channel:        "group",
		SenderID:       payload.RequesterID,
		ChatID:         payload.TaskID,
		TraceID:        env.CorrelationID,
		IdempotencyKey: fmt.Sprintf("group:%s", payload.TaskID),
		Content:        payload.Content,
		Timestamp:      time.Now(),
		Metadata: map[string]any{
			"group_task_id":   payload.TaskID,
			"group_requester": payload.RequesterID,
			"description":     payload.Description,
		},
	})

	slog.Info("GroupRouter: task request routed to bus",
		"task_id", payload.TaskID, "from", payload.RequesterID)
}

func (r *GroupRouter) handleTaskResponse(env *GroupEnvelope) {
	data, err := json.Marshal(env.Payload)
	if err != nil {
		return
	}
	var payload TaskResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	slog.Info("GroupRouter: task response received",
		"task_id", payload.TaskID, "from", payload.ResponderID, "status", payload.Status)

	// Route into bus as a group response
	r.msgBus.PublishInbound(&bus.InboundMessage{
		Channel:        "group",
		SenderID:       payload.ResponderID,
		ChatID:         payload.TaskID,
		TraceID:        env.CorrelationID,
		IdempotencyKey: fmt.Sprintf("group-resp:%s:%s", payload.TaskID, payload.ResponderID),
		Content:        fmt.Sprintf("[Task Response from %s] Status: %s\n%s", payload.ResponderID, payload.Status, payload.Content),
		Timestamp:      time.Now(),
	})
}

func (r *GroupRouter) handleTrace(env *GroupEnvelope) {
	data, err := json.Marshal(env.Payload)
	if err != nil {
		return
	}
	var payload TracePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}

	// Store in group_traces table if timeline is available
	if r.manager.timeline != nil {
		var startedAt, endedAt *time.Time
		if payload.StartedAt != "" {
			if t, err := time.Parse(time.RFC3339, payload.StartedAt); err == nil {
				startedAt = &t
			}
		}
		if payload.EndedAt != "" {
			if t, err := time.Parse(time.RFC3339, payload.EndedAt); err == nil {
				endedAt = &t
			}
		}

		_ = r.manager.timeline.AddGroupTrace(&timeline.GroupTrace{
			TraceID:       payload.TraceID,
			SourceAgentID: env.SenderID,
			SpanID:        payload.SpanID,
			ParentSpanID:  payload.ParentSpanID,
			SpanType:      payload.SpanType,
			Title:         payload.Title,
			Content:       payload.Content,
			StartedAt:     startedAt,
			EndedAt:       endedAt,
			DurationMs:    payload.DurationMs,
		})
	}

	slog.Debug("GroupRouter: trace stored", "trace_id", payload.TraceID, "from", env.SenderID)
}
