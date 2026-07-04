---
name: MBTI
description: MBTI personality assessment — 16 types, 4 dichotomies, Jungian cognitive functions
colors:
  primary: "#0d7377"
  primary-hover: "#095b5e"
  primary-light: "#e8f3f4"
  primary-border: "#c5e0e2"
  ink: "#1a1917"
  ink-muted: "#555550"
  ink-subtle: "#999990"
  bord: "#e0dcd6"
  bord-light: "#ede9e3"
  surface: "#FCF9F6"
  bg: "#f0ede8"
  bg-alt: "#e8e4de"
  nt: "#7c3aed"
  nf: "#d97706"
  sj: "#059669"
  sp: "#2563eb"
  success: "#059669"
  warning: "#d97706"
  error: "#dc2626"
typography:
  display:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "clamp(2.25rem, 5vw, 4rem)"
    fontWeight: 400
    lineHeight: 1.1
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "clamp(1.75rem, 3vw, 2.5rem)"
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "clamp(1.25rem, 2vw, 1.5rem)"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.65
    letterSpacing: "0em"
  label:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.5
    letterSpacing: "0em"
  mono:
    fontFamily: "JetBrains Mono, monospace"
    fontSize: "0.875rem"
    fontWeight: 600
    lineHeight: 1.5
    letterSpacing: "0em"
rounded:
  sm: "8px"
  md: "12px"
  lg: "16px"
  full: "9999px"
spacing:
  "2xs": "4px"
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "24px"
  xl: "32px"
  "2xl": "48px"
  "3xl": "64px"
  "4xl": "80px"
  "5xl": "96px"
  "6xl": "128px"
elevation:
  level-1:
    boxShadow: "0 1px 3px rgba(26,25,23,0.06), 0 4px 12px rgba(26,25,23,0.04)"
    border: "1px solid #ede9e3"
  level-2:
    boxShadow: "0 4px 16px rgba(26,25,23,0.08), 0 12px 40px rgba(26,25,23,0.06)"
  navbar-scroll:
    boxShadow: "0 1px 12px rgba(26,25,23,0.06)"
    background: "rgba(252,249,246,0.85)"
    backdropFilter: "blur(16px)"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#ffffff"
    rounded: "{rounded.sm}"
    padding: "12px 28px"
    typography: "{typography.label}"
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
    textColor: "#ffffff"
    rounded: "{rounded.sm}"
    padding: "12px 28px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.sm}"
    padding: "12px 28px"
    border: "1px solid {colors.bord}"
  button-secondary-hover:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "12px 28px"
    border: "1px solid {colors.ink}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.sm}"
    padding: "8px 16px"
  button-ghost-hover:
    backgroundColor: "{colors.bg}"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "8px 16px"
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.md}"
    border: "1px solid {colors.bord-light}"
    boxShadow: "{elevation.level-1.boxShadow}"
  card-flat:
    backgroundColor: "{colors.bg}"
    rounded: "{rounded.md}"
    border: "none"
  card-hover:
    border: "1px solid {colors.bord}"
    boxShadow: "0 4px 16px rgba(26,25,23,0.08)"
    transition: "all 0.2s ease"
  input-field:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "12px 16px"
    border: "1px solid {colors.bord}"
  input-field-focus:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "12px 16px"
    border: "1px solid {colors.primary}"
  progress-bar:
    backgroundColor: "{colors.bord-light}"
    rounded: "{rounded.full}"
    height: "4px"
  progress-fill:
    rounded: "{rounded.full}"
    height: "100%"
    transition: "width 0.8s ease"
---

# Design System: MBTI

## 1. Overview

**Creative North Star: "Know Your Architecture"**

MBTI's visual system is a mirror for self-discovery — it
doesn't judge, it reveals. The interface creates space for
introspection and understanding, not distraction. Every pixel
serves clarity and psychological depth.

The system is **refined and confident**: modern enough to
feel premium, restrained enough to feel serious. Subtle
elevation and warm tones signal quality without overwhelming
the psychological weight of the content.

**Key Characteristics:**
- Warm off-white base — never pure white or pure black
- Intentional elevation — three levels, warm-toned shadows
- Editorial serif for display, clean sans for body
- One accent (Abyssal Teal), ≤10% per screen
- Modern but grounded — not a SaaS dashboard, not a quiz app

## 2. Colors

The palette is intentionally narrow. One accent carries the
emotional weight; warm neutrals provide structure. Color
signals, separates, and guides — never decorates.

### Primary
- **Abyssal Teal** (#0d7377): The single accent. Interactive
  elements, focus rings, active states, data highlights,
  brand mark. ≤10% of any screen. Rarity is the point.
- **Abyssal Teal Hover** (#095b5e): Darkened for hover states.
- **Abyssal Teal Light** (#e8f3f4): Subtle backgrounds,
  badge fills, icon containers.
- **Abyssal Teal Border** (#c5e0e2): Bordered containers
  needing a hint of brand.

### Neutral (Warm)
- **Ink** (#1a1917): Primary text. Warm near-black, not
  pure black — grounded, not harsh.
- **Ink Muted** (#555550): Secondary text, body copy.
  WCAG AA 4.5:1 against surface.
- **Ink Subtle** (#999990): Placeholders, metadata, captions.
  WCAG AA 4.5:1 against surface.
- **Bord** (#e0dcd6): Standard borders, dividers. Warm gray.
- **Bord Light** (#ede9e3): Subtle borders, card outlines.
- **Surface** (#FCF9F6): Primary background. Warm off-white.
- **Bg** (#f0ede8): Section alternation, flat card backgrounds.
- **Bg Alt** (#e8e4de): Deeper alternation, subtle fills.

### Data (Temperament Groups)
- **NT** (#7c3aed): Purple. Rational temperament (INTJ, INTP,
  ENTJ, ENTP). Score displays and progress bars only.
- **NF** (#d97706): Amber. Idealist temperament (INFJ, INFP,
  ENFJ, ENFP). Score displays and progress bars only.
- **SJ** (#059669): Emerald. Guardian temperament (ISTJ, ISFJ,
  ESTJ, ESFJ). Score displays and progress bars only.
- **SP** (#2563eb): Blue. Artisan temperament (ISTP, ISFP,
  ESTP, ESFP). Score displays and progress bars only.

### Semantic
- **Success** (#059669): Confirmation, positive states.
- **Warning** (#d97706): Cautionary indicators.
- **Error** (#dc2626): Error states, destructive actions.

### Named Rules

**The One Voice Rule.** Abyssal Teal is the only accent.
≤10% of any screen. Second accent = design drift.

**The Data-Only Rule.** NT, NF, SJ, SP colors appear only
in score displays, progress bars, and type badges.
Never in navigation, buttons, or decorative elements.

**The Warm Base Rule.** Surface is always #FCF9F6.
Never pure white (#ffffff) or pure black (#000000).
All neutrals carry a warm undertone.

**The Warm Shadow Rule.** Shadows use warm ink tone
(#1a1917), never pure black. Opacity stays below 0.10
— shadows suggest, they don't announce.

## 3. Typography

**Display/Headline:** DM Serif Display — editorial gravity,
psychological weight. Carries the emotional tone of the brand.

**Body/UI:** Inter — recedes into quiet readability.
Does not compete with the serif.

**Mono:** JetBrains Mono — for scores, data, percentages.

### Hierarchy
- **Display** (DM Serif Display, 400, clamp(2.25rem, 5vw,
  4rem), 1.1, -0.02em): Hero headlines only. text-wrap: balance.
- **Headline** (DM Serif Display, 400, clamp(1.75rem, 3vw,
  2.5rem), 1.2, -0.01em): Section headings. text-wrap: balance.
- **Title** (Inter, 600, clamp(1.25rem, 2vw, 1.5rem), 1.3,
  -0.01em): Card titles, subheadings. text-wrap: balance.
- **Body** (Inter, 400, 1rem, 1.65): Paragraphs, descriptions.
  Max 65–75ch. text-wrap: pretty.
- **Label** (Inter, 500, 0.875rem, 1.5): Buttons, nav, labels.
- **Mono** (JetBrains Mono, 600, 0.875rem, 1.5): Score values,
  percentages, data points.

### Named Rules

**The Serif-for-Gravity Rule.** DM Serif Display for display
and headline only. Inter for everything else. Never mix
within the same hierarchy level.

**The Weight-Only Hierarchy Rule.** Hierarchy through weight,
size, and font family — not color, tracking, or case changes.
No uppercase eyebrows. No wide-tracked labels.

## 4. Elevation

Three levels of intentional depth. Used to indicate
interactivity and layering — never decoration.

**Level 0 — Flat:** Page background, section containers.
No shadow, no border. Base surface (#FCF9F6).

**Level 1 — Raised:** Cards, input fields.
box-shadow: 0 1px 3px rgba(26,25,23,0.06),
0 4px 12px rgba(26,25,23,0.04);
border: 1px solid #ede9e3;

**Level 2 — Floating:** Dropdowns, tooltips, modals.
box-shadow: 0 4px 16px rgba(26,25,23,0.08),
0 12px 40px rgba(26,25,23,0.06);

**Navbar on scroll:**
background: rgba(252,249,246,0.85);
backdrop-filter: blur(16px);
box-shadow: 0 1px 12px rgba(26,25,23,0.06);

### Named Rules

**The Intentional Elevation Rule.** Three levels only.
Warm-toned, low opacity, purposeful. No colored shadows,
no neon glows, no material-design-style hard shadows.

**The No-Pure-Flat Rule.** Cards and interactive surfaces
use Level 1 elevation — not fully flat. Flatness is for
backgrounds and non-interactive containers only.

## 5. Components

### Buttons
- **Primary:** Abyssal Teal (#0d7377) background, white text,
  8px radius, 12px 28px padding.
  Hover: #095b5e. Transition: background 0.2s ease.
  Focus: 2px teal outline, 2px offset. No shadow, no glow.
- **Secondary:** Transparent, inkMuted text, 1px bord border.
  Hover: border → ink, text → ink.
- **Ghost:** Transparent, inkMuted text, 8px 16px padding.
  Hover: bg background, ink text.

### Navbar
- **Default:** background #FCF9F6, 1px bord-light bottom border.
  No blur, no shadow.
- **On scroll:** background rgba(252,249,246,0.85),
  backdrop-filter blur(16px),
  box-shadow 0 1px 12px rgba(26,25,23,0.06).
- Transition: all 0.3s ease on scroll trigger.
- Height: 56–64px max. Slim, invisible when possible.
- Layout: logo left, nav links center, CTA button right.
- Links: ghost style. No active indicator beyond URL.
- Respects prefers-reduced-motion: no blur/transition
  if reduced motion enabled.

### Hero
- Background: #FCF9F6 — always light, never dark.
- Headline: DM Serif Display, large, warm ink (#1a1917).
- One word or phrase in Abyssal Teal (#0d7377).
- Typographic decoration: oversized symbol or word at
  3–5% opacity (e.g. "∅", "≈", "tipe") — NOT illustration.
- Subtle animated gradient shimmer on headline
  (CSS only, respects prefers-reduced-motion).
- Two CTAs: primary teal + secondary ghost, side by side.
- Trust pills below CTAs: "Anonim" · "Gratis" · "Hasil Instan"
- NO photos, NO Lottie, NO SVG illustrations, NO dark hero.

### Cards
- Background: surface (#FCF9F6).
- Border: 1px solid bord-light (#ede9e3).
- Shadow: Level 1 elevation.
- Radius: 12px (rounded.md).
- Padding: 24–32px.
- Hover: border → bord (#e0dcd6),
  shadow → 0 4px 16px rgba(26,25,23,0.08).
  Transition: all 0.2s ease.

### Inputs
- Border: 1px solid bord (#e0dcd6).
- Background: surface (#FCF9F6).
- Radius: 8px. Padding: 12px 16px.
- Focus: border → primary (#0d7377). No glow.
- Placeholder: inkSubtle (#999990) — WCAG AA.
- Error: border → error (#dc2626).

### Progress Bars
- Height: 4px. Track: bord-light. Full-radius caps.
- Fill: temperament color — no gradients.
- Transition: width 0.8s ease.

### Footer
- Background: bg (#f0ede8) — slightly deeper than surface.
- Top border: 1px solid bord-light only.
- No dark inversion, no color change.
- Layout: logo + tagline left, 2–3 link columns right.
- Copyright: inkSubtle, small, bottom.

## 6. Accessibility

- WCAG AA minimum on all elements.
- AAA contrast target for body text and CTAs where palette allows.
- prefers-reduced-motion: all animations and transitions
  must be disabled or reduced when enabled.
- Color blindness: never use color as sole differentiator.
  Always pair with shape, label, or position.
- ARIA labels on all interactive elements.
- Focus-visible: 2px teal outline, 2px offset on all
  interactive elements.

## 7. Do's and Don'ts

### Do:
- Use Abyssal Teal sparingly — ≤10% per screen
- Use warm off-white (#FCF9F6) as base — never pure white
- Use DM Serif Display for hero and section headlines
- Use Level 1 elevation on cards — not fully flat
- Use glass navbar effect on scroll
- Use tonal layering for section depth
- Keep body text at inkMuted (#555550) — WCAG AA
- Cap body line length at 65–75ch
- Use temperament colors (NT, NF, SJ, SP) exclusively for score data
- Respect prefers-reduced-motion on all animations
- Pair color with shape/label for color blindness support

### Don't:
- Don't use pure white (#ffffff) or pure black (#000000)
- Don't use Inter for display or headline — use DM Serif Display
- Don't use black/ink for primary button — use teal
- Don't use gradients — backgrounds, text, or buttons
- Don't use glassmorphism outside navbar scroll state
- Don't use illustrations, Lottie, or SVG sketches
- Don't use uppercase tracked eyebrows above headings
- Don't use numbered section markers (01, 02, 03)
- Don't use colored shadows or neon glows
- Don't use temperament colors outside score displays
- Don't gamify — no badges, achievements, celebrations
- Don't use corporate wellness warmth or stock photos
- Don't use border-radius larger than 16px on cards
- Don't pair border + heavy shadow on the same element
