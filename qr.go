// imports
package main

import (
	"bytes"
	"os"

	"github.com/mdp/qrterminal/v3"
)

func generateQr(content string) {
	_, _ = os.Stdout.Write([]byte(renderQr(content)))
}

func renderQr(content string) string {
	var output bytes.Buffer
	qrterminal.GenerateHalfBlock(content, qrterminal.L, &output)
	return output.String()
}
