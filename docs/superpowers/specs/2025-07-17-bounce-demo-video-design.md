# Bounce Demo Video — Design Spec

**Date:** 2025-07-17  
**Topic:** 30-second product demo video  
**Status:** approved → implementation

## Overview

A 30-second product demo video for **Bounce** — Smart Sports Data Proxy for Portuguese basketball. Built with Remotion, the video showcases the API dashboard with animated charts and transitions. All on-screen text in Portuguese, no voiceover.

## Audience

- **Primary:** Developers interested in self-hosting / API integration
- **Secondary:** Sports organizations (clubs, federations) consuming basketball data

## Visual Style

- **Theme:** Dark (#0a0a0f → #1a1a2e animated gradient background)
- **Accent:** Bounce orange (#F97316)
- **Typography:** Inter (sans-serif, clean)
- **Cards:** Subtle frosted glass (glassmorphism), 1px opacity borders
- **Charts:** Stroke-dashoffset draw animations, smooth gradients
- **Transitions:** Slide + fade between scenes, custom easing

## Audio

- Instrumental background music (subtle)
- No voiceover — animated on-screen text only
- All text in **Portuguese**

## Timeline

```
0s ───── 3s ────────── 10s ───────────── 17s ──────────── 24s ─── 30s
[ INTRO ] [ CENA 1      ] [ CENA 2        ] [ CENA 3       ] [ CTA  ]
```

### Scene 0: Intro (0–3s)

- Bounce logo (🏀 + "Bounce" text) fades in from center
- Subtitle "Smart Sports Data Proxy" appears below
- Subtle particle/glow effect behind logo
- Transition out: scale down + fade

### Scene 1: Métricas (3–10s)

- 4 metric cards in a 2×2 grid, staggered entrance
- Cards: Requests Totais, Cache Hit Rate %, Uptime, Goroutines
- Animated number counters (counting up from 0 to target value)
- Each card: icon + value (large) + label (small) + subtle trend indicator
- Glassmorphism card style with orange accent on hover-glow
- Transition out: cards slide left + fade

### Scene 2: Gráficos (10–17s)

- Line chart: Requests/minuto — animated draw (stroke-dashoffset)
- Area chart below: Cache Hit Rate % — filled gradient
- X-axis time labels, Y-axis values
- Tooltip appears on hover point with exact timestamp
- Time range selector pill visible: 1m | 5m | 1h | 6h | 24h | 7d
- "Live WebSocket" indicator pulsing
- Transition out: chart compresses upward + fade

### Scene 3: API (17–24s)

- Endpoint list reveals one by one with slide-in from right
- Featured endpoints: `/api/games/live`, `/api/standings/{compID}`, `/api/elo`, `/api/predictions/{gameId}`
- Method badges (GET in orange)
- "REST API • JSON • WebSocket" tagline
- Code-like monospace styling for paths
- Transition out: elements scatter

### Scene 4: CTA (24–30s)

- "Experimenta a API" — large centered text, fade in
- URL: `github.com/mefrraz/bounce` below
- Subtle pulsing glow on CTA
- Docker quick-start one-liner appears: `docker run -d -p 3001:3001 ghcr.io/mefrraz/bounce`
- Fade to black at 30s

## Components

```
src/
├── Root.tsx                    # Composition entry, <Composition>
├── scenes/
│   ├── Intro.tsx               # Logo + subtitle
│   ├── Metrics.tsx             # 4 animated stat cards
│   ├── Charts.tsx              # Line + area charts
│   ├── ApiEndpoints.tsx        # Endpoint reveal list
│   └── Cta.tsx                 # Call to action
├── components/
│   ├── MetricCard.tsx          # Single metric with counter animation
│   ├── LineChart.tsx           # Animated SVG line chart
│   ├── AreaChart.tsx           # Animated SVG area chart
│   ├── EndpointRow.tsx         # Single API endpoint row
│   └── SceneTransition.tsx     # Scene wrapper with transitions
├── data/
│   └── dashboard.ts            # Mock metric values matching real Bounce dashboard
└── style/
    └── theme.ts                # Colors, fonts, spacing tokens
```

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Framework | Remotion (React) | Frame-perfect video rendering |
| Charts | SVG custom (stroke-dashoffset) | Full control, no heavy libs |
| Animations | `useCurrentFrame()` + `interpolate()` | Remotion-native, performant |
| Counter | `interpolate(frame, [in, out], [0, value])` | Smooth number counting |
| Transitions | `<TransitionSeries>` | Built-in scene sequencing |
| FPS | 30fps (900 frames total) | Standard for web video |
| Resolution | 1920×1080 | Full HD, 16:9 |

## Files Affected

- **New:** `bounce/video/` — entire Remotion project
- **No changes** to existing Bounce codebase

## Acceptance Criteria

- [ ] Video renders at 1920×1080, 30fps, exactly 30 seconds
- [ ] All 5 scenes transition smoothly
- [ ] Metric counters animate from 0 to target values
- [ ] Charts draw with stroke-dashoffset animation
- [ ] All on-screen text is in Portuguese
- [ ] Orange accent (#F97316) matches Bounce brand
- [ ] Final frame shows CTA and GitHub URL
- [ ] Output file: `bounce/video/out/bounce-demo.mp4`
