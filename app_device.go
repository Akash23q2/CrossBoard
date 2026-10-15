package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

const (
	isAppHeader      = "is_app"
	deviceUUIDHeader = "X-Device-UUID"
)

type deviceIdentity struct {
	UUID string `json:"device_uuid"`
	IP   string `json:"device_ip"`
}

type identityContextKey struct{}

var deviceIdentities = struct {
	sync.RWMutex
	byUUID map[string]deviceIdentity
}{byUUID: make(map[string]deviceIdentity)}

var activeDeviceUUID string

func connectDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isAppRequest(r) {
		http.Error(w, "is_app:true is required", http.StatusBadRequest)
		return
	}
	ip, err := requestIP(r)
	if err != nil {
		http.Error(w, "could not determine device IP", http.StatusBadRequest)
		return
	}

	uuid := requestedDeviceUUID(r)
	var identity deviceIdentity
	if uuid != "" {
		deviceIdentities.RLock()
		identity, _ = deviceIdentities.byUUID[uuid]
		deviceIdentities.RUnlock()
		if identity.UUID == "" || identity.IP != ip {
			http.Error(w, "device UUID is not valid for this IP", http.StatusForbidden)
			return
		}
	} else {
		uuid, err := newDeviceUUID()
		if err != nil {
			http.Error(w, "could not assign device UUID", http.StatusInternalServerError)
			return
		}
		identity = deviceIdentity{UUID: uuid, IP: ip}
	}
	deviceIdentities.Lock()
	deviceIdentities.byUUID[identity.UUID] = identity
	activeDeviceUUID = identity.UUID
	deviceIdentities.Unlock()
	appLogf("app device registered: %s", ip)

	w.Header().Set(deviceUUIDHeader, identity.UUID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(identity)
}

func requestedDeviceUUID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(deviceUUIDHeader))
}

func appIdentityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/connect" {
			next.ServeHTTP(w, r)
			return
		}

		uuid := requestedDeviceUUID(r)
		if uuid == "" {
			if isAppRequest(r) {
				http.Error(w, "device UUID is required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		ip, err := requestIP(r)
		if err != nil {
			http.Error(w, "could not determine device IP", http.StatusBadRequest)
			return
		}
		deviceIdentities.RLock()
		identity, ok := deviceIdentities.byUUID[uuid]
		deviceIdentities.RUnlock()
		if !ok || identity.IP != ip {
			http.Error(w, "device UUID is not valid for this IP", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), identityContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isAppRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get(isAppHeader)), "true") ||
		strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Is-App")), "true")
}

func requestIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("invalid peer address %q", host)
	}
	return ip.String(), nil
}

func activeDevice() (deviceIdentity, bool) {
	deviceIdentities.RLock()
	defer deviceIdentities.RUnlock()
	identity, ok := deviceIdentities.byUUID[activeDeviceUUID]
	return identity, ok
}

func newDeviceUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(uuid[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func identityFromRequest(r *http.Request) (deviceIdentity, bool) {
	identity, ok := r.Context().Value(identityContextKey{}).(deviceIdentity)
	return identity, ok
}
