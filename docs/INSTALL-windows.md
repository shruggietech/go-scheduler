# Installing go-scheduler on Windows

## Quick start (desktop) — one download

1. From the [latest release](https://github.com/shruggietech/go-scheduler/releases/latest),
   download **`go-scheduler-desktop_<ver>_windows_amd64.zip`**. It is self-contained — it
   includes the GUI **and** the daemon and CLI:

   ```
   gosched-gui.exe   # the desktop app
   goschedd.exe      # the background daemon (started automatically)
   gosched.exe       # the command-line tool (optional)
   ```

2. (Recommended) Verify the download against `SHA256SUMS.txt`:

   ```powershell
   Get-FileHash .\go-scheduler-desktop_*_windows_amd64.zip -Algorithm SHA256
   ```

3. Extract the zip (keep all three `.exe` files together in one folder).

4. **Run `gosched-gui.exe`.** That's it — the GUI starts the background daemon automatically the
   first time it can't find one running, so there's no setup. No console window appears.

The background daemon keeps running after you close the window, so your scheduled tasks keep
firing. To stop it, use Task Manager (end `goschedd.exe`) or `.\gosched.exe service stop` if you
installed it as a service (below).

## Start on boot (optional)

The auto-started daemon runs until the machine is restarted. If you want your tasks to keep
running **across reboots without anyone logging in**, install it as a Windows service. Open
**PowerShell as Administrator**, `cd` into the folder, then:

```powershell
.\gosched.exe service install     # registers the Windows service (admin required)
.\gosched.exe service start
.\gosched.exe service status      # expect: running
```

Once the service is installed, the GUI detects it and reuses it (it won't start a second
daemon — a single-instance lock prevents that).

## Using the CLI (optional)

```powershell
.\gosched.exe health              # daemon ok (version ...)

# A recurring task (quote the schedule)
.\gosched.exe task add backup --command "C:\Windows\System32\cmd.exe" --arg "/c" --arg "echo backup" --schedule "every weekday at 09:00"

# A one-off reminder
.\gosched.exe task add bday --command "C:\Windows\System32\cmd.exe" --arg "/c" --arg "echo happy birthday" --at "2026-08-04T09:00:00Z"

.\gosched.exe task list
.\gosched.exe runs                # run history
.\gosched.exe alerts --unacked    # overlaps, failures, missed runs
```

## Removing it

```powershell
.\gosched.exe service stop
.\gosched.exe service uninstall   # admin required (only if you installed the service)
```

Then delete the folder.

## Troubleshooting

- **The GUI opens but shows nothing / "daemon unreachable"** → the daemon didn't start. Make
  sure `goschedd.exe` is in the same folder as `gosched-gui.exe`. You can start it manually with
  `.\gosched.exe service start` (if installed) or just rerun `gosched-gui.exe`.
- **`service install` fails with an access/privilege error** → run PowerShell **as
  Administrator**.
- **SmartScreen / antivirus warning** → these binaries are unsigned. Verify the SHA-256 hash and
  allow it if it matches.
- **Tasks aren't running after a reboot** → the auto-started daemon does not survive reboots;
  install the service (see *Start on boot*) for that.
