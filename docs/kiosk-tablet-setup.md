# Kiosk tablet setup

Checklist for the troop Pixel Tablet as the Meeting Kiosk ([#32](https://github.com/markgoho/troop-813/issues/32)). Why each step: [research from #3](https://github.com/markgoho/troop-813/blob/research/pixel-tablet-kiosk-lock/docs/research/pixel-tablet-kiosk-lock.md). No MDM: app pinning with a PIN is the only OS lock.

## One-time setup

1. Settings > System > Multiple users > Add user: create a **Kiosk** user (restricted profile if offered, else a regular user, not a guest). Give it a PIN only Leaders know. Add no personal Google accounts.
2. Switch to the Kiosk user. Settings > Display > Screen timeout: maximum. Turn off battery saver.
3. Settings > Security & privacy > More security settings > App pinning: turn on **Use app pinning** and **Ask for PIN before unpinning**.
4. In Chrome, open `https://troop-813-hq.web.app` and install the app (menu > Install app / Add to Home screen). Open it from the Home screen and confirm there's no URL bar.
5. Optional: silence notifications and clear other apps off the Home screen.

## Before each Meeting

1. Dock or plug in the tablet. Switch to the Kiosk user and unlock with the PIN.
2. Open the installed HQ app and go to `/kiosk`.
3. The Admin unlocks the Kiosk, then taps **Start today's Meeting**. Confirm the Kiosk start screen shows.
4. Pin it: swipe up and hold for Overview, tap the HQ icon above its preview, tap **Pin**.
5. Check: swipe down does nothing, Back stays on the Kiosk. Hand the tablet to the first Scout.

## After the Meeting

1. Nothing to end: the Kiosk session expires at 22:00 America/New_York on its own (and sleeps after 30 min idle).
2. Unpin on the tablet: swipe up and hold, enter the PIN.
