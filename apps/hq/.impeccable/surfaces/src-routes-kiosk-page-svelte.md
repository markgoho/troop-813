---
version: 1
slug: "src-routes-kiosk-page-svelte"
primary_target: "src/routes/kiosk/+page.svelte"
related_targets:
  [
    "src/lib/Kiosk.svelte",
    "src/lib/KioskNameList.svelte",
    "src/lib/KioskRsvp.svelte",
    "src/lib/KioskSession.svelte",
  ]
---

# Meeting Kiosk (/kiosk)

Mode: Operate. Shared troop Pixel Tablet, fullscreen PWA, landscape or portrait. One Scout at a time, aged 11–17, often first-time holders.

Job: confirm who you are (records Attendance), answer pending RSVPs, pass the tablet to the named Scout. Copy and behaviour are fixed by spec #17; this pass is visual only.

Most-seen screen: the Handoff ("Pass the tablet to <Scout>. Are you <Scout>?"). The name list is for the first Scout, late arrivals and broken chains; it must not feel like a wall of names.

## Direction contract

THESIS: The prototype the user liked (branch `prototype/kiosk-flow`), pinned: one clear question per screen, the person's name as the hero. Refuses the visitor-kiosk grid of identical cards as the main experience.

OWN-WORLD: Prototype grammar kept: Barlow Condensed display over Atkinson Hyperlegible body, light neutral ground with white surfaces, 2px-outlined rounded buttons, filled primary, soft-tinted state pills, light and dark themes. Green accent replaced by troop burgundy; green survives only as success / Going. Troop logo (line art) in the shell.

STORY: A Scout picks up the tablet, sees their own name huge, taps Yes, answers each Event with one big tap, sees their answers, taps "Done, pass the tablet on" and sees who to find next.

FIRST VIEWPORT: Handoff: small "Pass the tablet to" line, the Scout's name in burgundy condensed display at monumental size, then "Are you <Scout>?" and three full-width-feeling buttons, Yes filled burgundy, the two escapes outlined. Thin shell: logo + "Troop 813 · Meeting" quiet at the top.

FORM: Pinned by the user (prototype look), overriding the roll; rolled seed:d358cb75416a31bc. Signature interaction: the big name, which the name list's tap and the Handoff share.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Unresolved

- Exact burgundy shade: flyer `#7a004b`, troop's is "a bit darker"; building with a darker value, user to confirm.
- Leader screens inherit this world in a later pass.
