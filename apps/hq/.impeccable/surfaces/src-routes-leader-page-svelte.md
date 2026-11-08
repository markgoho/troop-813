---
version: 1
slug: "src-routes-leader-page-svelte"
primary_target: "src/routes/(leader)/+page.svelte"
related_targets: ["src/lib/LeaderShell.svelte", "src/lib/leader.css"]
---

# Leader screens (/, /roster, /events, /meetings, /access and detail pages)

Mode: Operate. Leaders' phones first (390px), also usable on a laptop. Tabs: Tonight / Meetings / Events / Roster, plus Access for the Admin. Copy and behaviour fixed by spec #17; this pass is visual only.

## Direction contract

THESIS: The prototype's Leader screens (branch `prototype/leader-screens`), pinned: glance at status, act in one tap, every fix explained. Refuses the admin-dashboard look (sidebars, tables, KPI tiles).

OWN-WORLD: The Kiosk's world (app.css tokens, Barlow Condensed + Atkinson Hyperlegible, troop burgundy, green only for success) at phone density: a logo bar, a bottom tab bar (top on wide screens), lists as one bordered card with a row per item, uppercase soft pills, 1px-outlined buttons with a filled primary and red danger, inline confirms, large condensed headline counts.

STORY: A Leader opens Tonight during the Meeting, reads who's in and whether the Kiosk is awake, fixes a wrong check-in with Undo, then sets RSVPs on an Event page.

FIRST VIEWPORT: Tonight: bar, Kiosk status banner, "N checked in / N not yet" in condensed display (burgundy / muted), then the Checked in card: name on top, time and method below, Undo at the row end. Tab bar fixed at the bottom with the current tab marked.

FORM: Extension of the Kiosk world (pinned prototype), no new roll; seed:d358cb75416a31bc. Signature: the segmented Going / Not / — switch on the Event page.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
