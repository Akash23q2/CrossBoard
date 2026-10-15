package main

import (
	"fmt"
	"net"
	"sync"
)

var serverURLs = struct {
	sync.RWMutex
	tunnel string
	lan    string
}{}

func setServerURLs(tunnel, lan string) {
	serverURLs.Lock()
	serverURLs.tunnel = tunnel
	serverURLs.lan = lan
	serverURLs.Unlock()
}

func getServerURLs() (string, string) {
	serverURLs.RLock()
	defer serverURLs.RUnlock()
	return serverURLs.tunnel, serverURLs.lan
}

func refreshQRCodes() {
	serverURLs.RLock()
	tunnel, lan := serverURLs.tunnel, serverURLs.lan
	serverURLs.RUnlock()
	if tunnel == "" && lan == "" {
		appLogf("QR refresh requested, but no server URL is available")
		return
	}
	if tunnel != "" {
		appLogf("reprinting tunnel QR code for %s", tunnel)
		generateQr(tunnel)
	}
	if lan != "" {
		appLogf("reprinting local network QR code for %s", lan)
		generateQr(lan)
	}
}

func castActiveAppDevice() error {
	ip, err := activeCastDeviceIP()
	if err != nil {
		return err
	}
	if err := startScrcpy(ip, "", false, false); err != nil {
		return fmt.Errorf("could not start scrcpy for app device %s: %w", ip, err)
	}
	appLogf("started scrcpy for registered app device %s on ADB TCP port %s", ip, scrcpyTCPPort)
	return nil
}

func activeCastDeviceIP() (string, error) {
	device, ok := activeDevice()
	if !ok {
		return "", fmt.Errorf("no app device is registered")
	}
	ip := net.ParseIP(device.IP)
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("registered app device IP %q is not IPv4; scrcpy requires IPv4", device.IP)
	}
	return ip.To4().String(), nil
}
