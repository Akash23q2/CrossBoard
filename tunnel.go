//imports

package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"time"
)

func extractTunnelURL(output string) string {
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	return re.FindString(output)
}

func handleTunneling() (string, string) {
	url := fmt.Sprintf("http://localhost:%d", currentAppConfig().ServerPort)
	lan := ""
	if ip := getLocalIP(); ip != "" {
		lan = fmt.Sprintf("http://%s:%d", ip, currentAppConfig().ServerPort)
	}

	if _, err := exec.LookPath("cloudflared"); err != nil {
		appLogf("cloudflared not found; skipping Cloudflare tunnel, local network access remains available")
		return "", lan
	}
	cmd := exec.Command("cloudflared", "tunnel", "--url", url)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		appLogf("could not capture cloudflared output: %v", err)
		return "", lan
	}

	if err := cmd.Start(); err != nil {
		appLogf("could not start cloudflared: %v", err)
		return "", lan
	}

	tunnelURL := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		foundURL := false
		for scanner.Scan() {
			if match := extractTunnelURL(scanner.Text()); match != "" && !foundURL {
				tunnelURL <- match
				foundURL = true
			}
		}
		if !foundURL {
			tunnelURL <- ""
		}
		if err := cmd.Wait(); err != nil {
			appLogf("cloudflared stopped: %v", err)
		}
	}()

	timer := time.NewTimer(time.Duration(currentAppConfig().TunnelStartupTimeout) * time.Second)
	defer timer.Stop()
	select {
	case url := <-tunnelURL:
		if url == "" {
			return "", lan
		}
		return url, lan
	case <-timer.C:
		if err := cmd.Process.Kill(); err != nil {
			appLogf("could not stop cloudflared after startup timeout: %v", err)
		}
		appLogf("cloudflared tunnel startup timed out")
		return "", lan
	}
}

// getLocalIP prefers common USB tethering IPv4 ranges, then uses the first local IPv4 address.
func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var localIPs []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not ipv4
			}
			localIPs = append(localIPs, ip)
		}
	}
	return selectLocalIP(localIPs)
}

func selectLocalIP(ips []net.IP) string {
	for _, ip := range ips {
		if isUSBTetheringIP(ip) {
			return ip.String()
		}
	}
	if len(ips) > 0 {
		return ips[0].String()
	}
	return ""
}

func isUSBTetheringIP(ip net.IP) bool {
	tetheringRanges := []string{"192.168.42.0/24", "192.168.137.0/24", "172.20.10.0/28"}
	for _, cidr := range tetheringRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
