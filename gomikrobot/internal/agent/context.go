package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kamir/gomikrobot/internal/group"
	"github.com/kamir/gomikrobot/internal/provider"
	"github.com/kamir/gomikrobot/internal/session"
	"github.com/kamir/gomikrobot/internal/tools"
)

var bootstrapFiles = []string{
	"AGENTS.md",
	"SOUL.md",
	"USER.md",
	"TOOLS.md",
	"IDENTITY.md",
}

// ContextBuilder assembles the system prompt and messages.
type ContextBuilder struct {
	workspace string
	workRepo  string
	systemRepo string
	registry  *tools.Registry
}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder(workspace string, workRepo string, systemRepo string, registry *tools.Registry) *ContextBuilder {
	return &ContextBuilder{
		workspace:  workspace,
		workRepo:   workRepo,
		systemRepo: systemRepo,
		registry:   registry,
	}
}

// BuildSystemPrompt constructs the full system prompt from files and runtime info.
func (b *ContextBuilder) BuildSystemPrompt() string {
	var parts []string

	// 1. Core Identity & Runtime Info
	parts = append(parts, b.getIdentity())

	// 2. Bootstrap Files
	if bootstrap := b.loadBootstrapFiles(); bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	// 3. Memory
	if memory := b.loadMemory(); memory != "" {
		parts = append(parts, "# Memory\n\n"+memory)
	}

	// 4. Skills (Summary)
	if skills := b.buildSkillsSummary(); skills != "" {
		parts = append(parts, "# Skills\n\n"+skills)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

func (b *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")

	// Expand workspace path
	wsPath := b.workspace
	if strings.HasPrefix(wsPath, "~") {
		home, _ := os.UserHomeDir()
		wsPath = filepath.Join(home, wsPath[1:])
	}
	if abs, err := filepath.Abs(wsPath); err == nil {
		wsPath = abs
	}

	runtimeInfo := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	return fmt.Sprintf(`# GoMikroBot 🤖

You are GoMikroBot, a helpful, efficient AI assistant.
You have access to tools that allow you to:
- Read, write, and edit files
- Execute shell commands
- Search the web and fetch web pages
- Send messages to users

## Action Policy (Go Native)
When the user asks to create, plan, or document something, you must:
1. Create the required artifact(s) immediately in the repo.
2. Prefer these locations:
   - /requirements for behavior/specs
   - /tasks for plans and milestones
   - /docs for explanations or summaries
3. Report the exact file paths you wrote and a short summary.
Do not respond with advice-only when a concrete artifact is requested.
Writes are restricted to the work repo by default. To write elsewhere, prefix the path with '!'.
When asked to remember something, store it in /memory inside the work repo.

## Current Time
%s

## Runtime
%s

## Workspace
Your workspace is at: %s
- Work repo (exclusive write target): %s
- Memory files: %s/memory/MEMORY.md
- Daily notes: %s/memory/YYYY-MM-DD.md
- Custom skills: %s/skills/{skill-name}/SKILL.md

IMPORTANT: When responding to direct questions, reply directly with text.
Only use the 'message' tool when explicitly asked to send a message to a channel.
Always be helpful, accurate, and concise.
`, now, runtimeInfo, wsPath, b.workRepo, b.workRepo, b.workRepo, wsPath)
}

func (b *ContextBuilder) loadBootstrapFiles() string {
	var parts []string

	// Expand workspace
	wsPath := b.workspace
	if strings.HasPrefix(wsPath, "~") {
		home, _ := os.UserHomeDir()
		wsPath = filepath.Join(home, wsPath[1:])
	}

	for _, filename := range bootstrapFiles {
		path := filepath.Join(wsPath, filename)
		content, err := os.ReadFile(path)
		if err == nil {
			parts = append(parts, fmt.Sprintf("## %s\n\n%s", filename, string(content)))
		}
	}

	return strings.Join(parts, "\n\n")
}

func (b *ContextBuilder) loadMemory() string {
	// Prefer work repo memory
	base := b.workRepo
	if base == "" {
		base = b.workspace
	}
	if strings.HasPrefix(base, "~") {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, base[1:])
	}

	path := filepath.Join(base, "memory", "MEMORY.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func (b *ContextBuilder) buildSkillsSummary() string {
	// Simple summary for now - listing registered tools
	// In the future, this should scan the skills/ directory like the Python version
	// For now, we rely on the registry

	tools := b.registry.List()
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("You have the following tools available:\n")
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	// Auto-load skills from the bot system repo (not the work repo).
	if skills := b.loadSystemRepoSkills(); skills != "" {
		sb.WriteString("\n\nSystem repo skills:\n")
		sb.WriteString(skills)
	}

	return sb.String()
}

func (b *ContextBuilder) loadSystemRepoSkills() string {
	base := b.systemRepoPath()
	if base == "" {
		return ""
	}

	var sb strings.Builder

	// 1) skills/*/SKILL.md
	skillsDir := filepath.Join(base, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(skillsDir, e.Name(), "SKILL.md")
			if data, err := os.ReadFile(path); err == nil {
				sb.WriteString(fmt.Sprintf("\n## %s\n\n%s\n", e.Name(), string(data)))
			}
		}
	}

	// 2) day2day guidance (if present)
	day2day := filepath.Join(base, "operations", "day2day", "README.md")
	if data, err := os.ReadFile(day2day); err == nil {
		sb.WriteString("\n## Day2Day Guidance\n\n")
		sb.WriteString(string(data))
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func (b *ContextBuilder) systemRepoPath() string {
	if b.systemRepo != "" {
		path := b.systemRepo
		if strings.HasPrefix(path, "~") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, path[1:])
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	wsPath := b.workspace
	if strings.HasPrefix(wsPath, "~") {
		home, _ := os.UserHomeDir()
		wsPath = filepath.Join(home, wsPath[1:])
	}
	if abs, err := filepath.Abs(wsPath); err == nil {
		wsPath = abs
	}
	path := filepath.Join(wsPath, "Bottibot-REPO-01")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// BuildIdentityEnvelope extracts identity info from soul files and registered tools
// to build an AgentIdentity for group collaboration.
func (b *ContextBuilder) BuildIdentityEnvelope(agentID, agentName, model string) group.AgentIdentity {
	// Extract first paragraph from SOUL.md
	soulSummary := ""
	wsPath := b.workspace
	if strings.HasPrefix(wsPath, "~") {
		home, _ := os.UserHomeDir()
		wsPath = filepath.Join(home, wsPath[1:])
	}
	soulPath := filepath.Join(wsPath, "SOUL.md")
	if data, err := os.ReadFile(soulPath); err == nil {
		lines := strings.Split(string(data), "\n")
		var para []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" && len(para) > 0 {
				break
			}
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				para = append(para, trimmed)
			}
		}
		soulSummary = strings.Join(para, " ")
	}

	// Collect tool names as capabilities
	var capabilities []string
	for _, tool := range b.registry.List() {
		capabilities = append(capabilities, tool.Name())
	}
	// Fallback: if registry is empty (e.g. group manager context), use default names.
	if len(capabilities) == 0 {
		capabilities = tools.DefaultToolNames()
	}

	// Determine active channels
	channels := []string{"cli"} // CLI is always available

	return group.AgentIdentity{
		AgentID:      agentID,
		AgentName:    agentName,
		SoulSummary:  soulSummary,
		Capabilities: capabilities,
		Channels:     channels,
		Model:        model,
		JoinedAt:     time.Now().Format(time.RFC3339),
		Status:       "active",
	}
}

// BuildMessages constructs the message list for the LLM.
func (b *ContextBuilder) BuildMessages(
	sess *session.Session,
	currentMessage string,
	channel string,
	chatID string,
	messageType string,
) []provider.Message {

	systemPrompt := b.BuildSystemPrompt()

	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}

	// Inject request context based on message type
	switch messageType {
	case "internal":
		systemPrompt += "\n\n## Request Context\nThis is an INTERNAL message from the bot owner. Treat as command/reflection. Full tool access. Respond concisely and directly. You may access system internals."
	case "external":
		systemPrompt += "\n\n## Request Context\nThis is an EXTERNAL request from an authorized user. Be helpful and professional. Do NOT expose system internals (paths, configs, keys). Prefer read-only operations. Tool access may be restricted by policy."
	}

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add recent history from session
	// We skip the last message in session because it's the current one we are about to add
	// (Session usually stores [User, Assistant, User...])
	// In the Loop.ProcessDirect, we added the user message to session BEFORE calling this.
	// So we should include all history EXCEPT the last one if we are appending it manually.
	// Actually, let's look at Loop.ProcessDirect:
	// sess.AddMessage("user", content) -> then calls BuildMessages
	// So the last message in session IS the current message.

	history := sess.GetHistory(50)

	// We want to format history for the LLM.
	// If the last message in history is the current message, we should exclude it from the "history" block
	// and add it as the explicit "Current message" at the end.

	var historyMessages []session.Message
	if len(history) > 0 && history[len(history)-1].Content == currentMessage {
		historyMessages = history[:len(history)-1]
	} else {
		historyMessages = history
	}

	for _, msg := range historyMessages {
		messages = append(messages, provider.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current message
	messages = append(messages, provider.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}
