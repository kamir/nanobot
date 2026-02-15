package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/kamir/gomikrobot/internal/agent"
	"github.com/kamir/gomikrobot/internal/bus"
	"github.com/kamir/gomikrobot/internal/channels"
	"github.com/kamir/gomikrobot/internal/config"
	"github.com/kamir/gomikrobot/internal/group"
	"github.com/kamir/gomikrobot/internal/memory"
	"github.com/kamir/gomikrobot/internal/policy"
	"github.com/kamir/gomikrobot/internal/provider"
	"github.com/kamir/gomikrobot/internal/timeline"
	"github.com/kamir/gomikrobot/internal/tools"
	"github.com/spf13/cobra"
)

func newTraceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the agent gateway (WhatsApp, etc)",
	Run:   runGateway,
}

func runGateway(cmd *cobra.Command, args []string) {
	printHeader("🌐 GoMikroBot Gateway")
	fmt.Println("Starting GoMikroBot Gateway...")

	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}
	// 2. Setup Timeline (QMD)
	home, _ := os.UserHomeDir()
	timelinePath := fmt.Sprintf("%s/.gomikrobot/timeline.db", home)
	timeSvc, err := timeline.NewTimelineService(timelinePath)
	if err != nil {
		fmt.Printf("Failed to init timeline: %v\n", err)
		os.Exit(1)
	}

	// Seed default settings if missing
	seedSetting := func(key, value string) {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return
		}
		if v, err := timeSvc.GetSetting(key); err == nil && strings.TrimSpace(v) != "" {
			return
		}
		_ = timeSvc.SetSetting(key, value)
	}
	seedSetting("bot_repo_path", "/Users/kamir/GITHUB.kamir/nanobot/gomikrobot")
	seedSetting("default_work_repo_path", filepath.Join(home, "GoMikroBot-Workspace"))
	seedSetting("default_repo_search_path", home)
	seedSetting("kafscale_lfs_proxy_url", "http://localhost:8080")

	// Resolve work repo path (settings override config)
	workRepoPath := cfg.Paths.WorkRepoPath
	if v, err := timeSvc.GetSetting("work_repo_path"); err == nil && strings.TrimSpace(v) != "" {
		workRepoPath = strings.TrimSpace(v)
	}
	if warn, err := config.EnsureWorkRepo(workRepoPath); err != nil {
		fmt.Printf("Work repo error: %v\n", err)
	} else if warn != "" {
		fmt.Printf("Work repo warning: %s\n", warn)
	}
	var workRepoMu sync.RWMutex
	getWorkRepo := func() string {
		workRepoMu.RLock()
		defer workRepoMu.RUnlock()
		return workRepoPath
	}
	// Resolve system repo path (settings override config)
	systemRepoPath := cfg.Paths.SystemRepoPath
	if v, err := timeSvc.GetSetting("bot_repo_path"); err == nil && strings.TrimSpace(v) != "" {
		systemRepoPath = strings.TrimSpace(v)
	}

	// Helper: resolve repo from query param (?repo=identity → systemRepoPath, else work repo)
	resolveRepo := func(r *http.Request) string {
		if r.URL.Query().Get("repo") == "identity" {
			return systemRepoPath
		}
		return getWorkRepo()
	}

	// 3. Setup Bus
	msgBus := bus.NewMessageBus()

	// 4. Setup Providers
	oaProv := provider.NewOpenAIProvider(cfg.Providers.OpenAI.APIKey, cfg.Providers.OpenAI.APIBase, cfg.Model.Name)
	var prov provider.LLMProvider = oaProv

	if cfg.Providers.LocalWhisper.Enabled {
		prov = provider.NewLocalWhisperProvider(cfg.Providers.LocalWhisper, oaProv)
	}

	// 4b. Setup Policy Engine
	policyEngine := policy.NewDefaultEngine()
	// Allow Tier 2 (shell) by default for the personal bot — the shell tool
	// already has its own deny-pattern and allow-list safety layer.
	policyEngine.MaxAutoTier = 2
	// External users (non-owner) are restricted to read-only tools (tier 0).
	policyEngine.ExternalMaxTier = 0

	// 4c. Setup Memory System (requires Embedder-capable provider)
	var memorySvc *memory.MemoryService
	if embedder, ok := prov.(provider.Embedder); ok {
		vecStore := memory.NewSQLiteVecStore(timeSvc.DB(), 1536)
		memorySvc = memory.NewMemoryService(vecStore, embedder)
		fmt.Println("🧠 Memory system initialized")
	} else {
		fmt.Println("ℹ️  Memory system disabled (provider does not support embeddings)")
	}

	// 4d. Setup Group Collaboration (conditional)
	grpState := &groupState{}

	// Helper: build a group manager from config + settings
	buildGrpManager := func(grpCfg config.GroupConfig) *group.Manager {
		if grpCfg.LFSProxyURL == "" || grpCfg.LFSProxyURL == "http://localhost:8080" {
			if url, err := timeSvc.GetSetting("kafscale_lfs_proxy_url"); err == nil && url != "" {
				grpCfg.LFSProxyURL = url
			}
		}
		registry := tools.NewRegistry()
		ctxBuilder := agent.NewContextBuilder(cfg.Paths.Workspace, workRepoPath, systemRepoPath, registry)
		agentID := grpCfg.AgentID
		if agentID == "" {
			hostname, _ := os.Hostname()
			agentID = fmt.Sprintf("gomikrobot-%s", hostname)
		}
		identity := ctxBuilder.BuildIdentityEnvelope(agentID, "GoMikroBot", cfg.Model.Name)
		return group.NewManager(grpCfg, timeSvc, identity)
	}

	// Helper: start Kafka consumer + router, returns cancel func
	startGrpKafka := func(grpCfg config.GroupConfig, mgr *group.Manager, parentCtx context.Context) context.CancelFunc {
		if grpCfg.KafkaBrokers == "" {
			return func() {}
		}
		kafkaCtx, kafkaCancel := context.WithCancel(parentCtx)
		topics := mgr.Topics()
		consumerGroup := grpCfg.ConsumerGroup
		if consumerGroup == "" {
			consumerGroup = grpCfg.AgentID
			if consumerGroup == "" {
				hostname, _ := os.Hostname()
				consumerGroup = fmt.Sprintf("gomikrobot-%s", hostname)
			}
		}
		kafkaConsumer := group.NewKafkaConsumer(
			grpCfg.KafkaBrokers,
			consumerGroup,
			[]string{topics.Announce, topics.Requests, topics.Responses, topics.Traces},
		)
		router := group.NewGroupRouter(mgr, msgBus, kafkaConsumer)
		go func() {
			if err := router.Run(kafkaCtx); err != nil {
				fmt.Printf("⚠️ Group router stopped: %v\n", err)
			}
		}()
		fmt.Println("📡 Kafka consumer started for group topics")
		return kafkaCancel
	}

	if cfg.Group.Enabled && cfg.Group.GroupName != "" {
		mgr := buildGrpManager(cfg.Group)
		grpState.SetManager(mgr, nil)
		fmt.Println("🤝 Group collaboration enabled:", cfg.Group.GroupName)
	} else {
		// Check if group was activated via settings
		if active, err := timeSvc.GetSetting("group_active"); err == nil && active == "true" {
			if gn, err := timeSvc.GetSetting("group_name"); err == nil && gn != "" {
				cfg.Group.GroupName = gn
				cfg.Group.Enabled = true
				mgr := buildGrpManager(cfg.Group)
				grpState.SetManager(mgr, nil)
				fmt.Println("🤝 Group collaboration restored from settings:", cfg.Group.GroupName)
			}
		}
	}

	// Build group publisher for the loop (nil-safe)
	var groupPublisher agent.GroupTracePublisher
	if grpState.Manager() != nil {
		groupPublisher = &groupTraceAdapter{mgr: grpState.Manager()}
	}

	// 5. Setup Loop
	loop := agent.NewLoop(agent.LoopOptions{
		Bus:            msgBus,
		Provider:       prov,
		Timeline:       timeSvc,
		Policy:         policyEngine,
		MemoryService:  memorySvc,
		GroupPublisher: groupPublisher,
		Workspace:      cfg.Paths.Workspace,
		WorkRepo:       workRepoPath,
		SystemRepo:     systemRepoPath,
		WorkRepoGetter: getWorkRepo,
		Model:          cfg.Model.Name,
		MaxIterations:  cfg.Model.MaxToolIterations,
	})

	// 5b. Index soul files (non-blocking background)
	if memorySvc != nil {
		go func() {
			indexer := memory.NewSoulFileIndexer(memorySvc, cfg.Paths.Workspace)
			if err := indexer.IndexAll(context.Background()); err != nil {
				fmt.Printf("⚠️ Soul file indexing error: %v\n", err)
			}
		}()
	}

	// 6. Setup Channels
	// WhatsApp
	wa := channels.NewWhatsAppChannel(cfg.Channels.WhatsApp, msgBus, prov, timeSvc)

	// 7. Start Everything
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start Channels
	if err := wa.Start(ctx); err != nil {
		fmt.Printf("Failed to start WhatsApp: %v\n", err)
	}

	// Route web UI outbound to WhatsApp and timeline
	msgBus.Subscribe("webui", func(msg *bus.OutboundMessage) {
		go func() {
			webUserID, err := strconv.ParseInt(msg.ChatID, 10, 64)
			if err != nil {
				fmt.Printf("⚠️ webui outbound invalid web_user_id: %s\n", msg.ChatID)
				return
			}
			jid, ok, err := timeSvc.GetWebLink(webUserID)
			if err != nil {
				fmt.Printf("⚠️ webui outbound link lookup error: %v\n", err)
			}
			status := "no_link"
			jid = strings.TrimSpace(jid)
			if ok && jid != "" {
				jid = normalizeWhatsAppJID(jid)
				status = "queued"
			} else {
				fmt.Printf("⚠️ webui outbound no WhatsApp link for web_user_id=%d\n", webUserID)
			}

			// Check silent mode and optional override
			forceSend := true
			if user, err := timeSvc.GetWebUser(webUserID); err == nil {
				forceSend = user.ForceSend
			}
			if status != "no_link" && timeSvc.IsSilentMode() && !forceSend {
				fmt.Printf("🔇 webui outbound suppressed (silent mode) to %s web_user_id=%d\n", jid, webUserID)
				status = "suppressed"
			} else if status != "no_link" {
				// Send via WhatsApp channel; bypass silent when forceSend is enabled
				if timeSvc.IsSilentMode() && forceSend {
					sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := wa.Send(sendCtx, &bus.OutboundMessage{
						Channel: "whatsapp",
						ChatID:  jid,
						Content: msg.Content,
					}); err != nil {
						fmt.Printf("⚠️ webui outbound direct send error: %v\n", err)
						status = "error"
					} else {
						status = "sent"
					}
				} else {
					msgBus.PublishOutbound(&bus.OutboundMessage{
						Channel: "whatsapp",
						ChatID:  jid,
						Content: msg.Content,
					})
					status = "queued"
				}
			}

			// Log outbound to timeline for Web UI visibility (always)
			_ = timeSvc.AddEvent(&timeline.TimelineEvent{
				EventID:        fmt.Sprintf("WEBUI_ACK_%d", time.Now().UnixNano()),
				Timestamp:      time.Now(),
				SenderID:       "AGENT",
				SenderName:     "Agent",
				EventType:      "SYSTEM",
				ContentText:    msg.Content,
				Classification: fmt.Sprintf("WEBUI_OUTBOUND->%s force=%v status=%s", jid, forceSend, status),
				Authorized:     true,
			})
			fmt.Printf("📤 WebUI outbound status=%s to=%s\n", status, jid)
		}()
	})

	// Start Delivery Worker
	deliveryWorker := agent.NewDeliveryWorker(timeSvc, msgBus)
	go deliveryWorker.Run(ctx)

	// Start Bus Dispatcher
	go msgBus.DispatchOutbound(ctx)

	// Start Local HTTP Server for Local Network access
	// Start Local HTTP Server for Local Network access (API)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			msg := r.URL.Query().Get("message")
			if msg == "" {
				http.Error(w, "Missing message parameter", http.StatusBadRequest)
				return
			}

			session := r.URL.Query().Get("session")
			if session == "" {
				session = "local:default"
			}

			fmt.Printf("🌐 Local Network Request: %s\n", msg)
			traceID := newTraceID()
			_ = timeSvc.AddEvent(&timeline.TimelineEvent{
				EventID:        fmt.Sprintf("LOCAL_IN_%d", time.Now().UnixNano()),
				TraceID:        traceID,
				Timestamp:      time.Now(),
				SenderID:       session,
				SenderName:     "Local",
				EventType:      "TEXT",
				ContentText:    msg,
				Classification: "LOCAL_INBOUND",
				Authorized:     true,
			})
			resp, err := loop.ProcessDirectWithTrace(ctx, msg, session, traceID)
			if err != nil {
				_ = timeSvc.AddEvent(&timeline.TimelineEvent{
					EventID:        fmt.Sprintf("LOCAL_OUT_%d", time.Now().UnixNano()),
					TraceID:        traceID,
					Timestamp:      time.Now(),
					SenderID:       "AGENT",
					SenderName:     "Agent",
					EventType:      "SYSTEM",
					ContentText:    err.Error(),
					Classification: "LOCAL_OUTBOUND status=error",
					Authorized:     true,
				})
				fmt.Printf("📤 Local outbound status=error session=%s\n", session)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = timeSvc.AddEvent(&timeline.TimelineEvent{
				EventID:        fmt.Sprintf("LOCAL_OUT_%d", time.Now().UnixNano()),
				TraceID:        traceID,
				Timestamp:      time.Now(),
				SenderID:       "AGENT",
				SenderName:     "Agent",
				EventType:      "SYSTEM",
				ContentText:    resp,
				Classification: "LOCAL_OUTBOUND status=sent",
				Authorized:     true,
			})
			fmt.Printf("📤 Local outbound status=sent session=%s\n", session)
			fmt.Fprint(w, resp)
		})

		addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
		fmt.Printf("📡 API Server listening on http://%s\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("API Server Error: %v\n", err)
		}
	}()

	// Start Dashboard Server
	go func() {
		mux := http.NewServeMux()

		// API: Timeline
		mux.HandleFunc("/api/v1/timeline", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit == 0 {
				limit = 100
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			sender := r.URL.Query().Get("sender")
			traceID := r.URL.Query().Get("trace_id")

			events, err := timeSvc.GetEvents(timeline.FilterArgs{
				Limit:    limit,
				Offset:   offset,
				SenderID: sender,
				TraceID:  traceID,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(events)
		})

		// API: Trace (GET)
		mux.HandleFunc("/api/v1/trace/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			traceID := strings.TrimPrefix(r.URL.Path, "/api/v1/trace/")
			traceID = strings.TrimSpace(traceID)
			if traceID == "" {
				http.Error(w, "trace_id required", http.StatusBadRequest)
				return
			}

			events, err := timeSvc.GetEvents(timeline.FilterArgs{
				Limit:   500,
				TraceID: traceID,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			type span struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Title    string `json:"title"`
				Time     string `json:"time"`
				Duration string `json:"duration"`
				Output   string `json:"output"`
			}

			spans := make([]span, 0, len(events))
			for _, e := range events {
				spanType := "EVENT"
				switch {
				case strings.Contains(e.Classification, "INBOUND") || e.SenderName == "User":
					spanType = "INBOUND"
				case strings.Contains(e.Classification, "OUTBOUND") || e.SenderName == "Agent":
					spanType = "OUTBOUND"
				case strings.Contains(e.Classification, "LLM"):
					spanType = "LLM"
				case strings.Contains(e.Classification, "TOOL"):
					spanType = "TOOL"
				}
				spans = append(spans, span{
					ID:       e.EventID,
					Type:     spanType,
					Title:    e.Classification,
					Time:     e.Timestamp.Format("15:04:05"),
					Duration: "",
					Output:   "",
				})
			}

			// Also fetch task + policy decisions for this trace
			var taskInfo map[string]any
			if task, err := timeSvc.GetTaskByTraceID(traceID); err == nil && task != nil {
				taskInfo = map[string]any{
					"task_id":          task.TaskID,
					"status":           task.Status,
					"delivery_status":  task.DeliveryStatus,
					"prompt_tokens":    task.PromptTokens,
					"completion_tokens": task.CompletionTokens,
					"total_tokens":     task.TotalTokens,
					"channel":          task.Channel,
					"created_at":       task.CreatedAt,
					"completed_at":     task.CompletedAt,
				}
			}

			var policyDecisions []map[string]any
			if decisions, err := timeSvc.ListPolicyDecisions(traceID); err == nil {
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

			json.NewEncoder(w).Encode(map[string]any{
				"trace_id":         traceID,
				"spans":            spans,
				"task":             taskInfo,
				"policy_decisions": policyDecisions,
			})
		})

		// API: Policy Decisions (GET)
		mux.HandleFunc("/api/v1/policy-decisions", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			traceID := r.URL.Query().Get("trace_id")
			if traceID == "" {
				http.Error(w, "trace_id required", http.StatusBadRequest)
				return
			}

			decisions, err := timeSvc.ListPolicyDecisions(traceID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(decisions)
		})

		// API: Trace Graph (GET)
		mux.HandleFunc("/api/v1/trace-graph/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			traceID := strings.TrimPrefix(r.URL.Path, "/api/v1/trace-graph/")
			traceID = strings.TrimSpace(traceID)
			if traceID == "" {
				http.Error(w, "trace_id required", http.StatusBadRequest)
				return
			}

			graph, err := timeSvc.GetTraceGraph(traceID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(graph)
		})

		// API: Group Status (GET)
		mux.HandleFunc("/api/v1/group/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			mgr := grpState.Manager()
			if mgr == nil {
				json.NewEncoder(w).Encode(map[string]any{
					"active":       false,
					"group_name":   "",
					"member_count": 0,
				})
				return
			}
			json.NewEncoder(w).Encode(mgr.Status())
		})

		// API: Group Members (GET)
		mux.HandleFunc("/api/v1/group/members", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			members, err := timeSvc.ListGroupMembers()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if members == nil {
				members = []timeline.GroupMemberRecord{}
			}
			json.NewEncoder(w).Encode(members)
		})

		// API: Group Join (POST)
		mux.HandleFunc("/api/v1/group/join", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var body struct {
				GroupName    string `json:"group_name"`
				LFSProxyURL string `json:"lfs_proxy_url"`
				KafkaBrokers string `json:"kafka_brokers"`
				AgentID      string `json:"agent_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			groupName := strings.TrimSpace(body.GroupName)
			if groupName == "" {
				http.Error(w, "group_name required", http.StatusBadRequest)
				return
			}

			// Leave existing group if active
			if mgr := grpState.Manager(); mgr != nil && mgr.Active() {
				leaveCtx, leaveCancel := context.WithTimeout(ctx, 5*time.Second)
				_ = mgr.Leave(leaveCtx)
				leaveCancel()
				grpState.Clear()
			}

			// Build new config
			grpCfg := cfg.Group
			grpCfg.GroupName = groupName
			grpCfg.Enabled = true
			if body.LFSProxyURL != "" {
				grpCfg.LFSProxyURL = body.LFSProxyURL
			}
			if body.KafkaBrokers != "" {
				grpCfg.KafkaBrokers = body.KafkaBrokers
			}
			if body.AgentID != "" {
				grpCfg.AgentID = body.AgentID
			}

			mgr := buildGrpManager(grpCfg)

			joinCtx, joinCancel := context.WithTimeout(ctx, 15*time.Second)
			defer joinCancel()
			if err := mgr.Join(joinCtx); err != nil {
				http.Error(w, fmt.Sprintf("join failed: %v", err), http.StatusInternalServerError)
				return
			}

			setupGroupBusSubscription(mgr, msgBus)
			kafkaCancel := startGrpKafka(grpCfg, mgr, ctx)
			grpState.SetManager(mgr, kafkaCancel)

			// Persist settings
			_ = timeSvc.SetSetting("group_name", groupName)
			_ = timeSvc.SetSetting("group_active", "true")
			if body.LFSProxyURL != "" {
				_ = timeSvc.SetSetting("kafscale_lfs_proxy_url", body.LFSProxyURL)
			}

			json.NewEncoder(w).Encode(mgr.Status())
		})

		// API: Group Leave (POST)
		mux.HandleFunc("/api/v1/group/leave", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			mgr := grpState.Manager()
			if mgr == nil {
				http.Error(w, "not in a group", http.StatusBadRequest)
				return
			}

			leaveCtx, leaveCancel := context.WithTimeout(ctx, 10*time.Second)
			defer leaveCancel()
			if err := mgr.Leave(leaveCtx); err != nil {
				http.Error(w, fmt.Sprintf("leave failed: %v", err), http.StatusInternalServerError)
				return
			}

			grpState.Clear()
			_ = timeSvc.SetSetting("group_active", "false")

			json.NewEncoder(w).Encode(map[string]string{"status": "left"})
		})

		// API: Group Config (GET/POST)
		mux.HandleFunc("/api/v1/group/config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}

			if r.Method == "GET" {
				mgr := grpState.Manager()
				if mgr == nil {
					json.NewEncoder(w).Encode(map[string]any{
						"enabled":       cfg.Group.Enabled,
						"group_name":    cfg.Group.GroupName,
						"lfs_proxy_url": cfg.Group.LFSProxyURL,
						"api_key":       maskSecret(cfg.Group.LFSProxyAPIKey),
						"kafka_brokers": cfg.Group.KafkaBrokers,
						"consumer_group": cfg.Group.ConsumerGroup,
						"agent_id":      cfg.Group.AgentID,
						"poll_interval_ms": cfg.Group.PollIntervalMs,
					})
					return
				}
				grpCfg := mgr.Config()
				json.NewEncoder(w).Encode(map[string]any{
					"enabled":       grpCfg.Enabled,
					"group_name":    grpCfg.GroupName,
					"lfs_proxy_url": grpCfg.LFSProxyURL,
					"api_key":       maskSecret(grpCfg.LFSProxyAPIKey),
					"kafka_brokers": grpCfg.KafkaBrokers,
					"consumer_group": grpCfg.ConsumerGroup,
					"agent_id":      mgr.AgentID(),
					"poll_interval_ms": grpCfg.PollIntervalMs,
				})
				return
			}

			if r.Method == "POST" {
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid body", http.StatusBadRequest)
					return
				}
				requiresRejoin := false
				for key, val := range body {
					switch key {
					case "lfs_proxy_url":
						_ = timeSvc.SetSetting("kafscale_lfs_proxy_url", val)
						requiresRejoin = true
					case "api_key":
						_ = timeSvc.SetSetting("kafscale_lfs_proxy_api_key", val)
						requiresRejoin = true
					case "kafka_brokers":
						_ = timeSvc.SetSetting("kafka_brokers", val)
						requiresRejoin = true
					case "consumer_group":
						_ = timeSvc.SetSetting("kafka_consumer_group", val)
						requiresRejoin = true
					case "agent_id":
						_ = timeSvc.SetSetting("group_agent_id", val)
						requiresRejoin = true
					case "poll_interval_ms":
						_ = timeSvc.SetSetting("group_poll_interval_ms", val)
					}
				}
				json.NewEncoder(w).Encode(map[string]any{
					"status":         "ok",
					"requires_rejoin": requiresRejoin,
				})
				return
			}

			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})

		// API: Group Tasks Submit (POST)
		mux.HandleFunc("/api/v1/group/tasks/submit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			mgr := grpState.Manager()
			if mgr == nil || !mgr.Active() {
				http.Error(w, "not in a group", http.StatusBadRequest)
				return
			}

			var body struct {
				Description string `json:"description"`
				Content     string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.Description) == "" {
				http.Error(w, "description required", http.StatusBadRequest)
				return
			}

			taskID := newTraceID()
			submitCtx, submitCancel := context.WithTimeout(ctx, 10*time.Second)
			defer submitCancel()
			if err := mgr.SubmitTask(submitCtx, taskID, body.Description, body.Content); err != nil {
				http.Error(w, fmt.Sprintf("submit failed: %v", err), http.StatusInternalServerError)
				return
			}

			// Persist to local DB
			_ = timeSvc.InsertGroupTask(&timeline.GroupTaskRecord{
				TaskID:      taskID,
				Description: body.Description,
				Content:     body.Content,
				Direction:   "outgoing",
				RequesterID: mgr.AgentID(),
				Status:      "pending",
			})

			json.NewEncoder(w).Encode(map[string]string{"status": "submitted", "task_id": taskID})
		})

		// API: Group Tasks List (GET)
		mux.HandleFunc("/api/v1/group/tasks", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			direction := r.URL.Query().Get("direction")
			status := r.URL.Query().Get("status")
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit == 0 {
				limit = 50
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

			tasks, err := timeSvc.ListGroupTasks(direction, status, limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if tasks == nil {
				tasks = []timeline.GroupTaskRecord{}
			}
			json.NewEncoder(w).Encode(tasks)
		})

		// API: Group Traces (GET)
		mux.HandleFunc("/api/v1/group/traces", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			agentID := r.URL.Query().Get("agent_id")
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit == 0 {
				limit = 50
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

			traces, err := timeSvc.ListAllGroupTraces(limit, offset, agentID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if traces == nil {
				traces = []timeline.GroupTrace{}
			}
			json.NewEncoder(w).Encode(traces)
		})

		// API: Settings (GET/POST)
		mux.HandleFunc("/api/v1/settings", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}

			if r.Method == "POST" {
				var body struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid body", http.StatusBadRequest)
					return
				}
				if err := timeSvc.SetSetting(body.Key, body.Value); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				fmt.Printf("⚙️ Setting changed: %s = %s\n", body.Key, body.Value)
				// Auto-reload WhatsApp auth when allowlist/denylist changes
				if body.Key == "whatsapp_allowlist" || body.Key == "whatsapp_denylist" || body.Key == "whatsapp_pair_token" {
					wa.ReloadAuth()
				}
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				return
			}

			// GET: return all requested settings
			key := r.URL.Query().Get("key")
			if key != "" {
				val, err := timeSvc.GetSetting(key)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"key": key, "value": ""})
					return
				}
				json.NewEncoder(w).Encode(map[string]string{"key": key, "value": val})
				return
			}
			// Return silent_mode by default
			json.NewEncoder(w).Encode(map[string]bool{"silent_mode": timeSvc.IsSilentMode()})
		})

		// API: Work Repo (GET/POST)
		mux.HandleFunc("/api/v1/workrepo", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}

			switch r.Method {
			case "GET":
				workRepoMu.RLock()
				current := workRepoPath
				workRepoMu.RUnlock()
				json.NewEncoder(w).Encode(map[string]string{"path": current})
			case "POST":
				var body struct {
					Path string `json:"path"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid body", http.StatusBadRequest)
					return
				}
				newPath := strings.TrimSpace(body.Path)
				if newPath == "" {
					http.Error(w, "path required", http.StatusBadRequest)
					return
				}
				// If multiple absolute paths got concatenated, keep the last one.
				if idx := strings.LastIndex(newPath, "/Users/"); idx > 0 {
					newPath = newPath[idx:]
				}
				if idx := strings.LastIndex(newPath, "C:\\"); idx > 0 {
					newPath = newPath[idx:]
				}
				if strings.HasPrefix(newPath, "~") {
					home, _ := os.UserHomeDir()
					newPath = filepath.Join(home, newPath[1:])
				}
				if !filepath.IsAbs(newPath) {
					if abs, err := filepath.Abs(newPath); err == nil {
						newPath = abs
					}
				}
				if warn, err := config.EnsureWorkRepo(newPath); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if warn != "" {
					fmt.Printf("Work repo warning: %s\n", warn)
				}
				if err := timeSvc.SetSetting("work_repo_path", newPath); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				workRepoMu.Lock()
				workRepoPath = newPath
				workRepoMu.Unlock()
				json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": newPath})
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// API: Repo Tree (GET)
		mux.HandleFunc("/api/v1/repo/tree", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			base := resolveRepo(r)
			repoPath := base
			sub := strings.TrimSpace(r.URL.Query().Get("path"))
			if sub != "" {
				repoPath = filepath.Join(repoPath, sub)
			}
			items, err := listRepoTree(repoPath, base)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(items)
		})

		// API: Repo File (GET)
		mux.HandleFunc("/api/v1/repo/file", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			repo := resolveRepo(r)
			rel := strings.TrimSpace(r.URL.Query().Get("path"))
			if rel == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			full := filepath.Join(repo, rel)
			if !isWithin(repo, full) {
				http.Error(w, "path outside repo", http.StatusBadRequest)
				return
			}
			data, err := os.ReadFile(full)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !utf8.Valid(data) {
				json.NewEncoder(w).Encode(map[string]string{"path": rel, "content": "[binary file]"})
				return
			}
			if len(data) > 200_000 {
				json.NewEncoder(w).Encode(map[string]string{"path": rel, "content": string(data[:200_000]) + "\n... (truncated)"})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"path": rel, "content": string(data)})
		})

		// API: Repo Status (GET)
		mux.HandleFunc("/api/v1/repo/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			rp := resolveRepo(r)
			out, err := runGit(rp, "status", "-sb")
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"status": "", "error": err.Error()})
				return
			}
			remote, _ := runGit(rp, "remote", "-v")
			json.NewEncoder(w).Encode(map[string]string{"status": out, "remote": remote})
		})

		// API: Repo Search (GET)
		mux.HandleFunc("/api/v1/repo/search", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			root, _ := timeSvc.GetSetting("default_repo_search_path")
			root = strings.TrimSpace(root)
			if root == "" {
				json.NewEncoder(w).Encode(map[string]any{"root": "", "repos": []string{}})
				return
			}
			if strings.HasPrefix(root, "~") {
				home, _ := os.UserHomeDir()
				root = filepath.Join(home, root[1:])
			}
			if abs, err := filepath.Abs(root); err == nil {
				root = abs
			}

			entries, err := os.ReadDir(root)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]any{"root": root, "repos": []string{}})
				return
			}

			repos := make([]string, 0, len(entries))
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				path := filepath.Join(root, e.Name())
				if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
					repos = append(repos, path)
				}
			}

			json.NewEncoder(w).Encode(map[string]any{"root": root, "repos": repos})
		})

		// API: GitHub Auth Status (GET)
		mux.HandleFunc("/api/v1/repo/gh-auth", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			out, err := runGh(resolveRepo(r), "auth", "status", "-h", "github.com")
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"status": "not_authenticated", "detail": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "detail": out})
		})

		// API: Repo Branches (GET)
		mux.HandleFunc("/api/v1/repo/branches", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			out, err := runGit(resolveRepo(r), "branch", "--format=%(refname:short)")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			lines := []string{}
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					lines = append(lines, line)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"branches": lines})
		})

		// API: Repo Checkout Branch (POST)
		mux.HandleFunc("/api/v1/repo/checkout", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			var body struct {
				Branch string `json:"branch"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			branch := strings.TrimSpace(body.Branch)
			if branch == "" {
				http.Error(w, "branch required", http.StatusBadRequest)
				return
			}
			out, err := runGit(resolveRepo(r), "checkout", branch)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"result": out})
		})

		// API: Repo Log (GET)
		mux.HandleFunc("/api/v1/repo/log", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			limit := strings.TrimSpace(r.URL.Query().Get("limit"))
			if limit == "" {
				limit = "20"
			}
			out, err := runGit(resolveRepo(r), "log", "--oneline", "-n", limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			lines := []string{}
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					lines = append(lines, line)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"commits": lines})
		})

		// API: Repo File Diff (GET)
		mux.HandleFunc("/api/v1/repo/diff-file", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			rel := strings.TrimSpace(r.URL.Query().Get("path"))
			if rel == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			out, err := runGit(resolveRepo(r), "diff", "--", rel)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"diff": out})
		})

		// API: Repo Diff (GET)
		mux.HandleFunc("/api/v1/repo/diff", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")
			rel := strings.TrimSpace(r.URL.Query().Get("path"))
			args := []string{"diff"}
			if rel != "" {
				args = append(args, "--", rel)
			}
			out, err := runGit(resolveRepo(r), args...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"diff": out})
		})

		// API: Repo Commit (POST)
		mux.HandleFunc("/api/v1/repo/commit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			msg := strings.TrimSpace(body.Message)
			if msg == "" {
				http.Error(w, "message required", http.StatusBadRequest)
				return
			}
			rp := resolveRepo(r)
			if _, err := runGit(rp, "add", "-A"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out, err := runGit(rp, "commit", "-m", msg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"result": out})
		})

		// API: Repo Pull (POST)
		mux.HandleFunc("/api/v1/repo/pull", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			out, err := runGit(resolveRepo(r), "pull", "--ff-only")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"result": out})
		})

		// API: Repo Push (POST)
		mux.HandleFunc("/api/v1/repo/push", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			out, err := runGit(resolveRepo(r), "push")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"result": out})
		})

		// API: Repo Init (POST)
		mux.HandleFunc("/api/v1/repo/init", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			var body struct {
				RemoteURL string `json:"remote_url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			repo := resolveRepo(r)
			if warn, err := config.EnsureWorkRepo(repo); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if warn != "" {
				fmt.Printf("Work repo warning: %s\n", warn)
			}
			if strings.TrimSpace(body.RemoteURL) != "" {
				_, _ = runGit(repo, "remote", "remove", "origin")
				if _, err := runGit(repo, "remote", "add", "origin", strings.TrimSpace(body.RemoteURL)); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		// API: Repo PR (POST) using gh
		mux.HandleFunc("/api/v1/repo/pr", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "OPTIONS" {
				return
			}
			var body struct {
				Title string `json:"title"`
				Body  string `json:"body"`
				Base  string `json:"base"`
				Head  string `json:"head"`
				Draft bool   `json:"draft"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.Title) == "" {
				http.Error(w, "title required", http.StatusBadRequest)
				return
			}
			args := []string{"pr", "create", "--title", body.Title, "--body", body.Body}
			if body.Base != "" {
				args = append(args, "--base", body.Base)
			}
			if body.Head != "" {
				args = append(args, "--head", body.Head)
			}
			if body.Draft {
				args = append(args, "--draft")
			}
			out, err := runGh(resolveRepo(r), args...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"result": out})
		})

		// API: Web Users (GET/POST)
		mux.HandleFunc("/api/v1/webusers", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}

			switch r.Method {
			case "GET":
				users, err := timeSvc.ListWebUsers()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if users == nil {
					users = []timeline.WebUser{}
				}
				if err := json.NewEncoder(w).Encode(users); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "POST":
				var body struct {
					Name string `json:"name"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid body", http.StatusBadRequest)
					return
				}
				user, err := timeSvc.CreateWebUser(strings.TrimSpace(body.Name))
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				json.NewEncoder(w).Encode(user)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// API: Web User Force Send (POST)
		mux.HandleFunc("/api/v1/webusers/force", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var body struct {
				WebUserID int64 `json:"web_user_id"`
				ForceSend bool  `json:"force_send"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if body.WebUserID == 0 {
				http.Error(w, "web_user_id required", http.StatusBadRequest)
				return
			}
			if err := timeSvc.SetWebUserForceSend(body.WebUserID, body.ForceSend); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		// API: Web Links (GET/POST)
		mux.HandleFunc("/api/v1/weblinks", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}

			switch r.Method {
			case "GET":
				idStr := r.URL.Query().Get("web_user_id")
				webUserID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					http.Error(w, "invalid web_user_id", http.StatusBadRequest)
					return
				}
				jid, ok, err := timeSvc.GetWebLink(webUserID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !ok {
					jid = ""
				}
				json.NewEncoder(w).Encode(map[string]string{
					"web_user_id":  idStr,
					"whatsapp_jid": jid,
				})
			case "POST":
				var body struct {
					WebUserID   int64  `json:"web_user_id"`
					WhatsAppJID string `json:"whatsapp_jid"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid body", http.StatusBadRequest)
					return
				}
				if body.WebUserID == 0 {
					http.Error(w, "web_user_id required", http.StatusBadRequest)
					return
				}
				if strings.TrimSpace(body.WhatsAppJID) == "" {
					if err := timeSvc.UnlinkWebUser(body.WebUserID); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				} else {
					jid := normalizeWhatsAppJID(strings.TrimSpace(body.WhatsAppJID))
					if err := timeSvc.LinkWebUser(body.WebUserID, jid); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// API: Web Chat Send
		mux.HandleFunc("/api/v1/webchat/send", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				return
			}
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var body struct {
				WebUserID int64  `json:"web_user_id"`
				Message   string `json:"message"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if body.WebUserID == 0 || strings.TrimSpace(body.Message) == "" {
				http.Error(w, "web_user_id and message required", http.StatusBadRequest)
				return
			}

			user, err := timeSvc.GetWebUser(body.WebUserID)
			if err != nil {
				http.Error(w, "web user not found", http.StatusBadRequest)
				return
			}
			traceID := newTraceID()

			// Resolve link (optional) and maybe forward the input itself to WhatsApp
			jid, ok, err := timeSvc.GetWebLink(body.WebUserID)
			if err != nil {
				http.Error(w, "link lookup failed", http.StatusInternalServerError)
				return
			}
			if ok && jid != "" {
				jid = normalizeWhatsAppJID(jid)
				forceSend := user.ForceSend
				status := "queued"

				if timeSvc.IsSilentMode() && !forceSend {
					fmt.Printf("🔇 webui input suppressed (silent mode) to %s web_user_id=%d\n", jid, body.WebUserID)
					status = "suppressed"
				} else if timeSvc.IsSilentMode() && forceSend {
					sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := wa.Send(sendCtx, &bus.OutboundMessage{
						Channel: "whatsapp",
						ChatID:  jid,
						TraceID: traceID,
						Content: body.Message,
					}); err != nil {
						fmt.Printf("⚠️ webui input direct send error: %v\n", err)
						status = "error"
					} else {
						status = "sent"
					}
				} else {
					msgBus.PublishOutbound(&bus.OutboundMessage{
						Channel: "whatsapp",
						ChatID:  jid,
						TraceID: traceID,
						Content: body.Message,
					})
					status = "queued"
				}

				_ = timeSvc.AddEvent(&timeline.TimelineEvent{
					EventID:        fmt.Sprintf("WEBUI_INPUT_ACK_%d", time.Now().UnixNano()),
					TraceID:        traceID,
					Timestamp:      time.Now(),
					SenderID:       "AGENT",
					SenderName:     "Agent",
					EventType:      "SYSTEM",
					ContentText:    body.Message,
					Classification: fmt.Sprintf("WEBUI_INPUT_OUTBOUND->%s force=%v status=%s", jid, forceSend, status),
					Authorized:     true,
				})
			}

			// Log inbound from Web UI
			_ = timeSvc.AddEvent(&timeline.TimelineEvent{
				EventID:        fmt.Sprintf("WEBUI_IN_%d", time.Now().UnixNano()),
				TraceID:        traceID,
				Timestamp:      time.Now(),
				SenderID:       fmt.Sprintf("webui:%s", user.Name),
				SenderName:     user.Name,
				EventType:      "TEXT",
				ContentText:    body.Message,
				Classification: "WEBUI_INBOUND",
				Authorized:     true,
			})

			// Publish inbound to agent
			msgBus.PublishInbound(&bus.InboundMessage{
				Channel:        "webui",
				SenderID:       fmt.Sprintf("webui:%s", user.Name),
				ChatID:         fmt.Sprintf("%d", body.WebUserID),
				TraceID:        traceID,
				IdempotencyKey: "web:" + traceID,
				Content:        body.Message,
				Timestamp:      time.Now(),
				Metadata: map[string]any{
					bus.MetaKeyMessageType: bus.MessageTypeExternal,
				},
			})

			json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
		})

		// API: Tasks List (GET)
		mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			status := r.URL.Query().Get("status")
			channel := r.URL.Query().Get("channel")
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit == 0 {
				limit = 50
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

			tasks, err := timeSvc.ListTasks(status, channel, limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if tasks == nil {
				tasks = []timeline.AgentTask{}
			}
			json.NewEncoder(w).Encode(tasks)
		})

		// API: Task Detail (GET)
		mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
			taskID = strings.TrimSpace(taskID)
			if taskID == "" {
				http.Error(w, "task_id required", http.StatusBadRequest)
				return
			}

			task, err := timeSvc.GetTask(taskID)
			if err != nil {
				http.Error(w, "task not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(task)
		})

		// Static: Media
		mediaDir := filepath.Join(cfg.Paths.Workspace, "media")
		fs := http.FileServer(http.Dir(mediaDir))
		mux.Handle("/media/", http.StripPrefix("/media/", fs))

		// SPA: Timeline
		mux.HandleFunc("/timeline", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "web/timeline.html")
		})

		// SPA: Group Management
		mux.HandleFunc("/group", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "web/group.html")
		})

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/timeline", http.StatusFound)
			}
		})

		if cfg.Gateway.DashboardPort == 0 {
			cfg.Gateway.DashboardPort = 18791
		}
		addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.DashboardPort)
		fmt.Printf("🖥️  Dashboard listening on http://%s\n", addr)

		// Startup Probe
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("❌ Dashboard Server FAILED to start: %v\n", err)
			cancel() // Stop the whole gateway if dashboard fails
		}
	}()

	// Start Agent Loop in background
	go func() {
		if err := loop.Run(ctx); err != nil {
			fmt.Printf("Agent loop crashed: %v\n", err)
			cancel()
		}
	}()

	// Start Group Collaboration (if configured)
	if mgr := grpState.Manager(); mgr != nil {
		// Subscribe bus for group outbound
		setupGroupBusSubscription(mgr, msgBus)

		// Join group
		go func() {
			joinCtx, joinCancel := context.WithTimeout(ctx, 15*time.Second)
			defer joinCancel()
			if err := mgr.Join(joinCtx); err != nil {
				fmt.Printf("⚠️ Group join failed: %v\n", err)
			} else {
				fmt.Printf("🤝 Joined group: %s\n", mgr.GroupName())
			}
		}()

		// Start Kafka consumer if brokers are configured
		kafkaCancel := startGrpKafka(cfg.Group, mgr, ctx)
		grpState.SetManager(mgr, kafkaCancel)
	}

	fmt.Println("Gateway running. Press Ctrl+C to stop.")
	<-sigChan

	fmt.Println("Shutting down...")
	// Leave group cleanly
	if mgr := grpState.Manager(); mgr != nil && mgr.Active() {
		leaveCtx, leaveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = mgr.Leave(leaveCtx)
		leaveCancel()
	}
	grpState.Clear()
	wa.Stop()
	loop.Stop()
	timeSvc.Close()
}

func normalizeWhatsAppJID(jid string) string {
	jid = strings.TrimSpace(jid)
	if jid == "" {
		return jid
	}
	if strings.Contains(jid, "@") {
		return jid
	}
	// Default to user JID.
	return jid + "@s.whatsapp.net"
}

type RepoItem struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Depth int    `json:"depth"`
	Size  int64  `json:"size"`
}

func listRepoTree(root, repoRoot string) ([]RepoItem, error) {
	items := []RepoItem{}
	base := repoRoot
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if path == root {
			return nil
		}
		rel, _ := filepath.Rel(base, path)
		rel = filepath.ToSlash(rel)
		depth := strings.Count(rel, "/")
		info, _ := d.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		itemType := "file"
		if d.IsDir() {
			itemType = "dir"
		}
		items = append(items, RepoItem{
			Path:  rel,
			Name:  d.Name(),
			Type:  itemType,
			Depth: depth,
			Size:  size,
		})
		return nil
	})
	return items, err
}

func runGit(repo string, args ...string) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("work repo not configured")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runGh(repo string, args ...string) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("work repo not configured")
	}
	cmd := exec.Command("gh", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

// groupState manages the lifecycle of the group manager at runtime.
type groupState struct {
	mu     sync.RWMutex
	mgr    *group.Manager
	cancel context.CancelFunc // cancels Kafka consumer goroutine
}

func (gs *groupState) Manager() *group.Manager {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.mgr
}

func (gs *groupState) SetManager(mgr *group.Manager, cancel context.CancelFunc) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.mgr = mgr
	gs.cancel = cancel
}

func (gs *groupState) Clear() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.cancel != nil {
		gs.cancel()
	}
	gs.mgr = nil
	gs.cancel = nil
}

// groupTraceAdapter adapts group.Manager to the agent.GroupTracePublisher interface.
type groupTraceAdapter struct {
	mgr *group.Manager
}

func (a *groupTraceAdapter) Active() bool {
	return a.mgr.Active()
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func (a *groupTraceAdapter) PublishTrace(ctx context.Context, payload interface{}) error {
	// Accept either TracePayload or map[string]string
	switch p := payload.(type) {
	case group.TracePayload:
		return a.mgr.PublishTrace(ctx, p)
	case map[string]string:
		tp := group.TracePayload{
			TraceID:  p["trace_id"],
			SpanType: p["span_type"],
			Title:    p["title"],
			Content:  p["content"],
		}
		return a.mgr.PublishTrace(ctx, tp)
	default:
		return fmt.Errorf("unsupported trace payload type: %T", payload)
	}
}
