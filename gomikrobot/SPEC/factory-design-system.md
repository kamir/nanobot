# Factory Design System

Industrial/factory visual theme for GoMikroBot web interfaces.
Applied across: `web/group.html`, `web/index.html`, `electron/src/renderer/views/ModePicker.vue`.

---

## Color Palette

### Primary Brand
| Name | Hex | Usage |
|------|-----|-------|
| Industrial Orange | `#ff6b35` | Primary accent, pipes, gears, CTAs |
| Amber/Gold | `#fbbf24` | Secondary accent, warnings, highlights |
| Gear Yellow | `#fcd34d` | Gear centers, bright accents |

### Mode Accents
| Mode | Hex | Usage |
|------|-----|-------|
| Timeline/Standalone | `#a855f7` | Purple accent |
| Group Management/Full | `#ff6b35` | Orange accent |
| Approvals/Remote | `#22c55e` | Green accent |

### Status
| State | Hex | CSS Class |
|-------|-----|-----------|
| Active/Success | `#22c55e` | `.badge-active`, `.status-active` |
| Warning/Stale | `#eab308` | `.badge-stale`, `.status-stale` |
| Error/Failed | `#ef4444` | `.badge-failed` |
| Info/Blue | `#58a6ff` | `.span-INBOUND` |

### Backgrounds (dark gradient, darkest → lightest)
| Name | Hex | Usage |
|------|-----|-------|
| Sky Top | `#060810` | SVG gradient top |
| Main BG | `#0a0e14` | Body background |
| Card BG | `#0d1117` | Card surfaces |
| Ground | `#161b22` | Factory ground, conveyor |
| Grid Lines | `#1a1f2b` | Subtle grid overlay (opacity 0.3) |
| Conveyor Detail | `#111820`, `#1a2030` | Conveyor belt segments |
| Borders | `#21262d`, `#30363d` | Card/panel borders |

### Text
| Level | Hex | Usage |
|-------|-----|-------|
| Primary | `#c9d1d9` | Main content |
| Secondary | `#6b7280` | Labels, muted text |
| Tertiary | `#9ca3af` | Subtle info |
| Bright | `#f0f6fc` | Headers, emphasis |
| Pipe Labels | `#d1d5db` | SVG text on pipes |

---

## Typography

- **Font Family**: `'JetBrains Mono', ui-monospace, monospace`
- **Weights**: 400 (regular), 500 (medium), 600 (semibold), 700 (bold), 800 (extra-bold)

| Element | Size | Weight | Letter-Spacing | Extra |
|---------|------|--------|----------------|-------|
| Logo | 30px | 800 | 0.12em | — |
| Page Headers | 18-20px | 700-800 | 0.06em | — |
| Section Headers | 14-16px | 700 | 0.06em | — |
| `.ind-label` | 9px | — | 0.2em | `text-transform: uppercase` |
| `.ind-value` | 1.5rem | 800 | 0.04em | — |
| Pipe Labels | 10px | 600 | 0.15em | SVG `text-anchor: middle` |
| Small/Subtle | 8-11px | — | 0.05em-0.3em | — |

---

## SVG Component Library

### Gear
Construction: concentric circles + tooth rects.

```
Outer Ring:  <circle> stroke-dasharray, transparent fill, rotating animation
Hub:         <circle> fill="#ff6b35" opacity="0.12-0.25"
Center Dot:  <circle> fill="#fcd34d" opacity="0.6-0.9", r=2-4
Teeth:       6-8 <rect> elements, rotated every 45-60deg around center
             width=2, height=3.5, rx=0.5, fill="#ff6b35" opacity="0.3"
```

Rotation: `animation: gear-spin 20s linear infinite` (slow) or `12s reverse` (fast).

### Pipe
Layered construction from bottom to top:

```
1. Glow:        Large rect, high blur filter, opacity 0.04-0.15
2. Body:        Main rect, rounded ends (rx = thickness/2), solid fill
3. 3D Shine:    White rect overlay, opacity 0.03-0.04
4. Liquid Fill:  Colored rect, height proportional to message count
5. Valves:      Triple-circle at each end (see Valve below)
6. Bubbles:     Animated circles flowing along pipe path
7. Pressure:    Small dot at valve, pulses when active
```

Liquid level formula: `level = msgCount === 0 ? 0.15 : min(0.95, 0.15 + log2(msgCount + 1) * 0.1)`

### Valve (Triple-Circle)
```
Outer Ring:   r=6, fill=pipeColor, opacity 0.12-0.15
Middle Shell: r=4, fill=#0a0e14 (background color)
Inner Dot:    r=2, fill=pipeColor, opacity 0.5-0.7, optional pulse animation
```

### Conveyor Belt
```xml
<pattern id="conveyor" width="24" height="10">
  <rect width="24" height="10" fill="#111820"/>
  <rect x="2" y="2.5" width="8" height="5" rx="1" fill="#1a2030"/>
  <rect x="14" y="2.5" width="8" height="5" rx="1" fill="#1a2030"/>
</pattern>
```
Bolt circles at separator points: `r=3`, `fill=#21262d`, `stroke=#30363d`.

### Pressure Dot
Small indicator circle at valve endpoints.
- Idle: `r=4`, opacity 0.3
- Active: pulses opacity 0.7-1.0 with `pressure-active` class

### Flow Bubbles
```xml
<circle r="2-4" fill="pipeColor" opacity="0.15-0.5">
  <animateMotion path="M start,y L end,y" dur="3-5s" repeatCount="indefinite"/>
</circle>
```
- Active pipes: dur 2.5-5s
- Stale/idle pipes: dur 10-13s (slow drift)

### Grid Background
```xml
<pattern id="grid" width="60" height="60">
  <line x1="60" y1="0" x2="60" y2="60" stroke="#1a1f2b" opacity="0.3"/>
  <line x1="0" y1="60" x2="60" y2="60" stroke="#1a1f2b" opacity="0.3"/>
</pattern>
```

---

## Animation Specifications

| Name | Keyframes | Duration | Easing | Usage |
|------|-----------|----------|--------|-------|
| `gear-spin` | rotate(0deg → 360deg) | 20s | linear | Gear ring rotation |
| `gear-spin-rev` | rotate(360deg → 0deg) | 12s | linear | Counter-rotating gear |
| `idle-pulse` | opacity 0.4 ↔ 0.55 | 4s | ease-in-out | Stale/idle pipes |
| `pressure-blink` | opacity 0.7 ↔ 1.0 | 1.5s | ease-in-out | Active pressure dots |
| `scan-line` | translateY(-100% → 100vh) | 6-7s | linear | CRT scan line overlay |
| `ind-scan` | left -30% → 130% | 8s | linear | Card top border glow sweep |
| `float-up` | translateY(0) → translateY(-40px), opacity 0.4 → 0 | 4-6s | ease-out | Steam particles |
| `pulse-ring` | box-shadow: 0 0 0 0 → 0 0 0 6px (fade) | 2s | — | Status indicator glow |
| `valve-pulse` | opacity 0.3 ↔ 0.8 | 1.2s | ease-in-out | Valve inner dot pulse |
| `topic-pulse-anim` | opacity 0.6 ↔ 1.0 | 2s | ease-in-out | Topic name emphasis |

Steam particle stagger: delays of 0.8s, 1.5s, 2.5s between particles.

---

## CSS Class Naming Conventions

### Industrial Cards
| Class | Purpose |
|-------|---------|
| `.ind-card` | Base card: `bg:#0d1117`, `border:rgba(255,107,53,0.15)`, `border-radius:14px` |
| `.ind-card::before` | Top accent line: 2px gradient `transparent → rgba(255,107,53,0.4) → transparent` |
| `.ind-scanline` | Adds animated scan line `::after` pseudo-element |
| `.ind-value` | Large monospace number: `1.5rem`, weight 800 |
| `.ind-label` | Small uppercase label: `9px`, `letter-spacing:0.2em` |

### Industrial Glow Variants
| Class | Color | Hex |
|-------|-------|-----|
| `.ind-glow-amber` | Amber | `#fbbf24` + `text-shadow: 0 0 12px rgba(251,191,36,0.3)` |
| `.ind-glow-green` | Green | `#22c55e` + matching shadow |
| `.ind-glow-blue` | Blue | `#58a6ff` + matching shadow |
| `.ind-glow-purple` | Purple | `#a855f7` + matching shadow |
| `.ind-glow-orange` | Orange | `#ff6b35` + matching shadow |
| `.ind-glow-red` | Red | `#ef4444` + matching shadow |

### Pipe Classes
| Class | Purpose |
|-------|---------|
| `.pipe-body` | Main pipe rect, `cursor:pointer`, hover `brightness(1.4)` |
| `.pipe-idle` | Applies `idle-pulse` animation (stale pipes) |
| `.pipe-label` | SVG text: pipe name |
| `.pipe-count` | SVG text: message count |
| `.pipe-count-bg` | Background rect for count readability |
| `.pipe-glow` | Pointer-events none glow overlay |
| `.valve-ring` | Valve circles, `transition: opacity 0.3s` |
| `.pressure-dot` | Small indicator, `transition: fill 0.5s, opacity 0.5s` |
| `.pressure-active` | Applies `pressure-blink` animation |

### Badge Classes
| Class | BG Color | Text Color |
|-------|----------|------------|
| `.badge-active` | `rgba(34,197,94,0.15)` | `#4ade80` |
| `.badge-stale` | `rgba(234,179,8,0.15)` | `#facc15` |
| `.badge-pending` | `rgba(59,130,246,0.15)` | `#60a5fa` |
| `.badge-completed` | `rgba(34,197,94,0.15)` | `#4ade80` |
| `.badge-failed` | `rgba(239,68,68,0.15)` | `#f87171` |

### Span Type Colors (Timeline)
| Class | Color | Type |
|-------|-------|------|
| `.span-INBOUND` | `#58a6ff` | Incoming messages |
| `.span-LLM` | `#a855f7` | LLM processing |
| `.span-TOOL` | `#fb923c` | Tool execution |
| `.span-OUTBOUND` | `#22c55e` | Outgoing messages |
| `.span-POLICY` | `#ef4444` | Policy decisions |
| `.span-EVENT` | `#6366f1` | System events |

---

## SVG Filter Definitions

```xml
<!-- Standard glow -->
<filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
  <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur"/>
  <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
</filter>

<!-- Strong glow (stdDeviation=6) -->
<filter id="glow-strong" ...> stdDeviation="6" </filter>

<!-- Pipe glow (moderate, narrow x) -->
<filter id="pipe-glow" x="-20%" y="-50%" width="140%" height="200%">
  <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur"/>
  <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
</filter>

<!-- Bubble blur -->
<filter id="bubble-blur">
  <feGaussianBlur in="SourceGraphic" stdDeviation="2"/>
</filter>
```

**Important**: SVG filters are scoped to their parent `<svg>` element. If referencing a filter from another SVG, define a local copy.

---

## Layout Patterns

### Card Grid
- Mobile: 1 column
- Tablet: 2 columns
- Desktop: 3-4 columns
- Gap: 16-20px
- Card padding: 24px
- Card border-radius: 14-16px

### Card Hover
```css
transform: translateY(-4px);
box-shadow: 0 8px 32px rgba(255,107,53,0.15);
border-color: rgba(255,107,53,0.4);
```

### Glass Morphism
```css
background: rgba(22,27,34,0.6-0.8);
backdrop-filter: blur(8-10px);
border: 1px solid rgba(48,54,61,0.4-0.5);
```

### Opacity Strategy
| Layer | Opacity Range | Examples |
|-------|---------------|---------|
| Background elements | 0.03-0.08 | Grid lines, far glow |
| Structural elements | 0.1-0.15 | Pipe bodies, gear rings |
| Interactive elements | 0.2-0.4 | Flow bubbles, hover states |
| Active/visible | 0.5-1.0 | Text, active indicators |

---

## Transition Defaults

| Type | Duration | Easing |
|------|----------|--------|
| Standard card | 0.3-0.4s | `cubic-bezier(0.4, 0, 0.2, 1)` |
| Hover effects | 0.2-0.4s | ease-out |
| Opacity changes | 0.15s | linear |
| Gear rotation | continuous | linear |
| Pulse animations | symmetric | ease-in-out |

---

## Usage Guidelines

1. **New pages** should import `JetBrains Mono` from Google Fonts (weights: 400,500,600,700,800)
2. **Background**: Always use the dark gradient (`#0a0e14` base) with grid pattern overlay
3. **Decorative SVGs**: Include at least 2 gear clusters and 1 horizontal pipe per page
4. **Scan line overlay**: Apply `.ind-scanline` to hero sections for the CRT effect
5. **Color discipline**: Use orange (`#ff6b35`) as primary accent; never mix with other warm tones
6. **Filter scoping**: Each `<svg>` element needs its own filter definitions
7. **Animation performance**: Keep animated elements to ≤20 per viewport; prefer `transform`/`opacity` over layout-triggering properties
