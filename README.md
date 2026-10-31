<p align="center">
  <img src="assets/branding/yawa-logo-master.png" alt="Yawa logo" width="120">
</p>

# yawa - a customizable Windows hotkey launcher

**Why “yawa”?** `Yawa` is the Bisaya word I blurt out when a laptop full of open apps freezes while I am searching for—or reopening—something. Yawa turns that frustration into configurable commands and global shortcuts.

> A fast, local-first Windows 11 launcher built with Go. Open projects, applications, folders, files, and URLs from the terminal or configurable global shortcuts.

Yawa v1 ships three focused executables: `yawa.exe` for the CLI, `yawa-ui.exe` for visual customization, and the headless `yawa-agent.exe` for global shortcuts and the system tray. All three use the same local configuration file.

## Install on Windows 11

### Install from a GitHub Release (recommended)

Download the installer from the latest GitHub Release, then run it. The installer verifies the release archive's SHA-256 checksum, installs all three executables under `%LOCALAPPDATA%\Programs\yawa`, adds that directory to your user `PATH`, and creates a **Yawa UI** Start menu shortcut.

```powershell
$installer = Join-Path $env:TEMP "install-yawa.ps1"
Invoke-WebRequest "https://github.com/charleszardd/yawa/releases/latest/download/install.ps1" -OutFile $installer
powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer
```

Open **Yawa UI** from the Start menu or run `yawa-ui` in a new terminal. Create a command, save it, then select **Start** in the sidebar. Enable **Start with Windows** if you want shortcuts available automatically after sign-in. The UI may be closed afterward; the lightweight tray agent keeps running. No administrator privileges or Windows service are required.

### Install directly from GitHub with Go

Use `go install`, not `go get`. This requires Go to be installed and the GitHub repository to be public, or your machine to be authenticated for the private repository.

```powershell
go install github.com/charleszardd/yawa/cmd/yawa@latest
go install github.com/charleszardd/yawa/cmd/yawa-agent@latest
go install -tags production github.com/charleszardd/yawa/cmd/yawa-ui@latest
```

Go writes the executables to `$(go env GOPATH)\bin`. Add that directory to your user `PATH`, open a new terminal, then run `yawa-ui`. Pin a release such as `@v1.0.0` instead of `@latest` when you need a reproducible installation. GitHub Release installation is preferred because its GUI executables are built without console windows and its archive is checksum-verified.

## Overview

`yawa` is a small Windows `.exe` that lets you:

- Create, edit, remove, and run named commands for **any installed desktop application**, files, folders, and URLs from CMD.
- Assign global shortcuts to those commands, even when CMD is not focused.
- Optionally select a folder or file and pass its path to **any configured app** (VS Code, Visual Studio, Explorer, Photoshop, VLC, etc.).
- Customize command aliases, target app executable paths, arguments, paths, working directories, and hotkeys through a TOML file or CLI management commands—no recompilation.
- Keep a managed background hotkey agent in the Windows system tray, reload it after changes, and optionally start it at Windows sign-in.

**Example experience:**

```bat
yawa run code
:: Opens a folder picker, then opens the selected folder in VS Code.

yawa run api
:: Opens a saved project in VS Code.

yawa list
:: Prints all configured commands and hotkeys.

yawa agent
:: Runs the terminal-mode agent for development or diagnostics.
```

For normal daily use, start the background agent from Yawa UI. Pressing a configured shortcut then invokes its command without opening a terminal.

> **Important:** A shortcut works globally only while the agent is running and only if Windows or another application does not already own it. `Ctrl+1` is supported, but it is a poor default because browsers, editors, and terminals commonly use it for tab switching. Prefer defaults such as `Ctrl+Alt+1`. `yawa` must report collisions without preventing other valid bindings from loading. A normal user process does not operate on the Windows secure desktop and should not elevate itself automatically.

## Product assessment

This is a good project for Go and a useful Windows tool. It has a narrow core and a clear user benefit. Go is especially suitable for the configuration, CLI, validation, process management, and concurrency layers. The Win32 and COM integration stays behind focused Windows-only interfaces rather than leaking into the command engine.

The main engineering risks are not raw performance. They are shortcut conflicts, Windows thread-affinity rules, quoting paths correctly, duplicate resident agents, configuration corruption, and actions that block the Win32 message loop. The architecture below addresses those risks from the start.

The product stays intentionally small: one command registry with CLI, UI, and hotkey invocation paths. The v1 tray, startup integration, and live reload all reuse that core.

## Tech stack

| Layer               | Choice                                                                                   | Purpose                                                                     |
| ------------------- | ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Language            | Go                                                                                       | Build a single `.exe` and implement the CLI, config, and execution engine |
| CLI                 | Cobra                                                                                    | Stable subcommands, validation, help, and shell completion                  |
| Config              | TOML parser, e.g. `github.com/BurntSushi/toml`                                         | Read human-friendly command definitions                                     |
| Windows integration | `golang.org/x/sys/windows` / small Windows-specific wrapper                            | Register and dispatch global hotkeys using Win32 APIs                       |
| Picker              | Windows native file/folder dialog (`IFileOpenDialog`; `FOS_PICKFOLDERS` for folders) | Pick a file or directory for any command                                    |
| Execution           | `os/exec`, `exec.Command`                                                            | Launch programs using explicit executable and argument arrays               |
| Data storage        | `%APPDATA%\yawa\config.toml`                                                           | Versioned per-user configuration                                            |
| Logging             | `log/slog`                                                                             | Structured local logs with sensitive arguments omitted                      |

**Note:** Windows API message loops and COM dialogs require careful OS-thread handling in Go. Keep them in dedicated Windows-only modules; lock the relevant goroutine to an OS thread using `runtime.LockOSThread()` when appropriate and initialize COM for dialogs correctly.

## System architecture

```text
                         Windows PC
 ┌────────────────────────────────────────────────────────┐
 │                                                        │
 │  CMD / PowerShell                 Global Shortcut       │
 │  yawa run api                     Ctrl+Alt+1            │
 │         │                             │                 │
 │         ▼                             ▼                 │
 │  CLI Command Parser            Agent (resident)        │
 │         │                      RegisterHotKey           │
 │         └────────────┬────────────────┘                 │
 │                      ▼                                 │
 │               Command Registry                         │
 │            config.toml + validation                    │
 │                      │                                 │
 │                      ▼                                 │
 │               Action Dispatcher                        │
 │            ┌─────────┼──────────┐                      │
 │            ▼         ▼          ▼                      │
 │         app launch   folder     URL                    │
 │         / project    picker     opener                 │
 │            │         │          │                      │
 │            └─────────┴──────────┘                      │
 │                      ▼                                 │
 │           Windows process / shell APIs                 │
 └────────────────────────────────────────────────────────┘
```

**Design principle:** Both terminal commands and hotkeys invoke the **same validated action dispatcher**. Avoid two different implementations for the same action.

### Runtime design

- `yawa-agent.exe` is the managed, single long-running owner of registered hotkeys. A named mutex enforces one background agent per Windows user. `yawa agent` remains available as a terminal-mode diagnostic alternative.
- The UI controls the agent through named Windows events, reports whether shortcuts are actually active, and reloads bindings after saves. A separate readiness event prevents a running-but-failed process from being shown as healthy.
- **Start with Windows** writes a quoted, per-user `HKCU\...\Run` entry. Enabling it also starts the agent immediately and rolls the registry change back if startup fails.
- The Win32 message-loop goroutine is locked to its OS thread. It translates `WM_HOTKEY` into a small immutable job and immediately queues it; it never launches applications or opens dialogs itself.
- A bounded dispatcher executes jobs. Repeated presses are de-bounced and `MOD_NOREPEAT` is used where supported. A full queue produces a warning rather than unbounded goroutines.
- COM picker work runs on a dedicated, locked OS thread initialized as an STA. Cancellation is a normal outcome, not an error.
- CLI invocations load and validate the same registry and call the same dispatcher. Inter-process control can be added later if single-instance picker behavior requires it; named pipes are the preferred Windows transport.
- Shutdown stops accepting jobs, unregisters every successfully registered hotkey, drains in-flight work with a timeout, releases COM resources, and releases the mutex.

## Command model: generic, not editor-specific

Every command is a user-defined alias to an **action**. None of the actions know anything special about VS Code.

| Type            | Behavior                                                                                         | Example                                                               |
| --------------- | ------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------- |
| `exec`        | Run a specific executable with an argument array; optionally inject a saved path                 | Open VS Code, Visual Studio, Blender, Photoshop, or a custom `.exe` |
| `pick_folder` | Show native folder picker, then invoke any target executable with the selected path              | Pick repository → VS Code; pick folder → Explorer or another IDE    |
| `pick_file`   | Show native file picker, then invoke any target executable with selected file                    | Pick image → paint app; pick video → VLC                            |
| `open_path`   | Open a configured file or directory with its Windows default associated application, or Explorer | Open Downloads, a PDF, or a spreadsheet                               |
| `url`         | Open a valid URL in default browser                                                              | GitHub, documentation                                                 |

The `executable` can be an absolute path (recommended for GUI apps), or a known application available on PATH. Use `exec.Command` with separate arguments; for Windows shell-associated files/URLs, use `ShellExecuteEx` or equivalent. A selected path is substituted into **one** argument, never concatenated into shell text.

### Editable configuration

File: `%APPDATA%\yawa\config.toml`

```toml
[general]
config_version = 1
log_level = "info"

# 1. Folder picker → VS Code (example only)
[[commands]]
name = "code"
description = "Choose a folder and open it in VS Code"
type = "pick_folder"
executable = "code"
args = ["--new-window", "{selected_path}"]
start_dir = "desktop"
hotkey = "Ctrl+Alt+1"

# 2. Saved folder → another application (Visual Studio IDE)
# Configure the actual devenv.exe location on your PC.
[[commands]]
name = "visual-studio"
description = "Open a saved solution in Visual Studio"
type = "exec"
executable = 'C:\Program Files\Microsoft Visual Studio\2022\Community\Common7\IDE\devenv.exe'
args = ['D:\Projects\SampleApp\SampleApp.sln']
hotkey = "Ctrl+2"

# 3. Pick an image → any editor
[[commands]]
name = "edit-image"
type = "pick_file"
executable = "mspaint.exe"
args = ["{selected_path}"]
start_dir = "pictures"
hotkey = "Ctrl+Alt+I"

# 4. Saved directory → default Windows file manager
[[commands]]
name = "downloads"
type = "open_path"
path = "{downloads}"
hotkey = "Ctrl+Alt+D"

# 5. Any GUI app without a path
[[commands]]
name = "calculator"
type = "exec"
executable = "calc.exe"
args = []

# 6. URL → default browser
[[commands]]
name = "github"
type = "url"
url = "https://github.com"
hotkey = "Ctrl+Alt+G"
```

> TOML note: Windows paths use **single-quoted literal strings** to avoid backslash escaping, e.g. `executable = 'C:\Program Files\App\app.exe'`. The application paths above are examples; adjust them to your installed versions.

### Config rules

- `name` is unique, case-insensitive, and matches `[a-z0-9-]+`.
- `type` is one of `exec`, `pick_folder`, `pick_file`, `open_path`, `url`.
- `hotkey` is optional; malformed or duplicate shortcuts fail validation.
- `exec`, `pick_folder`, `pick_file` need an `executable`; only picker types need a `{selected_path}` placeholder in at least one argument.
- `args` is an array of distinct process arguments, **never** a command-line string passed through a shell.
- `working_dir` is optional for `exec` and picker commands; check that it exists before launch.
- `path` is required for `open_path`; allow known-directory placeholders like `{downloads}` and `{desktop}` or a normal absolute path.
- `start_dir` accepts `desktop`, `documents`, `downloads`, `pictures`, or an existing absolute path.
- A target application receives the picked path **only if configured to** (some applications require special flags or a solution/project file).
- Resolve application executables explicitly; report when missing. Avoid assuming an application is installed or accepting arbitrary executable paths obtained from untrusted input.
- URL launches allow `https://` by default; `http://` may be enabled as an explicit preference.
- Common shortcuts such as `Ctrl+1` are accepted but discouraged; registration failures must identify the conflicting binding while other hotkeys continue working.
- UI saves reload a running background agent automatically. TOML or CLI edits require selecting **Reload** in the UI/tray or restarting a terminal-mode agent.

### Customization workflows

**From CMD :**

```bat
yawa command add calculator --type exec --exe calc.exe
yawa command add downloads --type open_path --path "%USERPROFILE%\Downloads" --hotkey "Ctrl+Alt+D"
yawa command edit calculator --hotkey "Ctrl+3"
yawa command show calculator
yawa command remove calculator
yawa run downloads
```

`command add/edit/remove` must persist changes atomically to TOML; preserve other commands, validate before writing, and back up the previous version. Env-var expansion (such as `%USERPROFILE%`) is applied only to *documented path fields* during execution, not arbitrary `args` unless explicitly designed. Alternatively, use the config editor to create complex commands.

**From TOML:** Copy a `[[commands]]` block, change `name`, `type`, `executable`, `args`/`path`, and optional `hotkey`. Then run `yawa validate`.

## CLI specification

```text
yawa --help
yawa list
yawa run <command-name>
yawa validate
yawa command add <name> --type <type> [type-specific flags]
yawa command edit <name> [flags]
yawa command show <name>
yawa command remove <name>
yawa config path
yawa config edit
yawa agent
yawa version
```

Expected behavior:

- `command add/edit/show/remove`: Manages custom commands, validates updates, and persists user config atomically.
- `run`: Reads config, validates the requested command, executes it, and returns a meaningful exit status.
- `list`: Prints command aliases, descriptions, and assigned hotkeys.
- `validate`: Finds syntax errors, duplicate aliases, malformed arguments, and invalid shortcut definitions.
- `config path`: Shows config location.
- `config edit`: Opens config in the default text editor (or prints the path if editor unavailable).
- `agent`: Registers hotkeys, processes Windows hotkey messages, logs failures, and unregisters shortcuts on shutdown.

Example:

```powershell
PS> yawa list
NAME      HOTKEY        ACTION
code            Ctrl+Alt+1    Pick folder -> VS Code
visual-studio   Ctrl+2        Open Visual Studio solution
edit-image      Ctrl+Alt+I    Pick file -> Paint
downloads       Ctrl+Alt+D    Open Downloads
github          Ctrl+Alt+G    Open GitHub
```

## Developer setup (Windows 11)

Prerequisites:

1. Install a supported Go release.
2. Install Visual Studio Code (optional, used by example commands).
3. Ensure the VS Code `code` CLI is available in your `PATH`, or configure the absolute executable path.
4. Install GNU Make if you want to use the development shortcuts below.

### Development with Make

These commands run from source and do not use `dist`:

```powershell
make frontend-install       # First-time frontend dependency setup
make check                  # Frontend build, Go tests, and static analysis
make ui                     # Run the Wails settings UI from source
make agent                  # Run the global hotkey agent from source
make background-agent       # Run the managed tray agent from source
make cli ARGS="list"        # Run any CLI command
make cli ARGS="run code"    # Execute a configured command
```

Use a non-default configuration when needed:

```powershell
make validate CONFIG=".\configs\example.toml"
make agent CONFIG=".\configs\example.toml"
```

`make build` is optional and writes development executables to the ignored `bin\` directory. Production/release artifacts remain the responsibility of `scripts\build.ps1`.

```powershell
# Verify environment
go version
where.exe code

# Clone and enter the project (replace <owner> with the repository owner)
git clone https://github.com/charleszardd/yawa.git
cd yawa

go mod tidy
go test ./...
make build

# Try the CLI
.\bin\yawa.exe validate
.\bin\yawa.exe run code

# In a terminal that remains open:
.\bin\yawa.exe agent
```

The Go module path is `github.com/charleszardd/yawa` and must stay aligned with the repository URL.

### Desktop customization UI

Launch the Wails settings application after building:

```powershell
.\dist\yawa-ui.exe
```

The UI can create, search, edit, validate, test, and delete commands. It detects common installed applications, provides a native `.exe` browser, and validates executable paths before saving. Fields change with the selected action type, and picker actions explain the required `{selected_path}` argument. Saved changes are written atomically to the same `%APPDATA%\yawa\config.toml` used by the CLI.

Use the sidebar to start, reload, or stop the background agent. Saving a command reloads a running agent automatically. **Start with Windows** is opt-in and per-user. The settings window does not need to remain open for shortcuts to work; use the tray icon to reopen Yawa, reload shortcuts, or exit.

UI development uses Wails v2 and React with TypeScript:

```powershell
cd .\cmd\yawa-ui\frontend
npm ci
npm run build
cd ..\..\..
go build -tags production -o .\dist\yawa-ui.exe .\cmd\yawa-ui
```

Windows 11 normally includes the required WebView2 runtime. Use `wails doctor` if the UI cannot initialize its webview.

If a blank Yawa window remains after an earlier failed launch, close every existing Yawa window and run `make ui` again. Yawa uses a single-instance lock and stores WebView2 state in the stable, writable `%LOCALAPPDATA%\yawa\webview2` directory so `go run` does not depend on a temporary executable directory.

Yawa writes bounded, structured JSON logs to `%LOCALAPPDATA%\yawa\yawa.log`; the previous file is retained as `yawa.log.1` after rotation. Command names, types, and errors are logged, but argument arrays are deliberately omitted.

Exit codes are stable for automation:

|  Code | Meaning                        |
| ----: | ------------------------------ |
| `0` | Success                        |
| `1` | General/internal failure       |
| `2` | Invalid configuration or usage |
| `3` | Command not found              |
| `4` | Action launch failure          |
| `5` | Hotkey agent failure           |

For daily use, activate global shortcuts from Yawa UI and optionally enable **Start with Windows**. Use `yawa agent` only when you specifically want foreground terminal diagnostics.

**Make the command available everywhere:** Add the absolute `yawa\bin` directory to your user `PATH`, open a fresh terminal, then run `yawa list`. Do not overwrite the existing PATH value.

## Reliability and security guidelines

- **No implicit shell execution:** Do not run `cmd /c` or PowerShell for arbitrary user-entered strings. Shell-specific actions can be added later as an explicit, clearly labeled command type.
- **Treat config as trusted but validate it:** Only launch the executables or URLs declared in the user's local config.
- **Don't assume hotkey registration succeeds:** Windows and other apps can reserve or consume shortcut combinations.
- **Don't block the Win32 message pump:** A folder picker or slow process must not freeze shortcut handling.
- **Don't elevate automatically:** Run with normal user privileges by default; don't request administrator access for routine use.
- **No telemetry by default:** Store config and logs locally; redact sensitive command arguments in logs where possible.
- **No busy polling:** Use Windows hotkey messages rather than repeatedly checking keyboard state.
- **Graceful shutdown:** Release registered shortcuts and native resources.
- **Single instance:** Allow only one hotkey agent per user and explain how to find or stop an existing instance.
- **Bounded concurrency:** Never create an unbounded goroutine or process storm from repeated hotkey input.
- **Atomic configuration:** Write a sibling temporary file, flush and close it, replace the destination, and retain one recoverable backup. Reject unsupported `config_version` values.
- **Stable identity:** Generate a deterministic, collision-free numeric registration ID for each validated hotkey during agent startup; do not use array positions that change after edits.
- **Actionable diagnostics:** Include command name and error category, but never log secrets or the full argument list by default.

## Success criteria

- [X] Users can create, update, inspect, and delete arbitrary named commands using CMD or TOML.
- [X] A command can launch **any configured application** with arguments, or open a file, folder, or URL.
- [X] Both file and folder pickers use the generic application dispatcher—not VS Code-specific code.
- [X] `yawa run code` uses the native Windows folder picker and passes the selected folder as one argument.
- [X] A configured command can open a saved project/solution in any selected IDE without a picker.
- [X] `yawa agent` registers `Ctrl+Alt+1` and other valid configured hotkeys.
- [X] Hotkeys work while CMD is unfocused, provided they are not reserved or intercepted.
- [X] Commands and hotkeys can be changed in `config.toml` without recompilation.
- [X] Invalid configuration and shortcut collisions produce actionable error messages.
- [X] Closing the agent unregisters its hotkeys.
- [X] The UI can start, reload, stop, and monitor a headless tray agent.
- [X] Start with Windows is opt-in, per-user, and requires no elevation.

## Future direction

yawa remains a keyboard-first launcher, **not a full desktop search app**. Future overlays or integrations can reuse the same validated command-dispatch core.

**Project philosophy:** One registry, multiple ways to invoke it—terminal, UI, and global shortcuts.
