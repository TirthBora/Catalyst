# Catalyst

> **Centralized, browser-accessible second-screen development workspace companion.**

Code on one screen. Observe, monitor, and interact with your development environment on another.

---

## What is Catalyst?

Catalyst transforms your secondary device—such as an iPad, Android tablet, phone, or secondary monitor—into an interactive real-time development dashboard. 

Instead of constantly switching between your code editor, browser preview, terminal logs, and system monitors on a single display, Catalyst exposes a high-performance local workspace you can open in any browser on your network.

---

## Key Features

* **🖥️ Live Application Preview**: Built-in reverse proxy with WebSocket HMR support for Next.js, Vite, React, Vue, Flask, and static HTML. Works seamlessly across devices even when dev servers bind only to `localhost`.
* **⚡ Real-Time Hot Reload**: Recursive filesystem monitoring using `fsnotify` with 100ms debouncing, dynamic subfolder discovery, and cache-busting preview refreshes.
* **📱 Instant Mobile Pairing**: Connect any phone or tablet instantly by scanning an on-screen SVG QR code or typing a friendly 6-character PIN (`CAT-XXXX`).
* **🎛️ One-Click Dev Server Runner**: Start, stop, and restart vetted project development scripts (`npm run dev`, `go run .`, `cargo run`) directly from your secondary screen with pairing token authorization.
* **📜 Streaming Terminal Log Viewer**: View compiler logs, dev server stdout, and stderr live on your second screen with auto-scroll and stream coloring.
* **📂 Multi-Workspace Switcher**: Register and switch between different project directories dynamically through the dashboard dropdown without restarting the server.
* **🩺 Environment Doctor & Diagnostics**: Inspect installed compiler toolchains (Go, Node.js, npm, Python, Git, Rust) and validate workspace dependency health.
* **⚙️ Persistent Configuration Engine**: Customize preview targets, device frames, auto-reload behavior, and custom ignored directories saved in `.catalyst/config.json`.
* **🎨 Linear / Raycast Aesthetic**: Refined dark theme UI optimized for tablets and phones, with device frame emulation (Desktop, Tablet 768px, Mobile 375px).

---

## Quickstart

### 1. Using the CLI

Run Catalyst directly from any project root:

```bash
# Start the development workspace on default port 9999
catalyst dev

# Or specify a custom port and directory
catalyst dev -p 8080 -d /path/to/project

# Check status of the running daemon
catalyst status

# Diagnose your local development environment
catalyst doctor

# Stop the running daemon cleanly
catalyst stop

# View version and build info
catalyst version
```

### 2. Using the VS Code Extension

1. Open your project in **Visual Studio Code**.
2. Catalyst automatically boots in the background when your workspace opens.
3. Click the **`$(broadcast) Catalyst Active`** item in the status bar or run the command `Catalyst: Open Workspace`.
4. Scan the pairing QR code or open `http://<your-ip>:9999` on your tablet or phone.

---

## Configuration (`.catalyst/config.json`)

Catalyst automatically creates and maintains a workspace configuration file:

```json
{
  "port": "9999",
  "targetUrl": "/preview/",
  "customIgnore": ["data", "docs", "cache"],
  "defaultDevice": "full",
  "autoReload": true,
  "theme": "dark"
}
```

Settings can also be modified in real-time using the **⚙️ Settings** modal on the dashboard.

---

## Building from Source

### Prerequisites
* Go 1.22+
* Node.js 20+ & npm

### Build Commands

```bash
# Build the local host binary (bin/catalyst.exe or bin/catalyst)
npm run build:daemon

# Cross-compile for Windows, Linux, and macOS (amd64 and arm64)
npm run build:all

# Run the complete Go test suite
npm run test:go

# Compile the VS Code extension
npm run compile
```

---

## License

MIT
