//imports

package main

import (
	"golang.design/x/clipboard"
)

func readClipboard() string {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
	content := string(clipboard.Read(clipboard.FmtText))
	// fmt.Println("read content", content)
	return content
}

func writeClipboard(content string) {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
	clipboard.Write(clipboard.FmtText, []byte(content))
	// fmt.Println("wrote content", content)
}
