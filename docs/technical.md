# Technical overview

English | [中文](technical.zh.md)

## Technology stack

| Layer | Technology |
|---|---|
| Desktop framework | Wails v2.12.0 |
| Backend | Go 1.25.x, independent Go module |
| Frontend | Vue 3, TypeScript, Vite, Pinia, Vue Router |
| Icons | lucide-vue-next |
| Styling | Native CSS and design tokens |
| Package manager | pnpm 10.x |
| Testing | Go testing, Vitest, Vue Test Utils |
| Release | GitHub Actions and GitHub Releases |

## Upstream constraints and architecture decisions

npc comes from [djylb/nps](https://github.com/djylb/nps). The application uses npc 0.34.10 with core 0.34.0.

| Upstream fact | Technical decision |
|---|---|
| npc supports command-line and -config file modes | The application always starts npc with a configuration file |
| The [common] section has one server_addr | One profile maps to one server |
| Command-line mode accepts comma-separated servers but puts vkey in argv | Imports split servers into profiles; startup uses configuration files |
| StartFromFile calls os.Exit(1) on configuration errors, and logging/state are globals | The application validates before start and isolates npc in a subprocess |
| npc logs through zerolog ConsoleWriter | The application captures stdout and uses -log=stdout and -log_color=false |
| The C SDK covers only basic connection, state, and log snapshots | The project does not embed the SDK |
| Upstream go.mod requires Go 1.27; the project uses Go 1.25.x | The GUI does not import nps modules and builds independently |
| Upstream assets use the os_arch_client.tar.gz pattern | The application maps assets per platform and verifies the GitHub asset digest |
| Upstream uses GPLv3 | The application does not redistribute npc and downloads it from upstream |

The architecture uses a **sidecar subprocess**. The GUI owns configuration, processes, logs, and updates; npc continues to run as the official binary with its full configuration surface.

## Overall architecture

~~~text
Vue 3 UI
  ↕ Wails bindings / events
Go App facade
  ├─ Store
  ├─ ConfigGen
  ├─ BinaryManager
  ├─ ProcessManager
  ├─ LogHub
  ├─ UpdateManager
  └─ System
~~~

| Component | Responsibility |
|---|---|
| Store | Profiles, settings, atomic JSON persistence |
| ConfigGen | Form validation, npc.conf generation, CLI arguments, NPS command parsing |
| BinaryManager | npc detection, upstream download, SHA-256, version checks |
| ProcessManager | Process lifecycle, PID registry, stop, reconnect |
| LogHub | stdout/stderr parsing, ring buffer, batched events |
| UpdateManager | Client release checks, helper replacement, rollback |
| System | Tray, launch at login, notifications |

The backend exposes profile, process, log, settings, and update methods through Wails bindings. It publishes runtime state, logs, update progress, and notification events through Wails events. The frontend uses Pinia for profiles, runtime state, logs, and updates.

## Startup flow

1. When a user saves a profile, App.SaveProfile validates it and writes profiles.json.
2. The application generates profiles/<id>/npc.conf with 0600 permissions and computes its configuration hash.
3. App.StartProfile resolves the npc path, and ProcessManager starts the subprocess with `npc -config=... -log=stdout -debug=false -log_color=false -log_level=... -disable_remote_update=true`.
4. LogHub receives stdout and stderr, strips ANSI escapes, parses timestamps and levels, updates the state machine, and writes the ring buffer and log file.
5. State changes are pushed through runtime:state; logs are pushed in batches through runtime:logs.
6. When a user stops a connection, the Unix build sends SIGTERM to the process group and sends SIGKILL after a timeout. Windows uses Process.Kill or taskkill as a fallback.

## State machine and logs

~~~text
stopped -> starting -> running -> online
                 |         |
                 |         +-> reconnecting -> online
                 |         |
                 +--------> error
online/reconnecting -> stopping -> stopped
~~~

- running means the process is alive but has not produced a successful bridge log; this covers P2P and secret modes.
- online is triggered by Successful connection with server.
- reconnecting is triggered by The connection server failed, Failed to connect, Reconnecting..., and Client closed.
- Configuration and authentication errors are recognized from loading error, missing [common], web_user may have been occupied, and similar logs.
- When a process exits without a user stop, the application keeps its exit code and error summary.

The log path uses a ring buffer and batched delivery. The default buffer keeps 5000 lines. The log console filters by profile, level, and keyword and supports follow mode, line wrapping, clearing, and export. Log files use 0600 permissions and rotate by size.

## Configuration, data, and security

### Data directories

| Platform | Directory |
|---|---|
| macOS | ~/Library/Application Support/npc-gui/ |
| Windows | %AppData%/npc-gui/ |
| Linux | ~/.config/npc-gui/ |

The data directory contains:

- profiles.json, settings.json, and binary.json store application configuration.
- profiles/<id>/npc.conf stores profile configuration with 0600 permissions.
- logs/<id>/session-*.log stores runtime logs with 0600 permissions.
- updates/ stores client update staging and logs.
- runtime/<id>.json stores PID records for stale-process cleanup.

### Configuration generation

- Validated form fields generate npc.conf.
- The application starts npc with -config.
- Advanced configuration supports extra common keys and extra INI sections.

### Security boundaries

- All commands go through exec.Command and never through a shell.
- vkey is written only to 0600 configuration files and profile JSON, never to process arguments.
- Configuration previews mask secrets by default; exporting real configuration or profiles with secrets shows a warning.
- npc downloads verify the GitHub asset digest, and extraction rejects path traversal and symbolic links.
- Client updates use only the repository injected at build time; release builds do not allow users to change the update URL.
- Logs may contain web credentials printed by npc, so export requires a review of the sharing scope.

## System integration

- The tray uses fyne.io/systray RunWithExternalLoop to join the Wails event loop. macOS creates NSStatusItem on the main thread through dispatch_async.
- The tray menu reads online count, running count, profile count, and launch-at-login state. It provides show-window, start-all, stop-all, and quit actions.
- Launch at login uses a LaunchAgent on macOS, the HKCU Run key on Windows, and an autostart desktop entry on Linux.
- The autostart command includes --hidden, so the application starts hidden and hands control to the tray.
- Desktop notifications use beeep for connection errors, recovery, and new client versions.
- Tray changes take effect on the next launch, which avoids destroying NSStatusItem while the Wails event loop is running.

## Client updates

### Version sources

- The version comes from a git tag; the build removes the v prefix and injects buildinfo.Version through ldflags.
- The repository comes from github.repository in CI and is injected into buildinfo.Repo.
- Local development builds use Version 0.1.0-dev and an empty Repo; update state is disabled and the client does not access GitHub.
- Development can point NPC_GUI_UPDATE_REPO and NPC_GUI_UPDATE_API at a test release source; release builds ignore the API override.

### Release assets

| Platform | Asset |
|---|---|
| macOS arm64 | npc-gui_<version>_darwin_arm64.app.zip |
| macOS amd64 | npc-gui_<version>_darwin_amd64.app.zip |
| Windows amd64 | npc-gui_<version>_windows_amd64.exe |
| Linux amd64 | npc-gui_<version>_linux_amd64.tar.gz |
| All | checksums.txt |

macOS updates the complete .app; Windows and Linux update the executable.

### Check, download, and verify

- The client queries GET https://api.github.com/repos/<repo>/releases/latest for the latest version.
- The latest response is cached at updates/latest.json; automatic checks run at most once every 24 hours.
- Version comparison uses semver; the latest endpoint excludes drafts and prereleases.
- Assets are selected by runtime.GOOS and runtime.GOARCH; when no asset matches, the UI opens the release page.
- Downloads are staged under updates/<version>/.
- SHA-256 comes from the GitHub asset digest and checksums.txt; the client rejects automatic updates when both are missing.
- zip and tar extraction reject path traversal, symbolic links, and oversized files.

### Apply and rollback

1. After the user confirms restart-and-update, the application stops all npc processes.
2. The application starts the helper with `helper --apply-update --target <exe/app> --staged <bin> --staged-app <app> --kind app|binary --old-pid <pid> --relaunch <path>`.
3. The helper waits up to 90 seconds for the old PID to exit.
4. On macOS, the helper renames the current .app to .app.old, copies the new .app into place, and restores the old version on failure.
5. On Windows and Linux, the helper renames the target executable to .old, writes the new file, and restores the old version on failure.
6. The helper starts the new version and writes updates/last-result.json; logs go to updates/update.log.

When the macOS application is translocated or in a read-only directory, the client refuses automatic updates and offers the Applications folder or a manual download. When the target directory is not writable, the client offers a manual download and does not request elevation.

## Build and release

### Development builds

Local development builds use a dev version and do not check for client updates. Build with:

~~~bash
./scripts/build-local.sh
~~~

Build outputs include the macOS application at build/bin/npc-gui.app and dist/npc-gui_<version>_darwin_arm64.app.zip.

Release builds receive the repository and tag version from CI:

~~~bash
VERSION=0.1.1 REPO=owner/npc-gui PLATFORM=darwin/arm64 ./scripts/build-local.sh
~~~

### Release flow

- CI reads github.repository and injects buildinfo.Repo.
- Pushing a v* tag triggers the Release workflow.
- The Release workflow builds on native macOS arm64/amd64, Windows amd64, and Linux amd64 runners.
- Release assets include .app.zip, .exe, .tar.gz, and checksums.txt.

## Project structure

~~~text
npc-gui/
├── app.go
├── main.go
├── internal/
│   ├── model/
│   ├── store/
│   ├── configgen/
│   ├── npcbin/
│   ├── npcrun/
│   ├── update/
│   ├── tray/
│   ├── autostart/
│   ├── notify/
│   ├── buildinfo/
│   └── platform/
├── frontend/
├── build/
├── scripts/
├── docs/
└── .github/workflows/
~~~

## Uninstall and cleanup

- The application has no standalone uninstaller. Disable launch at login before uninstalling to remove the LaunchAgent, Run key, or autostart file.
- Application data lives in the user data directory. Delete that directory to remove configuration, secrets, logs, and update caches.
- Tray, notifications, and managed npc processes belong to the application process. They disappear when the application exits, and stale npc processes are removed on the next launch.
- If the application is deleted without disabling launch at login, the operating system login item remains and must be removed manually.

## Testing and acceptance

Go unit tests cover store, configgen, ghrelease, npcbin, npcrun, update, autostart, and helper. Frontend checks include vue-tsc, Vitest, and the Vite production build. Release acceptance is owned by [manual-test.md](manual-test.md).

## Related documents

- [business.md](business.md) records product positioning, users, scenarios, and scope.
- [manual-test.md](manual-test.md) records the release acceptance checklist.
- [README.md](../README.md) records the project entry, features, and quick start.
