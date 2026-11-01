# Manual acceptance

English | [中文](manual-test.zh.md)

This checklist covers local checks before a release. Each item states an observable expected result.

## Environment

Isolated testing uses a workspace data directory:

~~~bash
export NPC_GUI_DATA_DIR="$PWD/.cache/appdata"
wails dev
~~~

A browser can call the real Go backend at http://localhost:34115. The frontend mock mode starts with pnpm dev.

## Setup

- [ ] Without npc, the application opens the setup page; the download button shows progress and supports cancellation.
- [ ] After download, the setup page shows the version and path and allows the next step.
- [ ] After selecting a local npc, the application runs -version and verifies the version.
- [ ] Invalid files are rejected with a readable error.
- [ ] Automatic detection finds npc on PATH or in the GUI-managed directory.

## Profiles

- [ ] Creating a profile validates required fields.
- [ ] Pasting ./npc -server="..." -vkey="..." -type="tcp" into the new-profile page fills name, server, vkey, and connection type.
- [ ] A multi-server command fills only the first server and suggests the Import Command flow for batch creation.
- [ ] A server address with multiple comma-separated values is rejected.
- [ ] vkey is masked by default; show and hide work.
- [ ] Saving a profile generates profiles/<id>/npc.conf with 0600 permissions.
- [ ] The preview masks vkey by default; revealing the secret shows plaintext.
- [ ] Exporting configuration opens the system save dialog.
- [ ] Editing a running profile shows the pending-restart state.
- [ ] Pasting an NPS command splits it into multiple profiles without executing the command.
- [ ] Duplicate sections and [common] override errors are rejected.

## Processes and state

- [ ] Starting an unreachable server moves state from starting to reconnecting and streams logs.
- [ ] After stop, the process disappears within 5 seconds and the runtime PID file is removed.
- [ ] Two profiles run in parallel with independent state, logs, and PIDs.
- [ ] After stop, the card shows PID -- and does not retain the old PID.
- [ ] Without the tray, closing the window with running connections asks for confirmation; confirming stops all connections and exits, while cancelling keeps the application running.
- [ ] A normal exit leaves no npc processes.
- [ ] Enabling launch at login creates ~/Library/LaunchAgents/com.npcgui.desktop.plist on macOS, an HKCU Run value on Windows, or ~/.config/autostart/npc-gui.desktop on Linux.
- [ ] Starting with --hidden keeps the window hidden; the tray icon is available and can show the window.
- [ ] The tray menu provides start-all, stop-all, launch-at-login toggle, and quit.
- [ ] With close-to-tray enabled, closing the window only hides it and npc keeps running; quitting from the tray stops all processes.
- [ ] Connection errors, recovery, and new client versions produce desktop notifications; disabling notifications stops them.
- [ ] After a forced application exit, the next launch ends the previous npc process during startup and shows no stale-process warning.
- [ ] When a stale process command line cannot be verified, the application does not kill it and only displays a warning.

## Logs

- [ ] All-logs and per-profile logs switch correctly.
- [ ] Level and keyword filters produce correct results.
- [ ] Follow mode and scrolling behave correctly.
- [ ] Clearing affects only the active profile.
- [ ] Exported files match the visible log content.
- [ ] Log files exist with 0600 permissions.

## npc versions

- [ ] Automatic download of the latest npc matches the SHA-256 from the GitHub asset digest.
- [ ] After download, npc -version prints the expected version.
- [ ] Selecting a candidate version switches the active npc.
- [ ] Upstream download failures show a readable error and a retry path.

## Client updates

- [ ] Development builds show disabled and do not access GitHub.
- [ ] Release builds, or a test source configured through NPC_GUI_UPDATE_REPO and NPC_GUI_UPDATE_API, can find a new version.
- [ ] Download progress, cancellation, and checksum errors are reported correctly.
- [ ] Restart-and-update stops connections first; the helper replaces the application and restarts it.
- [ ] The old .old backup exists and restores on replacement failure.
- [ ] macOS AppTranslocation or read-only directories produce a clear message.
- [ ] The update log can be opened from settings.

## Build

Backend tests:

~~~bash
go test ./...
go vet ./...
~~~

Frontend checks and build:

~~~bash
cd frontend
pnpm typecheck
pnpm test
pnpm build
cd ..
~~~

Production packaging:

~~~bash
wails build -clean
~~~

Launch the application:

~~~bash
open build/bin/npc-gui.app
~~~
