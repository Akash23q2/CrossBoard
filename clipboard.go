//imports

package main

import (
	"fmt"

	"golang.design/x/clipboard"
)

func readClipboard() (string, error) {
	err := clipboard.Init()
	if err != nil {
		return "", err
	}
	content := string(clipboard.Read(clipboard.FmtText))
	return content, nil
}

func writeClipboard(content string) error {
	err := clipboard.Init()
	if err != nil {
		return err
	}
	if changed := clipboard.Write(clipboard.FmtText, []byte(content)); changed == nil {
		return fmt.Errorf("clipboard write failed")
	}
	return nil
}
