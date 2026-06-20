# Installing go-scheduler on Windows

go-scheduler has three programs:

| Program | What it does | Which archive |
|---------|--------------|---------------|
| `goschedd.exe` | The **daemon/service** — runs your scheduled tasks in the background | `go-scheduler_<ver>_windows_amd64.zip` |
| `gosched.exe` | The **CLI** — create and manage tasks, groups, triggers | `go-scheduler_<ver>_windows_amd64.zip` |
| `gosched-gui.exe` | The **desktop GUI** — a window for managing everything visually | `go-scheduler-gui_<ver>_windows_amd64.zip` |

The CLI and GUI are **clients**: they talk to the daemon. Nothing runs until the daemon is
installed and started, so you need **both** archives.

## 1. Download

From the [latest release](https://github.com/shruggietech/go-scheduler/releases/latest), download:

- `go-scheduler_<ver>_windows_amd64.zip` (daemon + CLI)
- `go-scheduler-gui_<ver>_windows_amd64.zip` (GUI)

(Use the `arm64` archives instead if you're on a Windows-on-ARM device. The GUI is currently
published for `amd64` only.)

## 2. Verify the downloads (recommended)

The release includes `SHA256SUMS.txt`. In PowerShell:

```powershell
Get-FileHash .\go-scheduler_*_windows_amd64.zip -Algorithm SHA256
```

Compare the printed hash against the matching line in `SHA256SUMS.txt`.

## 3. Extract

Extract both archives. The simplest layout is to put all three `.exe` files in one folder,
for example `C:\Tools\go-scheduler\`:

```
C:\Tools\go-scheduler\
  goschedd.exe
  gosched.exe
  gosched-gui.exe
```

> Keeping `gosched-gui.exe` next to `gosched.exe` lets `gosched gui` find and launch it, and
> lets the service installer find `goschedd.exe`.

## 4. Install and start the service (requires Administrator)

Open **PowerShell as Administrator**, `cd` into the folder, then:

```powershell
.\gosched.exe service install     # registers the Windows service (admin required)
.\gosched.exe service start
.\gosched.exe health              # expect: daemon ok (version 0.2.0)
```

The service is registered to start on boot, so your tasks keep running across reboots without
anyone being logged in.

## 5. Use it

**GUI** — just run it (no console window appears):

```powershell
.\gosched-gui.exe
```

…or launch it from the CLI:

```powershell
.\gosched.exe gui
```

**CLI** — for example:

```powershell
# A recurring task (note: quote the schedule)
.\gosched.exe task add backup --command "C:\Windows\System32\cmd.exe" --arg "/c" --arg "echo backup" --schedule "every weekday at 09:00"

# A one-off reminder
.\gosched.exe task add bday --command "C:\Windows\System32\cmd.exe" --arg "/c" --arg "echo happy birthday" --at "2026-08-04T09:00:00Z"

.\gosched.exe task list
.\gosched.exe runs               # run history
.\gosched.exe alerts --unacked   # alerts (overlaps, failures, missed runs)
```

## Managing / removing the service

```powershell
.\gosched.exe service status
.\gosched.exe service stop
.\gosched.exe service uninstall   # admin required
```

## Troubleshooting

- **`service install` fails with an access/privilege error** → run PowerShell **as
  Administrator**.
- **`gosched health` says the daemon is unreachable** → make sure the service is running
  (`.\gosched.exe service status`); start it with `.\gosched.exe service start`.
- **The GUI opens but shows nothing / errors** → the daemon isn't running, or isn't reachable.
  Start it (step 4) and reopen the GUI. The GUI reconnects to the live event stream
  automatically.
- **`gosched gui` says the GUI binary wasn't found** → put `gosched-gui.exe` in the same folder
  as `gosched.exe` (or run `gosched-gui.exe` directly).
- **SmartScreen / antivirus warning** → these binaries are unsigned. Verify the SHA-256 hash
  (step 2) and allow it if it matches.
