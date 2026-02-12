package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kamir/gomikrobot/internal/bus"
	"github.com/kamir/gomikrobot/internal/config"
	"github.com/kamir/gomikrobot/internal/provider"
	"github.com/kamir/gomikrobot/internal/timeline"
	"github.com/skip2/go-qrcode"

	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// WhatsAppChannel implements a native WhatsApp client.
type WhatsAppChannel struct {
	BaseChannel
	client    *whatsmeow.Client
	config    config.WhatsAppConfig
	container *sqlstore.Container
	provider  provider.LLMProvider
	timeline  *timeline.TimelineService
	sendFn    func(ctx context.Context, msg *bus.OutboundMessage) error
	mu        sync.Mutex
}

// NewWhatsAppChannel creates a new WhatsApp channel.
func NewWhatsAppChannel(cfg config.WhatsAppConfig, messageBus *bus.MessageBus, prov provider.LLMProvider, tl *timeline.TimelineService) *WhatsAppChannel {
	return &WhatsAppChannel{
		BaseChannel: BaseChannel{Bus: messageBus},
		config:      cfg,
		provider:    prov,
		timeline:    tl,
	}
}

func (c *WhatsAppChannel) Name() string { return "whatsapp" }

func (c *WhatsAppChannel) Start(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	// Setup logging
	dbLog := waLog.Stdout("Database", "WARN", true)
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Initialize database
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".gomikrobot", "whatsapp.db")

	os.MkdirAll(filepath.Dir(dbPath), 0755)

	// sqlstore.New(ctx, driver, url, log)
	container, err := sqlstore.New(ctx, "sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbLog)
	if err != nil {
		return fmt.Errorf("failed to init whatsapp db: %w", err)
	}
	c.container = container

	// Get first device
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Create client
	c.client = whatsmeow.NewClient(deviceStore, clientLog)
	c.client.AddEventHandler(c.eventHandler)

	// Login if needed
	if c.client.Store.ID == nil {
		// No session, need to pair
		qrChan, _ := c.client.GetQRChannel(context.Background())
		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		fmt.Println("WhatsApp: Scan this QR code to login:")
		for evt := range qrChan {
			if evt.Event == "code" {
				// 1. Save to file
				home, _ := os.UserHomeDir()
				qrPath := filepath.Join(home, ".gomikrobot", "whatsapp-qr.png")
				err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrPath)
				if err == nil {
					fmt.Printf("\n🖼️  WhatsApp Login QR Code saved to: %s\n", qrPath)
					fmt.Println("Please open this file on your computer and scan it with your phone.")
				}
			} else {
				fmt.Println("WhatsApp: Login event:", evt.Event)
			}
		}
	} else {
		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		fmt.Println("WhatsApp: Connected")
	}

	// Subscribe to outbound messages
	c.Bus.Subscribe(c.Name(), func(msg *bus.OutboundMessage) {
		go func() {
			c.handleOutbound(msg)
		}()
	})

	return nil
}

func (c *WhatsAppChannel) Stop() error {
	if c.client != nil {
		c.client.Disconnect()
	}
	if c.container != nil {
		c.container.Close()
	}
	return nil
}

func (c *WhatsAppChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("client not initialized")
	}

	jid, err := types.ParseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	// Use Protobuf message
	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	_, err = c.client.SendMessage(ctx, jid, waMsg)

	return err
}

func (c *WhatsAppChannel) handleOutbound(msg *bus.OutboundMessage) {
	// Check silent mode — never send if enabled
	if c.timeline != nil && c.timeline.IsSilentMode() {
		fmt.Printf("🔇 Silent Mode: suppressed outbound to %s reason=silent_mode channel=%s\n", msg.ChatID, c.Name())
		return
	}
	sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.sendOutbound(sendCtx, msg); err != nil {
		fmt.Printf("Error sending whatsapp message: %v\n", err)
	}
}

func (c *WhatsAppChannel) sendOutbound(ctx context.Context, msg *bus.OutboundMessage) error {
	if c.sendFn != nil {
		return c.sendFn(ctx, msg)
	}
	return c.Send(ctx, msg)
}

func (c *WhatsAppChannel) eventHandler(evt interface{}) {
	// SUPER DEBUG: Log every event type
	// fmt.Printf("🔔 WhatsApp Event: %T\n", evt)

	switch v := evt.(type) {
	case *events.Message:
		// Improved content extraction
		content := ""
		mediaPath := "" // Declare outside scope

		if v.Message.GetConversation() != "" {
			content = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage().GetText() != "" {
			content = v.Message.GetExtendedTextMessage().GetText()
		} else if v.Message.GetImageMessage() != nil {
			content = "[Image Message]"
			img := v.Message.GetImageMessage()
			data, err := c.client.Download(context.Background(), img)
			if err == nil {
				ext := "jpg"
				if strings.Contains(img.GetMimetype(), "png") {
					ext = "png"
				}
				fileName := fmt.Sprintf("%s.%s", v.Info.ID, ext)
				home, _ := os.UserHomeDir()
				dirPath := filepath.Join(home, ".gomikrobot", "workspace", "media", "images")
				os.MkdirAll(dirPath, 0755) // Ensure dir exists
				filePath := filepath.Join(dirPath, fileName)
				os.WriteFile(filePath, data, 0644)

				mediaPath = filePath
				fmt.Printf("📸 Image saved to %s\n", filePath)

				// Optional: Describe image using Vision API? (For later)
			} else {
				fmt.Printf("❌ Image download error: %v\n", err)
			}
		} else if v.Message.GetAudioMessage() != nil {
			content = "[Audio Message]"
			audio := v.Message.GetAudioMessage()
			data, err := c.client.Download(context.Background(), audio)
			if err == nil {
				ext := "ogg"
				if strings.Contains(audio.GetMimetype(), "mp4") {
					ext = "m4a"
				}
				fileName := fmt.Sprintf("%s.%s", v.Info.ID, ext)
				home, _ := os.UserHomeDir()
				filePath := filepath.Join(home, ".gomikrobot", "workspace", "media", "audio", fileName)
				os.WriteFile(filePath, data, 0644)

				mediaPath = filePath // Capture it

				fmt.Printf("🔊 Audio saved to %s\n", filePath)

				// Transcribe
				transcript, err := c.provider.Transcribe(context.Background(), &provider.AudioRequest{
					FilePath: filePath,
				})
				if err == nil {
					fmt.Printf("📝 Transcript: %s\n", transcript.Text)
					content = "[Audio Transcript]: " + transcript.Text
					// Note: Transcript echo removed - no automatic response
				} else {
					fmt.Printf("❌ Transcription error: %v\n", err)
				}
			} else {
				fmt.Printf("❌ Download error: %v\n", err)
			}
		} else if v.Message.GetDocumentMessage() != nil {
			doc := v.Message.GetDocumentMessage()
			docTitle := doc.GetTitle()
			if docTitle == "" {
				docTitle = doc.GetFileName()
			}
			content = fmt.Sprintf("[Document: %s]", docTitle)

			data, err := c.client.Download(context.Background(), doc)
			if err == nil {
				// Determine extension from mimetype or filename
				ext := "bin"
				if doc.GetFileName() != "" {
					parts := strings.Split(doc.GetFileName(), ".")
					if len(parts) > 1 {
						ext = parts[len(parts)-1]
					}
				} else if strings.Contains(doc.GetMimetype(), "pdf") {
					ext = "pdf"
				}

				fileName := fmt.Sprintf("%s.%s", v.Info.ID, ext)
				home, _ := os.UserHomeDir()
				dirPath := filepath.Join(home, ".gomikrobot", "workspace", "media", "documents")
				os.MkdirAll(dirPath, 0755)
				filePath := filepath.Join(dirPath, fileName)
				os.WriteFile(filePath, data, 0644)

				mediaPath = filePath
				fmt.Printf("📄 Document saved to %s (%s, %d bytes)\n", filePath, doc.GetMimetype(), len(data))
			} else {
				fmt.Printf("❌ Document download error: %v\n", err)
			}
		} else {
			// Fallback: try to see if there's any text at all
			content = v.Message.String()
			fmt.Printf("🔍 Unknown message structure, raw: %s\n", content)
		}

		fmt.Printf("📩 Message Event from %s (IsFromMe: %v)\n", v.Info.Sender, v.Info.IsFromMe)
		fmt.Printf("📝 Content: %s\n", content)

		// For testing: allow messages from self (but we should normally block this to avoid loops)
		// If you want to disable self-chat again later, uncomment the block below.
		/*
			if v.Info.IsFromMe {
				return
			}
		*/

		sender := v.Info.Sender.User
		isAuthorized := c.isAllowed(sender)

		if !isAuthorized {
			fmt.Printf("🚫 Unauthorized sender: %s\n", sender)
			// Continue to process and log, but don't respond or publish to bus
		}

		if content == "" {
			return
		}

		// Classify intent (for logging purposes only - no automatic responses)
		category, _ := c.classifyMessage(context.Background(), content)

		// Log Inbound Event (with authorization status)
		c.logEvent(v.Info.ID, sender, "TEXT", content, mediaPath, category, isAuthorized)

		// Publish to bus only if authorized
		if isAuthorized {
			c.Bus.PublishInbound(&bus.InboundMessage{
				Channel:   c.Name(),
				SenderID:  sender,
				ChatID:    v.Info.Chat.String(),
				Content:   content,
				Timestamp: v.Info.Timestamp,
			})
		}
	}
}

func (c *WhatsAppChannel) logEvent(evtID, sender, evtType, content, media, classification string, authorized bool) {
	if c.timeline == nil {
		return
	}
	err := c.timeline.AddEvent(&timeline.TimelineEvent{
		EventID:        evtID,
		Timestamp:      time.Now(), // or v.Info.Timestamp if available
		SenderID:       sender,
		SenderName:     "User", // TODO: Lookup contact name
		EventType:      evtType,
		ContentText:    content,
		MediaPath:      media,
		Classification: classification,
		Authorized:     authorized,
	})
	if err != nil {
		fmt.Printf("⚠️ Failed to log timeline event: %v\n", err)
	}
}

func shorten(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// Simple heuristic to detect English
func isEnglish(text string) bool {
	commonEnglish := []string{"the", "and", "is", "in", "to", "for", "with", "you", "are", "what", "how", "why", "who", "hello", "hi"}
	lower := strings.ToLower(text)
	score := 0
	words := strings.Fields(lower)
	if len(words) == 0 {
		return false
	}

	for _, w := range words {
		for _, eng := range commonEnglish {
			if w == eng {
				score++
			}
		}
	}

	// If meaningful percentage of words are common English stop words
	return float64(score)/float64(len(words)) > 0.15
}

func (c *WhatsAppChannel) isAllowed(sender string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}
	for _, allowed := range c.config.AllowFrom {
		if allowed == sender {
			return true
		}
	}
	return false
}

// classifyMessage uses the LLM to classify the intent of the message.
func (c *WhatsAppChannel) classifyMessage(ctx context.Context, content string) (category string, summary string) {
	sysPrompt := `You are an intent classifier for a personal agent.
Classify the user message into one of these categories:
1. EMERGENCY: Danger, critical system failure, urgent health issue.
2. APPOINTMENT: Request for a meeting, call, or scheduling.
3. ASSISTANCE: General questions, research, coding help, or anything else.

Output logic:
- Return a JSON object: {"category": "...", "summary": "..."}
- "summary" should be a very short (max 10 words) summary of the request content.
- If it's a technical question, it's ASSISTANCE.
- If it's "The server is down!", it's EMERGENCY.
- If it's "Can we meet at 5?", it's APPOINTMENT.
`

	resp, err := c.provider.Chat(ctx, &provider.ChatRequest{
		Model: "gpt-4o",
		Messages: []provider.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: content},
		},
		MaxTokens: 100,
	})

	if err != nil {
		fmt.Printf("⚠️ Classification failed: %v\n", err)
		return "ASSISTANCE", shorten(content, 30)
	}

	// Clean code blocks if present
	txt := strings.TrimSpace(resp.Content)
	txt = strings.TrimPrefix(txt, "```json")
	txt = strings.TrimSuffix(txt, "```")

	var result struct {
		Category string `json:"category"`
		Summary  string `json:"summary"`
	}

	// Fallback to simple unmarshal or just default
	if err := json.Unmarshal([]byte(txt), &result); err != nil {
		fmt.Printf("⚠️ JSON parse error: %v (raw: %s)\n", err, txt)
		return "ASSISTANCE", shorten(content, 30)
	}

	return strings.ToUpper(result.Category), result.Summary
}
