package channels

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/kamir/gomikrobot/internal/bus"
	"github.com/kamir/gomikrobot/internal/config"
	"github.com/kamir/gomikrobot/internal/timeline"
)

func newTestTimeline(t *testing.T) *timeline.TimelineService {
	t.Helper()
	baseDir := t.TempDir()
	dbPath := filepath.Join(baseDir, "timeline.db")
	timeSvc, err := timeline.NewTimelineService(dbPath)
	if err != nil {
		t.Fatalf("failed to create timeline service: %v", err)
	}
	t.Cleanup(func() {
		_ = timeSvc.Close()
		_ = os.RemoveAll(baseDir)
	})
	return timeSvc
}

func TestWhatsAppSilentModeSuppressesOutbound(t *testing.T) {
	timeSvc := newTestTimeline(t)
	msgBus := bus.NewMessageBus()

	cfg := config.WhatsAppConfig{Enabled: true}
	wa := NewWhatsAppChannel(cfg, msgBus, nil, timeSvc)

	var called int32
	wa.sendFn = func(ctx context.Context, msg *bus.OutboundMessage) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	wa.handleOutbound(&bus.OutboundMessage{
		Channel: wa.Name(),
		ChatID:  "12345@s.whatsapp.net",
		Content: "test",
	})

	if atomic.LoadInt32(&called) != 0 {
		t.Fatalf("expected send to be suppressed in silent mode")
	}
}

func TestWhatsAppSilentModeDisabledAllowsOutbound(t *testing.T) {
	timeSvc := newTestTimeline(t)
	if err := timeSvc.SetSetting("silent_mode", "false"); err != nil {
		t.Fatalf("failed to disable silent mode: %v", err)
	}

	msgBus := bus.NewMessageBus()
	cfg := config.WhatsAppConfig{Enabled: true}
	wa := NewWhatsAppChannel(cfg, msgBus, nil, timeSvc)

	var called int32
	wa.sendFn = func(ctx context.Context, msg *bus.OutboundMessage) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	wa.handleOutbound(&bus.OutboundMessage{
		Channel: wa.Name(),
		ChatID:  "12345@s.whatsapp.net",
		Content: "test",
	})

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected send to occur when silent mode is disabled")
	}
}
