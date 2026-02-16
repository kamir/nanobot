# Skill Authoring Guide

How to create skills for GoMikroBot, inspired by the [flow-coach](https://github.com/kamir/flow-coach) documentation-as-architecture pattern.

---

## Directory Structure

```
skills/{skill-name}/
  SKILL.md          -- Entry point: loaded into system prompt automatically
  docs/             -- Reference docs (loaded on demand, not at startup)
  examples/         -- Example configs, templates, sample outputs
```

Skills live in the bot system repo under `skills/`. The context builder (`internal/agent/context.go:loadSystemRepoSkills()`) scans `skills/*/SKILL.md` at startup and injects each into the system prompt.

---

## SKILL.md Structure

### Frontmatter (optional)

```markdown
---
name: day2day
version: 1.0
trigger: natural language or dtu/dtp commands
domains: task-management, planning
---
```

### Sections

1. **Overview**: One paragraph describing what the skill does
2. **Trigger Conditions**: When should the LLM activate this skill (keywords, patterns, message types)
3. **Rules**: Numbered, imperative rules the LLM must follow (read-before-write, date computation, etc.)
4. **Tool Usage**: Which tools to use and how (prefer edit_file over write_file, etc.)
5. **Output Format**: How responses should be formatted
6. **Examples** (optional): Concrete input/output pairs

### Writing Rules

- Rules must be **imperative and unambiguous**: "You MUST read the file before answering" not "It would be good to read the file"
- Number each rule for easy reference
- Specify exactly which tool to use for each operation
- Include date computation instructions if the skill deals with time
- Preserve user's original language/formatting unless explicitly asked to change it

---

## Task Assessment Dimensions

Each skill should document its assessment profile — which dimensions matter and how to weight them. This helps the agent route requests appropriately.

| Dimension | Range | When to Use |
|-----------|-------|-------------|
| Complexity | 0-1 | How many steps/tools required |
| Duration | 0-1 | Expected time (0=seconds, 1=hours) |
| Memory | 0-1 | How much prior context needed |
| Coordination | 0-1 | Multi-agent involvement (group) |
| Security | 0-1 | Risk level of operations |

Example for Day2Day: `complexity=0.2, duration=0.1, memory=0.6, coordination=0.1, security=0.1`

---

## Cognitive Modes

Skills can suggest a preferred cognitive mode that gets injected into the system prompt:

| Mode | Best For | Prompt Hint |
|------|----------|-------------|
| convergent | Bug fixes, precise tasks | "Focus on the specific problem. Be systematic and precise." |
| divergent | Brainstorming, creative tasks | "Explore multiple possibilities. Be creative." |
| critical | Security reviews, audits | "Analyze carefully. Question assumptions. Check edge cases." |
| systems | Architecture, design | "Think holistically. Consider connections and dependencies." |
| adaptive | Mixed/unknown tasks | "Adjust your approach based on what you discover." |

Default mode is `adaptive` when no skill matches.

---

## Integration Points

### Loading
Skills are loaded by `internal/agent/context.go:loadSystemRepoSkills()`:
- Scans `{system_repo}/skills/*/SKILL.md`
- Content appended to system prompt under `## {skill-name}`
- Loaded once at gateway startup (restart to reload changes)

### Day2Day Guidance
The special path `{system_repo}/operations/day2day/README.md` is also loaded (separate from skills directory).

### Future: On-Demand Loading
Skills in `docs/` subdirectories are not loaded at startup. Future plan: the LLM can request additional documentation via a `load_skill_doc` tool.

---

## Example: Minimal Skill

```markdown
# Weather Skill

Provides weather information for the user's location.

## Trigger Conditions
- User asks about weather, temperature, forecast
- Keywords: "weather", "Wetter", "temperature", "rain", "forecast"

## Rules

### RULE 1: Use web_search for current data
Always use `web_search` to get current weather. Never answer from memory.

### RULE 2: Include location in response
State the location and date in your response so the user can verify.

### RULE 3: Prefer metric units
Default to Celsius and km/h unless the user requests imperial.

## Output Format
- Temperature: "Currently 12°C in Berlin"
- Forecast: Bullet list for next 3 days
```

---

## Checklist for New Skills

- [ ] Created `skills/{name}/SKILL.md` with all required sections
- [ ] Rules are numbered and imperative
- [ ] Trigger conditions are clear and unambiguous
- [ ] Tool usage specified (which tools, read-before-write patterns)
- [ ] Output format documented
- [ ] Assessment dimensions documented
- [ ] Cognitive mode suggested
- [ ] Tested: restart gateway, send matching message, verify skill activates
