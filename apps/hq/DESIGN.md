---
name: Troop 813 HQ
description: Meeting Kiosk and Leader screens for Troop 813, Spencerport NY; one clear question per screen, in troop burgundy.
colors:
  bg: "oklch(96.1% 0.003 354.7)"
  surface: "oklch(100.0% 0 0)"
  ink: "oklch(22.9% 0.015 346.7)"
  muted: "oklch(47.5% 0.019 348.3)"
  line: "oklch(87.6% 0.012 351.0)"
  accent: "oklch(32.2% 0.121 355.3)"
  accent-hover: "oklch(27.4% 0.102 354.9)"
  accent-ink: "oklch(100.0% 0 0)"
  accent-soft: "oklch(93.2% 0.018 351.2)"
  fill: "oklch(32.2% 0.121 355.3)"
  fill-hover: "oklch(27.4% 0.102 354.9)"
  fill-ink: "oklch(100.0% 0 0)"
  ok: "oklch(49.9% 0.098 154.5)"
  ok-soft: "oklch(92.6% 0.020 159.7)"
  warn: "oklch(49.2% 0.110 66.0)"
  warn-soft: "oklch(94.8% 0.038 81.5)"
  bad: "oklch(48.5% 0.150 28.1)"
  bad-soft: "oklch(91.6% 0.030 25.3)"
  focus: "oklch(32.2% 0.121 355.3)"
  dark-bg: "oklch(18.5% 0.009 352.0)"
  dark-surface: "oklch(22.7% 0.012 350.4)"
  dark-ink: "oklch(93.5% 0.010 352.4)"
  dark-muted: "oklch(73.6% 0.018 349.7)"
  dark-line: "oklch(33.1% 0.018 349.0)"
  dark-accent: "oklch(80% 0.075 355)"
  dark-accent-hover: "oklch(86% 0.06 355)"
  dark-accent-ink: "oklch(20% 0.06 357)"
  dark-accent-soft: "oklch(29.2% 0.047 348.0)"
  dark-fill: "oklch(44% 0.14 355)"
  dark-fill-hover: "oklch(50% 0.15 355)"
  dark-fill-ink: "oklch(100% 0 0)"
  dark-ok: "oklch(77.4% 0.097 159.7)"
  dark-ok-soft: "oklch(31.8% 0.035 159.7)"
  dark-warn: "oklch(79.5% 0.120 77.6)"
  dark-warn-soft: "oklch(30.9% 0.040 82.5)"
  dark-bad: "oklch(73.9% 0.125 26.5)"
  dark-bad-soft: "oklch(28.7% 0.042 24.6)"
  dark-focus: "oklch(80% 0.075 355)"
typography:
  display:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "clamp(3rem, min(18cqi, 22vh, 11rem, calc(190cqi / var(--len, 8))), 11rem)"
    fontWeight: 700
    lineHeight: 0.95
    letterSpacing: "0.01em"
  headline:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "3rem"
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: "0.01em"
  stat:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "2.75rem"
    fontWeight: 700
    lineHeight: 1
  title:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "2.25rem"
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: "0.01em"
  section:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "1.375rem"
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: "0.01em"
  prompt:
    fontFamily: "Atkinson Hyperlegible, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "1.75rem"
    fontWeight: 700
    lineHeight: 1.45
  body:
    fontFamily: "Atkinson Hyperlegible, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.45
  meta:
    fontFamily: "Atkinson Hyperlegible, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.45
  label:
    fontFamily: "Atkinson Hyperlegible, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 700
    letterSpacing: "0.06em"
  shell:
    fontFamily: "Barlow Condensed, Arial Narrow, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 600
    letterSpacing: "0.06em"
rounded:
  sm: "8px"
  card: "10px"
  md: "12px"
  pill: "999px"
spacing:
  xs: "0.25rem"
  sm: "0.5rem"
  md: "0.75rem"
  lg: "1rem"
  xl: "1.5rem"
components:
  kiosk-button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "0.75rem 1.5rem"
    height: "4rem"
  kiosk-button-primary:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.fill-ink}"
    rounded: "{rounded.md}"
    padding: "0.75rem 1.5rem"
    height: "4rem"
  kiosk-button-primary-hover:
    backgroundColor: "{colors.fill-hover}"
  kiosk-button-quiet:
    textColor: "{colors.muted}"
    rounded: "{rounded.md}"
    padding: "0.75rem 1.5rem"
  leader-button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 0.875rem"
    height: "2.75rem"
  leader-button-primary:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.fill-ink}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 0.875rem"
    height: "2.75rem"
  leader-button-primary-hover:
    backgroundColor: "{colors.fill-hover}"
  leader-button-danger:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.bad}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 0.875rem"
    height: "2.75rem"
  leader-button-danger-solid:
    backgroundColor: "{colors.bad}"
    textColor: "{colors.surface}"
    rounded: "{rounded.sm}"
  pill:
    backgroundColor: "{colors.bg}"
    textColor: "{colors.muted}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0.125rem 0.5rem"
  pill-ok:
    backgroundColor: "{colors.ok-soft}"
    textColor: "{colors.ok}"
  pill-warn:
    backgroundColor: "{colors.warn-soft}"
    textColor: "{colors.warn}"
  pill-bad:
    backgroundColor: "{colors.bad-soft}"
    textColor: "{colors.bad}"
  pill-live:
    backgroundColor: "{colors.fill}"
    textColor: "{colors.fill-ink}"
  list-card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.card}"
  list-row:
    padding: "0.625rem 0.875rem"
    height: "3.25rem"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 0.75rem"
    height: "2.75rem"
  alert:
    backgroundColor: "{colors.bad-soft}"
    textColor: "{colors.bad}"
    rounded: "{rounded.sm}"
    padding: "0.75rem 0.875rem"
  banner:
    backgroundColor: "{colors.accent-soft}"
    rounded: "{rounded.card}"
    padding: "0.75rem 0.875rem"
  tab-bar:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.muted}"
    height: "3.5rem"
  tab-bar-current:
    textColor: "{colors.accent}"
---

# Design System: Troop 813 HQ

## Overview

**Creative North Star: "One Question, One Name"**

Troop 813 HQ is a light, plain working tool in troop burgundy. The Kiosk asks one question per screen and makes the person's name the hero, set in burgundy condensed capitals large enough to read at arm's length on a shared tablet. The Leader screens carry the same world at phone density: a text-only shell bar, a tab bar, lists drawn as one bordered card with a row per item, soft status pills, and large condensed counts.

Everything sits on a near-white, faintly rose-tinted ground with white surfaces and hairline borders. There is no decoration beyond the troop's monoline logo badge, shown large on the lock, sign-in and waiting screens only. Colour carries meaning: burgundy marks the brand, the current choice and the main action; green, amber and red appear only as state. Both surfaces ship light and dark themes from the same tokens.

The world was pinned by the user from the prototypes (`prototype/kiosk-flow`, `prototype/leader-screens`), with troop burgundy replacing the prototype's green. It refuses two things: the visitor-kiosk grid of identical cards as the main Kiosk experience, and the admin-dashboard look (sidebars, tables, KPI tiles) on the Leader screens.

**Key Characteristics:**

- Condensed display (Barlow Condensed) over a legibility-first body (Atkinson Hyperlegible).
- One burgundy accent; green only for success and Going.
- Flat: white cards on a tinted ground, 1px or 2px outlines, no drop shadows.
- Two densities from one token set: Kiosk at a 1.25rem base with 4rem targets, Leader at 1rem with 2.75rem targets.
- Light and dark themes; in dark mode the accent splits into light-rose text and deep-burgundy fills.

## Colors

One deep troop burgundy against warm-neutral greys with a faint rose cast, plus three state colours that each come with a soft tint.

### Primary

- **Troop Burgundy** (`accent`, `fill`): the brand colour. As `accent` it colours the Handoff's hero name, links, the current tab, headline counts and big stats. As `fill` it is every primary button, the checked segment of a switch, the `live` pill and text selection. In light mode the two are the same value; they are separate tokens so dark mode can split them.
- **Deep Burgundy** (`accent-hover`, `fill-hover`): hover state for burgundy fills.
- **Rose Wash** (`accent-soft`): the background of the Leader's informational banner (for example the Kiosk status banner on Tonight).
- **PWA chrome:** the web manifest's `theme_color` and `background_color` and the `theme-color` meta tag stay in hex (`#5e0a36`), the sRGB equivalent of Troop Burgundy, because those fields need hex. Everything in CSS is oklch.

### Tertiary (state)

- **Going Green** (`ok`, `ok-soft`): success and Going only. The drawn check mark on a checked-in name, the Going answer chip on the Kiosk summary, the `ok` pill, the Going count, the checked Going segment.
- **Last-Chance Amber** (`warn`, `warn-soft`): the Kiosk's last-chance reminder, the `warn` pill and the warning banner.
- **Problem Red** (`bad`, `bad-soft`): alerts, danger buttons, inline delete/undo confirms, the Not going answer, count and checked segment.

### Neutral

- **Lodge Mist** (`bg`): the page ground, faintly rose-grey. Also the background of the neutral pill and the cleared (—) segment.
- **Paper White** (`surface`): cards, buttons, inputs, the Leader shell bar and tab bar.
- **Charcoal Ink** (`ink`): body text and outlined-button text.
- **Pencil Grey** (`muted`): secondary text, meta lines, the shell label, inactive tabs, quiet buttons.
- **Hairline** (`line`): every border and row divider.
- **Focus** (`focus`): the focus ring; follows the accent in both themes.

### Named Rules

**The Green Means Going Rule.** Green (`ok`) appears only for success and Going. Burgundy is the accent; never use green as a brand colour, a link or a primary button.

**The Text-Versus-Fill Rule.** Use `--accent` for burgundy text and strokes and `--fill` (with `--fill-ink`) for burgundy backgrounds. In dark mode the accent becomes a light rose for legible text on the dark ground while fills stay deep burgundy with white text; using `--accent` as a background breaks dark mode.

**The oklch Source Rule.** Colours are authored in oklch. Hex appears only where a platform field requires it (the manifest and `theme-color`).

## Typography

**Display Font:** Barlow Condensed 600/700 (with Arial Narrow, system-ui)
**Body Font:** Atkinson Hyperlegible 400/700 (with system-ui, -apple-system, Segoe UI)

**Character:** A tall, tight condensed face for names, headings and numbers, paired with a body face designed for legibility, which matters for young readers and a tablet at arm's length. Both are self-hosted through fontsource.

### Hierarchy

- **Display** (700, `clamp()` up to 11rem, 0.95): the Kiosk Handoff's hero name only. Sized from the stage's width (`cqi`) and the name's length (`--len`) so a long name stays on one line.
- **Headline** (700, 3rem, 1.05): Kiosk screen titles (3.5rem for the Event name on an RSVP screen).
- **Stat** (700, 2.5 to 2.75rem, 1 to 1.1): Leader headline counts ("N checked in", burgundy, beside a muted "N not yet") and big stat numbers.
- **Title** (700, 2.25rem, 1.05): Leader page titles.
- **Section** (700, 1.375rem): Leader section headings.
- **Prompt** (Atkinson 700, 1.75rem): the Kiosk's one question per screen ("Are you …?", the RSVP ask).
- **Body** (400, 1rem, 1.45): Leader text. The Kiosk raises its base to 1.25rem, and its sub-lines and status text to 1.375rem.
- **Meta** (400, 0.875rem, muted): times, methods, emails, reasons and notes under a row's name.
- **Label** (Atkinson 700, 0.75rem, 0.06em, uppercase): status pills only.
- **Shell** (Barlow Condensed 600, 1.0625rem Leader / 1.125rem Kiosk, 0.06em, uppercase, muted): the shell bar's troop name ("Troop 813 HQ", "Troop 813 · Meeting").

### Named Rules

**The Condensed For Names And Numbers Rule.** Barlow Condensed sets names, headings, counts and the shell label. Everything a person reads as a sentence, including the Kiosk's questions and all buttons, is Atkinson Hyperlegible.

**The Tabular Numbers Rule.** Times, attendance counts and big stats use `font-variant-numeric: tabular-nums`.

## Layout

The Kiosk is a two-row grid: a thin text-only shell bar ("Troop 813 · Meeting") over a stage centred vertically, at most 60rem wide, with 1.5rem gaps and side padding that respects safe areas. The stage is a size container so the hero name scales with it. Answer buttons sit in a wrapping row (`flex: 1 1 14rem`, 1rem gap); in portrait orientation they stack full-width. The Handoff's Yes spans the full row at 5rem tall, with the two escapes side by side beneath it. The name list alone may widen to `min(76rem, 100vw - 3rem)` and uses an auto-fill grid of 11rem minimum columns so about 30 names fit one landscape screen.

The Leader screens are phone-first (390px): a text-only shell bar, a single 42rem-max column with 1rem gaps and 1.25rem top padding, and a tab bar fixed to the bottom (6rem bottom padding keeps content clear of it). At 48rem and wider the tab bar moves to the top under the shell bar, centred, and the column's bottom padding drops to 3rem. Everything respects `env(safe-area-inset-*)`.

Rhythm: 0.25 / 0.5 / 0.75 / 1 / 1.5rem gaps; list rows pad 0.625rem by 0.875rem with a 3.25rem minimum height.

### Named Rules

**The Two Densities Rule.** One token set, two densities: Kiosk targets are at least 4rem tall on a 1.25rem base; Leader targets are at least 2.75rem tall on a 1rem base. A new Kiosk screen uses the Kiosk vocabulary; a new Leader screen uses `leader.css`.

## Elevation & Depth

Flat. Depth comes from tonal layering only: white surfaces on the tinted ground, separated by 1px hairline borders. There are no drop shadows anywhere. The only `box-shadow` in the system is the current-tab marker, an inset 3px burgundy bar (`inset 0 3px 0 var(--accent)` at the top edge on phones, `inset 0 -3px 0` at the bottom edge on wide screens).

### Named Rules

**The Flat Paper Rule.** Separate layers with `--surface` against `--bg` and a `--line` border, never with a drop shadow.

## Shapes

Soft, consistent corners: 12px (`--radius`) on Kiosk buttons and Kiosk summary rows, 8px (`--radius-sm`) on Leader buttons, inputs, alerts, confirms and the segmented switch, 10px on Leader list cards, form cards and banners, and fully rounded pills (999px). Outlines are 2px on the Kiosk and 1px on the Leader screens. A dashed outline means "not available": the Kiosk's no-answer chip and a disabled Leader danger button.

The troop logo is a circular monoline badge drawn in `currentColor` in a 380-unit viewBox: two clearly separated rings (r=177 and r=156, 2.2-unit stroke) around mountains, trees, sky and lettering (2.8-unit stroke), round caps and joins. Strokes scale with the art. It appears only as a large crest: 12rem on the Kiosk lock and unlock screens, 10rem on the sign-in and waiting pages.

### Named Rules

**The Crest Not Icon Rule.** The logo is used at 10rem or larger, never as a small icon. Its trees, stars and lettering cannot render legibly at around 32px at 1x, so shell bars carry the troop name as text instead.

## Components

### Buttons

Large, outlined and plain; the main action is filled.

- **Kiosk:** 4rem tall, 0.75rem by 1.5rem padding, 1.375rem bold text, 2px `--line` outline, 12px corners, `--surface` background. Primary (`.primary`, `.yes`) fills with `--fill`. The Handoff's Yes and the RSVP Done button grow to 5rem and 1.75rem. Quiet buttons are transparent, muted, regular weight.
- **Leader:** 2.75rem tall, 0.5rem by 0.875rem padding, 0.9375rem bold, 1px `--line` outline, 8px corners. Primary (`.primary` and every `type="submit"`) fills with `--fill`. Danger has red text on the outlined button; `.danger.solid` fills red with surface-coloured text for the final confirm. A disabled danger button turns muted with a dashed outline at full opacity; other disabled buttons drop to 45% opacity (55% on the Kiosk).
- **Hover / Focus:** hover turns the outline burgundy (`--accent`) and darkens fills to `--fill-hover`; transitions run 160ms on `--ease`. On the Kiosk a pressed button scales to 0.98 when motion is allowed. Focus is a 3px `--focus` outline offset 2px everywhere.

### Chips

- **Status pill (Leader):** uppercase 0.75rem bold label, 0.06em tracking, fully rounded, 0.125rem by 0.5rem. Neutral (`--bg` / `--muted`), `ok`, `warn`, `bad` (soft tint with the strong colour as text), and `live` (burgundy fill).
- **Answer chip (Kiosk summary):** 1.125rem bold, fully rounded, 0.25rem by 0.875rem. Going is `ok`, Not going is `bad`; no answer is muted with a 2px dashed `--line` outline.

### Cards / Containers

- **List card (Leader):** one `--surface` card per list, 1px `--line` border, 10px corners, a row per item divided by hairlines. Rows wrap: the name (bold, links in `--ink`, underline on hover) first, muted meta beneath, the row's action pushed to the end.
- **Form card (Leader):** the same card with 1rem padding and a 0.75rem grid gap.
- **Summary row (Kiosk):** a `--surface` row with a 1px border and 12px corners: the Event name, a muted detail line, the answer chip and a Change button.
- **Banner (Leader):** `--accent-soft` (or `--warn-soft`) panel, 10px corners, wrapping text and action.
- **Inline confirm (Leader):** a `--bad-soft` panel inside the row, bold question on its own line, then the consequences and the buttons.
- **Alert:** `--bad-soft` panel with bold `--bad` text, 8px corners. The Kiosk's last-chance reminder is the same shape in `--warn-soft` / `--warn`.

### Inputs / Fields

- **Style:** full-width, 2.75rem tall, 0.5rem by 0.75rem, 1px `--line` border, 8px corners, `--surface` background, 1rem regular text. Labels stack above in 0.875rem bold muted text. Checkboxes are 1.5rem with `accent-color: var(--fill)`.
- **Focus:** the border turns `--accent` in addition to the shared 3px focus ring.

### Navigation

- **Shell bar:** text only, no logo: the uppercase condensed troop name in muted grey ("Troop 813 HQ" on Leader screens, a `--surface` strip with a hairline under it; "Troop 813 · Meeting" on the Kiosk, without the surface fill).
- **Tab bar (Leader):** Tonight / Meetings / Events / Roster (plus Access for the Admin), bold 0.875rem muted labels in 3.5rem-tall cells. The current tab turns burgundy with the inset 3px bar. Fixed to the bottom on phones; at 48rem it moves to the top under the shell bar, centred, 0.9375rem labels in 3rem cells.

### Hero Name (signature, Kiosk)

The Handoff shows a small muted "Pass the tablet to" lead in body type, then the Scout's name in Display type in `--accent`, kept on one line, then the prompt and the buttons. When the name changes it rises 0.12em into place from 60% opacity over 420ms on `--ease` (only when motion is allowed); it never fades in from nothing.

### Name List (Kiosk)

A grid of outlined name buttons (4rem tall, 1.5rem text, left-aligned). A checked-in name drops its outline and background, goes muted and regular weight, and gains a drawn green check mark (a masked SVG, with a hidden "✓" in the text for screen readers), so checked-in state never relies on colour alone.

### Segmented RSVP Switch (signature, Leader)

A radiogroup drawn as one 8px-cornered outlined control with hairline dividers; each option is at least 3rem by 2.75rem with the real radio input stretched invisibly over it. The checked option fills: Going in `--ok`, Not going in `--bad` (both with `--surface` text), and the cleared "—" option in `--bg` with `--ink` text. Elsewhere the default checked fill is `--fill`. Keyboard focus draws the 3px ring inset (offset -3px) on the option.

### Shared Vocabulary And Specificity

**The Zero-Specificity Rule.** Shared styles carry no weight so a screen's own styles always win. The base element rules in `app.css` (headings, paragraphs, form controls, buttons, links, `:focus-visible`) sit inside `:where()`. The Kiosk's shared vocabulary in `Kiosk.svelte` puts the `.kiosk` scope inside `:where()`, so those rules weigh only as much as their target selector and any scoped class in a Kiosk screen beats them. Every selector in `leader.css` sits wholly inside `:where(.leader …)`, so it has zero specificity: a component's `<style>` always wins, and within `leader.css` the later rule wins. Pseudo-elements cannot go inside `:where()`, so those few rules keep a normal selector. Breaking this (an unwrapped shared selector) has caused real bugs where shared rules overrode screen styles.

## Do's and Don'ts

### Do:

- **Do** author every colour as an oklch custom property in `app.css`, with a dark-mode value, and reference it with `var(--…)`.
- **Do** use `--accent` for burgundy text and `--fill` / `--fill-ink` for burgundy backgrounds.
- **Do** keep green (`--ok`) for success and Going, amber (`--warn`) for warnings, red (`--bad`) for problems and Not going.
- **Do** wrap every new shared rule in `:where()` (the whole selector in `leader.css`, the `.kiosk` scope in the Kiosk vocabulary) so screen styles override it and source order decides.
- **Do** give Kiosk controls at least a 4rem height and Leader controls at least 2.75rem.
- **Do** draw lists on the Leader screens as one bordered card with a row per item.
- **Do** pair state colour with a shape or text (check mark, pill label, dashed outline) so meaning never relies on colour alone.
- **Do** gate motion behind `prefers-reduced-motion: no-preference` and start entrances from a visible state.
- **Do** use the logo through `Logo.svelte` (it follows the text colour), at 12rem on Kiosk lock screens and 10rem on gate pages.
- **Do** use fictional names from `bun run seed` in any mock, fixture or screenshot.

### Don't:

- **Don't** use green as a brand or accent colour, or burgundy for success.
- **Don't** use `--accent` as a background fill; it turns light rose in dark mode.
- **Don't** add drop shadows; the inset tab marker is the only box-shadow.
- **Don't** add a shared selector outside `:where()`.
- **Don't** make a grid of identical cards the main Kiosk experience; the Handoff's single big name is the main screen and the name list stays secondary.
- **Don't** build Leader screens as an admin dashboard: no sidebars, data tables or KPI tiles.
- **Don't** use the logo below 10rem or as a small icon in a bar, button or tab.
- **Don't** put real Scout names in designs, fixtures or screenshots.
- **Don't** hard-code hex colours in CSS; hex lives only in the manifest and `theme-color`.
