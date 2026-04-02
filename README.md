<p align="center">
  <img src="./crossboard_icon.ico" width="96" height="96" alt="CrossBoard Logo" />
</p>

<svg width="100%" height="110" viewBox="0 0 1000 110" xmlns="http://www.w3.org/2000/svg" style="margin-bottom: 16px;">
  <defs>
    <linearGradient id="a" x1="0" x2="1" y1="0" y2="0">
      <stop offset="0%" stop-color="#00ffe0"/>
      <stop offset="100%" stop-color="#ff6b35"/>
    </linearGradient>
  </defs>
  <rect x="0" y="0" width="1000" height="110" rx="20" ry="20" fill="rgba(17,17,24,0.95)" />
  <path fill="none" stroke="url(#a)" stroke-width="4" d="M 25 80 Q 80 20 150 80 T 275 80 T 400 80 T 525 80 T 650 80 T 775 80 T 900 80"/>
  <text x="50%" y="60" dominant-baseline="middle" text-anchor="middle" font-family="IBM Plex Mono, monospace" font-size="30" fill="#fff">CrossBoard - Stable, Fast, Minimal</text>
</svg>

# CrossBoard 

> CrossBoard is the only tool you need when you live and breathe code and need to ship clipboard syncing, remote input, and file transfer into your server from one tiny launcher.

![Status](https://img.shields.io/badge/status-POC%20%2F%20Ready-%2300ffea)
![License](https://img.shields.io/badge/license-MIT-%23f0db4f)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux-%23008cff)
![Download](https://img.shields.io/badge/download-crosboard.exe-%23ff6b35)

---

## 💥 TL;DR (because why not)

- `crossboard.exe`: remote clipboard + file browser + remote keyboard/mouse in one process
- no phone app required, no same-network restrictions (phone on cellular + PC on wifi works)
- Cloudflared is installed once; CrossBoard handles tunnel setup automatically
- built in Go for stability and lightweight performance; 1 binary, no reporting/tracking garbage
- made because every other clipboard share tool (including KDE/Win integration) felt broken, slow, clumsy, and intrusive

---

## 🆚 Comparison: KDE, Microsoft Phone Link, CrossBoard

| Feature | KDE Connect / GSConnect | Microsoft Phone Link | CrossBoard (this project) |
|---|---|---|---|
| Clipboard sync | Yes (local network) | Yes (requires Windows + mobile app) | Yes (web UI + tunnel) |
| File transfer | Yes (local network, manual browse) | Yes (limited) | Yes (upload/download in app dir) |
| Remote input | Yes (mouse/keyboard) | No | Yes (remote keyboard/mouse over websocket) |
| Cross-network | No (same LAN) | Yes (if Windows device visible) | Yes (cloudflared tunnel, phone on cellular + PC on Wi-Fi) |
| Setup complexity | Medium | Medium | Low (single binary, no install on phone) |
| Privacy/control | Good, open source | Mixed, proprietary | Good, open source, local root dir locked |

---

## 🔧 Prereqs (because I never have time to set this up again)

1. Install `cloudflared` first (required):
   - Windows: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation
   - Linux: `sudo apt install cloudflared` or: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation
2. Install Go (1.20+)
3. Clone project:
```bash
cd C:\stuffs\projects
git clone https://github.com/youruser/crossboard.git
cd crossboard
```

---

## 🚀 Build (Windows + Linux)

### Linux
```bash
go build -o crossboard
./crossboard
```

### Windows
```powershell
go build -o crossboard.exe
./crossboard.exe
```

---

## Start Cloudflare Tunnel (mandatory for remote)

`cloudflared` only needs to be installed; the local app handles the tunnel setup automatically (no manual login required).

```bash
# install cloudflared
# on Linux: sudo apt install cloudflared (or download from Cloudflare)
# on Windows: download installer from Cloudflare

# run crossboard (it will auto-connect to cloudflared tunnel)
./crossboard       # Linux
./crossboard.exe   # Windows
```

Then access via generated tunnel URL and scan QR.

---

## Features (because life is too short for weak clipboard tools)

- Remote clipboard get/send
- Remote keyboard/mouse via websocket
- File browser + folder navigation + back button
- Cross Device File Sync [In the folder where exe is located]
- Upload file to server (in-app directory only)
- Download from executable dir only, safe path handling, no folder escape (`../`) abuse
- Dark neon theme + minimal UI
- QR generation for phone access (no app install needed)
- Multiple devices supported; Go concurrency handles load neatly
- Works when phone is on cellular and PC on Wi-Fi (truly cross-network)

---

## 🛡️ Security design notes

- `appHome` is locked to executable directory
- `sanitizePath()` restricts listing + upload to `appHome`
- binary serves only valid folder children
- no folder path injection

---

## 📦 File upload/download behavior

### Browse files
- click `BROWSE FILES`, hits `/listfiles?folder=.`
- supports subfolder navigation via same endpoint

### Download
- `/getfile/<path>` (from app folder root)
- linked by UI item

### Upload
- `/uploadfile` plus `folder=.` (current folder from UI state)
- 500MB max in `r.ParseMultipartForm(500 << 20)`

---

## 🚦 One-minute “no-life dev” quick start

1. launch crossboard
2. scan QR from phone
3. everything works, optionally add more devices (the server handles it)

---

## 🧪 Quick endpoint check

```bash
curl http://localhost:8888/health
curl http://localhost:8888/listfiles?folder=.
curl http://localhost:8888/getfile/file.go --output ./downloaded_file.go
curl -F "file=@./README.md" -F "folder=." http://localhost:8888/uploadfile
```

---

## 📥 Prebuilt binary

Download this directly (raw GitHub content):

**https://raw.githubusercontent.com/youruser/crossboard/main/crossboard.exe**

---

## 🧾 License

MIT
