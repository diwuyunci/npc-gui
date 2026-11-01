# Business overview

English | [中文](business.zh.md)

## Product overview

NPC GUI is a desktop client for NPS. It manages [djylb/nps](https://github.com/djylb/nps) npc reverse-tunnel processes and turns configuration, process control, log reading, and version maintenance into one desktop workflow.

The product targets developers and operators who keep reverse-tunnel connections running for long periods. One application manages multiple NPS server profiles. Each profile owns its configuration, process, state, and logs.

## Target users

- Developers who switch between multiple NPS servers.
- Operators who maintain several reverse-tunnel connections at once.
- Users who need connection state and logs without opening a terminal.
- Users who want one place to manage npc and client versions.

## Core scenarios

### Multi-server connection management

Users create one profile per NPS server and configure the address, vkey, connection type, proxy, authentication, and reconnect settings. The application starts one npc process per profile, so multiple connections run in parallel.

### Launch command import

The NPS admin console generates an npc launch command. Users paste the command into the import entry, and the application parses the server address, vkey, and connection type into profile fields. The command itself is never executed.

### Connection monitoring and troubleshooting

The application shows connection state, PID, uptime, reconnect count, and the last error. The log console filters by profile, level, and keyword, and supports export for diagnosing connection, authentication, or network failures.

### Version maintenance

The application detects the local npc binary and can download npc from upstream GitHub Releases, verify its SHA-256 digest, and check its version. The client updates itself from this repository's GitHub Releases. npc and client updates are independent.

### Long-running operation

The application provides a system tray, launch at login, and desktop notifications. With the tray enabled, closing the window hides it while connections keep running. The tray restores the window and starts or stops all connections.

## Core capabilities

- **Connection profiles** cover server address, vkey, TCP/TLS/KCP/QUIC/WS/WSS, proxy, DNS/NTP, reconnect settings, authentication, and rate limits.
- **Command fill** populates basic fields after a user pastes an npc launch command and does not execute the command.
- **Multi-instance operation** runs one npc process per profile; profiles do not affect each other.
- **Managed configuration** generates a separate npc.conf per profile and keeps vkey out of process arguments.
- **Live state** reports online, reconnecting, error, and stopped states together with runtime data and the last error.
- **Log console** supports filtering, search, follow mode, line wrapping, clearing, and export; log files use 0600 permissions.
- **NPC core management** supports detection, local selection, upstream download, SHA-256 verification, and version checks.
- **Client updates** use GitHub Releases for checks, downloads, verification, helper replacement, and rollback.
- **System integration** provides tray actions, launch at login, and notifications for connection errors, recovery, and new versions.

## Workflows

### First use

1. The application opens the setup page and offers automatic npc download, local file selection, or PATH detection.
2. After npc is ready, the user creates the first connection profile.
3. The user enters the server address, vkey, and connection type, or pastes an NPS launch command.
4. The application saves the profile, generates its configuration, and starts the connection.
5. The dashboard shows runtime state and the log page shows npc output.

### Daily operation

1. The user starts, stops, or restarts connections from the dashboard or profile page.
2. The application updates connection state and pushes state and logs to the UI.
3. The user filters or exports logs per profile.
4. The user hides the window to the tray while connections keep running.
5. The user restores the window from the tray or starts and stops all connections there.

### Version maintenance

1. The application checks for npc core and client updates.
2. After user confirmation, the application downloads and verifies update assets.
3. npc updates take effect on the next process start; the helper replaces and restarts the client.
4. Failed updates keep the old version or restore a backup.

## Scope

- One profile maps to one server; multiple servers require multiple profiles.
- The current scope excludes NPS server Web APIs, tunnel CRUD, and traffic statistics.
- The current scope excludes npc system-service installation.
- The current scope excludes code signing, Apple notarization, DMG, and NSIS installers.
- npc binaries are not bundled; the application downloads them from upstream or uses a local file.
- Client and npc updates use separate release sources and do not affect each other.

## Value

- The application turns command-line operations into a visual workflow and lowers the daily cost of using npc.
- Separate configuration files and processes let multiple servers run in parallel.
- Centralized management of vkey, configuration, logs, and versions removes repeated work.
- State, logs, and notifications shorten the time needed to diagnose connection failures.
- The GitHub Releases update path reduces maintenance for both the client and npc.

The technical architecture, system integration, and update mechanisms are documented in [technical.md](technical.md).
