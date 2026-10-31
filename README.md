# yawa — a customizable Windows hotkey launcher

> A fast, local-first Windows 11 launcher built with Go. Open projects, applications, folders, files, and URLs from the terminal or configurable global shortcuts.

## Overview

`yawa` is a small Windows `.exe` that lets you:

- Create, edit, remove, and run named commands for **any installed desktop application**, files, folders, and URLs from CMD.
- Assign global shortcuts to those commands, even when CMD is not focused.
- Optionally select a folder or file and pass its path to **any configured app** (VS Code, Visual Studio, Explorer, Photoshop, VLC, etc.).
- Customize command aliases, target app executable paths, arguments, paths, working directories, and hotkeys through a TOML file or CLI management commands—no recompilation.
- Keep a background hotkey agent running in the Windows system tray (later milestone), or start it from a terminal.

The name comes from a Bisaya expression Charles often uses when software gets annoying. The product goal follows from that: remove small, repeated annoyances without becoming another heavy launcher.

**Example experience:**

```bat
yawa run code
:: Opens a folder picker, then opens the selected folder in VS Code.

yawa run api
:: Opens a saved project in VS Code.

yawa list
:: Prints all configured commands and hotkeys.

yawa agent
:: Keeps global shortcuts active while this process runs.
```

With `yawa agent` running, pressing a configured shortcut invokes its command without opening a terminal.

> **Important:** A shortcut works globally only while the agent is running and only if Windows or another application does not already own it. `Ctrl+1` is supported, but it is a poor default because browsers, editors, and terminals commonly use it for tab switching. Prefer defaults such as `Ctrl+Alt+1`. `yawa` must report collisions without preventing other valid bindings from loading. A normal user process does not operate on the Windows secure desktop and should not elevate itself automatically.

## Product assessment

This is a good project for Go and a useful Windows tool. It has a narrow core, a clear user benefit, and can be distributed as one executable. Go is especially suitable for the configuration, CLI, validation, process management, and concurrency layers. The small amount of Win32 and COM integration should live behind focused Windows-only interfaces rather than leaking into the command engine.

The main engineering risks are not raw performance. They are shortcut conflicts, Windows thread-affinity rules, quoting paths correctly, duplicate resident agents, configuration corruption, and actions that block the Win32 message loop. The architecture below addresses those risks from the start.

The initial product should stay intentionally small: a command registry plus two invocation paths—CLI and hotkeys. A tray UI, startup integration, and live reload are valuable only after the core is reliable.

## Goals and non-goals

### MVP goals

1. Single Windows executable (`yawa.exe`) written in Go.
2. `run`, `list`, `validate`, `config path`, and `agent` subcommands.
3. User-editable command aliases and shortcut bindings.
4. Global shortcut listener (`RegisterHotKey` + Windows message loop).
5. Generic app/file/folder/URL launcher with optional native file or folder picker, not hardcoded to VS Code.
6. Create/edit/delete commands via CLI as well as by editing TOML.
7. Safe application execution without passing input through `cmd.exe` by default.
8. Helpful errors and logs for missing apps, invalid configuration, and hotkey conflicts.

### Not in the first MVP

- Floating search UI, plugin marketplace, macros, arbitrary script downloads.
- System tray UI and automatic Windows startup (planned for v1.1).
- macOS and Linux support (Windows-only to start).

## Tech stack

| Layer | Choice | Purpose |
|---|---|---|
| Language | Go | Build a single `.exe` and implement the CLI, config, and execution engine |
| CLI | Cobra | Stable subcommands, validation, help, and shell completion |
| Config | TOML parser, e.g. `github.com/BurntSushi/toml` | Read human-friendly command definitions |
| Windows integration | `golang.org/x/sys/windows` / small Windows-specific wrapper | Register and dispatch global hotkeys using Win32 APIs |
| Picker | Windows native file/folder dialog (`IFileOpenDialog`; `FOS_PICKFOLDERS` for folders) | Pick a file or directory for any command |
| Execution | `os/exec`, `exec.Command` | Launch programs using explicit executable and argument arrays |
| Data storage | `%APPDATA%\yawa\config.toml` | Versioned per-user configuration |
| Logging | `log/slog` | Structured local logs with sensitive arguments omitted |

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

- `yawa agent` is the single long-running owner of registered hotkeys. Enforce one agent per Windows user with a named mutex.
- The Win32 message-loop goroutine is locked to its OS thread. It translates `WM_HOTKEY` into a small immutable job and immediately queues it; it never launches applications or opens dialogs itself.
- A bounded dispatcher executes jobs. Repeated presses are de-bounced and `MOD_NOREPEAT` is used where supported. A full queue produces a warning rather than unbounded goroutines.
- COM picker work runs on a dedicated, locked OS thread initialized as an STA. Cancellation is a normal outcome, not an error.
- CLI invocations load and validate the same registry and call the same dispatcher. Inter-process control can be added later if single-instance picker behavior requires it; named pipes are the preferred Windows transport.
- Shutdown stops accepting jobs, unregisters every successfully registered hotkey, drains in-flight work with a timeout, releases COM resources, and releases the mutex.

## Command model: generic, not editor-specific

Every command is a user-defined alias to an **action**. None of the actions know anything special about VS Code.

| Type | Behavior | Example |
|---|---|---|
| `exec` | Run a specific executable with an argument array; optionally inject a saved path | Open VS Code, Visual Studio, Blender, Photoshop, or a custom `.exe` |
| `pick_folder` | Show native folder picker, then invoke any target executable with the selected path | Pick repository → VS Code; pick folder → Explorer or another IDE |
| `pick_file` | Show native file picker, then invoke any target executable with selected file | Pick image → paint app; pick video → VLC |
| `open_path` | Open a configured file or directory with its Windows default associated application, or Explorer | Open Downloads, a PDF, or a spreadsheet |
| `url` | Open a valid URL in default browser | GitHub, documentation |

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
- After editing commands or hotkeys, restart `agent` for MVP (support live reload in v1.1).

### Customization workflows

**From CMD (implemented in Phase 2):**

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

## Suggested project structure

```text
yawa/
├── README.md
├── go.mod
├── go.sum
├── cmd/
│   └── yawa/
│       └── main.go             # CLI entry point
├── internal/
│   ├── cli/                    # run/list/validate/config/agent commands
│   ├── config/                 # load, parse, validate TOML
│   ├── commands/               # registry and action dispatcher
│   ├── launcher/               # generic app, path, and URL invocation
│   ├── hotkeys/                # Windows RegisterHotKey + message loop
│   └── dialogs/                # native file and folder pickers
├── configs/
│   └── example.toml
├── scripts/
│   └── build.ps1
└── tests/
    └── integration/           # Windows integration tests
```

Use Windows build tags (`//go:build windows`) for native OS modules, and keep parsing, validation, and registry code platform-neutral where practical.

## Developer setup (Windows 11)

Prerequisites:

1. Install a supported Go release.
2. Install Visual Studio Code (optional, used by example commands).
3. Ensure the VS Code `code` CLI is available in your `PATH`, or configure the absolute executable path.

```powershell
# Verify environment
go version
where.exe code

# Clone and enter the project (replace <owner> with the repository owner)
git clone https://github.com/<owner>/yawa.git
cd yawa

# After implementing the files shown above:
go mod tidy
go test ./...
go build -trimpath -o .\bin\yawa.exe .\cmd\yawa

# Try the CLI
.\bin\yawa.exe validate
.\bin\yawa.exe run code

# In a terminal that remains open:
.\bin\yawa.exe agent
```

When bootstrapping a new repository rather than cloning one, run `go mod init github.com/<owner>/yawa` once using the final hosting path.

**Make the command available everywhere:** Add the absolute `yawa\bin` directory to your user `PATH`, open a fresh terminal, then run `yawa list`. Do not overwrite the existing PATH value.

## Implementation plan

### Phase 1 — Generic action engine and CLI

- Define a typed command registry and action interface. Do not include VS Code-specific logic.
- Implement the `exec`, `open_path`, and `url` action types; support arbitrary configured installed apps, arguments, and optional `working_dir`.
- Add `list`, `run`, `validate`, `config path`, and `version`.
- For executables use `exec.Command(executable, args...)`; for default-associated files, folders, and URLs use Windows shell APIs.
- Validate types, named commands, file paths, and executable resolution.
- Test launching unrelated apps (e.g. Calculator), opening a directory, and opening a PDF/URL.

**Acceptance:** `yawa run calculator` launches Calculator and `yawa run downloads` opens Downloads. Neither is coded as a special case.

### Phase 2 — User-defined commands and persistence

- Implement `yawa command add`, `edit`, `show`, `remove` with type-specific flags and interactive prompts where necessary.
- Permit users to choose **which app**, saved file/path, arguments, starting directory, working directory, and optional hotkey.
- Validate and atomically write TOML via a temporary file and rename; back up config before changes and avoid overwriting concurrent edits.
- Support edit-by-file (`config edit`) for advanced configurations without recompiling.
- Test creating, updating, removing, and persisting aliases across restarts.

**Acceptance:** A user can add a command for a new `.exe` entirely through CMD, run it, modify its arguments/hotkey, and remove it without changing Go source.

### Phase 3 — Native file and folder pickers for any app

- Implement Windows COM dialogs for both folder and file selection.
- Add `pick_folder` and `pick_file` actions with configurable `executable` and `args`.
- Substitute `{selected_path}` as a single argument; support Unicode and paths with spaces.
- Cancel cleanly without launching an app; gracefully report incompatible arguments.
- Test folder → VS Code, file → Paint, and folder/file → another custom configured application.

**Acceptance:** `yawa run code` opens VS Code with the chosen folder; `yawa run edit-image` opens the selected image in Paint. Both use the same generic picker dispatcher.

### Phase 4 — Global keyboard shortcut agent

- Parse configurable hotkeys (e.g. `Ctrl+Alt+1`, `Ctrl+Alt+I`) to Win32 modifiers and keycodes; use `MOD_NOREPEAT`.
- Register via `RegisterHotKey` and process messages in a dedicated Windows thread/message loop.
- Dispatch hotkey invocations to the **same action engine** used by `yawa run`.
- Log collisions, deregister on shutdown, prevent duplicate agents with a per-user named mutex, and keep all action work off the message-loop thread.
- Use a bounded work queue and debounce repeated launches so a held or repeatedly pressed key cannot create an unbounded number of processes.
- Support new key bindings by restarting the agent in MVP; add live reload later.

**Acceptance:** After `yawa agent`, a configured `Ctrl+Alt+1` launches the folder picker and `Ctrl+Alt+I` launches the file picker, even when CMD is unfocused.

### Phase 5 — Reliability, build, and distribution

- Add structured logs, redaction for sensitive arguments, informative errors, and meaningful exit codes.
- Include config examples for VS Code, Visual Studio IDE, Windows Explorer, Paint, media players, and arbitrary `.exe` apps.
- Add tests for bad paths, missing applications, invalid TOML, duplicate shortcuts, Unicode paths, process arguments, and cancellation.
- Build and document `yawa.exe`, PATH installation, required user permissions, and a Windows 11 smoke test.
- Optional v1.1: system tray, auto-start at login, config/hotkey reload, fuzzy search, command history.

**Acceptance:** New commands and hotkeys are added without code edits, VS Code is optional, and a different Windows 11 machine can configure its own apps and paths independently.

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

- [ ] Users can create, update, inspect, and delete arbitrary named commands using CMD or TOML.
- [ ] A command can launch **any configured application** with arguments, or open a file, folder, or URL.
- [ ] Both file and folder pickers work with different applications—not only VS Code.

- [ ] `yawa.exe` is callable from any CMD / PowerShell directory.
- [ ] `yawa run code` shows a Windows folder picker and opens the chosen folder in VS Code.
- [ ] A configured command opens a saved project/solution in the user-selected IDE without a picker.
- [ ] `yawa agent` registers `Ctrl+Alt+1` and other valid configured hotkeys.
- [ ] Hotkeys work while CMD is unfocused, provided they are not reserved or intercepted.
- [ ] Commands and hotkeys can be changed in `config.toml` without recompilation.
- [ ] Invalid configuration and shortcut collisions produce actionable error messages.
- [ ] Closing the agent unregisters its hotkeys.

## Future direction

yawa starts as a keyboard-first CLI utility, **not a full desktop search app**. Once the CLI and hotkey agent are stable, a search overlay or settings interface can be added without replacing the command-dispatch core.

**Project philosophy:** One registry, multiple ways to invoke it—terminal commands today, keyboard shortcuts always, and an optional UI later.

