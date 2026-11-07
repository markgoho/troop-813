# Locking a Pixel Tablet into the Meeting Kiosk

Research for [#3](https://github.com/markgoho/troop-813/issues/3) (child of map [#1](https://github.com/markgoho/troop-813/issues/1)). Researched 2026-11-07.

**Setting:** A Leader signs into the SvelteKit app on a troop-owned Google Pixel Tablet and opens the **Meeting Kiosk**. The tablet goes Scout to Scout (ages ~11–17) by **Handoff**. Scouts must stay out of other apps, browser tabs, the URL bar and Leader-only screens. There is no MDM.

## TL;DR

- **The only no-MDM OS lock is Android app pinning ("screen pinning").** Turn on "ask for PIN before unpinning" and a Scout can't unpin without the device PIN. Pinning blocks the notification shade, Quick Settings and launching other apps or tasks. **Back stays enabled**, and Home/Overview stay visible because the unpin gesture uses them.
- **Pin an installed PWA (WebAPK), not a Chrome tab.** `display: standalone` hides the URL bar and Chrome's tab UI. `fullscreen` hides the status bar too. A pinned PWA can only be exited with the unpin gesture plus the PIN.
- **Chrome for Android has no kiosk mode.** Chrome Enterprise kiosk is ChromeOS-only and needs enrolled devices. Real Android "lock task mode" needs a device-owner app (DPC). You can get one without MDM through `adb dpm set-device-owner` on a factory-reset tablet, but for this use case it's overkill.
- **Use a dedicated tablet user for the Kiosk**, not the Leader's own profile. The Leader's email, notifications and saved sessions should never be on the pinned screen.
- **The web app can't trap Back.** Chrome skips history entries that a page pushed without a user gesture, and the Navigation API can't cancel back/forward traversals. So make Back harmless instead. The server must enforce the Kiosk's scope: Leader routes reject a Kiosk session. Leaving the Kiosk takes Leader re-auth. Idle time resets to the Kiosk start screen.
- **Screen Wake Lock works in Chrome Android (Chrome 84+).** It needs HTTPS, it's released whenever the page is hidden, and it must be re-acquired on `visibilitychange`.

## 1. Android app pinning (screen pinning)

**How it works on Pixel Tablet** ([Pixel Tablet Help: Pin and unpin screens](https://support.google.com/googlepixeltablet/answer/13592197?hl=en)):

- Enable it: Settings > Security & privacy > More security settings > App pinning > "Use app pinning". Then "you'll see an option to ask for your PIN, pattern, or password before unpinning." Some steps need Android 14+.
- Pin: open the app, swipe up and hold to open Overview, tap the app's icon at the top of its preview, then tap **Pin**.
- Unpin: swipe up and hold (gesture nav), or touch and hold Back + Overview (3-button nav). "If prompted, enter your PIN, pattern, or password."
- Generic Android help says the same: "When app pinning is on, you need to enter your PIN, pattern, or password before you can unpin." ([Android Help 9455138](https://support.google.com/android/answer/9455138))

**Is a PIN required to unpin?** Only if the toggle is on, and only if the profile has a secure screen lock. AOSP reads `Settings.Secure.LOCK_TO_APP_EXIT_LOCKED`. If that setting is unset, the default is "is the lock screen secure?" On unpin it calls `lockNow()` + `requireCredentialEntry()`. ([AOSP `LockTaskController.java`](https://android.googlesource.com/platform/frameworks/base/+/refs/heads/main/services/core/java/com/android/server/wm/LockTaskController.java), `shouldLockKeyguard` / `lockKeyguardIfNeeded`)

**What pinning blocks** (AOSP `LockTaskController`):

- Status bar: pinned mode applies `STATUS_BAR_MASK_PINNED = DISABLE_MASK & ~DISABLE_BACK & ~DISABLE_HOME & ~DISABLE_RECENT`. Shade expansion (`DISABLE_EXPAND`, so no notification shade, Quick Settings or user switcher from the shade), notification icons and alerts, clock and system info are all disabled. **Back, Home and Overview remain visible.** Home and Overview are there for the unpin gesture. Back is a live button that the web app sees.
- Launching other tasks: `isLockTaskModeViolation` blocks starting any non-allowlisted task while a locked or pinned task exists, and shows a toast. Without a DPC nothing is allowlisted, so an in-app link that would open Chrome, Maps, a `mailto:` handler and so on should be blocked. Emergency-call and wireless-emergency-alert intents are exceptions.
- Keyguard: in pinned mode the code calls `disableKeyguard(...)` (`setKeyguardState`, `LOCK_TASK_MODE_PINNED` branch). So after a screen timeout, the tablet should wake straight back to the pinned app without asking for the PIN. **Unverified on device:** Android's keyguard-disable has historically been a no-op on some secure-lock configurations. Test it. Either way, keep the screen awake (section 5) so it doesn't matter.

**Pinning vs lock task mode:** "Screen pinning appears similar (and provides an immersive experience) but the person using the device can exit the mode whenever they want." Lock task mode hides Home and Overview, blanks the status bar and has no user exit, but "Only apps that have been allowlisted by a device policy controller (DPC) can run" in it. ([developer.android.com: Lock task mode](https://developer.android.com/work/dpc/dedicated-devices/lock-task-mode)) "Exit whenever they want" means exit with the PIN, when the PIN toggle is on.

## 2. Installed PWA (WebAPK) with standalone/fullscreen display

- **Display modes** ([MDN: manifest `display`](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest/Reference/display)):
  - `fullscreen`: "Opens the app with browser UI elements hidden and uses the entirety of the available display area."
  - `standalone`: "The browser will exclude UI elements such as a URL bar but can still include other UI elements such as the status bar."
  - If a mode isn't supported, the browser falls back: `fullscreen` → `standalone` → `minimal-ui` → `browser`.
  - `@media (display-mode: …)` tells the page which mode it actually got.
- **So yes, an installed PWA hides the URL bar and tabs.** `minimal-ui` doesn't: it "possibly [includes] a way to view the app's URL." Avoid it.
- **WebAPK:** On Android, Chrome "automatically generates and installs a special APK of your app". It shows up "in the app launcher, in Android's app settings" ([web.dev: WebAPKs](https://web.dev/articles/webapks)). It's a separate app, so it can be pinned separately from Chrome.
- **Scope:** URLs outside the manifest `scope` "should be opened in the browser" (same source). While pinned, that browser launch is a new task and should be blocked (section 1). Still, the Kiosk should contain no out-of-scope links.
- **Can a pinned PWA be exited?** Only with the unpin gesture plus the PIN (section 1). Back inside the PWA navigates its own history. Back at the root of the history would normally background the app, and backgrounding is what pinning prevents. **Verify on device.** Either way, don't let Back reach the root (section 6).
- The Fullscreen API (`element.requestFullscreen()`) needs a user gesture and is exited by Back or a swipe. It adds nothing over `display: fullscreen` in the manifest. Skip it.

## 3. Chrome "kiosk-like" options without MDM

- **Chrome for Android has no kiosk mode.** There's no `--kiosk` flag and no user-facing lock setting. Chrome Enterprise kiosk ("websites, PWAs, IWAs in locked-down kiosk mode") is a **ChromeOS** feature configured in the Google Admin console. It needs enrolled devices ([Chrome Enterprise Help: Add web apps to Chrome kiosks](https://support.google.com/chrome/a/answer/9781496?hl=en), [Enroll ChromeOS devices](https://support.google.com/chrome/a/answer/1360534)). It doesn't apply to a Pixel Tablet.
- **Real lock task mode without an MDM vendor:** set a DPC as device owner with `adb shell dpm set-device-owner <component>`, e.g. Google's sample TestDPC. This only works on an unprovisioned device: "if `USER_SETUP_COMPLETE` has ever been set, the device is considered provisioned and the device owner can't be set," which in practice means a factory reset with no accounts added ([AOSP: Provision for device management](https://source.android.com/docs/devices/admin/provision), [Test device management](https://source.android.com/docs/devices/admin/testing-setup), [Dedicated devices cookbook](https://developer.android.com/work/dpc/dedicated-devices/cookbook)). The DPC then allowlists the PWA's WebAPK package for lock task mode. **Verdict:** possible, but it means a wiped, dedicated tablet and maintaining a debug-grade admin app. Pinning plus a PIN gives most of the benefit. Keep this as a fallback if Scouts beat pinning.
- Third-party kiosk-browser apps exist. They're out of scope here (not primary sources), and most need device-owner provisioning anyway to be stronger than pinning.

## 4. Pixel Tablet multi-user / guest profiles

From Pixel Tablet Help ([Delete, switch, or add users](https://support.google.com/googlepixeltablet/answer/13591529?hl=en), [Use or delete guests](https://support.google.com/googlepixeltablet/answer/13567147?hl=en), [Change guest and user settings](https://support.google.com/googlepixeltablet/answer/13592148?hl=en)):

- Only the device owner can add users: Settings > System > Multiple users > Add user. Each user has their own Home screens, accounts and apps, and can have their own screen lock.
- **Guest:** for short-term use. On exit, "The device will delete all the data from this guest" unless the guest chooses to save. The guest gets wiped and re-created, so a PWA install and sign-in wouldn't persist. **Poor fit.**
- **Restricted profile:** "lets you control what features and content a user can see" (tablets only). AOSP describes it as using the parent user's accounts, with the parent choosing which apps are available ([AOSP: Support multiple users](https://source.android.com/devices/tech/admin/multi-user)). **I couldn't confirm it still exists on the current Pixel Tablet build.** Check Settings > System > Multiple users > Add user for it. If it's there, it's the best fit: an allowlist of just Chrome plus the PWA.
- Settings can turn on switching and adding users from the lock screen ("Add users from lock screen") and through Quick Settings > User. Pinning disables shade expansion, so Scouts can't reach the user switcher while the screen is pinned.
- **Recommendation:** a dedicated "Kiosk" secondary user (restricted profile if available) with its own PIN that only Leaders know. Inside it: Chrome, the installed Kiosk PWA and no personal accounts. The Leader's personal profile then never ends up in a Scout's hands.

## 5. Screen Wake Lock in Chrome Android

- Chrome shipped Screen Wake Lock in **Chrome 84**, desktop and Android alike ([developer.chrome.com: Wake Lock](https://developer.chrome.com/docs/capabilities/web-apis/wake-lock)).
- It needs HTTPS. `navigator.wakeLock.request("screen")` returns a `WakeLockSentinel`. The request can be rejected for "power save mode or low battery level", and the lock is "automatically released when document becomes inactive". To keep it, "re-acquire screen wake lock … when document becomes active (listen for `visibilitychange`)." A released sentinel can't be reused ([MDN: Screen Wake Lock API](https://developer.mozilla.org/en-US/docs/Web/API/Screen_Wake_Lock_API)). Permissions-Policy `screen-wake-lock` defaults to `self`, so the top-level app is fine.
- **Back it up at the OS level:** set Settings > Display > Screen timeout to the maximum in the Kiosk user, and keep the tablet on its charging dock or plugged in during Meetings. Wake lock fails quietly in battery saver.

## 6. What the web app itself must do

**Back navigation can't be blocked reliably, so make it harmless.**

- Chromium marks an entry "skip on back/forward UI" when "this page does a navigation without ever receiving a user gesture … to intervene against pages that manipulate the history" (covers `history.pushState` and client redirects) ([Chromium `navigation_entry_impl.h`, `should_skip_on_back_forward_ui_`](https://chromium.googlesource.com/chromium/src/+/refs/heads/main/content/browser/renderer_host/navigation_entry_impl.h)). The usual "push a sentinel entry on load, re-push on `popstate`" trick is therefore unreliable.
- The Navigation API: "you can't cancel a navigation via `preventDefault()` if the user is pressing the Back or Forward buttons" ([developer.chrome.com: Navigation API](https://developer.chrome.com/docs/web-platform/navigation-api)).
- **Design:** run the Kiosk as **one route** that keeps its internal state (which Scout, which step, the Handoff) in component state, not in URL history. Use `replaceState` for any URL changes so no back stack builds up. If you want a buffer entry, push it from a real tap handler, such as a Scout tapping their name: gesture-backed entries aren't skipped. A Back that does land should show the Kiosk start screen, never a Leader screen.

**Scope the session on the server.**

- Starting the Kiosk should swap the Leader's session for a **Kiosk-scoped session or token**. That session can only read the Roster for the Meeting and write Attendance and RSVPs. Every Leader-only route and API (SvelteKit `hooks.server.ts` / `+layout.server.ts` guards) rejects it. Then, even if a Scout gets to a Leader URL somehow, the server refuses.
- **Exit Kiosk = Leader re-auth.** The exit control asks for Leader credentials (or a Leader PIN checked on the server) and only then issues a full Leader session. Rate-limit the PIN.
- Persist Kiosk state on the server or in storage, so a reload, crash or "pull to refresh" lands back on the Kiosk with the same Meeting. Use `overscroll-behavior: none` on the page to suppress Chrome's pull-to-refresh.

**Idle handling**

- After N seconds with no input, return to the Kiosk start screen ("Tap your name") and drop any half-finished Scout entry. **Never** log out the Kiosk session on idle. A logout would need a Leader mid-Meeting.
- Optional: the Kiosk session expires at Meeting end (e.g. 9:00 pm Monday), then shows a locked screen that needs a Leader.

**Other hygiene**

- No `target="_blank"` links, no external, `tel:` or `mailto:` links, and no links outside the manifest `scope` anywhere in Kiosk views.
- Disable text selection and long-press context menus on Kiosk controls (`user-select: none`, `-webkit-touch-callout: none`), so there's no "open in new tab" or "search" path.
- Acquire the wake lock on Kiosk start and re-acquire it on `visibilitychange`. If the wake lock is denied, show a small "keep screen on" warning to the Leader.
- Check `matchMedia('(display-mode: fullscreen), (display-mode: standalone)')` and warn the Leader if the Kiosk is opened in a normal tab.

## Recommended setup checklist

One-time, by a Leader:

1. [ ] On the Pixel Tablet, Settings > System > Multiple users: create a **Kiosk** user (a restricted profile if offered, otherwise a regular user, not a guest). Give it its own **PIN known only to Leaders**. Add no personal Google accounts.
2. [ ] In the Kiosk user, Settings > Display > Screen timeout: maximum. Turn off battery saver.
3. [ ] In the Kiosk user, Settings > Security & privacy > More security settings > App pinning: turn on **Use app pinning** and **ask for PIN before unpinning**.
4. [ ] In the Kiosk user, open the troop site in Chrome and **install the app** (Add to Home screen / Install). Confirm it opens with no URL bar.
5. [ ] Optional: in the Kiosk user, mute notifications and remove unneeded apps from the Home screen.

Each Meeting:

6. [ ] Switch to the Kiosk user and unlock with the Leader PIN. Plug in or dock the tablet.
7. [ ] Open the installed PWA, sign in as a Leader and start the **Meeting Kiosk**. This swaps in the Kiosk-scoped session.
8. [ ] Open Overview, tap the app icon, then **Pin**.
9. [ ] Hand the tablet to the first Scout.
10. [ ] At the end: a Leader exits the Kiosk in-app (re-auth), then unpins with the gesture plus the PIN.

App requirements: a fixed-scope `display: fullscreen` manifest (fallback `standalone`); a single-route Kiosk with no back stack; a server-enforced Kiosk session; Leader re-auth to exit; idle reset to the start screen; wake lock re-acquired on `visibilitychange`; no external or out-of-scope links; pull-to-refresh disabled.

Verify on the real tablet before relying on it:

- (a) After a screen timeout while pinned, does it wake to the app or to the PIN lock screen?
- (b) Back at the root of the pinned PWA: nothing, or a toast?
- (c) Does an out-of-scope link while pinned get blocked with a toast?
- (d) Is a restricted profile offered?
