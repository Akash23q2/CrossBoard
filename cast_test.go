package main

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCastScreenOff(t *testing.T) {
	tests := []struct {
		name string
		url  string
		body string
		wantScreenOff bool
		wantIsPotrait bool
	}{
		{name: "unset", url: "/cast/screen"},
		{name: "query enabled", url: "/cast/screen?screen_off=true", wantScreenOff: true},
		{name: "portrait query", url: "/cast/screen?is_potrait=true", wantIsPotrait: true},
		{name: "body options", url: "/cast/screen", body: `{"screen_off":true,"is_potrait":true}`, wantScreenOff: true, wantIsPotrait: true},
		{name: "query overrides body", url: "/cast/screen?screen_off=false&is_potrait=false", body: `{"screen_off":true,"is_potrait":true}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.url, strings.NewReader(test.body))
			gotScreenOff, gotIsPotrait, err := castOptions(request)
			if err != nil {
				t.Fatalf("castOptions() error = %v", err)
			}
			if gotScreenOff != test.wantScreenOff || gotIsPotrait != test.wantIsPotrait {
				t.Errorf("castOptions() = (%t, %t), want (%t, %t)", gotScreenOff, gotIsPotrait, test.wantScreenOff, test.wantIsPotrait)
			}
		})
	}
}

func TestScrcpyArgs(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		screenOff   bool
		isPotrait   bool
		noAudio     bool
		want        []string
	}{
		{
			name:      "screen with screen off",
			screenOff: true,
			want:      []string{"--tcpip=192.168.1.20:2004", "-S"},
		},
		{
			name:        "app without screen off",
			packageName: "com.example.app",
			want: []string{
				"--tcpip=192.168.1.20:2004",
				"--new-display=1920x1080/240",
				"--start-app=com.example.app",
			},
		},
		{
			name:        "portrait app omits landscape resolution",
			packageName: "com.example.app",
			isPotrait:   true,
			want: []string{
				"--tcpip=192.168.1.20:2004",
				"--new-display",
				"--start-app=com.example.app",
			},
		},
		{
			name:        "app with screen off",
			packageName: "com.example.app",
			screenOff:   true,
			noAudio:     true,
			want: []string{
				"--tcpip=192.168.1.20:2004",
				"--new-display=1920x1080/240",
				"--start-app=com.example.app",
				"-S",
				"--no-audio",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := scrcpyArgs("192.168.1.20", test.packageName, test.screenOff, test.isPotrait, test.noAudio)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("scrcpyArgs() = %v, want %v", got, test.want)
			}
		})
	}
}