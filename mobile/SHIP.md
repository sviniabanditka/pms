# Shipping the mobile app

Local dev needs nothing extra. Builds and push require an Expo/EAS account.

## Dev

```bash
cd mobile
npm install
# from repo root once (regenerates mobile/gen from proto):
#   export PATH="$HOME/go/bin:$PATH" && buf generate
npx expo start          # then: i (iOS sim) · a (Android) · w (web)
```

Real device: set the API host in `mobile/.env`:

```
EXPO_PUBLIC_API_BASE=http://<your-LAN-ip>:8080
```

The Go server must be reachable on that IP (not just localhost).

## Push (needs EAS projectId)

`getExpoPushTokenAsync` requires an EAS `projectId`. Without it `registerForPush`
silently no-ops (see `lib/push.ts`).

```bash
npm i -g eas-cli
eas login
eas init            # creates the project + writes extra.eas.projectId into app config
```

Push only works on a real device (not web/simulator).

## Builds (EAS)

Profiles are in `eas.json` (development · preview · production).

```bash
# internal test build (installable IPA/APK):
eas build --profile preview --platform ios
eas build --profile preview --platform android

# production:
eas build --profile production --platform all
```

`ios.bundleIdentifier` / `android.package` = `com.stdtask.app` (change if taken).

## Submit to stores

```bash
eas submit --profile production --platform ios       # → TestFlight / App Store
eas submit --profile production --platform android    # → Play internal track
```

Needs Apple Developer + Google Play Console accounts and their credentials
(EAS walks you through the first time).

## OTA updates (JS-only changes, no rebuild)

```bash
eas update --branch preview        # or production
```

Requires `expo-updates` (add when you want OTA; not wired yet).

## Still TODO before public release

- Real app icon + splash logo (currently Expo placeholders; brand colors set).
- Store listing assets (screenshots, description, privacy policy).
- Optional: Sentry/crash reporting, `expo-updates` for OTA.
