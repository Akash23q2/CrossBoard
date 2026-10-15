package main

import "testing"

func TestActiveCastDeviceUsesRegisteredAppIP(t *testing.T) {
	uuid, err := newDeviceUUID()
	if err != nil {
		t.Fatal(err)
	}
	device := deviceIdentity{UUID: uuid, IP: "192.0.2.42"}
	deviceIdentities.Lock()
	previousActive := activeDeviceUUID
	deviceIdentities.byUUID[uuid] = device
	activeDeviceUUID = uuid
	deviceIdentities.Unlock()
	t.Cleanup(func() {
		deviceIdentities.Lock()
		delete(deviceIdentities.byUUID, uuid)
		activeDeviceUUID = previousActive
		deviceIdentities.Unlock()
	})

	got, err := activeCastDeviceIP()
	if err != nil {
		t.Fatal(err)
	}
	if got != device.IP {
		t.Fatalf("activeCastDeviceIP() = %q, want registered IP %q", got, device.IP)
	}
}

func TestActiveCastDeviceRequiresRegisteredIPv4(t *testing.T) {
	deviceIdentities.Lock()
	previousActive := activeDeviceUUID
	activeDeviceUUID = ""
	deviceIdentities.Unlock()
	t.Cleanup(func() {
		deviceIdentities.Lock()
		activeDeviceUUID = previousActive
		deviceIdentities.Unlock()
	})

	if _, err := activeCastDeviceIP(); err == nil {
		t.Fatal("activeCastDeviceIP() accepted when no device is registered")
	}
}
