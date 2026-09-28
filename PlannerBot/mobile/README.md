# 📱 Planner Bot — Mobile Companion App

A clean, offline-first Flutter companion app for **Planner Bot** that connects directly to your Laptop using **Encrypted P2P CRDT Sync**.

## ✨ Features
- **Cross-Platform**: Runs natively on Android and iOS.
- **Encrypted Synchronization**: Communicates directly over local Wi-Fi with your laptop using AES-256-GCM.
- **Zero Cloud Dependence**: Keeps all journaling and reminder data private.
- **CRDT Conflict-Free**: Offline edits on phone seamlessly merge with laptop edits.
- **Day Analysis View**: See daily productivity metrics and AI recommendations.

## 🚀 How to Run

### Prerequisites
- [Flutter SDK (>=3.0.0)](https://flutter.dev/docs/get-started/install)
- Android Studio / Xcode for emulators or physical device deployment

### Steps
```bash
cd mobile

# Fetch dependencies
flutter pub get

# Run on connected phone or emulator
flutter run
```

### Pairing with Laptop
1. Launch `planner.exe` on your laptop.
2. Check the Pairing Code displayed in the terminal or on the Web UI (e.g. `PLAN-XXXX-XXXX`).
3. Tap the Sync icon (top right) in the mobile app, enter the code, and tap **Save & Sync**.
4. Both devices will now continuously synchronize your reminders over Wi-Fi with zero data leakage!
