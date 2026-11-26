# Catalyst Architecture & Specification

## 1. System Overview

**Catalyst** is a local development companion that transforms a developer's secondary device (phone, tablet, laptop, or secondary monitor) into an interactive, real-time development workspace.

```
                              PRIMARY MACHINE
┌─────────────────────────────────────────────────────────────────────────────┐
│   VS Code Extension (TypeScript)                                            │
│   └── Multi-tier daemon binary resolution (Settings > Root > Ext > PATH)    │
│                                                                             │
│   Catalyst Go Engine (catalyst.exe)                                         │
│   ├── internal/cli         : Subcommand router (dev, stop, status, doctor)  │
│   ├── internal/workspace   : Multi-workspace registry & project switching   │
│   ├── internal/project     : Framework detection & runnable script extractor│
│   ├── internal/watcher     : Dynamic recursive fsnotify with 100ms debounce │
│   ├── internal/process     : Supervised dev process runner & stdout stream  │
│   ├── internal/proxy       : Reverse proxy for localhost dev servers & HMR  │
│   ├── internal/pairing     : PIN verification, device registry, QR generator│
│   ├── internal/monitor     : Host CPU and physical RAM metrics collector    │
│   ├── internal/runtime     : Toolchain inspection & workspace health doctor │
│   ├── internal/config      : Persistent workspace configuration engine      │
│   ├── internal/websocket   : Concurrent event broadcast hub                 │
│   └── internal/server      : HTTP router, REST APIs & embedded dashboard UI │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │ HTTP / WebSocket (:9999)
                                       ▼
                         SECONDARY DEVICE (Tablet/Phone)
┌─────────────────────────────────────────────────────────────────────────────┐
│   Catalyst Dashboard (Linear/Raycast aesthetic)                             │
│   ├── Live Application Preview with Device Frame Emulation                  │
│   ├── One-Click Dev Server Runner (npm run dev / go run / cargo run)        │
│   ├── Real-Time Terminal Output Stream (stdout / stderr)                   │
│   ├── File Change Activity Log with Action Pills (Write, Create, Delete)    │
│   ├── Multi-Workspace Project Switcher Dropdown                             │
│   ├── Live System Health Indicators (RAM %, CPU, Uptime)                    │
│   └── Mobile QR Code & PIN Pairing Verification                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Package Architecture

| Package | Purpose & Responsibilities |
| :--- | :--- |
| `cmd/catalyst` | Application main entry point, CLI argument dispatch, signal handling (`SIGINT`, `SIGTERM`), and graceful server shutdown. |
| `internal/cli` | Subcommand router (`dev`, `stop`, `status`, `doctor`, `version`, `help`), build version metadata, and localhost daemon status query. |
| `internal/config` | Persistent workspace configuration manager supporting `.catalyst/config.json` and `catalyst.config.json` with automatic merging and live reload. |
| `internal/monitor` | Cross-platform host metrics collector. Uses Windows API `GlobalMemoryStatusEx` for accurate physical RAM metrics, with POSIX fallbacks. |
| `internal/pairing` | Device pairing core: generates 6-character alphanumeric PINs, issues session tokens, tracks active secondary devices, and creates vector SVG QR codes. |
| `internal/process` | Supervised process runner capturing stdout/stderr with rolling 500-line buffers and preventing orphan processes via Windows process tree termination (`taskkill /T`). |
| `internal/project` | Detects project frameworks (`Go`, `Node.js`, `Next.js`, `Vite`, `React`, `Python/Flask/Django/FastAPI`, `Rust`, `Static`) and extracts runnable script manifests. |
| `internal/proxy` | Reverse proxy forwarding traffic to local development servers (`/proxy/{port}/*`), rewriting Host headers, and streaming WebSockets for HMR. |
| `internal/runtime` | Environment doctor checking compiler versions (`go`, `node`, `npm`, `git`, `python`, `rustc`) and validating workspace dependency health (`node_modules`, `.venv`). |
| `internal/server` | HTTP server multiplexer hosting REST APIs, static file preview at `/preview/`, reverse proxy at `/proxy/`, and embedded dashboard assets. |
| `internal/watcher` | Recursive `fsnotify` file monitor with dynamic subfolder registration, file action metadata (`write`, `create`, `delete`, `rename`), and 100ms debouncing. |
| `internal/websocket` | Thread-safe WebSocket hub maintaining client connections, heartbeat pings, and JSON event broadcasts. |
| `internal/workspace` | Multi-workspace registry tracking project paths, active statuses, and facilitating live project switching without daemon restarts. |

---

## 3. REST API Specification

### Workspace & Projects
* `GET /api/workspace`: Returns active project metadata, detected framework, root directory, and connected client count.
* `GET /api/workspaces`: Lists all registered workspaces.
* `POST /api/workspaces`: Registers a new project directory path into the registry.
* `POST /api/workspaces/switch`: Dynamically switches the active workspace, updates watcher root, and reloads configuration.

### Development Server Runner
* `GET /api/dev/scripts`: Returns runnable scripts extracted from project configuration and active dev server status.
* `POST /api/dev/start`: Launches a vetted project script (e.g., `dev`, `start`) within the workspace directory.
* `POST /api/dev/stop`: Terminates the running dev server.
* `POST /api/dev/restart`: Restarts the active dev server.

### Diagnostics & Monitoring
* `GET /api/system`: Returns host CPU cores, physical RAM utilization (used/total MB and percent), uptime, OS, and architecture.
* `GET /api/doctor`: Runs full toolchain inspection and returns diagnostic check items.
* `GET /api/processes`: Returns list of all supervised processes with execution state, PID, and recent log buffers.

### Configuration
* `GET /api/config`: Returns persistent workspace configuration.
* `POST /api/config`: Updates workspace configuration (target preview URL, device frame, auto-reload, custom ignore rules) and saves to disk.

### Pairing & Access Control
* `GET /api/pairing`: Returns active 6-character pairing PIN and list of connected devices.
* `POST /api/pairing/verify`: Verifies pairing PIN, authorizes device, and issues a session token.
* `GET /api/pairing/qr`: Delivers an SVG QR code encoding direct mobile pairing link.

### Lifecycle
* `POST /api/shutdown`: Gracefully terminates Catalyst daemon. **Restricted exclusively to loopback connections (127.0.0.1 / localhost).**

---

## 4. WebSocket Event Protocol

All real-time messages transmitted over `/ws` adhere to the standard envelope:
```json
{
  "type": "EVENT_NAME",
  "payload": { ... }
}
```

| Event Type | Payload | Trigger |
| :--- | :--- | :--- |
| `FILE_CHANGED` | `{"file": "...", "action": "write|create|delete|rename", "timestamp": "..."}` | Emitted when a project file is saved, created, or deleted. |
| `DEV_SERVER_STATE` | `{"running": true|false, "pid": 1234, "script": "dev"}` | Emitted when the managed dev server starts, stops, or fails. |
| `PROCESS_OUTPUT` | `{"id": "...", "stream": "stdout|stderr", "text": "...", "timestamp": "..."}` | Emitted for every line of compiler or dev server console output. |
| `PROCESS_STARTED` | `{"id": "...", "pid": 1234, "command": "..."}` | Emitted when a supervised process begins execution. |
| `PROCESS_STOPPED` | `{"id": "...", "state": "stopped|failed", "exitCode": 0}` | Emitted when a process exits. |
| `SYSTEM_STATS` | `{"memory": {"usedPct": 45.2, ...}, ...}` | Periodic broadcast (every 5s when clients are connected). |
| `CONFIG_UPDATED` | `{ ... Config ... }` | Emitted when workspace settings are modified. |
| `WORKSPACE_CHANGED`| `{"id": "...", "name": "...", "path": "...", "framework": "..."}` | Emitted when active workspace project is switched. |

---

## 5. Security Architecture

1. **Remote Code Execution (RCE) Prevention:**
   Catalyst strictly disallows unauthenticated execution of arbitrary commands from the network. The Dev Runner endpoint only permits execution of scripts extracted directly from the verified project manifest (`package.json`, `go.mod`, `Cargo.toml`).
2. **Localhost-Restricted Shutdown:**
   The `/api/shutdown` endpoint verifies `r.RemoteAddr` and rejects all non-loopback requests with `403 Forbidden`, preventing unauthorized remote network clients from terminating the developer's server.
3. **Session Pairing Protection:**
   Connecting secondary devices can be verified via a 6-character PIN or QR code, issuing a session token to protect private local network previews on shared Wi-Fi networks.
