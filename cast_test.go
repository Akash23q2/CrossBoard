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
		want bool
	}{
		{name: "unset", url: "/cast/screen", want: false},
		{name: "query enabled", url: "/cast/screen?screen_off=true", want: true},
		{name: "body enabled", url: "/cast/screen", body: `{"screen_off":true}`, want: true},
		{name: "query overrides body", url: "/cast/screen?screen_off=false", body: `{"screen_off":true}`, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.url, strings.NewReader(test.body))
			got, err := castScreenOff(request)
			if err != nil {
				t.Fatalf("castScreenOff() error = %v", err)
			}
			if got != test.want {
				t.Errorf("castScreenOff() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestScrcpyArgs(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		screenOff   bool
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
			name:        "app with screen off",
			packageName: "com.example.app",
			screenOff:   true,
			want: []string{
				"--tcpip=192.168.1.20:2004",
				"--new-display=1920x1080/240",
				"--start-app=com.example.app",
				"-S",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := scrcpyArgs("192.168.1.20", test.packageName, test.screenOff)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("scrcpyArgs() = %v, want %v", got, test.want)
			}
		})
	}
}