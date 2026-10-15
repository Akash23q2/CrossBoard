package main

import (
	"net/http"
	"strconv"
	"strings"
)

func helpPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/help" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := strings.ReplaceAll(helpHTML, "{{SERVER_PORT}}", strconv.Itoa(currentAppConfig().ServerPort))
	_, _ = w.Write([]byte(page))
}

const helpHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>CrossBoard help</title>
<style>
*{box-sizing:border-box}body{margin:0;padding:24px;background:#111118;color:#e8e8f0;font:15px system-ui,sans-serif}
main{max-width:820px;margin:auto}h1,h2{color:#00ffe0}h1{margin:0 0 8px}h2{font-size:1.1rem;margin:0 0 12px}
p,.note{color:#aaaabc;line-height:1.65}a{color:#00ffe0}a:hover{color:#fff}
.top{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:24px}
.top button{border:1px solid #e8e8f0;border-radius:9px;padding:12px 17px;background:transparent;color:#00ffe0;font:inherit;cursor:pointer;white-space:nowrap}
.top button:hover{border-color:#00ffe0;background:#1b1b27}
.actions{display:flex;gap:10px;flex-wrap:wrap}
.card{background:#1b1b27;border:1px solid #292936;border-radius:8px;padding:20px;margin:16px 0}
code,pre{font:13px ui-monospace,monospace}code{color:#ffbd8a}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#111118;border-radius:6px;padding:14px;color:#e8e8f0;line-height:1.6}
ul{padding-left:22px;color:#aaaabc;line-height:1.75}.note{font-size:.9rem}
@media(max-width:520px){body{padding:16px}.top{align-items:flex-start;flex-direction:column}.card{padding:16px}}
</style>
</head>
<body><main>
<div class="top"><div><h1>CrossBoard help</h1><p class="note">Setup tips and troubleshooting for this computer.</p></div><div class="actions"><button type="button" onclick="window.close()">Close</button></div></div>
<section class="card">
<h2>Cloudflare Tunnel</h2>
<p>Cloudflare Tunnel is optional. Install <code>cloudflared</code> on the computer running CrossBoard and restart CrossBoard. If it starts successfully, the tunnel URL is shown in the terminal dashboard; press <code>t</code> there to switch to its QR code. Without it, connect using the LAN address and QR code. Press <code>h</code> in the terminal dashboard to open this help page.</p>
<p><a href="https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation" target="_blank" rel="noopener noreferrer">Cloudflare cloudflared installation guide ↗</a></p>
</section>
<section class="card">
<h2>scrcpy / connect to an app device</h2>
<p>Install both <code>adb</code> (Android platform-tools) and <code>scrcpy</code> on the CrossBoard computer. Register/connect the app device, enable and make ADB-over-TCP reachable on port <code>2004</code>, then press <code>Alt+K</code> on the computer. The app device must be reachable directly from the computer; a CrossBoard Cloudflare web tunnel does not tunnel ADB traffic.</p>
<p><a href="https://github.com/Genymobile/scrcpy/blob/master/doc/README.md#installation" target="_blank" rel="noopener noreferrer">scrcpy installation guide ↗</a> · <a href="https://developer.android.com/tools/adb" target="_blank" rel="noopener noreferrer">Android Debug Bridge guide ↗</a></p>
</section>
<section class="card">
<h2>Firewall: allow LAN access</h2>
<p>Allow inbound TCP traffic to CrossBoard's configured server port (<code>{{SERVER_PORT}}</code>) from your trusted private network only. Restrict the rule to a private/trusted network profile; do not expose the port to the public internet.</p>
<h3>Linux · UFW</h3>
<p>These rules cover common private IPv4 ranges. If possible, narrow them to your specific LAN subnet.</p>
<pre>sudo ufw allow from 10.0.0.0/8 to any port {{SERVER_PORT}} proto tcp
sudo ufw allow from 172.16.0.0/12 to any port {{SERVER_PORT}} proto tcp
sudo ufw allow from 192.168.0.0/16 to any port {{SERVER_PORT}} proto tcp</pre>
<h3>Windows · PowerShell as Administrator</h3>
<p>Allow the configured CrossBoard port on Private networks only.</p>
<pre>New-NetFirewallRule -DisplayName CrossBoard -Direction Inbound -Action Allow -Protocol TCP -LocalPort {{SERVER_PORT}} -RemoteAddress LocalSubnet -Profile Private</pre>
<h3>macOS</h3>
<p>In System Settings → Network → Firewall → Options, allow incoming connections for CrossBoard (or the terminal running it). Keep the firewall enabled and only use CrossBoard on trusted networks.</p>
</section>
<section class="card">
<h2>Global keyboard shortcuts</h2>
<p>Global shortcuts are listened for by the CrossBoard computer and require the terminal/application to remain running. The browser dashboard's own keys only work while its terminal is focused.</p>
<ul><li><code>Alt+C</code>: send Ctrl+Shift+C to copy the selected text on the PC, then send that text to the registered app device. This avoids Ctrl+C's interrupt behavior in terminals; the focused application must support Ctrl+Shift+C for copying.</li><li><code>Alt+V</code>: request the app clipboard; if the device responds before the configured request timeout, CrossBoard writes it to the PC clipboard and sends Ctrl+V to paste into the focused app.</li><li><code>Alt+K</code>: start scrcpy for the registered app device.</li></ul>
<ul>
<li><strong>Linux:</strong> the CrossBoard user needs read access to keyboard event devices under <code>/dev/input/event*</code>. A common setup is adding the user to the <code>input</code> group, then logging out/in and restarting CrossBoard. This permits system-wide keyboard-event access; grant it only to trusted users.</li>
<li><strong>Windows:</strong> check that security software or policy is not blocking the global keyboard hook; restart CrossBoard after changing permissions.</li>
<li><strong>macOS:</strong> allow CrossBoard or its terminal app under System Settings → Privacy &amp; Security → Accessibility, then restart CrossBoard.</li>
</ul>
<p class="note">On Linux, the listener may report that no keyboard is readable. Check device permissions and restart after granting access. Dashboard keys remain available when the terminal is focused.</p>
</section>
</main></body></html>`
