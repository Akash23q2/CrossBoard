//go:build !windows

package main

import "github.com/go-vgo/robotgo"

func scrollHorizontal(delta int) {
	if delta != 0 {
		robotgo.Scroll(delta, 0)
	}
}
