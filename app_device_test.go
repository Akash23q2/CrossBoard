package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectDeviceAssignsUniqueUUIDsForSharedIP(t *testing.T) {
	first := connectTestDevice(t, "")
	second := connectTestDevice(t, "")
	if first.UUID == second.UUID {
		t.Fatal("devices sharing an IP received the same UUID")
	}
	if first.IP != "192.0.2.10" || second.IP != first.IP {
		t.Fatalf("unexpected device IPs: %q, %q", first.IP, second.IP)
	}

	reconnected := connectTestDevice(t, first.UUID)
	if reconnected.UUID != first.UUID {
		t.Fatalf("reconnect UUID = %q, want %q", reconnected.UUID, first.UUID)
	}
}

func TestConnectDeviceRejectsUUIDFromDifferentIP(t *testing.T) {
	identity := connectTestDevice(t, "")
	request := httptest.NewRequest(http.MethodPost, "/device/connect", nil)
	request.RemoteAddr = "192.0.2.11:1234"
	request.Header.Set(isAppHeader, "true")
	request.Header.Set(deviceUUIDHeader, identity.UUID)
	response := httptest.NewRecorder()

	connectDevice(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("connect status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestAppIdentityMiddlewareChecksUUIDAndIP(t *testing.T) {
	uuid, err := newDeviceUUID()
	if err != nil {
		t.Fatal(err)
	}
	identity := deviceIdentity{UUID: uuid, IP: "192.0.2.20"}
	deviceIdentities.Lock()
	deviceIdentities.byUUID[uuid] = identity
	deviceIdentities.Unlock()
	t.Cleanup(func() {
		deviceIdentities.Lock()
		delete(deviceIdentities.byUUID, uuid)
		deviceIdentities.Unlock()
	})

	handler := appIdentityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, ok := identityFromRequest(r); !ok || got != identity {
			t.Errorf("middleware identity = (%v, %t), want (%v, true)", got, ok, identity)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name       string
		ip         string
		uuid       string
		statusCode int
	}{
		{name: "valid identity", ip: "192.0.2.20:1234", uuid: uuid, statusCode: http.StatusNoContent},
		{name: "wrong peer IP", ip: "192.0.2.21:1234", uuid: uuid, statusCode: http.StatusForbidden},
		{name: "app without UUID", ip: "192.0.2.20:1234", statusCode: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			request.RemoteAddr = test.ip
			if test.uuid != "" {
				request.Header.Set(deviceUUIDHeader, test.uuid)
			} else {
				request.Header.Set(isAppHeader, "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.statusCode {
				t.Errorf("status = %d, want %d", response.Code, test.statusCode)
			}
		})
	}
}

func connectTestDevice(t *testing.T, uuid string) deviceIdentity {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/device/connect", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set(isAppHeader, "true")
	if uuid != "" {
		request.Header.Set(deviceUUIDHeader, uuid)
	}
	response := httptest.NewRecorder()
	connectDevice(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("connect status = %d, body = %s", response.Code, response.Body.String())
	}
	var identity deviceIdentity
	if err := json.NewDecoder(response.Body).Decode(&identity); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		deviceIdentities.Lock()
		delete(deviceIdentities.byUUID, identity.UUID)
		deviceIdentities.Unlock()
	})
	return identity
}
