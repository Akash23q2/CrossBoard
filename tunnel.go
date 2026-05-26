//imports

package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"regexp"
)

const port = 8080

func extractTunnelURL(output string) string {
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	return re.FindString(output)
}

func handleTunneling() (string, string) {
	url := fmt.Sprintf("http://localhost:%d", port)
	lan := ""
	if ip := getLocalIP(); ip != "" {
		lan = fmt.Sprintf("http://%s:%d", ip, port)
	}

	cmd := exec.Command("cloudflared", "tunnel", "--url", url)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		return "", lan
	}

	cmd.Start()

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if match := extractTunnelURL(line); match != "" {
			return match, lan
		}
	}

	return "", lan
}

// getLocalIP returns the first non-loopback IPv4 address we can find.
func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
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
			return ip.String()
		}
	}
	return ""
}
