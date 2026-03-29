//imports

package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
)

const port = 8080

func extractTunnelURL(output string) string {
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	return re.FindString(output)
}

func handleTunneling() string {
	url := fmt.Sprintf("http://localhost:%d", port)
	cmd := exec.Command("cloudflared", "tunnel", "--url", url)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	cmd.Start()

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if match := extractTunnelURL(line); match != "" {
			return match
		}
	}

	return ""
}
