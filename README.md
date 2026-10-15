<p align="center">
  <img src="./crossboard_icon.ico" width="88" alt="CrossBoard">
</p>

<h1 align="center">CrossBoard</h1>

<p align="center">
  Share your clipboard, files, and desktop controls between your computer and mobile devices.
</p>

<p align="center">
  <a href="https://github.com/Akash23q2/CrossBoard/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-MIT-00bfa6" alt="MIT License"></a>
  <img src="https://img.shields.io/badge/built%20with-Go-00add8" alt="Built with Go">
</p>

CrossBoard runs a small server on your computer and provides a browser-based client for connected devices. Use it over your local network, through USB tethering, or from another network with an optional Cloudflare Tunnel. A companion Android app can integrate directly with the server for clipboard and device features.

## Features

- **Clipboard sync:** the Android app can sync the device clipboard seamlessly without requiring shortcut presses. Desktop users can also use `Alt+C` and `Alt+V`.
- **Touchpad and remote input:** use the browser client for mouse and keyboard control; the Android app adds configurable multi-touch gestures.
- **File transfer:** browse folders, upload files, and download files through the web client. Set the shared root folder in local settings; by default it is the folder containing the CrossBoard executable.
- **Android app features:** launch Android apps directly, mirror the device screen, use configurable multi-touch gestures, and sync the clipboard seamlessly.
- **Screen mirroring:** start scrcpy for the connected Android device with `Alt+K` when ADB over TCP is available.
- **LAN and USB tethering:** CrossBoard prints a local URL and QR code. It prefers common USB-tethering IPv4 ranges when choosing a local address.
- **Cross-network access:** optionally install `cloudflared` to get a temporary Cloudflare Tunnel URL. Connect multiple devices from different network seamlessly and share/access clipboard or files. Local network access continues if Cloudflare Tunnel is unavailable.
- **Terminal dashboard:** view QR codes and recent logs, open help/settings, and use shortcuts while CrossBoard runs in an interactive terminal.
- **Configurable server and device settings:** server/device ports, request timeouts, ADB timeout, and file-browser root are saved in the user configuration directory.

## Quick start

### Run a release binary

Download a binary from the [Releases](https://github.com/Akash23q2/CrossBoard/releases) page, then run it:

```sh
# Linux
./crossboard
```

```powershell
# Windows
.\crossboard.exe
```

When the terminal dashboard appears, scan the LAN QR code with a device on the same network. If the device is connected over USB tethering, use the tethering URL printed by CrossBoard.

### Build from source

Install Go, clone the repository, and build:

```sh
git clone https://github.com/Akash23q2/CrossBoard.git
cd CrossBoard
go build -o crossboard .
./crossboard
```

On Windows:

```powershell
go build -o crossboard.exe .
.\crossboard.exe
```

## Connect across networks

For connections outside your LAN, install [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/) on the computer running CrossBoard, then restart CrossBoard. It starts a temporary tunnel automatically when available; the tunnel URL is shown in the terminal dashboard. Press `t` to switch the displayed QR between the LAN and tunnel URLs.

Without `cloudflared`, CrossBoard continues to serve over the LAN. A Cloudflare Tunnel provides access to the CrossBoard web server; it does **not** make a phone's private IP reachable from the computer for direct app-to-device traffic such as ADB or device clipboard endpoints.

## Android app

The companion Android app provides features beyond the browser client:

- Launch installed Android apps directly from CrossBoard.
- Mirror the Android screen using scrcpy.
- Use configurable multi-touch gestures for remote control.
- Keep the phone and computer clipboards in sync automatically, without pressing a shortcut for each transfer.

Install `adb` (Android platform-tools) and [scrcpy](https://github.com/Genymobile/scrcpy/blob/master/doc/README.md#installation) on the CrossBoard computer and enable ADB over TCP on the Android device for screen mirroring. Device availability and network reachability are required for app control and clipboard sync.

## Desktop shortcuts

These shortcuts are handled by the computer running CrossBoard:

| Shortcut | Action |
| --- | --- |
| `Alt+C` | Send Ctrl+Shift+C to copy selected text in the focused PC app, then send it to the registered app device. |
| `Alt+V` | Fetch device clipboard text, write it to the PC clipboard, then paste into the focused PC app. |
| `Alt+K` | Connect to the registered app device over ADB TCP port `2004` and start scrcpy. |

Keep the relevant PC application focused when using clipboard shortcuts. `Alt+C` sends Ctrl+Shift+C, avoiding Ctrl+C's interrupt behavior in terminals; the focused application must support Ctrl+Shift+C to copy selected text. The device must return clipboard data before the configured request timeout for `Alt+V` to trigger the paste.

For scrcpy, install both `adb` (Android platform-tools) and [scrcpy](https://github.com/Genymobile/scrcpy/blob/master/doc/README.md#installation) on the CrossBoard computer. The app device must have ADB-over-TCP enabled and reachable on port `2004`.

## Browser controls and gestures

Open the CrossBoard URL from a mobile device to use the browser client. In remote-control mode:

| Gesture | Action |
| --- | --- |
| One-finger move | Move the PC pointer |
| One-finger tap | Left-click |
| Two-finger tap | Right-click |
| Two-finger swipe | Scroll |
| Three- or four-finger swipe | Trigger a configurable keyboard shortcut |

Gesture shortcuts can be edited in the remote-control panel. Enter a key or combination such as `alt+tab` or `win+n`, or use `off` to disable a gesture. Gesture mappings are saved in that browser's local storage.

## File transfer

Use the web client's file browser to navigate folders, download files, and upload files. The file-browser root is configurable in Settings, opened with `s` from the terminal dashboard. By default, the root is the directory containing the CrossBoard executable. Only choose a directory you intend to share: its contents are made available to connected clients.

The server confines file operations to the configured root and rejects paths that escape it, including symlinks leading outside the root.

## Terminal dashboard

In an interactive terminal, CrossBoard opens a dashboard with the LAN QR code visible by default. Press:

| Key | Action |
| --- | --- |
| `h` | Open the web help page |
| `?` | Show global desktop shortcuts |
| `t` | Toggle between LAN and Cloudflare QR codes |
| `r` | Redraw the dashboard |
| `s` | Open local settings |
| `l` | Open recent logs |
| `k` | Start scrcpy for the registered app device |
| `q` | Exit CrossBoard |

Dashboard keys work while the terminal is focused. Non-interactive launches print regular logs instead.

## Settings and configuration

Settings are available from the terminal dashboard with `s` and are restricted to requests from the CrossBoard computer. They include:

- CrossBoard server port (default `5050`; restart required after changing)
- File-browser root folder
- App-device HTTP port (default `80`)
- App-device request timeout
- Cloudflare Tunnel startup timeout
- ADB connection timeout

Configuration is saved as `config.json` under the operating system's user configuration directory in `CrossBoard` (typically `~/.config/CrossBoard/config.json` on Linux).

## Network and permissions

Devices on the same LAN can connect using the printed local URL. For USB tethering, connect the phone to the computer with USB tethering enabled and use the URL/QR printed for that network interface. CrossBoard prioritizes common USB tethering address ranges when selecting its local IPv4 address.

If a firewall blocks access, allow inbound TCP to the configured CrossBoard server port from your trusted private network only. Do not expose the server port publicly.

On Linux, global keyboard shortcuts require read access to keyboard event devices under `/dev/input`. One common setup is adding the CrossBoard user to the `input` group and logging out and back in. This grants system-wide keyboard-event access, so only do this for a trusted user. On macOS, grant the terminal or CrossBoard Accessibility permission if the native keyboard hook is blocked.

## License

CrossBoard is licensed under the [MIT License](LICENSE).
