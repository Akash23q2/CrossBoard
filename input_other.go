//go:build !linux

package main

func pickBackend() inputBackend { return robotgoBackend{} }
